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
	yaml "gopkg.in/yaml.v2"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EtcdClusterStatus is the status of the etcd cluster.
type EtcdClusterStatus struct {
	IsHealthy     bool
	Members       map[string]*etcdserverpb.Member
	InSyncMembers map[string]bool
}

// KubernetesClusterStatus contains kubernetes cluster configurations
type KubernetesClusterStatus struct {
	Nodes []core.Node

	RBACRoleExists        bool
	RBACRoleBindingExists bool
}

// ClusterStatus represents the working cluster status.
// The structure reflects Cluster, of course.
type ClusterStatus struct {
	Name         string
	NodeStatuses map[string]*NodeStatus // keys are IP address strings.

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
	APIServer         KubeComponentStatus
	ControllerManager KubeComponentStatus
	Scheduler         KubeComponentStatus
	Proxy             KubeComponentStatus
	Kubelet           KubeletStatus
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

// KubeletStatus represents kubelet status and health
type KubeletStatus struct {
	ServiceStatus
	IsHealthy bool
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
			ns, err := c.getNodeStatus(ctx, inf, n, cluster)
			if err != nil {
				return fmt.Errorf("%s: %v", n.Address, err)
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

	var livingMaster *Node
	for _, n := range controlPlanes(cluster.Nodes) {
		ns := statuses[n.Address]
		if ns.APIServer.Running {
			livingMaster = n
			break
		}
	}

	if livingMaster != nil {
		cs.Kubernetes, err = c.getKubernetesClusterStatus(ctx, inf, livingMaster)
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
	agent := inf.Agent(node.Address)
	ce := Docker(agent)

	ss, err := ce.Inspect([]string{
		etcdContainerName,
		riversContainerName,
		kubeAPIServerContainerName,
		kubeControllerManagerContainerName,
		kubeSchedulerContainerName,
		kubeProxyContainerName,
		kubeletContainerName,
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

	status.APIServer = KubeComponentStatus{ss[kubeAPIServerContainerName], false}
	if status.APIServer.Running {
		status.APIServer.IsHealthy, err = c.checkAPIServerHalth(ctx, inf, node)
		if err != nil {
			log.Warn("failed to check API server health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}
	}

	status.ControllerManager = KubeComponentStatus{ss[kubeControllerManagerContainerName], false}
	if status.ControllerManager.Running {
		status.ControllerManager.IsHealthy, err = c.checkHealthz(ctx, inf, node.Address, 10252)
		if err != nil {
			log.Warn("failed to check controller manager health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}
	}

	status.Scheduler = KubeComponentStatus{ss[kubeSchedulerContainerName], false}
	if status.Scheduler.Running {
		status.Scheduler.IsHealthy, err = c.checkHealthz(ctx, inf, node.Address, 10251)
		if err != nil {
			log.Warn("failed to check scheduler health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}
	}

	// TODO: doe to the following bug, health status cannot be checked for proxy.
	// https://github.com/kubernetes/kubernetes/issues/65118
	status.Proxy = KubeComponentStatus{ss[kubeProxyContainerName], false}
	status.Proxy.IsHealthy = status.Proxy.Running

	status.Kubelet = KubeletStatus{ss[kubeletContainerName], false, "", false}
	if status.Kubelet.Running {
		status.Kubelet.IsHealthy, err = c.checkHealthz(ctx, inf, node.Address, 10248)
		if err != nil {
			log.Warn("failed to check kubelet health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}

		cfgData, _, err := agent.Run("cat /etc/kubernetes/kubelet/config.yml")
		if err == nil {
			v := struct {
				ClusterDomain string `yaml:"clusterDomain"`
				FailSwapOn    bool   `yaml:"failSwapOn"`
			}{}
			err = yaml.Unmarshal(cfgData, &v)
			if err == nil {
				status.Kubelet.Domain = v.ClusterDomain
				status.Kubelet.AllowSwap = !v.FailSwapOn
			}
		}
	}

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
	if err != nil {
		return clusterStatus, err
	}

	clusterStatus.IsHealthy = resp.ID != clientv3.NoLease

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
	if err != nil {
		return false
	}

	return resp.Header.Revision >= clusterRev
}

func (c Controller) getKubernetesClusterStatus(ctx context.Context, inf Infrastructure, n *Node) (KubernetesClusterStatus, error) {
	clientset, err := inf.K8sClient(n)
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

	_, err = clientset.RbacV1().ClusterRoles().Get(rbacRoleName, meta.GetOptions{})
	switch {
	case err == nil:
		s.RBACRoleExists = true
	case errors.IsNotFound(err):
	default:
		return KubernetesClusterStatus{}, err
	}

	_, err = clientset.RbacV1().ClusterRoleBindings().Get(rbacRoleBindingName, meta.GetOptions{})
	switch {
	case err == nil:
		s.RBACRoleBindingExists = true
	case errors.IsNotFound(err):
	default:
		return KubernetesClusterStatus{}, err
	}

	return s, nil
}

func (c Controller) checkHealthz(ctx context.Context, inf Infrastructure, addr string, port uint16) (bool, error) {
	url := "http://" + addr + ":" + strconv.FormatUint(uint64(port), 10) + "/healthz"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req = req.WithContext(ctx)
	resp, err := inf.HTTPClient().Do(req)
	if err != nil {
		return false, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	resp.Body.Close()

	return strings.TrimSpace(string(body)) == "ok", nil
}

func (c Controller) checkAPIServerHalth(ctx context.Context, inf Infrastructure, n *Node) (bool, error) {
	cliantset, err := inf.K8sClient(n)
	if err != nil {
		return false, err
	}
	_, err = cliantset.CoreV1().Namespaces().List(meta.ListOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}
