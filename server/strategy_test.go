package server

import (
	"sort"
	"testing"
	"time"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/clusterdns"
	"github.com/cybozu-go/cke/op/etcd"
	"github.com/cybozu-go/cke/op/k8s"
	"github.com/cybozu-go/cke/op/nodedns"
	"github.com/cybozu-go/cke/scheduler"
	"github.com/cybozu-go/cke/static"
	"github.com/google/go-cmp/cmp"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testClusterName      = "test"
	testServiceSubnet    = "12.34.56.0/24"
	testDefaultDNSDomain = "cluster.local"
	testDefaultDNSAddr   = "10.0.0.53"
)

var (
	testDefaultDNSServers = []string{"8.8.8.8"}
	testResources         = []cke.ResourceDefinition{
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
	Cluster   *cke.Cluster
	Status    *cke.ClusterStatus
	Resources []cke.ResourceDefinition
}

func (d testData) ControlPlane() (nodes []*cke.Node) {
	for _, n := range d.Cluster.Nodes {
		if n.ControlPlane {
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
	cluster.Options.Kubelet = cke.KubeletParams{
		Domain:                   testDefaultDNSDomain,
		ContainerRuntime:         "remote",
		ContainerRuntimeEndpoint: "/var/run/k8s-containerd.sock",
		ContainerLogMaxFiles:     10,
		ContainerLogMaxSize:      "10Mi",
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
		nodeStatuses[nodeName] = &cke.NodeStatus{Etcd: cke.EtcdStatus{ServiceStatus: cke.ServiceStatus{Running: false}, HasData: false}}
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
		NodeStatuses: nodeStatuses,
		Kubernetes: cke.KubernetesClusterStatus{
			ResourceStatuses: map[string]map[string]string{
				"Namespace/foo": {cke.AnnotationResourceRevision: "1"},
			},
			Nodes: nodeList,
		},
	}

	return testData{
		Cluster:   cluster,
		Status:    status,
		Resources: testResources,
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

func (d testData) withStoppedEtcd() testData {
	for _, n := range d.ControlPlane() {
		d.NodeStatus(n).Etcd.HasData = true
	}
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

func (d testData) withAPIServer(serviceSubnet string) testData {
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).APIServer
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.HyperkubeImage.Name()
		st.BuiltInParams = k8s.APIServerParams(d.ControlPlane(), n.Address, serviceSubnet, false, "")
	}
	return d
}

func (d testData) withControllerManager(name, serviceSubnet string) testData {
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).ControllerManager
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.HyperkubeImage.Name()
		st.BuiltInParams = k8s.ControllerManagerParams(name, serviceSubnet)
	}
	return d
}

func (d testData) withScheduler() testData {
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).Scheduler
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.HyperkubeImage.Name()
		st.BuiltInParams = k8s.SchedulerParams()
	}
	return d
}

func (d testData) withKubelet(domain, dns string, allowSwap bool) testData {
	for _, n := range d.Cluster.Nodes {
		st := &d.NodeStatus(n).Kubelet
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.HyperkubeImage.Name()
		st.ContainerLogMaxSize = "10Mi"
		st.ContainerLogMaxFiles = 10
		st.BuiltInParams = k8s.KubeletServiceParams(n, cke.KubeletParams{
			ContainerRuntime:         "remote",
			ContainerRuntimeEndpoint: "/var/run/k8s-containerd.sock",
		})
		st.Domain = domain
		st.AllowSwap = allowSwap
	}
	return d
}

func (d testData) withProxy() testData {
	for _, n := range d.Cluster.Nodes {
		st := &d.NodeStatus(n).Proxy
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.HyperkubeImage.Name()
		st.BuiltInParams = k8s.ProxyParams(n)
	}
	return d
}

func (d testData) withAllServices() testData {
	d.withRivers()
	d.withEtcdRivers()
	d.withHealthyEtcd()
	d.withAPIServer(testServiceSubnet)
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
		ks.ResourceStatuses[res.Key] = map[string]string{
			cke.AnnotationResourceRevision: "1",
		}
	}
	ks.ResourceStatuses["Deployment/kube-system/cluster-dns"][cke.AnnotationResourceImage] = cke.CoreDNSImage.Name()
	ks.ResourceStatuses["DaemonSet/kube-system/node-dns"][cke.AnnotationResourceImage] = cke.UnboundImage.Name()
	ks.ClusterDNS.ConfigMap = clusterdns.ConfigMap(testDefaultDNSDomain, testDefaultDNSServers)
	ks.ClusterDNS.ClusterIP = testDefaultDNSAddr
	ks.NodeDNS.ConfigMap = nodedns.ConfigMap(testDefaultDNSAddr, testDefaultDNSDomain, testDefaultDNSServers)

	ks.EtcdEndpoints = &corev1.Endpoints{
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

	return d
}

func (d testData) withEtcdBackup() testData {
	d.withK8sResourceReady()
	d.Cluster.EtcdBackup = cke.EtcdBackup{
		Enabled:  true,
		PVCName:  "etcdbackup-pvc",
		Schedule: "*/1 * * * *",
		Rotate:   14,
	}
	d.Status.Kubernetes.EtcdBackup.Pod = &corev1.Pod{
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "etcdbackup",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "etcdbackup-pvc",
						},
					},
				},
			},
		},
	}
	d.Status.Kubernetes.EtcdBackup.Service = &corev1.Service{}
	d.Status.Kubernetes.EtcdBackup.ConfigMap = &corev1.ConfigMap{
		Data: map[string]string{
			"config.yml": `backup-dir: /etcdbackup
listen: 0.0.0.0:8080
rotate: 14
etcd:
  endpoints: 
    - https://cke-etcd:2379
  tls-ca-file: /etcd-certs/ca
  tls-cert-file: /etcd-certs/cert
  tls-key-file: /etcd-certs/key
`,
		},
	}
	d.Status.Kubernetes.EtcdBackup.Secret = &corev1.Secret{}
	d.Status.Kubernetes.EtcdBackup.CronJob = &batchv1beta1.CronJob{
		Spec: batchv1beta1.CronJobSpec{
			Schedule: "*/1 * * * *",
		},
	}
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

func TestDecideOps(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Name        string
		Input       testData
		ExpectedOps []string
	}{
		{
			Name:        "BootRivers",
			Input:       newData(),
			ExpectedOps: []string{"etcd-rivers-bootstrap", "rivers-bootstrap"},
		},
		{
			Name:        "BootRivers2",
			Input:       newData().withHealthyEtcd(),
			ExpectedOps: []string{"etcd-rivers-bootstrap", "rivers-bootstrap"},
		},
		{
			Name: "RestartRivers",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.Image = ""
			}).withEtcdRivers(),
			ExpectedOps: []string{"rivers-restart"},
		},
		{
			Name: "RestartRivers2",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.BuiltInParams.ExtraArguments = nil
			}).withEtcdRivers(),
			ExpectedOps: []string{"rivers-restart"},
		},
		{
			Name: "RestartRivers3",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.ExtraParams.ExtraArguments = []string{"foo"}
			}).withEtcdRivers(),
			ExpectedOps: []string{"rivers-restart"},
		},
		{
			Name: "StartRestartRivers",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.Image = ""
				d.NodeStatus(d.ControlPlane()[1]).Rivers.Running = false
			}).withEtcdRivers(),
			ExpectedOps: []string{"rivers-bootstrap", "rivers-restart"},
		},

		{
			Name: "RestartEtcdRivers",
			Input: newData().withRivers().withEtcdRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).EtcdRivers.Image = ""
			}),
			ExpectedOps: []string{"etcd-rivers-restart"},
		},
		{
			Name: "RestartEtcdRivers2",
			Input: newData().withRivers().withEtcdRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).EtcdRivers.BuiltInParams.ExtraArguments = nil
			}),
			ExpectedOps: []string{"etcd-rivers-restart"},
		},
		{
			Name: "RestartEtcdRivers3",
			Input: newData().withRivers().withEtcdRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).EtcdRivers.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{"etcd-rivers-restart"},
		},
		{
			Name: "StartRestartEtcdRivers",
			Input: newData().withRivers().withEtcdRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).EtcdRivers.Image = ""
				d.NodeStatus(d.ControlPlane()[1]).EtcdRivers.Running = false
			}),
			ExpectedOps: []string{"etcd-rivers-bootstrap", "etcd-rivers-restart"},
		},

		{
			Name:        "EtcdBootstrap",
			Input:       newData().withRivers().withEtcdRivers(),
			ExpectedOps: []string{"etcd-bootstrap"},
		},
		{
			Name:        "EtcdStart",
			Input:       newData().withRivers().withEtcdRivers().withStoppedEtcd(),
			ExpectedOps: []string{"etcd-start"},
		},
		{
			Name:        "WaitEtcd",
			Input:       newData().withRivers().withEtcdRivers().withUnhealthyEtcd(),
			ExpectedOps: []string{"etcd-wait-cluster"},
		},
		{
			Name:  "BootK8s",
			Input: newData().withHealthyEtcd().withRivers().withEtcdRivers(),
			ExpectedOps: []string{
				"kube-apiserver-bootstrap",
				"kube-controller-manager-bootstrap",
				"kube-proxy-bootstrap",
				"kube-scheduler-bootstrap",
				"kubelet-bootstrap",
			},
		},
		{
			Name:  "BootK8s2",
			Input: newData().withHealthyEtcd().withRivers().withEtcdRivers().withAPIServer(testServiceSubnet),
			ExpectedOps: []string{
				"kube-controller-manager-bootstrap",
				"kube-proxy-bootstrap",
				"kube-scheduler-bootstrap",
				"kubelet-bootstrap",
			},
		},
		{
			Name:  "RestartAPIServer",
			Input: newData().withAllServices().withAPIServer("11.22.33.0/24"),
			ExpectedOps: []string{
				"kube-apiserver-restart",
			},
		},
		{
			Name: "RestartAPIServer2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).APIServer.Image = ""
			}),
			ExpectedOps: []string{
				"kube-apiserver-restart",
			},
		},
		{
			Name: "RestartAPIServer3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).APIServer.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-apiserver-restart",
			},
		},
		{
			Name:  "RestartControllerManager",
			Input: newData().withAllServices().withControllerManager("another", testServiceSubnet),
			ExpectedOps: []string{
				"kube-controller-manager-restart",
			},
		},
		{
			Name: "RestartControllerManager2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).ControllerManager.Image = ""
			}),
			ExpectedOps: []string{
				"kube-controller-manager-restart",
			},
		},
		{
			Name: "RestartControllerManager3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).ControllerManager.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-controller-manager-restart",
			},
		},
		{
			Name: "RestartScheduler",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Scheduler.BuiltInParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-scheduler-restart",
			},
		},
		{
			Name: "RestartScheduler2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Scheduler.Image = ""
			}),
			ExpectedOps: []string{
				"kube-scheduler-restart",
			},
		},
		{
			Name: "RestartScheduler3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Scheduler.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-scheduler-restart",
			},
		},
		{
			Name: "RestartScheduler4",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Scheduler.Extenders = []*scheduler.ExtenderConfig{{URLPrefix: `urlPrefix: http://127.0.0.1:8001`}}
			}),
			ExpectedOps: []string{
				"kube-scheduler-restart",
			},
		},
		{
			Name:  "RestartKubelet",
			Input: newData().withAllServices().withKubelet("foo.local", "10.0.0.53", false),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name:  "RestartKubelet2",
			Input: newData().withAllServices().withKubelet("", "10.0.0.53", true),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name: "RestartKubelet3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[0]).Kubelet.Image = ""
			}),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name: "RestartKubelet4",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[0]).Kubelet.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name: "RestartKubelet5",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[0]).Kubelet.Domain = "neco.local"
			}),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name: "RestartKubelet6",
			Input: newData().withAllServices().with(func(d testData) {
				d.Cluster.Options.Kubelet.ContainerLogMaxFiles = 20
			}),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name: "RestartKubelet7",
			Input: newData().withAllServices().with(func(d testData) {
				d.Cluster.Options.Kubelet.ContainerLogMaxSize = "1Gi"
			}),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name: "RestartKubelet8",
			Input: newData().withAllServices().with(func(d testData) {
				d.Cluster.Options.Kubelet.ContainerRuntime = "docker"
			}),
			ExpectedOps: []string{
				"wait-kubernetes",
			},
		},
		{
			Name: "RestartKubelet9",
			Input: newData().withAllServices().with(func(d testData) {
				d.Cluster.Options.Kubelet.ContainerRuntimeEndpoint = "/var/run/dockershim.sock"
			}),
			ExpectedOps: []string{
				"wait-kubernetes",
			},
		},
		{
			Name: "RestartKubelet10",
			Input: newData().withAllServices().with(func(d testData) {
				t := time.Now()
				d.NodeStatus(d.Cluster.Nodes[0]).Kubelet.StartedAt = t.Add(-time.Hour)
				d.Cluster.Nodes[0].GeneratedAt = &t
			}),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name: "RestartProxy",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[0]).Proxy.BuiltInParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-proxy-restart",
			},
		},
		{
			Name: "RestartProxy2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[0]).Proxy.Image = ""
			}),
			ExpectedOps: []string{
				"kube-proxy-restart",
			},
		},
		{
			Name: "RestartProxy3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[0]).Proxy.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-proxy-restart",
			},
		},
		{
			Name:        "WaitKube",
			Input:       newData().withAllServices(),
			ExpectedOps: []string{"wait-kubernetes"},
		},
		{
			Name:  "K8sResources",
			Input: newData().withK8sReady(),
			ExpectedOps: []string{
				"create-cluster-dns-configmap",
				"create-etcd-endpoints",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
				"resource-apply",
			},
		},
		{
			Name: "UpdateDNSService",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				svc := &corev1.Service{}
				svc.Spec.ClusterIP = "1.1.1.1"
				d.Status.Kubernetes.DNSService = svc
			}),
			ExpectedOps: []string{
				"update-cluster-dns-configmap",
				"update-node-dns-configmap",
			},
		},
		{
			Name: "DNSUpdate1",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Cluster.Options.Kubelet.Domain = "neco.local"
			}),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name: "DNSUpdate2",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Cluster.Options.Kubelet.Domain = "neco.local"
				for _, st := range d.Status.NodeStatuses {
					st.Kubelet.Domain = "neco.local"
				}
			}),
			ExpectedOps: []string{
				"update-cluster-dns-configmap",
				"update-node-dns-configmap",
			},
		},
		{
			Name: "DNSUpdate3",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Cluster.DNSServers = []string{"1.1.1.1"}
			}),
			ExpectedOps: []string{
				"update-cluster-dns-configmap",
				"update-node-dns-configmap",
			},
		},
		{
			Name: "NodeDNSUpdate",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.ClusterDNS.ClusterIP = "10.0.0.54"
			}),
			ExpectedOps: []string{
				"update-node-dns-configmap",
			},
		},
		{
			Name: "EtcdEndpointsUpdate1",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.EtcdEndpoints.Subsets = []corev1.EndpointSubset{}
			}),
			ExpectedOps: []string{"update-etcd-endpoints"},
		},
		{
			Name: "EtcdEndpointsUpdate2",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.EtcdEndpoints.Subsets[0].Ports = []corev1.EndpointPort{}
			}),
			ExpectedOps: []string{"update-etcd-endpoints"},
		},
		{
			Name: "EtcdEndpointsUpdate3",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Status.Kubernetes.EtcdEndpoints.Subsets[0].Addresses = []corev1.EndpointAddress{}
			}),
			ExpectedOps: []string{"update-etcd-endpoints"},
		},
		{
			Name: "UserResourceAdd",
			Input: newData().withK8sResourceReady().withResources(
				append(testResources, cke.ResourceDefinition{
					Key:        "ConfigMap/foo/bar",
					Kind:       cke.KindConfigMap,
					Namespace:  "foo",
					Name:       "bar",
					Revision:   1,
					Definition: []byte(`{"apiversion":"v1","kind":"ConfigMap","metadata":{"namespace":"foo","name":"bar"},"data":{"a":"b"}}`),
				})),
			ExpectedOps: []string{"resource-apply"},
		},
		{
			Name: "UserResourceUpdate",
			Input: newData().withK8sResourceReady().withResources(
				[]cke.ResourceDefinition{{
					Key:        "Namespace/foo",
					Kind:       cke.KindNamespace,
					Name:       "foo",
					Revision:   2,
					Definition: []byte(`{"apiversion":"v1","kind":"Namespace","metadata":{"name":"foo"}}`),
				}}),
			ExpectedOps: []string{"resource-apply"},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
		},
		{
			Name: "NodeLabelCP1",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.11",
					Labels: map[string]string{},
				},
			}),
			ExpectedOps: []string{"update-node"},
		},
		{
			Name: "NodeLabelCP2",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.14",
					Labels: map[string]string{op.CKELabelMaster: "true"},
				},
			}),
			ExpectedOps: []string{"update-node"},
		},
		{
			Name: "NodeLabelCP3",
			Input: newData().withNodes(corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "10.0.0.11",
					Labels: map[string]string{op.CKELabelMaster: "hoge"},
				},
			}),
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
		},
		{
			Name: "NodeTaintCP1",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				d.Cluster.TaintCP = true
			}),
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"update-node"},
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
			ExpectedOps: []string{"remove-node"},
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
			}),
			ExpectedOps: []string{"etcd-remove-member"},
		},
		{
			Name: "EtcdDestroyNonCPMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.14"] = &etcdserverpb.Member{Name: "10.0.0.14", ID: 3}
			}),
			ExpectedOps: []string{"etcd-destroy-member"},
		},
		{
			Name: "EtcdReAdd",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.13"].Name = ""
				d.Status.Etcd.Members["10.0.0.13"].ID = 0
			}),
			ExpectedOps: []string{"etcd-add-member"},
		},
		{
			Name: "EtcdIsNotGood",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				// a node is to be added
				delete(d.Status.Etcd.Members, "10.0.0.13")
				delete(d.Status.Etcd.InSyncMembers, "10.0.0.13")
				// but the cluster is not good enough
				delete(d.Status.Etcd.InSyncMembers, "10.0.0.12")
			}),
			ExpectedOps: nil,
		},
		{
			Name: "EtcdAdd",
			Input: newData().withAllServices().with(func(d testData) {
				delete(d.Status.Etcd.Members, "10.0.0.13")
				delete(d.Status.Etcd.InSyncMembers, "10.0.0.13")
			}),
			ExpectedOps: []string{"etcd-add-member"},
		},
		{
			Name: "EtcdRemoveHealthyNonClusterMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.100"] = &etcdserverpb.Member{Name: "10.0.0.100", ID: 3}
				d.Status.Etcd.InSyncMembers["10.0.0.100"] = true
			}),
			ExpectedOps: []string{"etcd-remove-member"},
		},
		{
			Name: "EtcdDestroyHealthyNonCPMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.14"] = &etcdserverpb.Member{Name: "10.0.0.14", ID: 14}
				d.Status.Etcd.InSyncMembers["10.0.0.14"] = true
			}),
			ExpectedOps: []string{"etcd-destroy-member"},
		},
		{
			Name: "EtcdRestart",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Etcd.Image = ""
			}),
			ExpectedOps: []string{"etcd-restart"},
		},
		{
			Name: "EtcdBackupCreate",
			Input: newData().withEtcdBackup().with(func(d testData) {
				d.Status.Kubernetes.EtcdBackup.ConfigMap = nil
				d.Status.Kubernetes.EtcdBackup.Secret = nil
				d.Status.Kubernetes.EtcdBackup.CronJob = nil
				d.Status.Kubernetes.EtcdBackup.Service = nil
				d.Status.Kubernetes.EtcdBackup.Pod = nil
			}),
			ExpectedOps: []string{"etcdbackup-configmap-create", "etcdbackup-job-create", "etcdbackup-pod-create", "etcdbackup-secret-create", "etcdbackup-service-create"},
		},
		{
			Name: "EtcdBackupConfigMapCreate",
			Input: newData().withEtcdBackup().with(func(d testData) {
				d.Status.Kubernetes.EtcdBackup.ConfigMap = nil
			}),
			ExpectedOps: []string{"etcdbackup-configmap-create"},
		},
		{
			Name: "EtcdBackupSecretCreate",
			Input: newData().withEtcdBackup().with(func(d testData) {
				d.Status.Kubernetes.EtcdBackup.Secret = nil
			}),
			ExpectedOps: []string{"etcdbackup-secret-create"},
		},
		{
			Name: "EtcdBackupJobCreate",
			Input: newData().withEtcdBackup().with(func(d testData) {
				d.Status.Kubernetes.EtcdBackup.CronJob = nil
			}),
			ExpectedOps: []string{"etcdbackup-job-create"},
		},
		{
			Name: "EtcdBackupPodCreate",
			Input: newData().withEtcdBackup().with(func(d testData) {
				d.Status.Kubernetes.EtcdBackup.Pod = nil
			}),
			ExpectedOps: []string{"etcdbackup-pod-create"},
		},
		{
			Name: "EtcdBackupServiceCreate",
			Input: newData().withEtcdBackup().with(func(d testData) {
				d.Status.Kubernetes.EtcdBackup.Service = nil
			}),
			ExpectedOps: []string{"etcdbackup-service-create"},
		},
		{
			Name: "EtcdBackupPodUpdate",
			Input: newData().withEtcdBackup().with(func(d testData) {
				d.Cluster.EtcdBackup.PVCName = "new-pvc-name"
			}),
			ExpectedOps: []string{"etcdbackup-pod-update"},
		},
		{
			Name: "EtcdBackupJobUpdate",
			Input: newData().withEtcdBackup().with(func(d testData) {
				d.Cluster.EtcdBackup.Schedule = "* */0 * * *"
			}),
			ExpectedOps: []string{"etcdbackup-job-update"},
		},
		{
			Name: "EtcdBackupConfigMapUpdate",
			Input: newData().withEtcdBackup().with(func(d testData) {
				d.Cluster.EtcdBackup.Rotate = 10
			}),
			ExpectedOps: []string{"etcdbackup-configmap-update"},
		},
		{
			Name: "EtcdBackupRemove",
			Input: newData().withEtcdBackup().with(func(d testData) {
				d.Cluster.EtcdBackup.Enabled = false
			}),
			ExpectedOps: []string{"etcdbackup-configmap-remove", "etcdbackup-job-remove", "etcdbackup-pod-remove", "etcdbackup-secret-remove", "etcdbackup-service-remove"},
		},
		{
			Name: "Clean",
			Input: newData().withK8sResourceReady().with(func(d testData) {
				st := d.Status.NodeStatuses["10.0.0.14"]
				st.Etcd.Running = true
				st.Etcd.HasData = true
				st.APIServer.Running = true
				st.ControllerManager.Running = true
				st.Scheduler.Running = true
				st.EtcdRivers.Running = true
			}),
			ExpectedOps: []string{
				"stop-etcd",
				"stop-etcd-rivers",
				"stop-kube-apiserver",
				"stop-kube-controller-manager",
				"stop-kube-scheduler",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ops := DecideOps(c.Input.Cluster, c.Input.Status, c.Input.Resources)
			if len(ops) == 0 && len(c.ExpectedOps) == 0 {
				return
			}
			opNames := make([]string, len(ops))
			for i, o := range ops {
				opNames[i] = o.Name()
			}
			sort.Strings(opNames)
			if !cmp.Equal(c.ExpectedOps, opNames) {
				t.Error("unexpected ops:", cmp.Diff(c.ExpectedOps, opNames))
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
