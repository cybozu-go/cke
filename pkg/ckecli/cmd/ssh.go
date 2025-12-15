package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func (c *SshConfig) Ssh(ctx context.Context, args []string) error {
	terminaMode := false

	user, remoteHost, command, err := ParseSSHArgs(args)
	if err != nil {
		log.Error("failed to parse SSH args", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	if len(command) == 0 {
		terminaMode = true
	}

	privateKey, err := loadPrivateKey(remoteHost)
	if err != nil {
		log.Error("failed to load private key", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	signer, err := createSignerFromKeyString(privateKey)
	if err != nil {
		log.Error("failed to convert Signer from private key", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	// SSH client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Create a TCP connection with a timeout
	dialer := net.Dialer{Timeout: c.ConnectionTimeout}
	rawConn, err := dialer.DialContext(ctx, "tcp", remoteHost)

	if err != nil {
		log.Error("failed to connect", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	defer rawConn.Close()

	// Perform SSH handshake over the existing TCP connection
	sshConn, chans, reqs, err := ssh.NewClientConn(rawConn, remoteHost, config)
	if err != nil {
		log.Error("failed to establish SSH connection", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	// Start keep-alive goroutine
	done := make(chan struct{})
	defer close(done)

	// Start keep-alive goroutine
	go c.keepAlive(ctx, sshConn)

	// Create a new ssh session
	session, err := client.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create session: %v\n", err)
		return err
	}
	defer session.Close()

	// Goroutine to monitor context cancellation
	go func() {
		select {
		case <-ctx.Done():
			// If the context is canceled, force the connection to close, and clean up this goroutine and exit.
			sshConn.Close()
			return
		case <-done:
			// If the SSH operation completes successfully, clean up this goroutine and exit.
			return
		}
	}()

	if terminaMode {
		// Launching the interactive shell
		// Set the terminal to Raw mode and save the original state.
		fd := int(os.Stdin.Fd())
		if !term.IsTerminal(fd) {
			return fmt.Errorf("standard input is not a terminal")
		}
		// Set the terminal to Raw mode (enabling input control with vi, etc.)
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return fmt.Errorf("failed to set terminal to Raw mode: %v", err)
		}
		// Restore the original state when the main function exits
		defer term.Restore(fd, oldState)

		// Retrieve the terminal size and use it for Pty requests
		// Default values
		initialWidth, initialHeight := 80, 24

		// Get the current terminal size
		w, h, err := term.GetSize(fd)
		if err == nil && w > 0 && h > 0 {
			initialWidth = w
			initialHeight = h
		} else {
			log.Error("failed to get terminal size, Using default values", map[string]interface{}{
				log.FnError: err,
			})
		}

		// Start asynchronous detection and notification of window size changes
		handleWindowChanges(session)

		// Connect standard input/output to the session
		session.Stdout = os.Stdout
		session.Stderr = os.Stderr
		stdinPipe, err := session.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to get Stdin pipe: %v", err)
		}

		// Copy user input (os.Stdin) to the session's input pipe
		go func() {
			defer stdinPipe.Close()
			if _, err := io.Copy(stdinPipe, os.Stdin); err != nil && err != io.EOF {
				log.Error("error copying user input", map[string]interface{}{
					log.FnError: err,
				})
			}
		}()

		// Request a pseudo-terminal (Pty)
		// A Pty is required for interactive shell operations.
		modes := ssh.TerminalModes{
			ssh.ECHO:          1,     // Echo is handled by the client, set to 1 for vi, etc. (adjust as needed)
			ssh.ISIG:          1,     // Enable signal processing like Ctrl+C
			ssh.ICANON:        1,     // Enable line input mode (adjusted by shell when using vi)
			ssh.TTY_OP_ISPEED: 14400, // Input speed
			ssh.TTY_OP_OSPEED: 14400, // Output speed
		}

		// Request a Pty with the retrieved size
		if err := session.RequestPty("xterm-256color", initialHeight, initialWidth, modes); err != nil {
			return fmt.Errorf("failed to request pseudo-terminal: %v", err)
		}

		// Start the shell
		if err := session.Shell(); err != nil {
			return fmt.Errorf("failed to start shell: %v", err)
		}

		// Wait for the session to end
		// This method returns when the user exits the shell with 'exit' or Ctrl+D.
		if err := session.Wait(); err != nil {
			// Wait() may return an error based on the shell's exit status.
			if _, ok := err.(*ssh.ExitError); !ok {
				log.Error("session exit error", map[string]interface{}{
					log.FnError: err,
				})
			}
		}
	} else {
		// Set a read/write deadline for the underlying connection
		rawConn.SetDeadline(time.Now().Add(c.SessionTimeout))

		// Run a remote command
		var bufStdOut bytes.Buffer
		var bufStdErr bytes.Buffer
		session.Stdout = &bufStdOut
		session.Stderr = &bufStdErr

		if err := session.Run(command); err != nil {
			if err == io.EOF {
				fmt.Println("Session timed out or closed.")
			} else {
				log.Error("command error", map[string]interface{}{
					log.FnError: err,
				})
			}
		}

		// Print command output
		fmt.Println("COMMAND:", command)
		fmt.Println("STDOUT:", session.Stdout)
		fmt.Println("STDERR:", session.Stderr)
	}

	return nil
}

// sshCmd represents the ssh command
var sshCmd = &cobra.Command{
	Use:   "ssh [user@]NODE [COMMAND...]",
	Short: "connect to the node via ssh",
	Long: `Connect to the node via ssh.

NODE is IP address or hostname of the node to be connected.

If COMMAND is specified, it will be executed on the node.
`,

	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			s := &SshConfig{
				ConnectionTimeout:     30 * time.Second,
				SessionTimeout:        60 * time.Second, // Intentionally short timeout
				KeepAliveInterval:     60 * time.Second,
				KeepAliveReplyTimeout: 10 * time.Second, // Time out of keep-alive reply from client to server
			}
			return s.Ssh(ctx, args)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
}
