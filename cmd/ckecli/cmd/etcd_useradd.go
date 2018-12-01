package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// etcdUserAddCmd represents the "etcd user-add" command
var etcdUserAddCmd = &cobra.Command{
	Use:   "user-add NAME PREFIX",
	Short: "add a user to CKE managed etcd",
	Long: `Add a user to etcd managed by CKE (not the one used by CKE).

NAME must not be "root" or "backup".
PREFIX limits the user's priviledge to keys having the prefix.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("wrong number of arguments")
		}

		switch args[0] {
		case "", "root", "backup":
			return errors.New("bad etcd username: " + args[0])
		}

		if args[1] == "" {
			return errors.New("bad etcd prefix: " + args[1])
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		prefix := args[1]

		well.Go(func(ctx context.Context) error {
			etcd, err := inf.NewEtcdClient(ctx, nil)
			if err != nil {
				return err
			}
			return cke.AddUserRole(ctx, etcd, username, prefix)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	etcdCmd.AddCommand(etcdUserAddCmd)
}
