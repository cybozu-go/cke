package server

import (
	"slices"
	"testing"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/clusterdns"
	"github.com/cybozu-go/cke/op/etcd"
	"github.com/cybozu-go/cke/op/k8s"
	"github.com/cybozu-go/cke/op/nodedns"
	"github.com/cybozu-go/cke/static"
	"github.com/google/go-cmp/cmp"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	proxyv1alpha1 "k8s.io/kube-proxy/config/v1alpha1"
	schedulerv1 "k8s.io/kube-scheduler/config/v1"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/ptr"
)

const (
	testClusterName      = "test"
	testServiceSubnet    = "12.34.56.0/24"
	testDefaultDNSDomain = "cluster.local"
	testDefaultDNSAddr   = "10.0.0.53"
)

var (
	testDefaultDNSServers = []string{"8.8.8.8"}
	testConstraints       = &cke.Constraints{
		ControlPlaneCount:        3,
		MinimumWorkers:           1,
		MaximumWorkers:           6,
		RebootMaximumUnreachable: 1,
	}
	testResources = []cke.ResourceDefinition{
		{
			Key:        "Namespace/foo",
			Kind:       "Namespace",
			Name:       "foo",
			Revision:   1,
			Definition: []byte(`{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"foo"}}`),
		},
	}
	nodeNames = []string{
		"10.0.0.11",
		"10.0.0.12",
		"10.0.0.13",
		"10.0.0.14",
		"10.0.0.15",
		"10.0.0.16",
	}
)

type testData struct {
	Cluster     *cke.Cluster
	Status      *cke.ClusterStatus
	Constraints *cke.Constraints
	Resources   []cke.ResourceDefinition
	RebootArgs  DecideOpsRebootArgs
}

func (d testData) ControlPlane() (nodes []*cke.Node) {
	for _, n := range d.Cluster.Nodes {
		if n.ControlPlane {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func (d testData) NonCPWorkers() (nodes []*cke.Node) {
	for _, n := range d.Cluster.Nodes {
		if !n.ControlPlane {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func (d testData) NodeStatus(n *cke.Node) *cke.NodeStatus {
	return d.Status.NodeStatuses[n.Address]
}

func newData() testData {
	cluster := &cke.Cluster{
		Name: testClusterName,
		Nodes: []*cke.Node{
			{Address: nodeNames[0], ControlPlane: true},
			{Address: nodeNames[1], ControlPlane: true},
			{Address: nodeNames[2], ControlPlane: true},
			{
				Address:     nodeNames[3],
				Labels:      map[string]string{"label1": "value"},
				Annotations: map[string]string{"annotation1": "value"},
				Taints: []corev1.Taint{
					{
						Key:    "taint1",
						Value:  "value1",
						Effect: corev1.TaintEffectNoSchedule,
					},
					{
						Key:    "taint2",
						Effect: corev1.TaintEffectPreferNoSchedule,
					},
				},
			},
			{Address: nodeNames[4]},
			{Address: nodeNames[5]},
		},
		ServiceSubnet: testServiceSubnet,
		DNSServers:    testDefaultDNSServers,
	}
	schedulerConfig := &unstructured.Unstructured{}
	schedulerConfig.SetGroupVersionKind(schedulerv1.SchemeGroupVersion.WithKind("KubeSchedulerConfiguration"))
	schedulerConfig.Object["parallelism"] = 999
	cluster.Options.Scheduler = cke.SchedulerParams{
		Config: schedulerConfig,
	}
	kubeletConfig := &unstructured.Unstructured{}
	kubeletConfig.SetGroupVersionKind(kubeletv1beta1.SchemeGroupVersion.WithKind("KubeletConfiguration"))
	kubeletConfig.Object["containerLogMaxSize"] = "20Mi"
	kubeletConfig.Object["clusterDomain"] = testDefaultDNSDomain
	cluster.Options.Kubelet = cke.KubeletParams{
		CRIEndpoint: "/var/run/k8s-containerd.sock",
		Config:      kubeletConfig,
	}

	nodeReadyStatus := corev1.NodeStatus{
		Conditions: []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			},
		},
	}

	nodeStatuses := make(map[string]*cke.NodeStatus)
	var nodeList []corev1.Node
	for _, nodeName := range nodeNames {
		nodeStatuses[nodeName] = &cke.NodeStatus{
			Etcd:         cke.EtcdStatus{ServiceStatus: cke.ServiceStatus{Running: false}, HasData: false, IsAddedMember: false},
			SSHConnected: true,
		}
		nodeList = append(nodeList, corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName,
			},
			Status: nodeReadyStatus,
		})
	}
	for i := 0; i < 3; i++ {
		nodeList[i].Labels = map[string]string{op.CKELabelMaster: "true"}
	}
	nodeList[3].Annotations = cluster.Nodes[3].Annotations
	nodeList[3].Labels = cluster.Nodes[3].Labels
	nodeList[3].Spec.Taints = cluster.Nodes[3].Taints
	status := &cke.ClusterStatus{
		ConfigVersion: cke.ConfigVersion,
		NodeStatuses:  nodeStatuses,
		Kubernetes: cke.KubernetesClusterStatus{
			ResourceStatuses: map[string]cke.ResourceStatus{
				"Namespace/foo": {Annotations: map[string]string{cke.AnnotationResourceRevision: "1"}},
			},
			Nodes: nodeList,
		},
		RepairQueue: cke.RepairQueueStatus{Enabled: true},
	}

	return testData{
		Cluster:     cluster,
		Status:      status,
		Constraints: testConstraints,
		Resources:   testResources,
	}
}

func (d testData) with(f func(data testData)) testData {
	f(d)
	return d
}

func (d testData) withResources(res []cke.ResourceDefinition) testData {
	d.Resources = res
	return d
}

func (d testData) withRivers() testData {
	for _, v := range d.Status.NodeStatuses {
		v.Rivers.Running = true
		v.Rivers.Image = cke.ToolsImage.Name()
		v.Rivers.BuiltInParams = op.RiversParams(d.ControlPlane(), op.RiversUpstreamPort, op.RiversListenPort)
	}
	return d
}

func (d testData) withEtcdRivers() testData {
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).EtcdRivers
		st.Running = true
		st.Image = cke.ToolsImage.Name()
		st.BuiltInParams = op.RiversParams(d.ControlPlane(), op.EtcdRiversUpstreamPort, op.EtcdRiversListenPort)
	}
	return d
}

func (d testData) withInitFailedEtcd() testData {
	for _, n := range d.ControlPlane() {
		d.NodeStatus(n).Etcd.HasData = true
	}
	return d
}

func (d testData) withStoppedEtcd() testData {
	for _, n := range d.ControlPlane() {
		d.NodeStatus(n).Etcd.HasData = true
		d.NodeStatus(n).Etcd.IsAddedMember = true
	}
	return d
}

func (d testData) withNotHasDataStoppedEtcd() testData {
	d.NodeStatus(d.ControlPlane()[0]).Etcd.HasData = false
	d.NodeStatus(d.ControlPlane()[0]).Etcd.IsAddedMember = false
	return d
}

func (d testData) withNotMarkedStoppedEtcd() testData {
	d.NodeStatus(d.ControlPlane()[0]).Etcd.IsAddedMember = false
	return d
}

func (d testData) withUnhealthyEtcd() testData {
	d.withStoppedEtcd()
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).Etcd
		st.Running = true
		st.Image = cke.EtcdImage.Name()
		st.BuiltInParams = etcd.BuiltInParams(n, nil, "")
	}
	return d
}

func (d testData) withHealthyEtcd() testData {
	d.withUnhealthyEtcd()
	st := &d.Status.Etcd
	st.IsHealthy = true
	st.Members = make(map[string]*etcdserverpb.Member)
	st.InSyncMembers = make(map[string]bool)
	for i, n := range d.ControlPlane() {
		st.Members[n.Address] = &etcdserverpb.Member{
			ID:   uint64(i),
			Name: n.Address,
		}
		st.InSyncMembers[n.Address] = true
	}
	return d
}

func (d testData) withAPIServer(serviceSubnet, domain string) testData {
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).APIServer
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.KubernetesImage.Name()
		st.BuiltInParams = k8s.APIServerParams(n.Address, serviceSubnet, false, "", "", domain)
	}
	return d
}

func (d testData) withControllerManager(name, serviceSubnet string) testData {
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).ControllerManager
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.KubernetesImage.Name()
		st.BuiltInParams = k8s.ControllerManagerParams(name, serviceSubnet)
	}
	return d
}

func (d testData) withScheduler() testData {
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).Scheduler
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.KubernetesImage.Name()
		st.BuiltInParams = k8s.SchedulerParams()

		leaderElect := true
		st.Config = &schedulerv1.KubeSchedulerConfiguration{}
		st.Config.Parallelism = ptr.To(int32(999))
		st.Config.ClientConnection.Kubeconfig = op.SchedulerKubeConfigPath
		st.Config.LeaderElection.LeaderElect = &leaderElect
	}
	return d
}

func (d testData) withKubelet(domain, dns string, allowSwap bool) testData {
	for _, n := range d.Cluster.Nodes {
		st := &d.NodeStatus(n).Kubelet
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.KubernetesImage.Name()
		st.BuiltInParams = k8s.KubeletServiceParams(n, cke.KubeletParams{
			CRIEndpoint: "/var/run/k8s-containerd.sock",
		})

		webhookEnabled := true
		st.Config = &kubeletv1beta1.KubeletConfiguration{
			ClusterDomain:         domain,
			RuntimeRequestTimeout: metav1.Duration{Duration: 15 * time.Minute},
			HealthzBindAddress:    "0.0.0.0",
			VolumePluginDir:       "/opt/volume/bin",
			ContainerLogMaxSize:   "20Mi",
			TLSCertFile:           "/etc/kubernetes/pki/kubelet.crt",
			TLSPrivateKeyFile:     "/etc/kubernetes/pki/kubelet.key",
			Authentication: kubeletv1beta1.KubeletAuthentication{
				X509:    kubeletv1beta1.KubeletX509Authentication{ClientCAFile: "/etc/kubernetes/pki/ca.crt"},
				Webhook: kubeletv1beta1.KubeletWebhookAuthentication{Enabled: &webhookEnabled},
			},
			Authorization: kubeletv1beta1.KubeletAuthorization{Mode: kubeletv1beta1.KubeletAuthorizationModeWebhook},
			ClusterDNS:    []string{n.Address},
		}
		if allowSwap {
			failSwapOn := !allowSwap
			st.Config.FailSwapOn = &failSwapOn
		}
	}
	return d
}

func (d testData) withProxy() testData {
	for _, n := range d.Cluster.Nodes {
		st := &d.NodeStatus(n).Proxy
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.KubernetesImage.Name()
		st.BuiltInParams = k8s.ProxyParams()
		st.Config = &proxyv1alpha1.KubeProxyConfiguration{}
		st.Config.HostnameOverride = n.Nodename()
		st.Config.MetricsBindAddress = "0.0.0.0"
		st.Config.Conntrack = proxyv1alpha1.KubeProxyConntrackConfiguration{
			TCPEstablishedTimeout: &metav1.Duration{Duration: 24 * time.Hour},
			TCPCloseWaitTimeout:   &metav1.Duration{Duration: 1 * time.Hour},
		}
		st.Config.ClientConnection.Kubeconfig = "/etc/kubernetes/proxy/kubeconfig"
	}
	return d
}

func (d testData) withAllServices() testData {
	d.withRivers()
	d.withEtcdRivers()
	d.withHealthyEtcd()
	d.withAPIServer(testServiceSubnet, testDefaultDNSDomain)
	d.withControllerManager(testClusterName, testServiceSubnet)
	d.withScheduler()
	d.withKubelet(testDefaultDNSDomain, testDefaultDNSAddr, false)
	d.withProxy()
	return d
}

func (d testData) withK8sReady() testData {
	for i, n := range d.Status.Kubernetes.Nodes {
		n.Status.Conditions = append(n.Status.Conditions, corev1.NodeCondition{
			Type:   corev1.NodeReady,
			Status: corev1.ConditionTrue,
		})
		d.Status.Kubernetes.Nodes[i] = n
	}

	d.withAllServices()
	d.Status.Kubernetes.IsControlPlaneReady = true
	return d
}

func (d testData) withK8sResourceReady() testData {
	d.withK8sReady()
	ks := &d.Status.Kubernetes
	for _, res := range static.Resources {
		ks.ResourceStatuses[res.Key] = cke.ResourceStatus{
			Annotations: map[string]string{cke.AnnotationResourceRevision: "1"},
		}
	}
	ks.ResourceStatuses["ClusterRole/system:cluster-dns"].Annotations[cke.AnnotationResourceRevision] = "2"
	ks.ResourceStatuses["Deployment/kube-system/cluster-dns"].Annotations[cke.AnnotationResourceImage] = cke.CoreDNSImage.Name()
	ks.ResourceStatuses["Deployment/kube-system/cluster-dns"].Annotations[cke.AnnotationResourceRevision] = "4"
	ks.ResourceStatuses["DaemonSet/kube-system/node-dns"].Annotations[cke.AnnotationResourceImage] = cke.UnboundImage.Name() + "," + cke.UnboundExporterImage.Name()
	ks.ResourceStatuses["DaemonSet/kube-system/node-dns"].Annotations[cke.AnnotationResourceRevision] = "4"
	ks.ClusterDNS.ConfigMap = clusterdns.ConfigMap(testDefaultDNSDomain, testDefaultDNSServers)
	ks.ClusterDNS.ClusterIP = testDefaultDNSAddr
	ks.NodeDNS.ConfigMap = nodedns.ConfigMap(testDefaultDNSAddr, testDefaultDNSDomain, testDefaultDNSServers, true)

	ks.MasterEndpoints = &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"endpointslice.kubernetes.io/skip-mirror": "true",
			},
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{IP: "10.0.0.11"},
					{IP: "10.0.0.12"},
					{IP: "10.0.0.13"},
				},
				Ports: []corev1.EndpointPort{{Name: "https", Port: 6443}},
			},
		},
	}
	endpointReady := true
	masterPortName := "https"
	var masterPort int32 = 6443
	ks.MasterEndpointSlice = &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"endpointslice.kubernetes.io/managed-by": "cke.cybozu.com",
				"kubernetes.io/service-name":             "kubernetes",
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{
					"10.0.0.11",
				},
				Conditions: discoveryv1.EndpointConditions{Ready: &endpointReady},
			},
			{
				Addresses: []string{
					"10.0.0.12",
				},
				Conditions: discoveryv1.EndpointConditions{Ready: &endpointReady},
			},
			{
				Addresses: []string{
					"10.0.0.13",
				},
				Conditions: discoveryv1.EndpointConditions{Ready: &endpointReady},
			},
		},
		Ports: []discoveryv1.EndpointPort{{Name: &masterPortName, Port: &masterPort}},
	}
	ks.EtcdService = &corev1.Service{
		Spec: corev1.ServiceSpec{
			Ports:     []corev1.ServicePort{{Port: 2379}},
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: corev1.ClusterIPNone,
		},
	}
	ks.EtcdEndpoints = &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"endpointslice.kubernetes.io/skip-mirror": "true",
			},
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{IP: "10.0.0.11"},
					{IP: "10.0.0.12"},
					{IP: "10.0.0.13"},
				},
				Ports: []corev1.EndpointPort{{Port: 2379}},
			},
		},
	}
	etcdPortName := ""
	var etcdPort int32 = 2379
	ks.EtcdEndpointSlice = &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"endpointslice.kubernetes.io/managed-by": "cke.cybozu.com",
				"kubernetes.io/service-name":             "cke-etcd",
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{
					"10.0.0.11",
				},
				Conditions: discoveryv1.EndpointConditions{Ready: &endpointReady},
			},
			{
				Addresses: []string{
					"10.0.0.12",
				},
				Conditions: discoveryv1.EndpointConditions{Ready: &endpointReady},
			},
			{
				Addresses: []string{
					"10.0.0.13",
				},
				Conditions: discoveryv1.EndpointConditions{Ready: &endpointReady},
			},
		},
		Ports: []discoveryv1.EndpointPort{{Name: &etcdPortName, Port: &etcdPort}},
	}

	return d
}

func (d testData) withNotReadyEndpoint(i int) testData {
	masterAddresses := d.Status.Kubernetes.MasterEndpoints.Subsets[0].Addresses
	etcdAddresses := d.Status.Kubernetes.EtcdEndpoints.Subsets[0].Addresses
	if i < 0 || i >= len(masterAddresses) {
		return d
	}
	d.Status.Kubernetes.MasterEndpoints.Subsets[0].Addresses = masterAddresses[0:i]
	d.Status.Kubernetes.MasterEndpoints.Subsets[0].NotReadyAddresses = masterAddresses[i:3]
	d.Status.Kubernetes.EtcdEndpoints.Subsets[0].Addresses = etcdAddresses[0:i]
	d.Status.Kubernetes.EtcdEndpoints.Subsets[0].NotReadyAddresses = etcdAddresses[i:3]
	endpointReady := false
	d.Status.Kubernetes.MasterEndpointSlice.Endpoints[i].Conditions.Ready = &endpointReady
	d.Status.Kubernetes.EtcdEndpointSlice.Endpoints[i].Conditions.Ready = &endpointReady
	return d
}

func (d testData) withNodes(nodes ...corev1.Node) testData {
	d.withK8sResourceReady()
OUTER:
	for _, argNode := range nodes {
		for i, testDataNode := range d.Status.Kubernetes.Nodes {
			if testDataNode.Name == argNode.Name {
				d.Status.Kubernetes.Nodes[i] = argNode
				continue OUTER
			}
		}
		d.Status.Kubernetes.Nodes = append(d.Status.Kubernetes.Nodes, argNode)
	}
	return d
}

func (d testData) withSSHNotConnectedCP() testData {
	n := d.ControlPlane()[0]
	st := d.NodeStatus(n)
	d.Status.NodeStatuses[n.Address] = &cke.NodeStatus{
		Labels: st.Labels,
	}

	return d
}

func (d testData) withSSHNotConnectedNonCPWorker(num int) testData {
	// If num is larger than num of non-cp worker, all of them treat as unreachable
	if num > len(d.NonCPWorkers()) {
		num = len(d.NonCPWorkers())
	}

	for i, n := range d.NonCPWorkers() {
		if i > num-1 {
			break
		}
		st := d.NodeStatus(n)
		d.Status.NodeStatuses[n.Address] = &cke.NodeStatus{
			Labels: st.Labels,
		}
	}

	return d
}

func (d testData) withSSHNotConnectedNodes() testData {
	d.withSSHNotConnectedCP()
	d.withSSHNotConnectedNonCPWorker(1)
	return d
}

func (d testData) withRebootCordon(i int) testData {
	if i < 0 || i >= len(d.Status.Kubernetes.Nodes) {
		return d
	}
	d.Status.Kubernetes.Nodes[i].Spec.Unschedulable = true
	d.Status.Kubernetes.Nodes[i].Annotations = map[string]string{
		op.CKEAnnotationReboot: "true",
	}
	return d
}

func (d testData) withRepairConfig() testData {
	d.Cluster.Repair = cke.Repair{
		RepairProcedures: []cke.RepairProcedure{
			{
				MachineTypes: []string{"type1"},
				RepairOperations: []cke.RepairOperation{
					{
						Operation: "op1",
						RepairSteps: []cke.RepairStep{
							{RepairCommand: []string{"repair0"}},
							{RepairCommand: []string{"repair1"}},
						},
					},
				},
			},
		},
	}
	return d
}

func (d testData) withRepairEntries(entries []*cke.RepairQueueEntry) testData {
	for i, entry := range entries {
		entry.Index = int64(i)
		if slices.Contains(nodeNames, entry.Address) {
			entry.Nodename = entry.Address
		}
		if entry.Status == "" {
			entry.Status = cke.RepairStatusQueued
		}
		if entry.StepStatus == "" {
			entry.StepStatus = cke.RepairStepStatusWaiting
		}
	}
	d.Status.RepairQueue.Entries = entries
	return d
}

func (d testData) withRepairDisabled() testData {
	d.Status.RepairQueue.Enabled = false
	return d
}

func (d testData) withRebootConfig() testData {
	d.Cluster.Reboot.RebootCommand = []string{"reboot"}
	d.Cluster.Reboot.BootCheckCommand = []string{"true"}
	return d
}

func (d testData) withRebootEntries(entries []*cke.RebootQueueEntry) testData {
	d.RebootArgs.RQEntries = entries
	return d
}

func (d testData) withNewlyDrained(entries []*cke.RebootQueueEntry) testData {
	d.RebootArgs.NewlyDrained = entries
	return d
}

func (d testData) withDrainCompleted(entries []*cke.RebootQueueEntry) testData {
	d.RebootArgs.DrainCompleted = entries
	return d
}

func (d testData) withDrainTimedout(entries []*cke.RebootQueueEntry) testData {
	d.RebootArgs.DrainTimedout = entries
	return d
}

func (d testData) withRebootDequeued(entries []*cke.RebootQueueEntry) testData {
	d.RebootArgs.RebootDequeued = entries
	return d
}

func (d testData) withRebootCancelled(entries []*cke.RebootQueueEntry) testData {
	d.RebootArgs.RebootCancelled = entries
	return d
}

func (d testData) withDisableProxy() testData {
	d.Cluster.Options.Proxy.Disable = true
	return d
}

type opData struct {
	Name      string
	TargetNum int
}

func TestDecideOps(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Name        string
		Input       testData
		ExpectedOps []opData
	}{
		{
			Name:        "BootRivers",
			Input:       newData(),
			ExpectedOps: []opData{{"rivers-bootstrap", 5}, {"etcd-rivers-bootstrap", 3}},
		},
		{
			Name:        "BootRivers2",
			Input:       newData().withHealthyEtcd().withSSHNotConnectedNodes(),
			ExpectedOps: []opData{{"rivers-bootstrap", 4}, {"etcd-rivers-bootstrap", 2}},
		},
		{
			Name: "RestartRivers",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.Image = ""
				d.NodeStatus(d.ControlPlane()[1]).Rivers.Image = ""
			}).withEtcdRivers().withHealthyEtcd().withSSHNotConnectedNodes(),
			ExpectedOps: []opData{{"rivers-restart", 1}},
		},
		{
			Name: "RestartRivers2",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.BuiltInParams.ExtraArguments = nil
				d.NodeStatus(d.ControlPlane()[1]).Rivers.BuiltInParams.ExtraArguments = nil
			}).withEtcdRivers().withHealthyEtcd().withSSHNotConnectedNodes(),
			ExpectedOps: []opData{{"rivers-restart", 1}},
		},
		{
			Name: "RestartRivers3",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.ExtraParams.ExtraArguments = []string{"foo"}
				d.NodeStatus(d.ControlPlane()[1]).Rivers.ExtraParams.ExtraArguments = []string{"foo"}
			}).withEtcdRivers().withHealthyEtcd().withSSHNotConnectedNodes(),
			ExpectedOps: []opData{{"rivers-restart", 1}},
		},
		{
			Name: "StartRestartRivers",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.Image = ""
				d.NodeStatus(d.ControlPlane()[1]).Rivers.Running = false
			}).withEtcdRivers(),
			ExpectedOps: []opData{{"rivers-bootstrap", 1}, {"rivers-restart", 1}},
		},
		{
			Name: "RestartEtcdRivers",
			Input: newData().withRivers().withEtcdRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).EtcdRivers.Image = ""
				d.NodeStatus(d.ControlPlane()[1]).EtcdRivers.Image = ""
			}).withHealthyEtcd().withSSHNotConnectedNodes(),
			ExpectedOps: []opData{{"etcd-rivers-restart", 1}},
		},
		{
			Name: "RestartEtcdRivers2",
			Input: newData().withRivers().withEtcdRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).EtcdRivers.BuiltInParams.ExtraArguments = nil
				d.NodeStatus(d.ControlPlane()[1]).EtcdRivers.BuiltInParams.ExtraArguments = nil
			}).withHealthyEtcd().withSSHNotConnectedNodes(),
			ExpectedOps: []opData{{"etcd-rivers-restart", 1}},
		},
		{
			Name: "RestartEtcdRivers3",
			Input: newData().withRivers().withEtcdRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).EtcdRivers.ExtraParams.ExtraArguments = []string{"foo"}
				d.NodeStatus(d.ControlPlane()[1]).EtcdRivers.ExtraParams.ExtraArguments = []string{"foo"}
			}).withHealthyEtcd().withSSHNotConnectedNodes(),
			ExpectedOps: []opData{{"etcd-rivers-restart", 1}},
		},
		{
			Name: "StartRestartEtcdRivers",
			Input: newData().withRivers().withEtcdRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).EtcdRivers.Image = ""
				d.NodeStatus(d.ControlPlane()[1]).EtcdRivers.Running = false
			}),
			ExpectedOps: []opData{{"etcd-rivers-bootstrap", 1}, {"etcd-rivers-restart", 1}},
		},
		{
			Name:        "EtcdBootstrap",
			Input:       newData().withRivers().withEtcdRivers(),
			ExpectedOps: []opData{{"etcd-bootstrap", 3}},
		},
		{
			Name:        "SkipEtcdBootstrap",
			Input:       newData().withRivers().withEtcdRivers().withSSHNotConnectedNodes(),
			ExpectedOps: nil,
		},
		{
			Name:        "SkipEtcdBootstrap2",
			Input:       newData().withRivers().withEtcdRivers().withInitFailedEtcd(),
			ExpectedOps: []opData{{"etcd-wait-cluster", 3}},
			// This wait will never succeed.
			// The recovery from this failure case may need deletion of the data volumes, so it should be handled manually.
			// The failure case is very rare.  It will not occur once after the etcd cluster started to work.
		},
		{
			Name:        "EtcdStart",
			Input:       newData().withRivers().withEtcdRivers().withStoppedEtcd(),
			ExpectedOps: []opData{{"etcd-start", 3}},
		},
		{
			Name: "EtcdStart2",
			Input: newData().withRivers().withEtcdRivers().withHealthyEtcd().withSSHNotConnectedNodes().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Etcd.Running = false
				d.NodeStatus(d.ControlPlane()[1]).Etcd.Running = false
			}),
			ExpectedOps: []opData{
				{"etcd-start", 1},
			},
		},
		{
			Name:  "EtcdStart3",
			Input: newData().withRivers().withEtcdRivers().withStoppedEtcd().withNotHasDataStoppedEtcd(),
			ExpectedOps: []opData{
				{"etcd-start", 2},
			},
		},
		{
			Name:  "EtcdStart4",
			Input: newData().withRivers().withEtcdRivers().withStoppedEtcd().withNotMarkedStoppedEtcd(),
			ExpectedOps: []opData{
				{"etcd-start", 2},
			},
		},
		{
			Name:        "WaitEtcd",
			Input:       newData().withRivers().withEtcdRivers().withUnhealthyEtcd(),
			ExpectedOps: []opData{{"etcd-wait-cluster", 3}},
		},
		{
			Name:  "BootK8s",
			Input: newData().withHealthyEtcd().withRivers().withEtcdRivers().withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-apiserver-restart", 2},
				{"kube-controller-manager-bootstrap", 2},
				{"kube-scheduler-bootstrap", 2},
				{"kubelet-bootstrap", 4},
				{"kube-proxy-bootstrap", 4},
			},
		},
		{
			Name:  "BootK8sFromPartiallyRunning",
			Input: newData().withHealthyEtcd().withRivers().withEtcdRivers().withAPIServer(testServiceSubnet, testDefaultDNSDomain),
			ExpectedOps: []opData{
				{"kube-controller-manager-bootstrap", 3},
				{"kube-scheduler-bootstrap", 3},
				{"kubelet-bootstrap", 5},
				{"kube-proxy-bootstrap", 5},
			},
		},
		{
			Name:  "BootK8sWithoutProxy",
			Input: newData().withHealthyEtcd().withRivers().withEtcdRivers().withSSHNotConnectedNodes().withDisableProxy(),
			ExpectedOps: []opData{
				{"kube-apiserver-restart", 2},
				{"kube-controller-manager-bootstrap", 2},
				{"kube-scheduler-bootstrap", 2},
				{"kubelet-bootstrap", 4},
			},
		},
		{
			Name:  "RestartAPIServer",
			Input: newData().withAllServices().withAPIServer("11.22.33.0/24", testDefaultDNSDomain).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-apiserver-restart", 2},
			},
		},
		{
			Name: "RestartAPIServer2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).APIServer.Image = ""
				d.NodeStatus(d.ControlPlane()[1]).APIServer.Image = ""
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-apiserver-restart", 1},
			},
		},
		{
			Name: "RestartAPIServer3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).APIServer.ExtraParams.ExtraArguments = []string{"foo"}
				d.NodeStatus(d.ControlPlane()[1]).APIServer.ExtraParams.ExtraArguments = []string{"foo"}
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-apiserver-restart", 1},
			},
		},
		{
			Name:  "RestartControllerManager",
			Input: newData().withAllServices().withControllerManager("another", testServiceSubnet).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-controller-manager-restart", 2},
			},
		},
		{
			Name: "RestartControllerManager2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).ControllerManager.Image = ""
				d.NodeStatus(d.ControlPlane()[1]).ControllerManager.Image = ""
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-controller-manager-restart", 1},
			},
		},
		{
			Name: "RestartControllerManager3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).ControllerManager.ExtraParams.ExtraArguments = []string{"foo"}
				d.NodeStatus(d.ControlPlane()[1]).ControllerManager.ExtraParams.ExtraArguments = []string{"foo"}
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-controller-manager-restart", 1},
			},
		},
		{
			Name: "RestartScheduler",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Scheduler.BuiltInParams.ExtraArguments = []string{"foo"}
				d.NodeStatus(d.ControlPlane()[1]).Scheduler.BuiltInParams.ExtraArguments = []string{"foo"}
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-scheduler-restart", 1},
			},
		},
		{
			Name: "RestartScheduler2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Scheduler.Image = ""
				d.NodeStatus(d.ControlPlane()[1]).Scheduler.Image = ""
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-scheduler-restart", 1},
			},
		},
		{
			Name: "RestartScheduler3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Scheduler.ExtraParams.ExtraArguments = []string{"foo"}
				d.NodeStatus(d.ControlPlane()[1]).Scheduler.ExtraParams.ExtraArguments = []string{"foo"}
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-scheduler-restart", 1},
			},
		},
		{
			Name: "RestartScheduler4",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Scheduler.Config.Parallelism = nil
				d.NodeStatus(d.ControlPlane()[1]).Scheduler.Config.Parallelism = nil
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-scheduler-restart", 1},
			},
		},
		{
			Name:  "RestartKubelet",
			Input: newData().withAllServices().withKubelet("foo.local", "10.0.0.53", false).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kubelet-restart", 4},
			},
		},
		{
			Name:  "RestartKubelet2",
			Input: newData().withAllServices().withKubelet("", "10.0.0.53", true).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kubelet-restart", 4},
			},
		},
		{
			Name: "RestartKubelet3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[3]).Kubelet.Image = ""
				d.NodeStatus(d.Cluster.Nodes[4]).Kubelet.Image = ""
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kubelet-restart", 1},
			},
		},
		{
			Name: "RestartKubelet4",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[3]).Kubelet.ExtraParams.ExtraArguments = []string{"foo"}
				d.NodeStatus(d.Cluster.Nodes[4]).Kubelet.ExtraParams.ExtraArguments = []string{"foo"}
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kubelet-restart", 1},
			},
		},
		{
			Name: "RestartKubelet5",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[3]).Kubelet.Config.ClusterDomain = "neco.local"
				d.NodeStatus(d.Cluster.Nodes[4]).Kubelet.Config.ClusterDomain = "neco.local"
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kubelet-restart", 1},
			},
		},
		{
			Name: "RestartKubelet6",
			Input: newData().withAllServices().with(func(d testData) {
				d.Cluster.Options.Kubelet.Config.Object["containerLogMaxFiles"] = 20
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kubelet-restart", 4},
			},
		},
		{
			Name: "RestartKubelet7",
			Input: newData().withAllServices().with(func(d testData) {
				d.Cluster.Options.Kubelet.Config.Object["containerLogMaxSize"] = "1Gi"
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kubelet-restart", 4},
			},
		},
		{
			Name: "RestartKubelet8",
			Input: newData().withAllServices().with(func(d testData) {
				d.Cluster.Options.Kubelet.CRIEndpoint = "/var/run/dockershim.sock"
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"wait-kubernetes", 1},
			},
		},
		{
			Name: "RestartKubelet9",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Kubernetes.Nodes = d.Status.Kubernetes.Nodes[:3]
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kubelet-restart", 2},
			},
		},
		{
			Name: "RestartProxy",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[3]).Proxy.BuiltInParams.ExtraArguments = []string{"foo"}
				d.NodeStatus(d.Cluster.Nodes[4]).Proxy.BuiltInParams.ExtraArguments = []string{"foo"}
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-proxy-restart", 1},
			},
		},
		{
			Name: "RestartProxy2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[3]).Proxy.Image = ""
				d.NodeStatus(d.Cluster.Nodes[4]).Proxy.Image = ""
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-proxy-restart", 1},
			},
		},
		{
			Name: "RestartProxy3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[3]).Proxy.ExtraParams.ExtraArguments = []string{"foo"}
				d.NodeStatus(d.Cluster.Nodes[4]).Proxy.ExtraParams.ExtraArguments = []string{"foo"}
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-proxy-restart", 1},
			},
		},
		{
			Name: "RestartProxy4",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[3]).Proxy.Config.Mode = cke.ProxyModeIPVS
				d.NodeStatus(d.Cluster.Nodes[4]).Proxy.Config.Mode = cke.ProxyModeIPVS
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{
				{"kube-proxy-restart", 1},
			},
		},
		{
			Name:  "StopProxy",
			Input: newData().withAllServices().withDisableProxy(),
			ExpectedOps: []opData{
				{"stop-kube-proxy", 5},
			},
		},
		{
			Name:        "WaitKube",
			Input:       newData().withAllServices(),
			ExpectedOps: []opData{{"wait-kubernetes", 1}},
		},
		{
			Name:  "K8sResources",
			Input: newData().withK8sReady(),
			ExpectedOps: []opData{
				{"resource-apply", 1},
				{"resource-apply", 1},
				{"resource-apply", 1},
				{"resource-apply", 1},
				{"resource-apply", 1},
				{"resource-apply", 1},
				{"resource-apply", 1},
				{"resource-apply", 1},
				{"resource-apply", 1},
				{"create-cluster-dns-configmap", 1},
				{"create-endpoints", 1},
				{"create-endpointslice", 1},
				{"create-etcd-service", 1},
				{"create-endpoints", 1},
				{"create-endpointslice", 1},
			},
		},
		{
			Name: "UpdateDNSService",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				svc := &corev1.Service{}
				svc.Spec.ClusterIP = "1.1.1.1"
				d.Status.Kubernetes.DNSService = svc
			}),
			ExpectedOps: []opData{
				{"update-cluster-dns-configmap", 1},
				{"update-node-dns-configmap", 1},
			},
		},
		{
			Name: "DNSUpdate1",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Cluster.Options.Kubelet.Config.Object["clusterDomain"] = "neco.local"
			}),
			ExpectedOps: []opData{
				{"kube-apiserver-restart", 3},
				{"kubelet-restart", 5},
			},
		},
		{
			Name: "DNSUpdate2",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Cluster.Options.Kubelet.Config.Object["clusterDomain"] = "neco.local"
				for _, st := range d.Status.NodeStatuses {
					st.Kubelet.Config.ClusterDomain = "neco.local"
				}
			}).withAPIServer(testServiceSubnet, "neco.local"),
			ExpectedOps: []opData{
				{"update-cluster-dns-configmap", 1},
				{"update-node-dns-configmap", 1},
			},
		},
		{
			Name: "DNSUpdate3",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Cluster.DNSServers = []string{"1.1.1.1"}
			}),
			ExpectedOps: []opData{
				{"update-cluster-dns-configmap", 1},
				{"update-node-dns-configmap", 1},
			},
		},
		{
			Name: "NodeDNSUpdate",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.ClusterDNS.ClusterIP = "10.0.0.54"
			}),
			ExpectedOps: []opData{
				{"update-node-dns-configmap", 1},
			},
		},
		{
			Name: "MasterEndpointsUpdate1",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.MasterEndpoints.Subsets = []corev1.EndpointSubset{}
			}),
			ExpectedOps: []opData{{"update-endpoints", 1}},
		},
		{
			Name: "MasterEndpointsUpdate2",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.MasterEndpoints.Subsets[0].Ports = []corev1.EndpointPort{}
			}),
			ExpectedOps: []opData{{"update-endpoints", 1}},
		},
		{
			Name: "MasterEndpointsUpdate3",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.MasterEndpoints.Subsets[0].Addresses = []corev1.EndpointAddress{}
			}),
			ExpectedOps: []opData{{"update-endpoints", 1}},
		},
		{
			Name: "MasterEndpointSliceUpdate1",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.MasterEndpointSlice.Endpoints[0].Addresses = []string{}
			}),
			ExpectedOps: []opData{{"update-endpointslice", 1}},
		},
		{
			Name: "MasterEndpointSliceUpdate2",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.MasterEndpointSlice.Ports[0] = discoveryv1.EndpointPort{}
			}),
			ExpectedOps: []opData{{"update-endpointslice", 1}},
		},
		{
			Name: "EtcdServiceUpdate",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.EtcdService.Spec.Ports = []corev1.ServicePort{}
			}),
			ExpectedOps: []opData{{"update-etcd-service", 1}},
		},
		{
			Name: "EtcdEndpointsUpdate1",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.EtcdEndpoints.Subsets = []corev1.EndpointSubset{}
			}),
			ExpectedOps: []opData{{"update-endpoints", 1}},
		},
		{
			Name: "EtcdEndpointsUpdate2",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.EtcdEndpoints.Subsets[0].Ports = []corev1.EndpointPort{}
			}),
			ExpectedOps: []opData{{"update-endpoints", 1}},
		},
		{
			Name: "EtcdEndpointsUpdate3",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.EtcdEndpoints.Subsets[0].Addresses = []corev1.EndpointAddress{}
			}),
			ExpectedOps: []opData{{"update-endpoints", 1}},
		},
		{
			Name: "EtcdEndpointSliceUpdate1",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.EtcdEndpointSlice.Endpoints[0].Addresses = []string{}
			}),
			ExpectedOps: []opData{{"update-endpointslice", 1}},
		},
		{
			Name: "EtcdEndpointSliceUpdate2",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.EtcdEndpointSlice.Ports[0] = discoveryv1.EndpointPort{}
			}),
			ExpectedOps: []opData{{"update-endpointslice", 1}},
		},
		{
			Name: "EndpointsUpdateWithRebootEntry",
			Input: newData().withK8sResourceReady().withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[2],
					Status: cke.RebootStatusDraining,
				},
			}),
			ExpectedOps: []opData{
				{"update-endpoints", 1},
				{"update-endpointslice", 1},
				{"update-endpoints", 1},
				{"update-endpointslice", 1},
			},
		},
		{
			Name: "EndpointsUpdateWithRebootEntry2",
			Input: newData().withK8sResourceReady().withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[2],
					Status: cke.RebootStatusQueued,
				},
			}).withNewlyDrained([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[2],
					Status: cke.RebootStatusQueued,
				},
			}),
			ExpectedOps: []opData{
				{"update-endpoints", 1},
				{"update-endpointslice", 1},
				{"update-endpoints", 1},
				{"update-endpointslice", 1},
			},
		},
		{
			Name: "EndpointsWithRebootEntry",
			Input: newData().withK8sResourceReady().withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[2],
					Status: cke.RebootStatusQueued,
				},
			}).withNewlyDrained([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[2],
					Status: cke.RebootStatusQueued,
				},
			}).withNotReadyEndpoint(2),
			ExpectedOps: []opData{{"reboot-drain-start", 1}},
		},
		{
			Name: "EndpointsWithCancelledRebootEntry",
			Input: newData().withK8sResourceReady().withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[2],
					Status: cke.RebootStatusCancelled,
				},
			}).withRebootCancelled([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[2],
					Status: cke.RebootStatusCancelled,
				},
			}),
			ExpectedOps: []opData{{"reboot-dequeue", 1}},
		},
		{
			Name: "UserResourceAdd",
			Input: newData().withK8sResourceReady().withResources(
				append(testResources, cke.ResourceDefinition{
					Key:        "ConfigMap/foo/bar",
					Kind:       "ConfigMap",
					Namespace:  "foo",
					Name:       "bar",
					Revision:   1,
					Definition: []byte(`{"apiversion":"v1","kind":"ConfigMap","metadata":{"namespace":"foo","name":"bar"},"data":{"a":"b"}}`),
				})),
			ExpectedOps: []opData{
				{"resource-apply", 1},
			},
		},
		{
			Name: "UserResourceUpdate",
			Input: newData().withK8sResourceReady().withResources(
				[]cke.ResourceDefinition{{
					Key:        "Namespace/foo",
					Kind:       "Namespace",
					Name:       "foo",
					Revision:   2,
					Definition: []byte(`{"apiversion":"v1","kind":"Namespace","metadata":{"name":"foo"}}`),
				}}),
			ExpectedOps: []opData{
				{"resource-apply", 1},
			},
		},
		{
			Name: "NodeLabel1",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "10.0.0.14",
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeLabel2",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "10.0.0.14",
					Labels:      map[string]string{"label1": "wrongvalue"},
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeLabel3",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "10.0.0.14",
					Labels: map[string]string{
						"label1":             "value",
						"cke.cybozu.com/foo": "bar",
					},
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeLabel4",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "10.0.0.14",
					Labels: map[string]string{
						"label1":                     "value",
						"sabakan.cke.cybozu.com/foo": "bar",
					},
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeLabel5",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "10.0.0.14",
					Labels: map[string]string{
						"label1":                         "value",
						"node-role.kubernetes.io/worker": "true",
					},
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeLabelCP1",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.11",
					Labels: map[string]string{},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeLabelCP2",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.14",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeLabelCP3",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.11",
					Labels: map[string]string{op.CKELabelMaster: "hoge"},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeAnnotation1",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.14",
					Labels: map[string]string{"label1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeAnnotation2",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "10.0.0.14",
					Labels:      map[string]string{"label1": "value"},
					Annotations: map[string]string{"annotation1": "wrongvalue"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeAnnotation3",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.14",
					Labels: map[string]string{"label1": "value"},
					Annotations: map[string]string{
						"annotation1":        "value",
						"cke.cybozu.com/foo": "bar",
					},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeTaint1",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "10.0.0.14",
					Labels:      map[string]string{"label1": "value"},
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Value:  "value2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeTaint2",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "10.0.0.14",
					Labels:      map[string]string{"label1": "value"},
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectNoExecute,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeTaint3",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "10.0.0.14",
					Labels:      map[string]string{"label1": "value"},
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint3",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeTaint4",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "10.0.0.14",
					Labels:      map[string]string{"label1": "value"},
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
						{
							Key:    "cke.cybozu.com/foo",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeTaintCP1",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Cluster.TaintCP = true
			}),
			ExpectedOps: []opData{{"update-node", 3}},
		},
		{
			Name: "NodeTaintCP2",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Cluster.TaintCP = true
			}).withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.11",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    op.CKETaintMaster,
							Value:  "hoge",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}, corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.12",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    op.CKETaintMaster,
							Value:  "hoge",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}, corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.13",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    op.CKETaintMaster,
							Value:  "hoge",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{},
		},
		{
			Name: "NodeTaintCP3",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Cluster.TaintCP = true
			}).withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.11",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    op.CKETaintMaster,
							Value:  "hoge",
							Effect: corev1.TaintEffectNoExecute,
						},
					},
				},
			}, corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.12",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    op.CKETaintMaster,
							Value:  "hoge",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}, corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.13",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    op.CKETaintMaster,
							Value:  "hoge",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeTaintCP4",
			Input: newData().withK8sResourceReady().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.11",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    op.CKETaintMaster,
							Value:  "hoge",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeTaintCP5",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Cluster.TaintCP = true
			}).withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.11",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    op.CKETaintMaster,
							Value:  "hoge",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}, corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.12",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    op.CKETaintMaster,
							Value:  "hoge",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}, corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.13",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    op.CKETaintMaster,
							Value:  "hoge",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}, corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "10.0.0.15",
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    op.CKETaintMaster,
							Value:  "hoge",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: []opData{{"update-node", 1}},
		},
		{
			Name: "NodeExtraAttrs",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "10.0.0.14",
					Labels: map[string]string{
						"label1":              "value",
						"acke.cybozu.com/foo": "bar",
					},
					Annotations: map[string]string{
						"annotation1":         "value",
						"acke.cybozu.com/foo": "bar",
					},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
						{
							Key:    "acke.cybozu.com/foo",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: nil,
		},
		{
			Name: "RemoveNonClusterNodes",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "10.0.0.20",
				},
			}),
			ExpectedOps: []opData{{"remove-node", 1}},
		},
		{
			Name: "AllGreen",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "10.0.0.14",
					Labels:      map[string]string{"label1": "value"},
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}),
			ExpectedOps: nil,
		},
		{
			Name: "EtcdRemoveNonClusterMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.100"] = &etcdserverpb.Member{Name: "10.0.0.100", ID: 3}
				d.Status.Etcd.InSyncMembers["10.0.0.100"] = false
			}),
			ExpectedOps: []opData{{"etcd-remove-member", 1}},
		},
		{
			Name: "SkipEtcdRemoveNonClusterMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.100"] = &etcdserverpb.Member{Name: "10.0.0.100", ID: 3}
				d.Status.Etcd.InSyncMembers["10.0.0.100"] = false
			}).withSSHNotConnectedNodes(),
			ExpectedOps: []opData{{"wait-kubernetes", 1}},
		},
		{
			Name: "EtcdDestroyNonCPMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.14"] = &etcdserverpb.Member{Name: "10.0.0.14", ID: 3}
				d.Status.Etcd.InSyncMembers["10.0.0.14"] = false
			}),
			ExpectedOps: []opData{{"etcd-destroy-member", 1}},
		},
		{
			Name: "EtcdDestroyNonCPMemberSSHNotConnected",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.14"] = &etcdserverpb.Member{Name: "10.0.0.14", ID: 3}
				d.Status.Etcd.InSyncMembers["10.0.0.14"] = false
			}).withSSHNotConnectedNonCPWorker(3),
			ExpectedOps: []opData{{"etcd-destroy-member", 0}},
		},
		{
			Name: "EtcdReAdd",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.13"].Name = ""
				d.Status.Etcd.Members["10.0.0.13"].ID = 0
				d.Status.Etcd.InSyncMembers["10.0.0.13"] = false
			}),
			ExpectedOps: []opData{{"etcd-add-member", 1}},
		},
		{
			Name: "EtcdMark",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.NodeStatuses["10.0.0.13"].Etcd.IsAddedMember = false
			}),
			ExpectedOps: []opData{{"etcd-mark-member", 1}},
		},
		{
			Name: "EtcdIsNotGood",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				// a node is to be added
				delete(d.Status.Etcd.Members, "10.0.0.13")
				delete(d.Status.Etcd.InSyncMembers, "10.0.0.13")
				// but the cluster is not good enough
				d.Status.Etcd.InSyncMembers["10.0.0.12"] = false
			}),
			ExpectedOps: nil,
		},
		{
			Name: "EtcdAdd",
			Input: newData().withAllServices().with(func(d testData) {
				delete(d.Status.Etcd.Members, "10.0.0.13")
				delete(d.Status.Etcd.InSyncMembers, "10.0.0.13")
			}),
			ExpectedOps: []opData{{"etcd-add-member", 1}},
		},
		{
			Name: "EtcdRemoveHealthyNonClusterMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.100"] = &etcdserverpb.Member{Name: "10.0.0.100", ID: 3}
				d.Status.Etcd.InSyncMembers["10.0.0.100"] = true
			}),
			ExpectedOps: []opData{{"etcd-remove-member", 1}},
		},
		{
			Name: "EtcdDestroyHealthyNonCPMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.14"] = &etcdserverpb.Member{Name: "10.0.0.14", ID: 14}
				d.Status.Etcd.InSyncMembers["10.0.0.14"] = true
			}),
			ExpectedOps: []opData{{"etcd-destroy-member", 1}},
		},
		{
			Name: "EtcdDestroyHealthyNonCPMemberSSHNotConnected",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.14"] = &etcdserverpb.Member{Name: "10.0.0.14", ID: 14}
				d.Status.Etcd.InSyncMembers["10.0.0.14"] = true
			}).withSSHNotConnectedNonCPWorker(3),
			ExpectedOps: []opData{{"etcd-destroy-member", 0}},
		},
		{
			Name: "EtcdRestart",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Etcd.Image = ""
			}),
			ExpectedOps: []opData{{"etcd-restart", 1}},
		},
		{
			Name: "Clean",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				for _, node := range []string{"10.0.0.14", "10.0.0.15"} {
					st := d.Status.NodeStatuses[node]
					st.Etcd.Running = true
					st.Etcd.HasData = true
					st.Etcd.IsAddedMember = true
					st.APIServer.Running = true
					st.ControllerManager.Running = true
					st.Scheduler.Running = true
					st.EtcdRivers.Running = true
				}
			}).withSSHNotConnectedNonCPWorker(1),
			ExpectedOps: []opData{
				{"stop-kube-apiserver", 1},
				{"stop-kube-controller-manager", 1},
				{"stop-kube-scheduler", 1},
				{"stop-etcd", 1},
				{"stop-etcd-rivers", 1},
			},
		},
		{
			Name: "Upgrade",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "10.0.0.14",
					Labels:      map[string]string{"label1": "value"},
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}).with(func(data testData) {
				data.Status.ConfigVersion = "1"
			}),
			ExpectedOps: []opData{{"upgrade", 3}},
		},
		{
			Name: "UpgradeAbort",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "10.0.0.14",
					Labels:      map[string]string{"label1": "value"},
					Annotations: map[string]string{"annotation1": "value"},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			}).with(func(data testData) {
				data.Status.ConfigVersion = "1"
			}).withSSHNotConnectedCP(),
			ExpectedOps: nil,
		},
		{
			Name:  "UncordonNodes",
			Input: newData().withK8sResourceReady().withRebootConfig().withRebootCordon(0),
			ExpectedOps: []opData{
				{"reboot-uncordon", 1},
			},
		},
		{
			Name: "SkipManuallyCordondedNodes",
			Input: newData().withK8sResourceReady().withRebootConfig().with(func(d testData) {
				d.Status.Kubernetes.Nodes[0].Spec.Unschedulable = true
			}),
			ExpectedOps: nil,
		},
		{
			Name: "Repair",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1"},
			}),
			ExpectedOps: []opData{
				{"repair-execute", 1},
			},
		},
		{
			Name: "RepairWithoutConfig",
			Input: newData().withK8sResourceReady().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1"},
			}),
			ExpectedOps: nil,
		},
		{
			Name: "RepairBadMachineType",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type2", Operation: "op1"},
			}),
			ExpectedOps: nil,
		},
		{
			Name: "RepairBadOperation",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "noop"},
				{Address: nodeNames[5], MachineType: "type1", Operation: "op1"},
			}),
			ExpectedOps: nil, // implementation dependent; bad entry consumes concurrency slot
		},
		{
			Name: "RepairOutOfCluster",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: "10.0.99.99", MachineType: "type1", Operation: "op1"},
			}),
			ExpectedOps: []opData{
				{"repair-execute", 1},
			},
		},
		{
			Name: "RepairApiServerHighPriority",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "noop"},
				{Address: nodeNames[0], MachineType: "type1", Operation: "op1"},
			}),
			ExpectedOps: []opData{
				{"repair-execute", 1}, // implementation dependent; cf. RepairBadOperation
			},
		},
		{
			Name: "RepairMaxConcurrent",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[0], MachineType: "type1", Operation: "op1"},
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1"},
				{Address: nodeNames[5], MachineType: "type1", Operation: "op1"},
			}).with(func(d testData) {
				max := 2
				d.Cluster.Repair.MaxConcurrentRepairs = &max
			}),
			ExpectedOps: []opData{
				{"repair-execute", 1},
				{"repair-execute", 1},
			},
		},
		{
			Name: "RepairMaxConcurrentSameMachine",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[0], MachineType: "type1", Operation: "op1"},
				{Address: nodeNames[0], MachineType: "type1", Operation: "op1"},
			}).with(func(d testData) {
				max := 2
				d.Cluster.Repair.MaxConcurrentRepairs = &max
			}),
			ExpectedOps: []opData{
				{"repair-execute", 1},
			},
		},
		{
			Name: "RepairMaxConcurrentApiServer",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[0], MachineType: "type1", Operation: "op1"},
				{Address: nodeNames[1], MachineType: "type1", Operation: "op1"},
			}).with(func(d testData) {
				max := 2
				d.Cluster.Repair.MaxConcurrentRepairs = &max
			}),
			ExpectedOps: []opData{
				{"repair-execute", 1},
			},
		},
		{
			Name: "RepairApiServerAnotherRebooting",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[0], MachineType: "type1", Operation: "op1"},
			}).withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{Index: 1, Node: nodeNames[2], Status: cke.RebootStatusRebooting},
			}).withRebootCordon(2).withNotReadyEndpoint(2),
			ExpectedOps: nil,
		},
		{
			Name: "RepairRebootingApiServer",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[2], MachineType: "type1", Operation: "op1"},
			}).withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{Index: 1, Node: nodeNames[2], Status: cke.RebootStatusRebooting},
			}).withRebootCordon(2).withNotReadyEndpoint(2),
			ExpectedOps: []opData{
				{"repair-execute", 1},
			},
		},
		{
			Name: "RepairDrain",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1"},
			}).with(func(d testData) {
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}),
			ExpectedOps: []opData{
				{"repair-drain-start", 1},
			},
		},
		{
			Name: "RepairDrainWaitCompletion",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusDraining,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second / 2)},
			}).with(func(d testData) {
				timeout := 60
				d.Cluster.Repair.EvictionTimeoutSeconds = &timeout
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}).withRebootCordon(4),
			ExpectedOps: nil,
		},
		{
			Name: "RepairDrainWaitCompletionExpire",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusDraining,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second * 2)},
			}).with(func(d testData) {
				timeout := 60
				d.Cluster.Repair.EvictionTimeoutSeconds = &timeout
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-drain-timeout", 1},
			},
		},
		{
			Name: "RepairDrainWaitCompletionDefaultTimeout",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusDraining,
					LastTransitionTime: time.Now().Add(-time.Duration(cke.DefaultRepairEvictionTimeoutSeconds) * time.Second / 2)},
			}).with(func(d testData) {
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}).withRebootCordon(4),
			ExpectedOps: nil,
		},
		{
			Name: "RepairDrainWaitCompletionExpireDefaultTimeout",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusDraining,
					LastTransitionTime: time.Now().Add(-time.Duration(cke.DefaultRepairEvictionTimeoutSeconds) * time.Second * 2)},
			}).with(func(d testData) {
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-drain-timeout", 1},
			},
		},
		{
			Name: "RepairDrainWaitRetryUncordon",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWaiting,
					DrainBackOffExpire: time.Now().Add(time.Duration(60) * time.Second)},
			}).with(func(d testData) {
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"reboot-uncordon", 1},
			},
		},
		{
			Name: "RepairDrainWaitRetry",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWaiting,
					DrainBackOffExpire: time.Now().Add(time.Duration(60) * time.Second)},
			}).with(func(d testData) {
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}),
			ExpectedOps: nil,
		},
		{
			Name: "RepairDrainWaitRetryExpire",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWaiting,
					DrainBackOffExpire: time.Now().Add(-time.Duration(60) * time.Second)},
			}).with(func(d testData) {
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}),
			ExpectedOps: []opData{
				{"repair-drain-start", 1},
			},
		},
		{
			Name: "RepairDrainCompleted",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusDraining},
			}).with(func(d testData) {
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
				d.Status.RepairQueue.DrainCompleted = map[string]bool{nodeNames[4]: true}
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-execute", 1},
			},
		},
		{
			Name: "RepairWatch",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWatching,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second / 2)},
			}).with(func(d testData) {
				watch := 60
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].WatchSeconds = &watch
			}).withRebootCordon(4),
			ExpectedOps: nil,
		},
		{
			Name: "RepairWatchExpire",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWatching,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second * 2)},
			}).with(func(d testData) {
				watch := 60
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].WatchSeconds = &watch
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-execute", 1}, // next step
			},
		},
		{
			Name: "RepairWatchExpireDefaultTimeout",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWatching,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second / 2)},
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-execute", 1}, // next step
			},
		},
		{
			Name: "RepairWatchExpireLastStep",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWatching, Step: 1,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second / 2)},
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-finish", 1}, // failed
			},
		},
		{
			Name: "RepairCompleted",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWatching,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second / 2)},
			}).with(func(d testData) {
				d.Status.RepairQueue.RepairCompleted = map[string]bool{nodeNames[4]: true}
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-finish", 1}, // succeeded
			},
		},
		{
			Name: "RepairSucceededUncordon",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusSucceeded},
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"reboot-uncordon", 1},
			},
		},
		{
			Name: "RepairSucceeded",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusSucceeded},
			}),
			ExpectedOps: nil,
		},
		{
			Name: "RepairFailedUncordon",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusFailed},
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"reboot-uncordon", 1},
			},
		},
		{
			Name: "RepairFailed",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusFailed},
			}),
			ExpectedOps: nil,
		},
		{
			Name: "RepairDeletedUncordon",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Deleted: true},
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"reboot-uncordon", 1},
			},
		},
		{
			Name: "RepairDeleted",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Deleted: true},
			}),
			ExpectedOps: []opData{
				{"repair-dequeue", 1},
			},
		},
		{
			Name: "RepairDisabled",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1"},
			}),
			ExpectedOps: nil,
		},
		{
			Name: "RepairDisabledDrain",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1"},
			}).with(func(d testData) {
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}),
			ExpectedOps: nil,
		},
		{
			Name: "RepairDisabledDrainWaitCompletion",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusDraining,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second / 2)},
			}).with(func(d testData) {
				timeout := 60
				d.Cluster.Repair.EvictionTimeoutSeconds = &timeout
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-drain-timeout", 1},
			},
		},
		{
			Name: "RepairDisabledDrainWaitCompletionExpire",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusDraining,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second * 2)},
			}).with(func(d testData) {
				timeout := 60
				d.Cluster.Repair.EvictionTimeoutSeconds = &timeout
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-drain-timeout", 1},
			},
		},
		{
			Name: "RepairDisabledDrainWaitRetry",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWaiting,
					DrainBackOffExpire: time.Now().Add(time.Duration(60) * time.Second)},
			}).with(func(d testData) {
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}),
			ExpectedOps: nil,
		},
		{
			Name: "RepairDisabledDrainWaitRetryExpire",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWaiting,
					DrainBackOffExpire: time.Now().Add(-time.Duration(60) * time.Second)},
			}).with(func(d testData) {
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
			}),
			ExpectedOps: nil,
		},
		{
			Name: "RepairDisabledDrainCompleted",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusDraining},
			}).with(func(d testData) {
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = true
				d.Status.RepairQueue.DrainCompleted = map[string]bool{nodeNames[4]: true}
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-drain-timeout", 1},
			},
		},
		{
			Name: "RepairDisabledWatch",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWatching,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second / 2)},
			}).with(func(d testData) {
				watch := 60
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].WatchSeconds = &watch
			}).withRebootCordon(4),
			ExpectedOps: nil,
		},
		{
			Name: "RepairDisabledWatchExpire",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWatching,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second * 2)},
			}).with(func(d testData) {
				watch := 60
				d.Cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].WatchSeconds = &watch
			}).withRebootCordon(4),
			ExpectedOps: nil,
		},
		{
			Name: "RepairDisabledWatchExpireLastStep",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWatching, Step: 1,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second / 2)},
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-finish", 1}, // failed
			},
		},
		{
			Name: "RepairDisabledCompleted",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Status: cke.RepairStatusProcessing, StepStatus: cke.RepairStepStatusWatching,
					LastTransitionTime: time.Now().Add(-time.Duration(60) * time.Second / 2)},
			}).with(func(d testData) {
				d.Status.RepairQueue.RepairCompleted = map[string]bool{nodeNames[4]: true}
			}).withRebootCordon(4),
			ExpectedOps: []opData{
				{"repair-finish", 1}, // succeeded
			},
		},
		{
			Name: "RepairDisabledDeleted",
			Input: newData().withK8sResourceReady().withRepairConfig().withRepairDisabled().withRepairEntries([]*cke.RepairQueueEntry{
				{Address: nodeNames[4], MachineType: "type1", Operation: "op1",
					Deleted: true},
			}),
			ExpectedOps: []opData{
				{"repair-dequeue", 1},
			},
		},
		{
			Name: "RebootWithoutConfig",
			Input: newData().withK8sResourceReady().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusQueued,
				},
			}),
			ExpectedOps: nil,
		},
		{
			Name: "Reboot",
			Input: newData().withK8sResourceReady().withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusQueued,
				},
				{
					Index:  2,
					Node:   nodeNames[5],
					Status: cke.RebootStatusQueued,
				},
			}).withNewlyDrained([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusQueued,
				},
			}),
			ExpectedOps: []opData{
				{"reboot-drain-start", 1},
			},
		},
		{
			Name: "SkipStartDrainTooManyUnreachableNodes",
			Input: newData().withK8sResourceReady().withSSHNotConnectedNonCPWorker(2).withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusQueued,
				},
			}).withNewlyDrained([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusQueued,
				},
			}),
			ExpectedOps: nil,
		},
		{
			Name: "DontSkipStartRebootDrainedTooManyUnreachableNodes",
			Input: newData().withK8sResourceReady().withSSHNotConnectedNonCPWorker(2).withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusDraining,
				},
			}).withDrainCompleted([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusDraining,
				},
			}),
			ExpectedOps: []opData{
				{"reboot-reboot", 1},
			},
		},
		{
			Name: "DontSkipDrainBackoffTooManyUnreachableNodes",
			Input: newData().withK8sResourceReady().withSSHNotConnectedNonCPWorker(2).withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusDraining,
				},
			}).withDrainTimedout([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusDraining,
				},
			}),
			ExpectedOps: []opData{
				{"reboot-drain-timeout", 1},
			},
		},
		{
			Name: "DontSkipRebootDequeueoffTooManyUnreachableNodes",
			Input: newData().withK8sResourceReady().withSSHNotConnectedNonCPWorker(2).withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusRebooting,
				},
			}).withRebootDequeued([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusRebooting,
				},
			}),
			ExpectedOps: []opData{
				{"reboot-dequeue", 1},
			},
		},
		{
			Name: "SkipRebootEtcdOutOfSync",
			Input: newData().withK8sResourceReady().withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusQueued,
				},
			}).withNewlyDrained([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusQueued,
				},
			}).with(func(d testData) {
				d.Status.Etcd.InSyncMembers["10.0.0.11"] = false
			}),
			ExpectedOps: nil,
		},
		{
			Name: "CancelReboot",
			Input: newData().withK8sResourceReady().withRebootConfig().withRebootEntries([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusCancelled,
				},
			}).withRebootCancelled([]*cke.RebootQueueEntry{
				{
					Index:  1,
					Node:   nodeNames[4],
					Status: cke.RebootStatusCancelled,
				},
			}),
			ExpectedOps: []opData{
				{"reboot-dequeue", 1},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ops, _ := DecideOps(c.Input.Cluster, c.Input.Status, c.Input.Constraints, c.Input.Resources, c.Input.RebootArgs, &Config{
				Interval:             0,
				CertsGCInterval:      0,
				MaxConcurrentUpdates: 5,
			})
			if len(ops) == 0 && len(c.ExpectedOps) == 0 {
				return
			}
			actual := make([]opData, len(ops))
			for i, o := range ops {
				actual[i].Name = o.Name()
				actual[i].TargetNum = len(o.Targets())
			}
			if !cmp.Equal(c.ExpectedOps, actual) {
				t.Error("unexpected opData:", cmp.Diff(c.ExpectedOps, actual))
			}
		OUT:
			for _, o := range ops {
				for i := 0; i < 100; i++ {
					commander := o.NextCommand()
					if commander == nil {
						continue OUT
					}
				}
				t.Fatalf("[%s] Operator.NextCommand() never finished: %s", c.Name, o.Name())
			}
		})
	}
}
