package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cybozu-go/log"
	"github.com/spf13/cobra"

	"github.com/cybozu-go/cke"
)

func detectSSHNode(arg string) string {
	nodeName := arg
	if strings.Contains(arg, "@") {
		nodeName = arg[strings.Index(arg, "@")+1:]
	}
	return nodeName
}

func createFifo() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	fifoFilePath := filepath.Join(usr.HomeDir, ".ssh", "ckecli-ssh-key-"+strconv.Itoa(os.Getpid()))
	_, err = os.Stat(fifoFilePath)
	if os.IsExist(err) {
		return fifoFilePath, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	err = os.MkdirAll(filepath.Join(usr.HomeDir, ".ssh"), 0o700)
	if err != nil {
		return "", err
	}

	err = syscall.Mkfifo(fifoFilePath, 0o600)
	if err != nil {
		return "", err
	}

	return fifoFilePath, err
}

func getPrivateKey(ctx context.Context, nodeName string) (string, error) {
	vc, err := inf.Vault()
	if err != nil {
		return "", err
	}

	secret, err := vc.Logical().ReadWithContext(ctx, cke.SSHSecret)
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
		return "", errors.New("no ssh private key for " + nodeName)
	}

	return mykey.(string), nil
}

func startSshAgent(ctx context.Context, privateKeyFile string) (map[string]string, error) {
	myEnv := make(map[string]string)

	cmd := exec.CommandContext(ctx, "ssh-agent", "-s")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// Set enviromental variable to communicate ssh-agent
	line := strings.Split(string(stdoutStderr), "\n")
	partOfLine := strings.Split(line[0], ";")
	kvPair1 := strings.Split(partOfLine[0], "=")
	myEnv[kvPair1[0]] = kvPair1[1]
	err = os.Setenv(kvPair1[0], kvPair1[1])
	if err != nil {
		log.Error("failed to set environment variable 1", map[string]any{
			log.FnError: err,
			"env":       kvPair1[0],
			"val":       kvPair1[1],
		})
		return nil, err
	}
	partOfLine = strings.Split(line[1], ";")
	kvPair2 := strings.Split(partOfLine[0], "=")
	myEnv[kvPair2[0]] = kvPair2[1]
	err = os.Setenv(kvPair2[0], kvPair2[1])
	if err != nil {
		log.Error("failed to set environment variable 2", map[string]any{
			log.FnError: err,
			"env":       kvPair2[0],
			"val":       kvPair2[1],
		})
		return nil, err
	}

	cmd = exec.CommandContext(ctx, "ssh-add", privateKeyFile)
	_, err = cmd.CombinedOutput()
	if err != nil {
		log.Error("failed to add the private key", map[string]any{
			log.FnError: err,
		})
		return nil, err
	}
	log.Debug("Successfuly added the private key", map[string]any{
		"env": kvPair1[0],
		"val": kvPair1[1],
	})

	return myEnv, nil
}

func killSshAgent() {
	// Use an independent context so cleanup still runs even when the
	// caller's context is already canceled (e.g. by a signal).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh-agent", "-k")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("failed to run ssh-agent -k", map[string]any{
			log.FnError: err,
		})
		return
	}
	log.Debug("killed ssh-agent", map[string]any{
		"stdout_stderr": string(stdoutStderr),
	})
}

func writeToFifo(ctx context.Context, fifo string, data string) error {
	// Opening a FIFO for writing blocks until a reader appears, so do it in
	// a goroutine and let this function return as soon as ctx is done.
	type result struct {
		f   *os.File
		err error
	}
	openDone := make(chan result, 1)
	go func() {
		f, err := os.OpenFile(fifo, os.O_WRONLY|os.O_APPEND, os.ModeNamedPipe)
		openDone <- result{f, err}
	}()

	var f *os.File
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r := <-openDone:
		if r.err != nil {
			return r.err
		}
		f = r.f
	}
	defer f.Close()

	if _, err := f.Write([]byte(data)); err != nil {
		return err
	}
	return nil
}

func sshSubMain(ctx context.Context, args []string) error {
	pipeFilename, err := createFifo()
	if err != nil {
		log.Error("failed to create the named pipe", map[string]any{
			log.FnError: err,
		})
		return err
	}
	defer os.Remove(pipeFilename)

	node := detectSSHNode(args[0])
	privateKey, err := getPrivateKey(ctx, node)
	if err != nil {
		log.Error("failed to get the private key for ssh", map[string]any{
			log.FnError: err,
		})
		return err
	}

	go func() {
		if _, err := startSshAgent(ctx, pipeFilename); err != nil {
			log.Error("failed to start ssh-agent for ssh", map[string]any{
				log.FnError: err,
				"node":      node,
			})
		}
	}()
	defer killSshAgent()

	if err = writeToFifo(ctx, pipeFilename, privateKey); err != nil {
		log.Error("failed to write the named pipe", map[string]any{
			log.FnError: err,
			"pipe":      pipeFilename,
		})
		return err
	}

	sshArgs := []string{
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=60",
	}
	sshArgs = append(sshArgs, args...)
	c := exec.CommandContext(ctx, "ssh", sshArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
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
		return sshSubMain(cmd.Context(), args)
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
}
