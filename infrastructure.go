package cke

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/etcdutil"
	vault "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var httpClient = &cmd.HTTPClient{
	Client: &http.Client{},
}

var vaultClient atomic.Value

func setVaultClient(client *vault.Client) {
	vaultClient.Store(client)
}

// Infrastructure presents an interface for infrastructure on CKE
type Infrastructure interface {
	Close()
	Agent(addr string) Agent
	Vault() (*vault.Client, error)
	Storage() Storage

	NewEtcdClient(endpoints []string) (*clientv3.Client, error)
	kubernetesClient(n *Node) (*kubernetes.Clientset, error)
	HTTPClient() *cmd.HTTPClient
}

// NewInfrastructure creates a new Infrastructure instance
func NewInfrastructure(ctx context.Context, c *Cluster, s Storage) (Infrastructure, error) {
	agents := make(map[string]Agent)
	defer func() {
		for _, a := range agents {
			a.Close()
		}
	}()

	mu := new(sync.Mutex)

	env := cmd.NewEnvironment(ctx)
	for _, n := range c.Nodes {
		node := n
		env.Go(func(ctx context.Context) error {
			a, err := SSHAgent(node)
			if err != nil {
				return errors.Wrap(err, node.Address)
			}

			mu.Lock()
			agents[node.Address] = a
			mu.Unlock()
			return nil
		})

	}
	env.Stop()
	err := env.Wait()
	if err != nil {
		return nil, err
	}

	// These assignments of the `agent` should be placed last.
	inf := &ckeInfrastructure{agents: agents, storage: s}
	agents = nil

	inf.serverCA, err = inf.Storage().GetCACertificate(ctx, "server")
	if err != nil {
		return nil, err
	}
	inf.etcdCert, inf.etcdKey, err = EtcdCA{}.issueRoot(ctx, inf)
	if err != nil {
		return nil, err
	}
	inf.kubeCA, err = inf.Storage().GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return nil, err
	}
	inf.kubeCert, inf.kubeKey, err = KubernetesCA{}.issueAdminCert(ctx, inf)
	if err != nil {
		return nil, err
	}

	return inf, nil
}

type ckeInfrastructure struct {
	agents   map[string]Agent
	storage  Storage
	serverCA string
	etcdCert string
	etcdKey  string
	kubeCA   string
	kubeCert string
	kubeKey  string
}

func (i ckeInfrastructure) Agent(addr string) Agent {
	return i.agents[addr]
}

func (i ckeInfrastructure) Vault() (*vault.Client, error) {
	v := vaultClient.Load()
	if v == nil {
		return nil, errors.New("vault is not connected")
	}
	return v.(*vault.Client), nil
}

func (i ckeInfrastructure) Storage() Storage {
	return i.storage
}

func (i ckeInfrastructure) Close() {
	for _, a := range i.agents {
		a.Close()
	}
	i.agents = nil
}

func (i ckeInfrastructure) NewEtcdClient(endpoints []string) (*clientv3.Client, error) {
	cfg := &etcdutil.Config{
		Endpoints: endpoints,
		Timeout:   etcdutil.DefaultTimeout,
		TLSCA:     i.serverCA,
		TLSCert:   i.etcdCert,
		TLSKey:    i.etcdKey,
	}
	return etcdutil.NewClient(cfg)
}

func (i ckeInfrastructure) kubernetesClient(n *Node) (*kubernetes.Clientset, error) {
	tlsCfg := rest.TLSClientConfig{
		CertData: []byte(i.kubeCert),
		KeyData:  []byte(i.kubeKey),
		CAData:   []byte(i.kubeCA),
	}
	cfg := &rest.Config{
		Host:            "https://" + n.Address + ":6443",
		TLSClientConfig: tlsCfg,
		Timeout:         5 * time.Second,
	}
	return kubernetes.NewForConfig(cfg)
}

func (i ckeInfrastructure) HTTPClient() *cmd.HTTPClient {
	return httpClient
}
