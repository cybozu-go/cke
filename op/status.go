package op

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetNodeStatus returns NodeStatus.
func GetNodeStatus(ctx context.Context, inf cke.Infrastructure, node *cke.Node, cluster *cke.Cluster) (*cke.NodeStatus, error) {
	status := &cke.NodeStatus{}
	agent := inf.Agent(node.Address)
	ce := inf.Engine(node.Address)

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

	status.Etcd = cke.EtcdStatus{ss[etcdContainerName], etcdVolumeExists}
	status.Rivers = ss[riversContainerName]

	status.APIServer = cke.KubeComponentStatus{ss[kubeAPIServerContainerName], false}
	if status.APIServer.Running {
		status.APIServer.IsHealthy, err = checkAPIServerHealth(ctx, inf, node)
		if err != nil {
			log.Warn("failed to check API server health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}
	}

	status.ControllerManager = cke.KubeComponentStatus{ss[kubeControllerManagerContainerName], false}
	if status.ControllerManager.Running {
		status.ControllerManager.IsHealthy, err = checkSecureHealthz(ctx, inf, node.Address, 10257)
		if err != nil {
			log.Warn("failed to check controller manager health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}
	}

	status.Scheduler = cke.KubeComponentStatus{ss[kubeSchedulerContainerName], false}
	if status.Scheduler.Running {
		status.Scheduler.IsHealthy, err = checkHealthz(ctx, inf, node.Address, 10251)
		if err != nil {
			log.Warn("failed to check scheduler health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}
	}

	// TODO: doe to the following bug, health status cannot be checked for proxy.
	// https://github.com/kubernetes/kubernetes/issues/65118
	status.Proxy = cke.KubeComponentStatus{ss[kubeProxyContainerName], false}
	status.Proxy.IsHealthy = status.Proxy.Running

	status.Kubelet = cke.KubeletStatus{ss[kubeletContainerName], false, "", false}
	if status.Kubelet.Running {
		status.Kubelet.IsHealthy, err = checkHealthz(ctx, inf, node.Address, 10248)
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

// GetEtcdClusterStatus returns EtcdClusterStatus
func GetEtcdClusterStatus(ctx context.Context, inf cke.Infrastructure, nodes []*cke.Node) (cke.EtcdClusterStatus, error) {
	clusterStatus := cke.EtcdClusterStatus{}

	var endpoints []string
	for _, n := range nodes {
		if n.ControlPlane {
			endpoints = append(endpoints, fmt.Sprintf("https://%s:2379", n.Address))
		}
	}

	cli, err := inf.NewEtcdClient(ctx, endpoints)
	if err != nil {
		return clusterStatus, err
	}
	defer cli.Close()

	clusterStatus.Members, err = getEtcdMembers(ctx, inf, cli)
	if err != nil {
		return clusterStatus, err
	}

	ct, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()
	resp, err := cli.Grant(ct, 10)
	if err != nil {
		return clusterStatus, err
	}

	clusterStatus.IsHealthy = resp.ID != clientv3.NoLease

	clusterStatus.InSyncMembers = make(map[string]bool)
	for name := range clusterStatus.Members {
		clusterStatus.InSyncMembers[name] = getEtcdMemberInSync(ctx, inf, name, resp.Revision)
	}

	return clusterStatus, nil
}

func getEtcdMembers(ctx context.Context, inf cke.Infrastructure, cli *clientv3.Client) (map[string]*etcdserverpb.Member, error) {
	ct, cancel := context.WithTimeout(ctx, timeoutDuration)
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

func getEtcdMemberInSync(ctx context.Context, inf cke.Infrastructure, address string, clusterRev int64) bool {
	endpoints := []string{fmt.Sprintf("https://%s:2379", address)}
	cli, err := inf.NewEtcdClient(ctx, endpoints)
	if err != nil {
		return false
	}
	defer cli.Close()

	ct, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()
	resp, err := cli.Get(ct, "health")
	if err != nil {
		return false
	}

	return resp.Header.Revision >= clusterRev
}

// GetKubernetesClusterStatus returns KubernetesClusterStatus
func GetKubernetesClusterStatus(ctx context.Context, inf cke.Infrastructure, n *cke.Node, cluster *cke.Cluster) (cke.KubernetesClusterStatus, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}

	s := cke.KubernetesClusterStatus{}

	_, err = clientset.CoreV1().ServiceAccounts("kube-system").Get("default", metav1.GetOptions{})
	switch {
	case err == nil:
		s.IsReady = true
	case errors.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	resp, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}
	s.Nodes = resp.Items

	_, err = clientset.RbacV1().ClusterRoles().Get(rbacRoleName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.RBACRoleExists = true
	case errors.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	_, err = clientset.RbacV1().ClusterRoleBindings().Get(rbacRoleBindingName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.RBACRoleBindingExists = true
	case errors.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	if len(cluster.DNSService) > 0 {
		fields := strings.Split(cluster.DNSService, "/")
		if len(fields) != 2 {
			panic("invalid dns_service in cluster.yml")
		}
		svc, err := clientset.CoreV1().Services(fields[0]).Get(fields[1], metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
		case err == nil:
			s.DNSService = svc
		default:
			return cke.KubernetesClusterStatus{}, err
		}
	}

	s.ClusterDNS, err = GetClusterDNSStatus(ctx, inf, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}

	s.NodeDNS, err = GetNodeDNSStatus(ctx, inf, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}

	s.EtcdBackup, err = GetEtcdBackupStatus(ctx, inf, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}

	ep, err := clientset.CoreV1().Endpoints("kube-system").Get(etcdEndpointsName,
		metav1.GetOptions{IncludeUninitialized: true})
	switch {
	case err == nil:
		s.EtcdEndpoints = ep
	case errors.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	return s, nil
}

// GetClusterDNSStatus returns ClusterDNSStatus
func GetClusterDNSStatus(ctx context.Context, inf cke.Infrastructure, n *cke.Node) (cke.ClusterDNSStatus, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return cke.ClusterDNSStatus{}, err
	}

	s := cke.ClusterDNSStatus{}

	_, err = clientset.CoreV1().ServiceAccounts("kube-system").Get(clusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ServiceAccountExists = true
	case errors.IsNotFound(err):
	default:
		return cke.ClusterDNSStatus{}, err
	}

	_, err = clientset.RbacV1().ClusterRoles().Get(clusterDNSRBACRoleName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.RBACRoleExists = true
	case errors.IsNotFound(err):
	default:
		return cke.ClusterDNSStatus{}, err
	}

	_, err = clientset.RbacV1().ClusterRoleBindings().Get(clusterDNSRBACRoleName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.RBACRoleBindingExists = true
	case errors.IsNotFound(err):
	default:
		return cke.ClusterDNSStatus{}, err
	}

	config, err := clientset.CoreV1().ConfigMaps("kube-system").Get(clusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ConfigMap = config
	case errors.IsNotFound(err):
	default:
		return cke.ClusterDNSStatus{}, err
	}

	deployment, err := clientset.AppsV1().Deployments("kube-system").Get(clusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.Deployment = deployment
	case errors.IsNotFound(err):
	default:
		return cke.ClusterDNSStatus{}, err
	}

	service, err := clientset.CoreV1().Services("kube-system").Get(clusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ServiceExists = true
		s.ClusterIP = service.Spec.ClusterIP
	case errors.IsNotFound(err):
	default:
		return cke.ClusterDNSStatus{}, err
	}

	return s, nil
}

// GetNodeDNSStatus returns NodeDNSStatus
func GetNodeDNSStatus(ctx context.Context, inf cke.Infrastructure, n *cke.Node) (cke.NodeDNSStatus, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return cke.NodeDNSStatus{}, err
	}

	s := cke.NodeDNSStatus{}

	config, err := clientset.CoreV1().ConfigMaps("kube-system").Get(nodeDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ConfigMap = config
	case errors.IsNotFound(err):
	default:
		return cke.NodeDNSStatus{}, err
	}

	ds, err := clientset.AppsV1().DaemonSets("kube-system").Get(nodeDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.DaemonSet = ds
	case errors.IsNotFound(err):
	default:
		return cke.NodeDNSStatus{}, err
	}

	return s, nil
}

// GetEtcdBackupStatus returns EtcdBackupStatus
func GetEtcdBackupStatus(ctx context.Context, inf cke.Infrastructure, n *cke.Node) (cke.EtcdBackupStatus, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return cke.EtcdBackupStatus{}, err
	}

	s := cke.EtcdBackupStatus{}

	config, err := clientset.CoreV1().ConfigMaps("kube-system").Get(EtcdBackupConfigMapName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ConfigMap = config
	case errors.IsNotFound(err):
	default:
		return cke.EtcdBackupStatus{}, err
	}

	pod, err := clientset.CoreV1().Pods("kube-system").Get(EtcdBackupPodName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.Pod = pod
	case errors.IsNotFound(err):
	default:
		return cke.EtcdBackupStatus{}, err
	}

	service, err := clientset.CoreV1().Services("kube-system").Get(EtcdBackupServiceName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.Service = service
	case errors.IsNotFound(err):
	default:
		return cke.EtcdBackupStatus{}, err
	}

	secret, err := clientset.CoreV1().Secrets("kube-system").Get(EtcdBackupSecretName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.Secret = secret
	case errors.IsNotFound(err):
	default:
		return cke.EtcdBackupStatus{}, err
	}

	job, err := clientset.BatchV1beta1().CronJobs("kube-system").Get(EtcdBackupJobName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.CronJob = job
	case errors.IsNotFound(err):
	default:
		return cke.EtcdBackupStatus{}, err
	}

	return s, nil
}

func checkHealthz(ctx context.Context, inf cke.Infrastructure, addr string, port uint16) (bool, error) {
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
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(body)) == "ok", nil
}

func checkSecureHealthz(ctx context.Context, inf cke.Infrastructure, addr string, port uint16) (bool, error) {
	url := "https://" + addr + ":" + strconv.FormatUint(uint64(port), 10) + "/healthz"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req = req.WithContext(ctx)
	client, err := inf.HTTPSClient(ctx)
	if err != nil {
		return false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(body)) == "ok", nil
}

func checkAPIServerHealth(ctx context.Context, inf cke.Infrastructure, n *cke.Node) (bool, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return false, err
	}
	_, err = clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}
