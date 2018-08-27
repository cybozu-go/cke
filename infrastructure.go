package cke

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/cmd"
	"github.com/pkg/errors"
)

var httpClient *cmd.HTTPClient = &cmd.HTTPClient{
	Client: &http.Client{},
}

// Infrastructure presents an interface for infrastructure on CKE
type Infrastructure interface {
	Close()
	Agent(addr string) Agent

	NewEtcdClient(endpoints []string) (*clientv3.Client, error)
	NewHTTPClient() *cmd.HTTPClient
}

// NewInfrastructure creates a new Infrastructure instance
func NewInfrastructure(ctx context.Context, c *Cluster) (Infrastructure, error) {
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
	inf := &ckeInfrastructure{agents: agents}
	agents = nil
	return inf, nil

}

type ckeInfrastructure struct {
	agents map[string]Agent
}

func (i ckeInfrastructure) Agent(addr string) Agent {
	return i.agents[addr]
}

func (i ckeInfrastructure) Close() {
	for _, a := range i.agents {
		a.Close()
	}
	i.agents = nil
}

func (i ckeInfrastructure) NewEtcdClient(endpoints []string) (*clientv3.Client, error) {
	// TODO support TLS
	return clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
	})
}

func (i ckeInfrastructure) NewHTTPClient() *cmd.HTTPClient {
	// TODO support TLS
	return httpClient
}
