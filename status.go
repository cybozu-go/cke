package cke

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"github.com/pkg/errors"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EtcdClusterStatus is the status of the etcd cluster.
type EtcdClusterStatus struct {
	IsHealthy     bool
	Members       map[string]*etcdserverpb.Member
	InSyncMembers map[string]bool
}

type KubernetesClusterStatus struct {
	Nodes []core.Node
}

// ClusterStatus represents the working cluster status.
// The structure reflects Cluster, of course.
type ClusterStatus struct {
	Name         string
	NodeStatuses map[string]*NodeStatus // keys are IP address strings.
	RBAC         bool                   // true if RBAC is enabled

	Etcd       EtcdClusterStatus
	Kubernetes KubernetesClusterStatus

	// TODO:
	// CoreDNS will be deployed as k8s Pods.
	// We probably need to use k8s API to query CoreDNS service status.
}

// NodeStatus status of a node.
type NodeStatus struct {
	Etcd              EtcdStatus
	Rivers            ServiceStatus
	APIServer         ServiceStatus
	ControllerManager KubeComponentStatus
	Scheduler         KubeComponentStatus
	Proxy             ServiceStatus
	Kubelet           KubeComponentStatus
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

// KubeComponentStatus represents service status and endpoint's health
type KubeComponentStatus struct {
	ServiceStatus
	IsHealthy bool
}

// GetClusterStatus consults the whole cluster and constructs *ClusterStatus.
func (c Controller) GetClusterStatus(ctx context.Context, cluster *Cluster, inf Infrastructure) (*ClusterStatus, error) {
	var mu sync.Mutex
	statuses := make(map[string]*NodeStatus)

	env := cmd.NewEnvironment(ctx)
	for _, n := range cluster.Nodes {
		n := n
		env.Go(func(ctx context.Context) error {
			ns, err := c.getNodeStatus(ctx, inf, n, cluster)
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

	var etcdRunning bool
	for _, n := range controlPlanes(cluster.Nodes) {
		ns := statuses[n.Address]
		if ns.Etcd.HasData {
			etcdRunning = true
			break
		}
	}

	if etcdRunning {
		cs.Etcd, err = c.getEtcdClusterStatus(ctx, inf, cluster.Nodes)
		if err != nil {
			log.Error("failed to get etcd cluster status", map[string]interface{}{
				log.FnError: err,
			})
			return nil, err
		}
	}
	return cs, nil

	var apiserverRunning bool
	for _, n := range controlPlanes(cluster.Nodes) {
		ns := statuses[n.Address]
		if ns.APIServer.Running {
			apiserverRunning = true
			break
		}
	}

	if apiserverRunning {
		cs.Kubernetes, err = c.getKubernetesClusterStatus(ctx, inf, cluster.Nodes)
		if err != nil {
			log.Error("failed to get kubernetes cluster status", map[string]interface{}{
				log.FnError: err,
			})
			return nil, err
		}
	}
	return cs, nil
}

func (c Controller) getNodeStatus(ctx context.Context, inf Infrastructure, node *Node, cluster *Cluster) (*NodeStatus, error) {
	status := &NodeStatus{}
	ce := Docker(inf.Agent(node.Address))

	ss, err := ce.Inspect([]string{
		etcdContainerName,
		riversContainerName,
		kubeAPIServerContainerName,
		kubeControllerManagerContainerName,
		kubeSchedulerContainerName,
		kubeletContainerName,
		kubeProxyContainerName,
	})
	if err != nil {
		return nil, err
	}

	etcdVolumeExists, err := ce.VolumeExists(etcdVolumeName(cluster.Options.Etcd))
	if err != nil {
		return nil, err
	}

	status.Etcd = EtcdStatus{ss[etcdContainerName], etcdVolumeExists}
	status.Rivers = ss[riversContainerName]
	status.APIServer = ss[kubeAPIServerContainerName]

	var health bool
	if ss[kubeControllerManagerContainerName].Running {
		health = c.checkHealthz(ctx, inf, node.Address, 10252) != nil
	}
	status.ControllerManager = KubeComponentStatus{ServiceStatus: ss[kubeControllerManagerContainerName], IsHealthy: health}

	health = false
	if ss[kubeSchedulerContainerName].Running {
		health = c.checkHealthz(ctx, inf, node.Address, 10251) != nil
	}
	status.Scheduler = KubeComponentStatus{ServiceStatus: ss[kubeSchedulerContainerName], IsHealthy: health}

	health = false
	if ss[kubeletContainerName].Running {
		health = c.checkHealthz(ctx, inf, node.Address, 10248) != nil
	}
	status.Kubelet = KubeComponentStatus{ServiceStatus: ss[kubeletContainerName], IsHealthy: health}

	// NOTE unable to get kube-proxy's health status
	// https://github.com/kubernetes/kubernetes/issues/65118
	status.Proxy = ss[kubeProxyContainerName]

	return status, nil
}

func (c Controller) getEtcdMembers(ctx context.Context, inf Infrastructure, cli *clientv3.Client) (map[string]*etcdserverpb.Member, error) {
	ct, cancel := context.WithTimeout(ctx, defaultEtcdTimeout)
	defer cancel()
	resp, err := cli.MemberList(ct)
	if err != nil {
		return nil, err
	}
	members := make(map[string]*etcdserverpb.Member)
	for _, m := range resp.Members {
		name, err := etcdGuessMemberName(m)
		if err != nil {
			return nil, err
		}
		members[name] = m
	}
	return members, nil
}

func (c Controller) getEtcdClusterStatus(ctx context.Context, inf Infrastructure, nodes []*Node) (EtcdClusterStatus, error) {
	clusterStatus := EtcdClusterStatus{}

	var endpoints []string
	for _, n := range nodes {
		if n.ControlPlane {
			endpoints = append(endpoints, fmt.Sprintf("https://%s:2379", n.Address))
		}
	}

	cli, err := inf.NewEtcdClient(endpoints)
	if err != nil {
		return clusterStatus, err
	}
	defer cli.Close()

	clusterStatus.Members, err = c.getEtcdMembers(ctx, inf, cli)
	if err != nil {
		return clusterStatus, err
	}

	ct, cancel := context.WithTimeout(ctx, defaultEtcdTimeout)
	defer cancel()
	resp, err := cli.Grant(ct, 10)
	clusterStatus.IsHealthy = err == nil && resp.ID != clientv3.NoLease

	clusterStatus.InSyncMembers = make(map[string]bool)
	for name := range clusterStatus.Members {
		clusterStatus.InSyncMembers[name] = c.getEtcdMemberInSync(ctx, inf, name, resp.Revision)
	}

	return clusterStatus, nil
}

func (c Controller) getEtcdMemberInSync(ctx context.Context, inf Infrastructure, address string, clusterRev int64) bool {
	endpoints := []string{fmt.Sprintf("https://%s:2379", address)}
	cli, err := inf.NewEtcdClient(endpoints)
	if err != nil {
		return false
	}
	defer cli.Close()

	ct, cancel := context.WithTimeout(ctx, defaultEtcdTimeout)
	defer cancel()
	resp, err := cli.Get(ct, "health")
	if err == nil && resp.Header.Revision >= clusterRev {
		return true
	}

	return false
}

func (c Controller) getKubernetesClusterStatus(ctx context.Context, inf Infrastructure, nodes []*Node) (KubernetesClusterStatus, error) {
	// TODO available high-reliability control planes to get cluster status
	var master *Node
	for _, n := range nodes {
		if n.ControlPlane {
			master = n
		}
	}
	clientset, err := inf.kubernetesClient(master)
	if err != nil {
		return KubernetesClusterStatus{}, err
	}
	resp, err := clientset.CoreV1().Nodes().List(meta.ListOptions{})
	if err != nil {
		return KubernetesClusterStatus{}, err
	}

	s := KubernetesClusterStatus{
		Nodes: resp.Items,
	}
	return s, nil
}

func (c Controller) checkHealthz(ctx context.Context, inf Infrastructure, addr string, port uint16) error {
	url := "http://" + addr + ":" + strconv.FormatUint(uint64(port), 10) + "/healthz"
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
	if err != nil {
		return err
	}
	resp.Body.Close()
	if strings.TrimSpace(string(body)) == "ok" {
		return errors.New("component does not healthy")
		return nil
	}
	return nil
}
