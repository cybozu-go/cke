package cke

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/etcdhttp"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/cmd"
	"github.com/pkg/errors"
)

// Infrastructure presents an interface for infrastructure on CKE
type Infrastructure interface {
	Close()
	Agent(addr string) Agent

	EtcdAddMember(ctx context.Context, addr string) ([]*etcdserverpb.Member, error)
	EtcdGetMembers(ctx context.Context) ([]*etcdserverpb.Member, error)
	EtcdRemoveMember(ctx context.Context, id uint64) error
	EtcdGetHealth(ctx context.Context, endpoint string) (etcdhttp.Health, error)

	// TODO Add kubernetes methods
}

// NewInfrastructure creates a new Infrastructure instance
func NewInfrastructure(c *Cluster) (Infrastructure, error) {
	agents := make(map[string]Agent)
	defer func() {
		for _, a := range agents {
			a.Close()
		}
	}()

	for _, n := range c.Nodes {
		a, err := SSHAgent(n)
		if err != nil {
			return nil, errors.Wrap(err, n.Address)
		}
		agents[n.Address] = a
	}

	// These assignments should be placed last.

	inf := &ckeInfrastructure{
		agents:  agents,
		cluster: c,
		http: &cmd.HTTPClient{
			Client: &http.Client{},
		},
	}
	agents = nil
	return inf, nil
}

type ckeInfrastructure struct {
	agents  map[string]Agent
	cluster *Cluster
	http    *cmd.HTTPClient
}

func (i ckeInfrastructure) Agent(addr string) Agent {
	return i.agents[addr]
}

func (i ckeInfrastructure) EtcdAddMember(ctx context.Context, addr string) ([]*etcdserverpb.Member, error) {
	cli, err := i.newEtcdClient()
	if err != nil {
		return nil, err
	}
	resp, err := cli.MemberAdd(ctx, []string{fmt.Sprintf("http://%s:2380", addr)})
	if err != nil {
		return nil, err
	}
	return resp.Members, nil
}

func (i ckeInfrastructure) EtcdGetMembers(ctx context.Context) ([]*etcdserverpb.Member, error) {
	cli, err := i.newEtcdClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	resp, err := cli.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Members, nil
}

func (i ckeInfrastructure) EtcdRemoveMember(ctx context.Context, id uint64) error {
	cli, err := i.newEtcdClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	_, err = cli.MemberRemove(ctx, id)
	return err
}

func (i ckeInfrastructure) EtcdGetHealth(ctx context.Context, endpoint string) (etcdhttp.Health, error) {
	req, err := http.NewRequest("GET", endpoint+"/health", nil)
	if err != nil {
		return etcdhttp.Health{}, err
	}
	req = req.WithContext(ctx)
	resp, err := i.http.Do(req)
	if err != nil {
		return etcdhttp.Health{}, err
	}
	var health etcdhttp.Health
	err = json.NewDecoder(resp.Body).Decode(&health)
	resp.Body.Close()
	if err != nil {
		return etcdhttp.Health{}, err
	}
	return health, nil

}

func (i ckeInfrastructure) Close() {
	for _, a := range i.agents {
		a.Close()
	}
	i.agents = nil
}

func (i ckeInfrastructure) newEtcdClient() (*clientv3.Client, error) {
	var endpoints []string
	for _, n := range i.cluster.Nodes {
		if n.ControlPlane {
			endpoints = append(endpoints, "http://"+n.Address+":2379")
		}
	}

	return clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
	})
}
