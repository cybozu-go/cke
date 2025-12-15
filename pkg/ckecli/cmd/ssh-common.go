package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	usr "os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type SshConfig struct {
	ConnectionTimeout     time.Duration
	SessionTimeout        time.Duration
	KeepAliveInterval     time.Duration
	KeepAliveReplyTimeout time.Duration
}

func createSignerFromKeyString(privateKey string) (ssh.Signer, error) {
	keyBytes := []byte(privateKey)
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", err)
	}
	return signer, nil
}

// Detects changes in the client terminal size and notifies the server.
func handleWindowChanges(session *ssh.Session) {
	// Get the file descriptor (FD) of the terminal
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return // Exit if not in a terminal environment
	}

	// Initially get and set the current size
	width, height, err := term.GetSize(fd)
	if err == nil {
		session.WindowChange(height, width)
	}

	// Channel to capture SIGWINCH signal (window size change)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)

	// Goroutine to wait for signals
	go func() {
		for range sigCh {
			w, h, err := term.GetSize(fd)
			if err != nil {
				log.Error("failed to get window size", map[string]interface{}{
					log.FnError: err,
				})
				continue
			}
			// Notify the server via SSH session if the size has changed
			if w != width || h != height {
				width, height = w, h
				if err := session.WindowChange(height, width); err != nil {
					log.Error("window size change notification error", map[string]interface{}{
						log.FnError: err,
					})
				}
			}
		}
	}()
}

// returns user, host, command, error
func ParseSSHArgs(args []string) (string, string, string, error) {
	if len(args) < 1 {
		return "", "", "", fmt.Errorf("not enough arguments")
	}

	// Parse user and host
	remoteHost := args[0]
	var user, host string
	if atIndex := bytes.IndexByte([]byte(remoteHost), '@'); atIndex != -1 {
		user = remoteHost[:atIndex]
		host = remoteHost[atIndex+1:]
	} else {
		currentUser, err := usr.Current()
		if err != nil {
			log.Error("failed to get current user information", map[string]interface{}{
				log.FnError: nil,
			})
			return "", "", "", err
		}
		user = currentUser.Username // Default user name
		host = remoteHost
	}
	if colonIndex := bytes.IndexByte([]byte(host), ':'); colonIndex == -1 {
		host = host + ":22" // Default port number
	}

	command := ""
	for i, arg := range args {
		if i < 1 {
			continue
		}
		if i > 1 {
			command += " "
		}
		command += arg
	}
	return user, host, command, nil
}

func (c *SshConfig) keepAlive(ctx context.Context, conn ssh.Conn) {
	if c.KeepAliveInterval == 0 {
		return
	}
	t := time.NewTicker(c.KeepAliveInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			// Prepare a channel for executing SendRequest asynchronously
			replyChan := make(chan error, 1)

			go func() {
				// keepalive request and then expect wantReply: true
				_, _, err := conn.SendRequest("keepalive@openssh.com", true, nil)
				replyChan <- err
			}()

			select {
			case err := <-replyChan:
				// The response was returned before the timeout.
				if err != nil {
					// Failed to send or error response
					log.Error("KeepAlive request/reply error, closing connection", map[string]interface{}{
						log.FnError: err,
					})
					conn.Close()
					return
				}
				// Successfully sent and received reply

			case <-time.After(c.KeepAliveReplyTimeout):
				// Timeout! No response from the server.
				log.Error("KeepAlive reply timeout Server is unresponsive, closing connection", map[string]interface{}{
					log.FnError: nil,
					"timeout":   c.KeepAliveReplyTimeout,
				})

				conn.Close() // Close the entire connection.
				return       // End the goroutine

			case <-ctx.Done():
				// Context cancellation (external factor)
				return
			}

		case <-ctx.Done():
			// The connection was canceled externally.
			return
		}
	}
}

func loadPrivateKey(nodeName string) (string, error) {
	log.Info("loadPrivateKey", map[string]interface{}{
		log.FnError: nil,
		"nodeName":  nodeName,
	})

	vc, err := inf.Vault()
	if err != nil {
		return "", err
	}
	secret, err := vc.Logical().Read(cke.SSHSecret)
	if err != nil {
		return "", err
	}
	if secret == nil {
		return "", errors.New("no ssh private keys")

	}
	privKeys := secret.Data

	mykey, ok := privKeys[nodeName]
	if !ok {
		mykey = privKeys[""]
	}
	if mykey == nil {
		log.Error("no ssh private key", map[string]interface{}{
			log.FnError: nil,
			"nodeName":  nodeName,
		})
		return "", errors.New("no ssh private key for " + nodeName)
	}

	return mykey.(string), nil
}

func ToCurrentDirAbs(path string) (string, error) {
	return filepath.Abs(path)
}

// returns 0:user, 1:host, 2:localPath, 3:remoteFile, 4:direction, error
func ParseSCPArgs(args []string) (string, string, string, string, string, error) {
	if len(args) < 2 {
		return "", "", "", "", "", fmt.Errorf("not enough arguments")
	}
	src := args[0]
	dst := args[1]

	// Remote format detection (user@host:path or host:path)
	isRemote := func(s string) bool {
		return strings.Contains(s, ":") || (strings.Contains(s, "@"))
	}

	var direction, user, host string
	var localPath, remoteSpec string

	switch {
	case isRemote(src) && !isRemote(dst):
		// Receive: remote -> local
		direction = "Receive"
		remoteSpec = src
		localPath = dst

	case !isRemote(src) && isRemote(dst):
		// Send: local -> remote
		direction = "Send"
		var err error
		localPath, err = ToCurrentDirAbs(src)
		if err != nil {
			return "", "", "", "", "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		remoteSpec = dst

	default:
		return "", "", "", "", "", fmt.Errorf("invalid scp arguments")
	}

	// Parse remoteSpec
	// user@host:/path/to/file or host:/path/to/file
	atIndex := strings.Index(remoteSpec, "@")
	currentUser, err := usr.Current()
	if err != nil {
		log.Error("failed to get current user information", map[string]interface{}{
			log.FnError: nil,
		})
		return "", "", "", "", "", err
	}
	if atIndex == -1 {
		user = currentUser.Username
	} else {
		user = remoteSpec[:atIndex]
	}

	colonIndex := strings.Index(remoteSpec, ":")
	if colonIndex == -1 {
		return "", "", "", "", "", fmt.Errorf("invalid remote spec format")
	}

	if atIndex != -1 {
		host = remoteSpec[atIndex+1 : colonIndex]
	} else {
		host = remoteSpec[:colonIndex]
	}

	remoteFile := remoteSpec[colonIndex+1:]

	if (strings.LastIndex(localPath, "/") + 1) == len(localPath) {
		localPath += filepath.Base(remoteFile)
	}

	// Add default port 22 if host does not contain a port
	if !strings.Contains(host, ":") {
		host = host + ":22"
	}

	return user, host, localPath, remoteFile, direction, nil
}
