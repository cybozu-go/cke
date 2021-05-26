package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/well"
	vault "github.com/hashicorp/vault/api"
	"go.etcd.io/etcd/clientv3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var httpClient = &well.HTTPClient{
	Client: &http.Client{},
}

// cliInfrastructure implements cke.Infrastructure for CLI usage.
type cliInfrastructure struct {
	vc   *vault.Client
	etcd *clientv3.Client
}

func (i *cliInfrastructure) Close() {
	if i.etcd != nil {
		i.etcd.Close()
	}
}

func (i *cliInfrastructure) Storage() cke.Storage {
	return storage
}

func (i *cliInfrastructure) Vault() (*vault.Client, error) {
	if i.vc != nil {
		return i.vc, nil
	}

	cfg, err := storage.GetVaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	vc, _, err := cke.VaultClient(cfg)
	if err != nil {
		return nil, err
	}

	i.vc = vc
	return vc, nil
}

// The second argument is not used.
func (i *cliInfrastructure) NewEtcdClient(ctx context.Context, _ []string) (*clientv3.Client, error) {
	if i.etcd != nil {
		return i.etcd, nil
	}

	cluster, err := storage.GetCluster(ctx)
	if err != nil {
		return nil, err
	}

	endpoints := []string{}
	for _, n := range cluster.Nodes {
		if n.ControlPlane {
			endpoints = append(endpoints, fmt.Sprintf("https://%s:2379", n.Address))
		}
	}
	if len(endpoints) == 0 {
		return nil, errors.New("no control plane")
	}

	serverCA, err := storage.GetCACertificate(ctx, cke.CAServer)
	if err != nil {
		return nil, err
	}
	etcdCert, etcdKey, err := cke.EtcdCA{}.IssueRoot(ctx, i)
	if err != nil {
		return nil, err
	}

	cfg := &etcdutil.Config{
		Endpoints: endpoints,
		Timeout:   etcdutil.DefaultTimeout,
		TLSCA:     serverCA,
		TLSCert:   etcdCert,
		TLSKey:    etcdKey,
	}
	etcd, err := etcdutil.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	i.etcd = etcd
	return etcd, nil
}

func (i *cliInfrastructure) K8sConfig(ctx context.Context, n *cke.Node) (*rest.Config, error) {
	panic("not implemented")
}

func (i *cliInfrastructure) K8sClient(ctx context.Context, n *cke.Node) (*kubernetes.Clientset, error) {
	c, k, err := cke.KubernetesCA{}.IssueUserCert(ctx, i, cke.RoleAdmin, cke.AdminGroup, "1h")
	if err != nil {
		return nil, err
	}
	ca, err := i.Storage().GetCACertificate(ctx, cke.CAKubernetes)
	if err != nil {
		return nil, err
	}
	tlsCfg := rest.TLSClientConfig{
		CertData: []byte(c),
		KeyData:  []byte(k),
		CAData:   []byte(ca),
	}
	cfg := &rest.Config{
		Host:            "https://" + n.Address + ":6443",
		TLSClientConfig: tlsCfg,
		Timeout:         5 * time.Second,
	}
	return kubernetes.NewForConfig(cfg)
}

func (i *cliInfrastructure) HTTPClient() *well.HTTPClient {
	return httpClient
}
func (i *cliInfrastructure) HTTPSClient(ctx context.Context) (*well.HTTPClient, error) {
	panic("not implemented")
}
func (i *cliInfrastructure) Agent(addr string) cke.Agent {
	panic("not implemented")
}
func (i *cliInfrastructure) Engine(addr string) cke.ContainerEngine {
	panic("not implemented")
}
