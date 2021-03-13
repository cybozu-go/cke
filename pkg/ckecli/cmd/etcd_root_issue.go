package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var etcdRootIssueOpts struct {
	Output string
}

// etcdRootIssueCmd represents the "etcd issue" command
var etcdRootIssueCmd = &cobra.Command{
	Use:   "root-issue",
	Short: "issue client certificates for etcd root",
	Long:  `Issue TLS client certificates for etcd root user.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		outputJSON := false
		switch etcdRootIssueOpts.Output {
		case "json":
			outputJSON = true
		case "file":
		default:
			return errors.New("invalid option: output=" + etcdRootIssueOpts.Output)
		}

		well.Go(func(ctx context.Context) error {
			cert, key, err := cke.EtcdCA{}.IssueRoot(ctx, inf)
			if err != nil {
				return err
			}

			cacert, err := storage.GetCACertificate(ctx, cke.CAServer)
			if err != nil {
				return err
			}

			if outputJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(cke.IssueResponse{
					Cert:   cert,
					Key:    key,
					CACert: cacert,
				})
			}

			cacertFile := "etcd-ca.crt"
			certFile := "etcd-root.crt"
			keyFile := "etcd-root.key"
			err = os.WriteFile(cacertFile, []byte(cacert), 0644)
			if err != nil {
				return err
			}
			err = os.WriteFile(certFile, []byte(cert), 0644)
			if err != nil {
				return err
			}
			err = os.WriteFile(keyFile, []byte(key), 0600)
			if err != nil {
				return err
			}
			fmt.Println("cert files: ", cacertFile, certFile, keyFile)
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	fs := etcdRootIssueCmd.Flags()
	fs.StringVar(&etcdRootIssueOpts.Output, "output", "json", `output format ("json" or "file")`)
	etcdCmd.AddCommand(etcdRootIssueCmd)
}
