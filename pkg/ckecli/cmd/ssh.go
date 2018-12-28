package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

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

func connectToNode(ctx context.Context, args []string) error {
	nodeName := args[0]
	cluster, err := storage.GetCluster(ctx)
	if err != nil {
		return err
	}
	var node *cke.Node
	for _, n := range cluster.Nodes {
		if n.Hostname == nodeName || n.Address == nodeName {
			node = n
			break
		}
	}
	if node == nil {
		return errors.New("the node is not defined in the cluster: " + nodeName)
	}

	usr, err := user.Current()
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(usr.HomeDir, ".ssh"), 700)
	if err != nil {
		return err
	}
	fifo := filepath.Join(usr.HomeDir, ".ssh", "ckecli-ssh-key-"+strconv.Itoa(os.Getpid()))
	err = syscall.Mkfifo(fifo, 0600)
	if err != nil {
		return err
	}
	defer os.Remove(fifo)

	vc, err := inf.Vault()
	if err != nil {
		return err
	}
	secret, err := vc.Logical().Read(cke.SSHSecret)
	if err != nil {
		return err
	}
	if secret == nil {
		return errors.New("no ssh private keys")
	}
	privKeys := secret.Data

	mykey, ok := privKeys[node.Address]
	if !ok {
		mykey = privKeys[""]
	}
	if mykey == nil {
		return errors.New("no ssh private key for " + nodeName)
	}

	go func() {
		writeToFifo(fifo, mykey.(string))
		time.Sleep(100 * time.Millisecond)
		// OpenSSH reads the private key file twice, it need to write key twice.
		writeToFifo(fifo, mykey.(string))
	}()

	sshArgs := []string{
		"-i", fifo,
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		node.User + "@" + node.Address,
	}
	sshArgs = append(sshArgs, args[1:]...)
	c := exec.Command("ssh", sshArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// leaderCmd represents the leader command
var sshCmd = &cobra.Command{
	Use:   "ssh NODE [COMMAND...]",
	Short: "connect to the node via ssh",
	Long: `connect to the node via ssh.

NODE is IP address or hostname of the node to be connected.

If COMMAND is specified, it will be executed on the node.
`,

	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return connectToNode(ctx, args)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
}
