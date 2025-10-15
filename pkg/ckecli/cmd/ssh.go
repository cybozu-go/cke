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

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
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

	err = os.MkdirAll(filepath.Join(usr.HomeDir, ".ssh"), 0700)
	if err != nil {
		return "", err
	}

	err = syscall.Mkfifo(fifoFilePath, 0600)
	if err != nil {
		return "", err
	}

	return fifoFilePath, err
}

func getPrivateKey(nodeName string) (string, error) {
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
		return "", errors.New("no ssh private key for " + nodeName)
	}

	return mykey.(string), nil
}

func sshAgent(ctx context.Context, privateKeyFile string) (map[string]string, error) {
	myEnv := make(map[string]string)
	sshArgs := []string{
		"-s",
	}

	cmd := exec.CommandContext(ctx, "ssh-agent", sshArgs...)
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
		log.Error("failed to set environment variable 1", map[string]interface{}{
			log.FnError: err,
			"Env":       kvPair1[0],
			"Val":       kvPair1[1],
		})
		return nil, err
	}
	partOfLine = strings.Split(line[1], ";")
	kvPair2 := strings.Split(partOfLine[0], "=")
	myEnv[kvPair2[0]] = kvPair2[1]
	err = os.Setenv(kvPair2[0], kvPair2[1])
	if err != nil {
		log.Error("failed to set environment variable 2", map[string]interface{}{
			log.FnError: err,
			"Env":       kvPair2[0],
			"Val":       kvPair2[1],
		})
		return nil, err
	}

	sshArgs2 := []string{
		privateKeyFile,
	}
	cmd0 := exec.CommandContext(ctx, "ssh-add", sshArgs2...)
	_, err = cmd0.CombinedOutput()
	if err != nil {
		log.Error("failed to add the private key", map[string]interface{}{
			log.FnError: err,
		})
		return nil, err
	}
	log.Info("Successfuly added the private key", map[string]interface{}{
		"Env": kvPair1[0],
		"Val": kvPair1[1],
	})

	return myEnv, nil
}

func killSshAgent(ctx context.Context) error {
	sshArgs := []string{
		"-k",
	}
	cmd := exec.CommandContext(ctx, "ssh-agent", sshArgs...)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("failed to run ssh-agent -k", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	log.Info("killed ssh-agent", map[string]interface{}{
		"message": string(stdoutStderr),
	})
	return nil
}

func writeToFifo(fifo string, data string) error {
	f, err := os.OpenFile(fifo, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModeNamedPipe)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write([]byte(data)); err != nil {
		return err
	}
	return nil
}

func sshSubMain(ctx context.Context, args []string) error {
	pipeFilename, err := createFifo()
	if err != nil {
		log.Error("failed to create the named pipe", map[string]interface{}{
			log.FnError: err,
			"fifo name": pipeFilename,
		})
		return err
	}

	node := detectSSHNode(args[0])
	pirvateKey, err := getPrivateKey(node)
	if err != nil {
		log.Error("failed to get the private key for ssh", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	go func() {
		if _, err := sshAgent(ctx, pipeFilename); err != nil {
			log.Error("failed to start ssh-agent for ssh", map[string]interface{}{
				log.FnError: err,
				"node name": node,
			})
		}
	}()

	if err = writeToFifo(pipeFilename, pirvateKey); err != nil {
		log.Error("failed to write the named pipe", map[string]interface{}{
			log.FnError:  err,
			"named pipe": pipeFilename,
		})
		return err
	}
	defer os.Remove(pipeFilename)
	defer killSshAgent(ctx)

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
		well.Go(func(ctx context.Context) error {
			return sshSubMain(ctx, args)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
}
