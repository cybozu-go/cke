package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cybozu-go/cke/op"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func backupGet(ctx context.Context, cmd *cobra.Command, filename string) error {
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
	url := fmt.Sprintf("http://%s:%d/api/v1/backup/%s", nodeIP, port, filename)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	resp, err := inf.HTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return err
	}
	defer f.Close()
	cmd.SetOutput(f)

	_, err = io.Copy(f, resp.Body)
	return err
}

var etcdBackupGetCmd = &cobra.Command{
	Use:   "get BACKUP_NAME",
	Short: "Get an etcd backup",
	Long: `Get an etcd backup.

You can find BACKUP_NAME using ckecli etcd backup list`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return backupGet(ctx, cmd, args[0])
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	etcdBackupCmd.AddCommand(etcdBackupGetCmd)
}
