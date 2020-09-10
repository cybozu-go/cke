package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cybozu-go/cke/op"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func backupList(ctx context.Context, cmd *cobra.Command) error {
	cfg, err := storage.GetCluster(ctx)
	if err != nil {
		return err
	}
	if !cfg.EtcdBackup.Enabled {
		return errors.New("etcd backup is disabled")
	}
	var controlPlane *cke.Node
	for _, node := range cfg.Nodes {
		if node.ControlPlane {
			controlPlane = node
			break
		}
	}
	if controlPlane == nil {
		return errors.New("control plane not found")
	}
	cs, err := inf.K8sClient(ctx, controlPlane)
	if err != nil {
		return err
	}

	svc, err := cs.CoreV1().Services("kube-system").Get(ctx, op.EtcdBackupAppName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if len(svc.Spec.Ports) == 0 {
		return errors.New("service.ports is empty")
	}
	port := svc.Spec.Ports[0].NodePort
	nodeIP := cfg.Nodes[0].Address
	url := fmt.Sprintf("http://%s:%d/api/v1/backup", nodeIP, port)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	resp, err := inf.HTTPClient().Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}

var etcdBackupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List etcd backups",
	Long: `List etcd backup file names.

The filenames are usually contained created date string.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return backupList(ctx, cmd)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	etcdBackupCmd.AddCommand(etcdBackupListCmd)
}
