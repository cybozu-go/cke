package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var kubernetesIssueOpts struct {
	TTL string
}

// kubernetesIssueCmd represents the "kubernetes issue" command
var kubernetesIssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "issue client certificates for k8s admin",
	Long:  `Issue TLS client certificates for k8s admin user.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			cluster, err := storage.GetCluster(ctx)
			if err != nil {
				return err
			}
			cpNodes := cke.ControlPlanes(cluster.Nodes)
			if len(cpNodes) == 0 {
				return errors.New("no control plane")
			}

			// TODO: Replace `server` from node IP to Ingress address.
			// Since there is no Ingress yet, use the node IP.
			server := "https://" + cpNodes[0].Address + ":6443"

			cacert, err := storage.GetCACertificate(ctx, "kubernetes")
			if err != nil {
				return err
			}
			cert, key, err := cke.KubernetesCA{}.IssueAdminCert(ctx, inf, kubernetesIssueOpts.TTL)
			if err != nil {
				return err
			}

			cfg := cke.AdminKubeconfig(cluster.Name, cacert, cert, key, server)
			src, err := clientcmd.Write(*cfg)
			if err != nil {
				return err
			}
			_, err = fmt.Println(string(src))
			return err
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	fs := kubernetesIssueCmd.Flags()
	fs.StringVar(&kubernetesIssueOpts.TTL, "ttl", "2h", "TTL of the certificate")
	kubernetesCmd.AddCommand(kubernetesIssueCmd)
}
