package cke

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"github.com/pkg/errors"
)

// EtcdNodeHealth represents the health status of a node in etcd cluster.
type EtcdNodeHealth int

// health statuses of a etcd node.
const (
	EtcdNodeUnhealthy EtcdNodeHealth = iota
	EtcdNodeHealthy
)

// EtcdClusterStatus is the status of the etcd cluster.
type EtcdClusterStatus struct {
	Members      map[string]*etcdserverpb.Member
	MemberHealth map[string]EtcdNodeHealth
}

// ClusterStatus represents the working cluster status.
// The structure reflects Cluster, of course.
type ClusterStatus struct {
	Name         string
	NodeStatuses map[string]*NodeStatus // keys are IP address strings.
	RBAC         bool                   // true if RBAC is enabled
	Client       *cmd.HTTPClient

	Etcd EtcdClusterStatus
	// TODO:
	// CoreDNS will be deployed as k8s Pods.
	// We probably need to use k8s API to query CoreDNS service status.
}

// NodeStatus status of a node.
type NodeStatus struct {
	Etcd              EtcdStatus
	Rivers            ServiceStatus
	APIServer         ServiceStatus
	ControllerManager ServiceStatus
	Scheduler         ServiceStatus
	Proxy             ServiceStatus
	Kubelet           ServiceStatus
	Labels            map[string]string // are labels for k8s Node resource.
}

// ServiceStatus represents statuses of a service.
//
// If Running is false, the service is not running on the node.
// ExtraXX are extra parameters of the running service, if any.
type ServiceStatus struct {
	Running       bool
	Image         string
	BuiltInParams ServiceParams
	ExtraParams   ServiceParams
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
func (c Controller) GetClusterStatus(ctx context.Context, cluster *Cluster, inf Infrastructure) (*ClusterStatus, error) {
	var mu sync.Mutex
	statuses := make(map[string]*NodeStatus)

	env := cmd.NewEnvironment(ctx)
	for _, n := range cluster.Nodes {
		n := n
		env.Go(func(ctx context.Context) error {
			a := inf.Agent(n.Address)
			ns, err := c.getNodeStatus(ctx, n, a, cluster)
			if err != nil {
				return errors.Wrap(err, n.Address)
			}
			mu.Lock()
			statuses[n.Address] = ns
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
	cs.Client = &cmd.HTTPClient{
		Client: &http.Client{},
	}

	cs.Etcd.Members, err = c.getEtcdMembers(ctx, cluster.Nodes)
	if err != nil {
		// Ignore err since the cluster may be on bootstrap
		log.Warn("failed to get etcd members", map[string]interface{}{
			log.FnError: err,
		})
	}
	cs.Etcd.MemberHealth = c.getEtcdMemberHealth(ctx, cs.Etcd.Members)

	// TODO: query k8s cluster status and store it to ClusterStatus.

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

	// rivers status
	ss, err = ce.Inspect("rivers")
	if err != nil {
		return nil, err
	}
	status.Rivers = *ss

	// kube-apiserver status
	ss, err = ce.Inspect("kube-apiserver")
	if err != nil {
		return nil, err
	}
	status.APIServer = *ss

	// kube-controller-manager status
	ss, err = ce.Inspect("kube-controller-manager")
	if err != nil {
		return nil, err
	}
	status.ControllerManager = *ss

	// kube-scheduler status
	ss, err = ce.Inspect("kube-scheduler")
	if err != nil {
		return nil, err
	}
	status.Scheduler = *ss

	// kubelet status
	ss, err = ce.Inspect("kubelet")
	if err != nil {
		return nil, err
	}
	status.Kubelet = *ss

	// kube-proxy status
	ss, err = ce.Inspect("kube-proxy")
	if err != nil {
		return nil, err
	}
	status.Proxy = *ss

	// TODO: get statuses of other services.

	return status, nil
}

func (c Controller) getEtcdMembers(ctx context.Context, nodes []*Node) (map[string]*etcdserverpb.Member, error) {
	var endpoints []string
	for _, n := range nodes {
		if n.ControlPlane {
			endpoints = append(endpoints, fmt.Sprintf("https://%s:2379", n.Address))
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
	endpoints := []string{fmt.Sprintf("https://%s:2379", address)}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		return EtcdNodeUnhealthy
	}
	defer cli.Close()

	_, err = cli.Get(ctx, "health")
	if err == nil || err == rpctypes.ErrPermissionDenied {
		return EtcdNodeHealthy
	}

	return EtcdNodeUnhealthy
}
