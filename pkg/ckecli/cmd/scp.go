package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/cybozu-go/log"
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

func scpSubMain(ctx context.Context, args []string) error {
	pipeFilename, err := createFifo()
	if err != nil {
		log.Error("failed to create named pipe", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	defer os.Remove(pipeFilename)

	node, err := detectSCPNode(args)
	if err != nil {
		log.Error("failed to find the node name for scp", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	pirvateKey, err := getPrivateKey(node)
	if err != nil {
		log.Error("failed to get the private key for scp", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	go func() {
		if _, err := startSshAgent(ctx, pipeFilename); err != nil {
			log.Error("failed to start ssh-agent for scp", map[string]interface{}{
				log.FnError: err,
				"node":      node,
			})
		}
	}()
	defer killSshAgent(ctx)

	if err = writeToFifo(pipeFilename, pirvateKey); err != nil {
		log.Error("failed to write the named pipe", map[string]interface{}{
			log.FnError: err,
			"pipe":      pipeFilename,
		})
		return err
	}

	scpArgs := []string{
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=60",
	}
	if scpParams.recursive {
		scpArgs = append(scpArgs, "-r")
	}

	scpArgs = append(scpArgs, args...)
	c := exec.CommandContext(ctx, "scp", scpArgs...)
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
	Use:   "scp [[user@]NODE1:]FILE1 ... [[user@]NODE2:]FILE2",
	Short: "copy files between hosts via scp",
	Long: `Copy files between hosts via scp.

NODE is IP address or hostname of the node.
`,

	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return scpSubMain(ctx, args)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	scpCmd.Flags().BoolVarP(&scpParams.recursive, "", "r", false, "recursively copy entire directories")
	rootCmd.AddCommand(scpCmd)
}
