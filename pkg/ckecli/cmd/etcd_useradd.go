package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/cybozu-go/cke"
)

// etcdUserAddCmd represents the "etcd user-add" command
var etcdUserAddCmd = &cobra.Command{
	Use:   "user-add NAME PREFIX",
	Short: "add a user to CKE managed etcd",
	Long: `Add a user to etcd managed by CKE (not the one used by CKE).

NAME must not be "root" or "backup".
PREFIX limits the user's privilege to keys having the prefix.`,

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
		ctx := cmd.Context()

		username := args[0]
		prefix := args[1]

		etcd, err := inf.NewEtcdClient(ctx, nil)
		if err != nil {
			return err
		}
		return cke.AddUserRole(ctx, etcd, username, prefix)
	},
}

func init() {
	etcdCmd.AddCommand(etcdUserAddCmd)
}
