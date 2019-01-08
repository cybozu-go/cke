package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

func detectNode(ctx context.Context, args []string) (*cke.Node, error) {
	var nodeName string
	for _, arg := range args {
		if strings.Contains(arg, ":") {
			nodeName = arg[:strings.Index(arg, ":")]
			break
		}
	}

	if len(nodeName) == 0 {
		return nil, errors.New("node name is not specified")
	}

	cluster, err := storage.GetCluster(ctx)
	if err != nil {
		return nil, err
	}
	var node *cke.Node
	for _, n := range cluster.Nodes {
		if n.Hostname == nodeName || n.Address == nodeName {
			node = n
			break
		}
	}
	if node == nil {
		return nil, errors.New("the node is not defined in the cluster: " + nodeName)
	}
	return node, nil
}

func replaceHostName(node *cke.Node, args []string) []string {
	replaced := make([]string, len(args))
	for i, arg := range args {
		if strings.Contains(arg, ":") {
			replaced[i] = node.Address + ":" + arg[strings.Index(arg, ":")+1:]
		} else {
			replaced[i] = arg
		}
	}

	return replaced
}

func scp(ctx context.Context, args []string) error {
	node, err := detectNode(ctx, args)
	if err != nil {
		return err
	}
	fifo, err := sshPrivateKey(node)
	if err != nil {
		return err
	}
	defer os.Remove(fifo)

	scpArgs := []string{
		"-i", fifo,
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		"-o", "User=" + node.User,
	}
	if scpParams.recursive {
		scpArgs = append(scpArgs, "-r")
	}

	args = replaceHostName(node, args)
	scpArgs = append(scpArgs, args...)

	fmt.Println(scpArgs)
	c := exec.Command("scp", scpArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

var scpParams struct {
	recursive bool
}

// scpCmd represents the scp command
var scpCmd = &cobra.Command{
	Use:   "scp [NODE1:]FILE1 ... [NODE2:]FILE2",
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
