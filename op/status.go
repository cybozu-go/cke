package op

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/static"
	"github.com/cybozu-go/log"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	schedulerv1beta1 "k8s.io/kube-scheduler/config/v1beta1"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
)

var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

// GetNodeStatus returns NodeStatus.
func GetNodeStatus(ctx context.Context, inf cke.Infrastructure, node *cke.Node, cluster *cke.Cluster) (*cke.NodeStatus, error) {
	status := &cke.NodeStatus{}
	agent := inf.Agent(node.Address)
	status.SSHConnected = agent != nil
	if !status.SSHConnected {
		return status, nil
	}

	ce := inf.Engine(node.Address)
	ss, err := ce.Inspect([]string{
		EtcdContainerName,
		RiversContainerName,
		EtcdRiversContainerName,
		KubeAPIServerContainerName,
		KubeControllerManagerContainerName,
		KubeSchedulerContainerName,
		KubeProxyContainerName,
		KubeletContainerName,
	})
	if err != nil {
		return nil, err
	}

	etcdVolumeExists, err := ce.VolumeExists(EtcdVolumeName(cluster.Options.Etcd))
	if err != nil {
		return nil, err
	}

	isAddedmember, err := ce.VolumeExists(EtcdAddedMemberVolumeName)
	if err != nil {
		return nil, err
	}

	status.Etcd = cke.EtcdStatus{
		ServiceStatus: ss[EtcdContainerName],
		HasData:       etcdVolumeExists && isAddedmember,
	}
	status.Rivers = ss[RiversContainerName]
	status.EtcdRivers = ss[EtcdRiversContainerName]

	status.APIServer = cke.KubeComponentStatus{
		ServiceStatus: ss[KubeAPIServerContainerName],
		IsHealthy:     false,
	}
	if status.APIServer.Running {
		status.APIServer.IsHealthy, err = checkAPIServerHealth(ctx, inf, node)
		if err != nil {
			log.Warn("failed to check API server health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}
	}

	status.ControllerManager = cke.KubeComponentStatus{
		ServiceStatus: ss[KubeControllerManagerContainerName],
		IsHealthy:     false,
	}
	if status.ControllerManager.Running {
		status.ControllerManager.IsHealthy, err = checkSecureHealthz(ctx, inf, node.Address, 10257)
		if err != nil {
			log.Warn("failed to check controller manager health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}
	}

	status.Scheduler = cke.SchedulerStatus{
		ServiceStatus: ss[KubeSchedulerContainerName],
		IsHealthy:     false,
	}

	if status.Scheduler.Running {
		status.Scheduler.IsHealthy, err = checkSecureHealthz(ctx, inf, node.Address, 10259)
		if err != nil {
			log.Warn("failed to check scheduler health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}

		cfgData, _, err := agent.Run(fmt.Sprintf("cat %s", SchedulerConfigPath))
		if err != nil {
			log.Error("failed to cat "+SchedulerConfigPath, map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
			return nil, err
		}
		config := &schedulerv1beta1.KubeSchedulerConfiguration{}
		_, _, err = decUnstructured.Decode(cfgData, nil, config)
		if err == nil {
			// Nullify TypeMeta for later comparison using reflect.DeepEqual
			if config.APIVersion == schedulerv1beta1.SchemeGroupVersion.String() {
				config.TypeMeta = metav1.TypeMeta{}
			}
			status.Scheduler.Config = config
		}
	}

	// TODO: due to the following bug, health status cannot be checked for proxy.
	// https://github.com/kubernetes/kubernetes/issues/65118
	status.Proxy = cke.KubeComponentStatus{
		ServiceStatus: ss[KubeProxyContainerName],
		IsHealthy:     false,
	}
	status.Proxy.IsHealthy = status.Proxy.Running

	status.Kubelet = cke.KubeletStatus{
		ServiceStatus: ss[KubeletContainerName],
		IsHealthy:     false,
	}
	if status.Kubelet.Running {
		status.Kubelet.IsHealthy, err = CheckKubeletHealthz(ctx, inf, node.Address, 10248)
		if err != nil {
			log.Warn("failed to check kubelet health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}

		cfgData, _, err := agent.Run("cat /etc/kubernetes/kubelet/config.yml")
		if err == nil {
			var v kubeletv1beta1.KubeletConfiguration
			_, _, err = decUnstructured.Decode(cfgData, nil, &v)
			if err == nil {
				// Nullify TypeMeta for later comparison using reflect.DeepEqual
				if v.APIVersion == kubeletv1beta1.SchemeGroupVersion.String() {
					v.TypeMeta = metav1.TypeMeta{}
				}
				status.Kubelet.Config = &v
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

	ct, cancel := context.WithTimeout(ctx, TimeoutDuration)
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
	ct, cancel := context.WithTimeout(ctx, TimeoutDuration)
	defer cancel()
	resp, err := cli.MemberList(ct)
	if err != nil {
		return nil, err
	}
	members := make(map[string]*etcdserverpb.Member)
	for _, m := range resp.Members {
		name, err := GuessMemberName(m)
		if err != nil {
			return nil, err
		}
		members[name] = m
	}
	return members, nil
}

// GuessMemberName returns etcd member's ip address
func GuessMemberName(m *etcdserverpb.Member) (string, error) {
	if len(m.Name) > 0 {
		return m.Name, nil
	}

	if len(m.PeerURLs) == 0 {
		return "", errors.New("empty PeerURLs")
	}

	u, err := url.Parse(m.PeerURLs[0])
	if err != nil {
		return "", err
	}
	h, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return "", err
	}
	return h, nil
}

func getEtcdMemberInSync(ctx context.Context, inf cke.Infrastructure, address string, clusterRev int64) bool {
	endpoints := []string{fmt.Sprintf("https://%s:2379", address)}
	cli, err := inf.NewEtcdClient(ctx, endpoints)
	if err != nil {
		return false
	}
	defer cli.Close()

	ct, cancel := context.WithTimeout(ctx, TimeoutDuration)
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

	_, err = clientset.CoreV1().ServiceAccounts("kube-system").Get(ctx, "default", metav1.GetOptions{})
	switch {
	case err == nil:
		s.IsControlPlaneReady = true
	case k8serr.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	resp, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}
	s.Nodes = resp.Items

	if len(cluster.DNSService) > 0 {
		fields := strings.Split(cluster.DNSService, "/")
		if len(fields) != 2 {
			panic("invalid dns_service in cluster.yml")
		}
		svc, err := clientset.CoreV1().Services(fields[0]).Get(ctx, fields[1], metav1.GetOptions{})
		switch {
		case k8serr.IsNotFound(err):
		case err == nil:
			s.DNSService = svc
		default:
			return cke.KubernetesClusterStatus{}, err
		}
	}

	s.ClusterDNS, err = getClusterDNSStatus(ctx, inf, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}

	s.NodeDNS, err = getNodeDNSStatus(ctx, inf, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}

	epAPI := clientset.CoreV1().Endpoints
	ep, err := epAPI(metav1.NamespaceDefault).Get(ctx, "kubernetes", metav1.GetOptions{})
	switch {
	case err == nil:
		s.MasterEndpoints = ep
	case k8serr.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	svc, err := clientset.CoreV1().Services(metav1.NamespaceSystem).Get(ctx, EtcdServiceName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.EtcdService = svc
	case k8serr.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	ep, err = epAPI(metav1.NamespaceSystem).Get(ctx, EtcdEndpointsName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.EtcdEndpoints = ep
	case k8serr.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	resources, err := inf.Storage().GetAllResources(ctx)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}
	resources = append(resources, static.Resources...)

	cfg, err := inf.K8sConfig(ctx, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}

	s.ResourceStatuses = make(map[string]cke.ResourceStatus)
	for _, res := range resources {
		obj := &unstructured.Unstructured{}
		_, gvk, err := decUnstructured.Decode(res.Definition, nil, obj)
		if err != nil {
			return cke.KubernetesClusterStatus{}, err
		}

		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return cke.KubernetesClusterStatus{}, fmt.Errorf("failed to find rest mapping for %s: %w", gvk.String(), err)
		}

		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			ns := obj.GetNamespace()
			if ns == "" {
				return cke.KubernetesClusterStatus{}, fmt.Errorf("no namespace for %s: name=%s", gvk.String(), obj.GetName())
			}
			dr = dyn.Resource(mapping.Resource).Namespace(ns)
		} else {
			dr = dyn.Resource(mapping.Resource)
		}

		obj, err = dr.Get(ctx, obj.GetName(), metav1.GetOptions{})
		if k8serr.IsNotFound(err) {
			continue
		}
		if err != nil {
			return cke.KubernetesClusterStatus{}, err
		}
		s.SetResourceStatus(res.Key, obj.GetAnnotations(), len(obj.GetManagedFields()) != 0)
	}

	return s, nil
}

func getClusterDNSStatus(ctx context.Context, inf cke.Infrastructure, n *cke.Node) (cke.ClusterDNSStatus, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return cke.ClusterDNSStatus{}, err
	}

	s := cke.ClusterDNSStatus{}

	config, err := clientset.CoreV1().ConfigMaps("kube-system").Get(ctx, ClusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ConfigMap = config
	case k8serr.IsNotFound(err):
	default:
		return cke.ClusterDNSStatus{}, err
	}

	service, err := clientset.CoreV1().Services("kube-system").Get(ctx, ClusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ClusterIP = service.Spec.ClusterIP
	case k8serr.IsNotFound(err):
	default:
		return cke.ClusterDNSStatus{}, err
	}

	return s, nil
}

func getNodeDNSStatus(ctx context.Context, inf cke.Infrastructure, n *cke.Node) (cke.NodeDNSStatus, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return cke.NodeDNSStatus{}, err
	}

	s := cke.NodeDNSStatus{}

	config, err := clientset.CoreV1().ConfigMaps("kube-system").Get(ctx, NodeDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ConfigMap = config
	case k8serr.IsNotFound(err):
	default:
		return cke.NodeDNSStatus{}, err
	}

	return s, nil
}

// CheckKubeletHealthz checks that Kubelet is healthy
func CheckKubeletHealthz(ctx context.Context, inf cke.Infrastructure, addr string, port uint16) (bool, error) {
	healthzURL := "http://" + addr + ":" + strconv.FormatUint(uint64(port), 10) + "/healthz"
	req, err := http.NewRequest("GET", healthzURL, nil)
	if err != nil {
		return false, err
	}
	req = req.WithContext(ctx)
	resp, err := inf.HTTPClient().Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(body)) == "ok", nil
}

func checkSecureHealthz(ctx context.Context, inf cke.Infrastructure, addr string, port uint16) (bool, error) {
	healthzURL := "https://" + addr + ":" + strconv.FormatUint(uint64(port), 10) + "/healthz"
	req, err := http.NewRequest("GET", healthzURL, nil)
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

	body, err := io.ReadAll(resp.Body)
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
	_, err = clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func containCommandOption(slice []string, optionName string) bool {
	for _, v := range slice {
		switch {
		case v == optionName:
			return true
		case strings.HasPrefix(v, optionName+"="):
			return true
		case strings.HasPrefix(v, optionName+" "):
			return true
		}
	}
	return false
}
