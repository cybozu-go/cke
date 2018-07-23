package cke

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/etcdhttp"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"github.com/pkg/errors"
)

// EtcdNodeHealth represents the health status of a node in etcd cluster.
type EtcdNodeHealth int

// health statuses of a etcd node.
const (
	EtcdNodeUnreachable EtcdNodeHealth = iota
	EtcdNodeHealthy
	EtcdNodeUnhealthy
)

// EtcdClusterStatus is the status of the etcd cluster.
type EtcdClusterStatus struct {
	Members      map[string]*etcdserverpb.Member
	MemberHealth map[string]EtcdNodeHealth
}

func (s EtcdClusterStatus) minHealthyNodes() int {
	return len(s.Members)/2 + 1
}

func (s EtcdClusterStatus) numHealthy() int {
	numHealthy := 0
	for _, h := range s.MemberHealth {
		if h == EtcdNodeHealthy {
			numHealthy++
		}
	}
	return numHealthy
}

func (s EtcdClusterStatus) removableNodeCount() int {
	diff := s.numHealthy() - s.minHealthyNodes()
	if diff <= 0 {
		return 0
	}
	return diff
}

func (s EtcdClusterStatus) addable() bool {
	newMinHealthy := (len(s.Members)+1)/2 + 1
	return s.numHealthy() >= newMinHealthy
}

// ClusterStatus represents the working cluster status.
// The structure reflects Cluster, of course.
type ClusterStatus struct {
	Name          string
	NodeStatuses  map[string]*NodeStatus // keys are IP address strings.
	Agents        map[string]Agent       // ditto.
	ServiceSubnet *net.IPNet
	RBAC          bool // true if RBAC is enabled

	Etcd EtcdClusterStatus
	// TODO:
	// CoreDNS will be deployed as k8s Pods.
	// We probably need to use k8s API to query CoreDNS service status.
}

// Destroy calls Close for all agents.
func (cs *ClusterStatus) Destroy() {
	for _, a := range cs.Agents {
		a.Close()
	}
	cs.Agents = nil
}

// NodeStatus status of a node.
type NodeStatus struct {
	Etcd       EtcdStatus
	APIServer  ServiceStatus
	Controller ServiceStatus
	Scheduler  ServiceStatus
	Proxy      ServiceStatus
	Kubelet    KubeletStatus
	Labels     map[string]string // are labels for k8s Node resource.
}

// IsControlPlane returns true if the node has been configured as a control plane.
func (ns *NodeStatus) IsControlPlane() bool {
	return ns.Etcd.HasData
}

// ServiceStatus represents statuses of a service.
//
// If Running is false, the service is not running on the node.
// ExtraXX are extra parameters of the running service, if any.
type ServiceStatus struct {
	Running        bool
	Image          string
	ExtraArguments []string
	ExtraBinds     []Mount
	ExtraEnvvar    map[string]string
}

// EtcdStatus is the status of kubelet.
type EtcdStatus struct {
	ServiceStatus
	HasData bool
}

// KubeletStatus is the status of kubelet.
type KubeletStatus struct {
	ServiceStatus
	Domain    string
	AllowSwap bool
}

// GetClusterStatus consults the whole cluster and constructs *ClusterStatus.
func (c Controller) GetClusterStatus(ctx context.Context, cluster *Cluster) (*ClusterStatus, error) {
	var mu sync.Mutex
	statuses := make(map[string]*NodeStatus)
	agents := make(map[string]Agent)
	defer func() {
		for _, a := range agents {
			a.Close()
		}
	}()

	env := cmd.NewEnvironment(ctx)
	for _, n := range cluster.Nodes {
		n := n
		env.Go(func(ctx context.Context) error {
			a, err := SSHAgent(n)
			if err != nil {
				return errors.Wrap(err, n.Address)
			}
			ns, err := c.getNodeStatus(ctx, n, a, cluster)
			if err != nil {
				return errors.Wrap(err, n.Address)
			}
			mu.Lock()
			statuses[n.Address] = ns
			agents[n.Address] = a
			mu.Unlock()
			return nil
		})
	}
	env.Stop()
	err := env.Wait()
	if err != nil {
		return nil, err
	}

	cs := new(ClusterStatus)
	cs.NodeStatuses = statuses

	cs.Etcd.Members, err = c.getEtcdMembers(ctx, cluster.Nodes)
	if err != nil {
		// Ignore err since the cluster may be on bootstrap
		log.Warn("failed to get etcd members", map[string]interface{}{
			log.FnError: err,
		})
	}
	cs.Etcd.MemberHealth = c.getEtcdMemberHealth(ctx, cs.Etcd.Members)

	// TODO: query k8s cluster status and store it to ClusterStatus.

	// These assignments should be placed last.
	cs.Agents = agents
	agents = nil
	return cs, nil
}

func (c Controller) getNodeStatus(ctx context.Context, node *Node, agent Agent, cluster *Cluster) (*NodeStatus, error) {
	status := &NodeStatus{}
	ce := Docker(agent)

	// etcd status
	ss, err := ce.Inspect("etcd")
	if err != nil {
		return nil, err
	}
	ok, err := ce.VolumeExists(etcdVolumeName(cluster.Options.Etcd))
	if err != nil {
		return nil, err
	}

	status.Etcd = EtcdStatus{*ss, ok}

	// TODO: get statuses of other services.

	return status, nil
}

func (c Controller) getEtcdMembers(ctx context.Context, nodes []*Node) (map[string]*etcdserverpb.Member, error) {
	var endpoints []string
	for _, n := range nodes {
		if n.ControlPlane {
			endpoints = append(endpoints, fmt.Sprintf("http://%s:2379", n.Address))
		}
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	resp, err := cli.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	members := make(map[string]*etcdserverpb.Member)
	for _, m := range resp.Members {
		name, err := etcdGuessMemberName(m)
		if err != nil {
			log.Warn("failed to guess etcd member name", map[string]interface{}{
				"member_id": m.ID,
				log.FnError: err,
			})
			continue
		}
		members[name] = m
	}
	return members, nil
}

func (c Controller) getEtcdMemberHealth(ctx context.Context, members map[string]*etcdserverpb.Member) map[string]EtcdNodeHealth {
	memberHealth := make(map[string]EtcdNodeHealth)
	for name := range members {
		memberHealth[name] = c.getEtcdHealth(ctx, name)
	}
	return memberHealth
}

func (c Controller) getEtcdHealth(ctx context.Context, address string) EtcdNodeHealth {
	req := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "http", Host: address + ":2379", Path: "/health"},
	}
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		return EtcdNodeUnreachable
	}
	defer resp.Body.Close()

	health := new(etcdhttp.Health)
	err = json.NewDecoder(resp.Body).Decode(health)
	if err != nil {
		return EtcdNodeUnhealthy
	}
	if health.Health == "true" {
		return EtcdNodeHealthy
	}
	return EtcdNodeUnhealthy
}
