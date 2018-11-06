package cke

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/well"
	vault "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var httpClient = &well.HTTPClient{
	Client: &http.Client{},
}

var vaultClient atomic.Value

type certCache struct {
	cert      []byte
	key       []byte
	timestamp time.Time
	lifetime  time.Duration
}

func (c *certCache) get(issue func() (cert, key []byte, err error)) (cert, key []byte, err error) {
	now := time.Now()
	if c.cert != nil {
		if now.Sub(c.timestamp) < c.lifetime {
			return c.cert, c.key, nil
		}
	}

	cert, key, err = issue()
	if err == nil {
		c.cert = cert
		c.key = key
		c.timestamp = now
	}
	return
}

var k8sCertCache = &certCache{
	lifetime: time.Hour * 24,
}

func setVaultClient(client *vault.Client) {
	vaultClient.Store(client)
}

// Infrastructure presents an interface for infrastructure on CKE
type Infrastructure interface {
	Close()
	Agent(addr string) Agent
	Engine(addr string) ContainerEngine
	Vault() (*vault.Client, error)
	Storage() Storage

	NewEtcdClient(ctx context.Context, endpoints []string) (*clientv3.Client, error)
	K8sClient(ctx context.Context, n *Node) (*kubernetes.Clientset, error)
	HTTPClient() *well.HTTPClient
}

type ckeInfrastructure struct {
	agents  map[string]Agent
	storage Storage

	etcdOnce sync.Once
	etcdErr  error
	serverCA string
	etcdCert string
	etcdKey  string

	kubeOnce sync.Once
	kubeErr  error
	kubeCA   string
	kubeCert []byte
	kubeKey  []byte
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

	env := well.NewEnvironment(ctx)
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

	// This assignment of the `agent` must be placed last.
	inf := &ckeInfrastructure{agents: agents, storage: s}
	agents = nil
	return inf, nil
}

// NewInfrastructureWithoutSSH creates a new Infrastructure instance that has no SSH agents
func NewInfrastructureWithoutSSH(s Storage) Infrastructure {
	return &ckeInfrastructure{storage: s}
}

func (i ckeInfrastructure) Agent(addr string) Agent {
	return i.agents[addr]
}

func (i ckeInfrastructure) Engine(addr string) ContainerEngine {
	return Docker(i.agents[addr])
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

func (i *ckeInfrastructure) NewEtcdClient(ctx context.Context, endpoints []string) (*clientv3.Client, error) {
	i.etcdOnce.Do(func() {
		serverCA, err := i.Storage().GetCACertificate(ctx, "server")
		if err != nil {
			i.etcdErr = err
			return
		}
		etcdCert, etcdKey, err := EtcdCA{}.IssueRoot(ctx, i)
		if err != nil {
			i.etcdErr = err
			return
		}

		i.serverCA = serverCA
		i.etcdCert = etcdCert
		i.etcdKey = etcdKey
	})

	if i.etcdErr != nil {
		return nil, i.etcdErr
	}

	cfg := &etcdutil.Config{
		Endpoints: endpoints,
		Timeout:   etcdutil.DefaultTimeout,
		TLSCA:     i.serverCA,
		TLSCert:   i.etcdCert,
		TLSKey:    i.etcdKey,
	}
	return etcdutil.NewClient(cfg)
}

func (i *ckeInfrastructure) K8sClient(ctx context.Context, n *Node) (*kubernetes.Clientset, error) {
	i.kubeOnce.Do(func() {
		kubeCA, err := i.Storage().GetCACertificate(ctx, "kubernetes")
		if err != nil {
			i.kubeErr = err
			return
		}

		issue := func() (cert, key []byte, err error) {
			c, k, e := KubernetesCA{}.IssueAdminCert(ctx, i, "25h")
			if e != nil {
				return nil, nil, e
			}
			return []byte(c), []byte(k), nil
		}
		kubeCert, kubeKey, err := k8sCertCache.get(issue)
		if err != nil {
			i.kubeErr = err
			return
		}

		i.kubeCA = kubeCA
		i.kubeCert = kubeCert
		i.kubeKey = kubeKey
	})

	if i.kubeErr != nil {
		return nil, i.kubeErr
	}

	tlsCfg := rest.TLSClientConfig{
		CertData: i.kubeCert,
		KeyData:  i.kubeKey,
		CAData:   []byte(i.kubeCA),
	}
	cfg := &rest.Config{
		Host:            "https://" + n.Address + ":6443",
		TLSClientConfig: tlsCfg,
		Timeout:         5 * time.Second,
	}
	return kubernetes.NewForConfig(cfg)
}

func (i ckeInfrastructure) HTTPClient() *well.HTTPClient {
	return httpClient
}
