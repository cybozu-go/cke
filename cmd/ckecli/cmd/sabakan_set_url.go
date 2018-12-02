package cmd

import (
	"context"
	"errors"
	"net/url"
	"path"
	"strings"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// sabakanSetURLCmd represents the "sabakan set-url" command
var sabakanSetURLCmd = &cobra.Command{
	Use:   "set-url URL",
	Short: "set URL of sabakan server",
	Long:  `Set URL of sabakan server and enable sabakan integration.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		u, err := url.Parse(args[0])
		if err != nil {
			return err
		}

		if !u.IsAbs() {
			return errors.New("invalid URL")
		}

		if strings.HasSuffix(u.Path, "/graphql") {
			u.Path = path.Join(u.Path, "/graphql")
		}

		well.Go(func(ctx context.Context) error {
			return storage.SetSabakanURL(ctx, u.String())
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanSetURLCmd)
}
