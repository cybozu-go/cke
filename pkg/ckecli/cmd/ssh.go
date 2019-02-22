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

func writeToFifo(fifo string, data string) {
	f, err := os.OpenFile(fifo, os.O_WRONLY, 0600)
	if err != nil {
		log.Error("failed to open fifo", map[string]interface{}{
			log.FnError: err,
			"fifo":      fifo,
		})
		return
	}
	defer f.Close()

	_, err = f.WriteString(data)
	if err != nil {
		log.Error("failed to write to fifo", map[string]interface{}{
			log.FnError: err,
			"fifo":      fifo,
		})
	}
}

func sshPrivateKey(nodeName string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(filepath.Join(usr.HomeDir, ".ssh"), 0700)
	if err != nil {
		return "", err
	}
	fifo := filepath.Join(usr.HomeDir, ".ssh", "ckecli-ssh-key-"+strconv.Itoa(os.Getpid()))
	err = syscall.Mkfifo(fifo, 0600)
	if err != nil {
		return "", err
	}

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

	go func() {
		writeToFifo(fifo, mykey.(string))
		time.Sleep(100 * time.Millisecond)
		// OpenSSH reads the private key file twice, it need to write key twice.
		writeToFifo(fifo, mykey.(string))
	}()

	return fifo, nil
}

func ssh(ctx context.Context, args []string) error {
	node := detectSSHNode(args[0])
	fifo, err := sshPrivateKey(node)
	if err != nil {
		return err
	}
	defer os.Remove(fifo)

	sshArgs := []string{
		"-i", fifo,
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
			return ssh(ctx, args)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
}
