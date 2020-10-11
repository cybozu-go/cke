package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var etcdIssueOpts struct {
	TTL    string
	Output string
}

// etcdIssueCmd represents the "etcd issue" command
var etcdIssueCmd = &cobra.Command{
	Use:   "issue NAME",
	Short: "issue client certificates for a user NAME",
	Long: `Issue TLS client certificates for etcd user authentication.

NAME is the username of etcd user to be authenticated.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		if len(username) == 0 {
			return errors.New("username is empty")
		}

		outputJSON := false
		switch etcdIssueOpts.Output {
		case "json":
			outputJSON = true
		case "file":
		default:
			return errors.New("invalid option: output=" + etcdIssueOpts.Output)
		}

		well.Go(func(ctx context.Context) error {
			cert, key, err := cke.IssueEtcdClientCertificate(inf, username, etcdIssueOpts.TTL)
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
			certFile := fmt.Sprintf("etcd-%s.crt", username)
			keyFile := fmt.Sprintf("etcd-%s.key", username)
			err = ioutil.WriteFile(cacertFile, []byte(cacert), 0644)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(certFile, []byte(cert), 0644)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(keyFile, []byte(key), 0600)
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
	fs := etcdIssueCmd.Flags()
	fs.StringVar(&etcdIssueOpts.TTL, "ttl", "87600h", "TTL of the certificate")
	fs.StringVar(&etcdIssueOpts.Output, "output", "json", `output format ("json" or "file")`)
	etcdCmd.AddCommand(etcdIssueCmd)
}
