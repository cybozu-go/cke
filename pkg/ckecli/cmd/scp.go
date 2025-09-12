package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

func detectSCPNode(args []string) (string, error) {
	var nodeName string
	for _, arg := range args {
		if strings.Contains(arg, ":") {
			nodeName = detectSSHNode(arg[:strings.Index(arg, ":")])
			break
		}
	}

	if len(nodeName) == 0 {
		return "", errors.New("node name is not specified")
	}

	return nodeName, nil
}

func scp(ctx context.Context, args []string) error {
	node, err := detectSCPNode(args)
	if err != nil {
		return err
	}
	fifo, err := createFifo()
	if err != nil {
		return err
	}

	defer os.Remove(fifo)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() error {
		defer wg.Done()
		scpArgs := []string{
			"-i", fifo,
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "StrictHostKeyChecking=no",
			"-o", "ConnectTimeout=60",
		}
		if scpParams.recursive {
			scpArgs = append(scpArgs, "-r")
		}

		scpArgs = append(scpArgs, args...)

		fmt.Println(scpArgs)
		c := exec.CommandContext(ctx, "scp", scpArgs...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}()

	err = sshPrivateKey(node, fifo)
	if err != nil {
		return err
	}
	wg.Wait()
	return nil
}

var scpParams struct {
	recursive bool
}

// scpCmd represents the scp command
var scpCmd = &cobra.Command{
	Use:   "scp [[user@]NODE1:]FILE1 ... [[user@]NODE2:]FILE2",
	Short: "copy files between hosts via scp",
	Long: `Copy files between hosts via scp.

NODE is IP address or hostname of the node.
`,

	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return scp(ctx, args)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	scpCmd.Flags().BoolVarP(&scpParams.recursive, "", "r", false, "recursively copy entire directories")
	rootCmd.AddCommand(scpCmd)
}
