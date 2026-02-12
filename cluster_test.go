package cke

import (
	"os"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	proxyv1alpha1 "k8s.io/kube-proxy/config/v1alpha1"
	schedulerv1 "k8s.io/kube-scheduler/config/v1"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

func testClusterYAML(t *testing.T) {
	t.Parallel()

	b, err := os.ReadFile("testdata/cluster.yaml")
	if err != nil {
		t.Fatal(err)
	}

	c := new(Cluster)
	err = yaml.Unmarshal(b, c)
	if err != nil {
		t.Fatal(err)
	}

	if c.Name != "test" {
		t.Error(`c.Name != "test"`)
	}
	if len(c.Nodes) != 1 {
		t.Fatal(`len(c.Nodes) != 1`)
	}

	node := c.Nodes[0]
	if node.Address != "1.2.3.4" {
		t.Error(`node.Address != "1.2.3.4"`)
	}
	if node.Hostname != "host1" {
		t.Error(`node.Hostname != "host1"`)
	}
	if node.User != "cybozu" {
		t.Error(`node.User != "cybozu"`)
	}
	if !node.ControlPlane {
		t.Error(`!node.ControlPlane`)
	}
	if node.Labels["label1"] != "value1" {
		t.Error(`node.Labels["label1"] != "value1"`)
	}

	if len(c.CPTolerations) != 1 {
		t.Fatal(`len(c.CPTolerations) != 1`)
	}
	if c.CPTolerations[0] != "foo.cybozu.com/transient" {
		t.Error(`c.CPTolerations[0] != "foo.cybozu.com/transient"`)
	}
	if c.ServiceSubnet != "12.34.56.00/24" {
		t.Error(`c.ServiceSubnet != "12.34.56.00/24"`)
	}
	if !cmp.Equal(c.DNSServers, []string{"1.1.1.1", "8.8.8.8"}) {
		t.Error(`!cmp.Equal(c.DNSServers, []string{"1.1.1.1", "8.8.8.8"})`)
	}
	if c.DNSService != "kube-system/dns" {
		t.Error(`c.DNSService != "kube-system/dns"`)
	}
	if len(c.Reboot.RebootCommand) != 1 {
		t.Fatal(`len(c.Reboot.RebootCommand) != 1`)
	}
	if c.Reboot.RebootCommand[0] != "true" {
		t.Error(`c.Reboot.RebootCommand[0] != "true"`)
	}
	if len(c.Reboot.BootCheckCommand) != 1 {
		t.Fatal(`len(c.Reboot.BootCheckCommand) != 1`)
	}
	if c.Reboot.BootCheckCommand[0] != "false" {
		t.Error(`c.Reboot.BootCheckCommand[0] != "false"`)
	}
	if c.Reboot.EvictionTimeoutSeconds == nil {
		t.Fatal(`c.Reboot.EvictionTimeoutSeconds == nil`)
	}
	if *c.Reboot.EvictionTimeoutSeconds != 60 {
		t.Error(`*c.Reboot.EvictionTimeoutSeconds != 60`)
	}
	if c.Reboot.MaxConcurrentReboots == nil {
		t.Error(`c.Reboot.MaxConcurrentReboots == nil`)
	}
	if *c.Reboot.MaxConcurrentReboots != 5 {
		t.Error(`*c.Reboot.MaxConcurrentReboots != 5`)
	}
	if c.Reboot.CommandTimeoutSeconds == nil {
		t.Fatal(`c.Reboot.CommandTimeoutSeconds == nil`)
	}
	if *c.Reboot.CommandTimeoutSeconds != 120 {
		t.Error(`*c.Reboot.CommandTimeoutSeconds != 120`)
	}
	if c.Reboot.CommandRetries == nil {
		t.Fatal(`c.Reboot.CommandRetries == nil`)
	}
	if *c.Reboot.CommandRetries != 3 {
		t.Error(`*c.Reboot.CommandRetries != 3`)
	}
	if c.Reboot.CommandInterval == nil {
		t.Fatal(`c.Reboot.CommandInterval == nil`)
	}
	if *c.Reboot.CommandInterval != 30 {
		t.Error(`*c.Reboot.CommandInterval != 30`)
	}
	if c.Reboot.EvictRetries == nil {
		t.Fatal(`c.Reboot.EvictRetries == nil`)
	}
	if *c.Reboot.EvictRetries != 10 {
		t.Error(`*c.Reboot.EvictRetries != 10`)
	}
	if c.Reboot.EvictInterval == nil {
		t.Fatal(`c.Reboot.EvictInterval == nil`)
	}
	if *c.Reboot.EvictInterval != 3 {
		t.Error(`*c.Reboot.EvictInterval != 3`)
	}
	if c.Reboot.ProtectedNamespaces == nil {
		t.Fatal(`c.Reboot.ProtectedNamespaces == nil`)
	}
	if c.Reboot.ProtectedNamespaces.MatchLabels == nil {
		t.Fatal(`c.Reboot.ProtectedNamespaces.MatchLabels == nil`)
	}
	if c.Reboot.ProtectedNamespaces.MatchLabels["app"] != "sample" {
		t.Error(`c.Reboot.ProtectedNamespaces.MatchLabels["app"] != "sample"`)
	}
	if len(c.Repair.RepairProcedures) != 1 {
		t.Fatal(`len(c.Repair.RepairProcedures) != 1`)
	}
	if !slices.Equal(c.Repair.RepairProcedures[0].MachineTypes, []string{"Cray-1", "Cray-2"}) {
		t.Error(`c.Repair.RepairProcedures[0].MachineTypes != {"Cray-1", "Cray-2"}`)
	}
	if len(c.Repair.RepairProcedures[0].RepairOperations) != 1 {
		t.Fatal(`len(c.Repair.RepairProcedures[0].RepairOperations) != 1`)
	}
	if c.Repair.RepairProcedures[0].RepairOperations[0].Operation != "unreachable" {
		t.Error(`c.Repair.RepairProcedures[0].RepairOperations[0].OperationName != "unreachable"`)
	}
	if len(c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps) != 2 {
		t.Fatal(`len(c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps) != 2`)
	}
	if !slices.Equal(c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].RepairCommand, []string{"reset", "remotely"}) {
		t.Error(`c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].RepairCommand != {"reset", "remotely"}`)
	}
	if c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandTimeoutSeconds == nil {
		t.Fatal(`c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandTimeoutSeconds == nil`)
	}
	if *c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandTimeoutSeconds != 10 {
		t.Error(`*c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandTimeoutSeconds != 10`)
	}
	if c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandRetries == nil {
		t.Fatal(`c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandRetries == nil`)
	}
	if *c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandRetries != 1 {
		t.Error(`*c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandRetries != 1`)
	}
	if c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandInterval == nil {
		t.Fatal(`c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandInterval == nil`)
	}
	if *c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandInterval != 5 {
		t.Error(`*c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandInterval != 5`)
	}
	if !c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain {
		t.Fatal(`!c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain`)
	}
	if c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].WatchSeconds == nil {
		t.Fatal(`c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].WatchSeconds == nil`)
	}
	if *c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].WatchSeconds != 60 {
		t.Error(`*c.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].WatchSeconds != 60`)
	}
	if !slices.Equal(c.Repair.RepairProcedures[0].RepairOperations[0].HealthCheckCommand, []string{"knock"}) {
		t.Error(`c.Repair.RepairProcedures[0].RepairOperations[0].HealthCheckCommand != {"knock"}`)
	}
	if c.Repair.RepairProcedures[0].RepairOperations[0].CommandTimeoutSeconds == nil {
		t.Fatal(`c.Repair.RepairProcedures[0].RepairOperations[0].CommandTimeoutSeconds == nil`)
	}
	if *c.Repair.RepairProcedures[0].RepairOperations[0].CommandTimeoutSeconds != 30 {
		t.Error(`*c.Repair.RepairProcedures[0].RepairOperations[0].CommandTimeoutSeconds != 30`)
	}
	if c.Repair.MaxConcurrentRepairs == nil {
		t.Fatal(`c.Repair.MaxConcurrentRepairs == nil`)
	}
	if *c.Repair.MaxConcurrentRepairs != 2 {
		t.Error(`*c.Repair.MaxConcurrentRepairs != 2`)
	}
	if c.Repair.ProtectedNamespaces == nil {
		t.Fatal(`c.Repair.ProtectedNamespaces == nil`)
	}
	if c.Repair.ProtectedNamespaces.MatchLabels["app"] != "protected" {
		t.Error(`c.Repair.ProtectedNamespaces.MatchLabels["app"] != "protected"`)
	}
	if c.Repair.EvictRetries == nil {
		t.Fatal(`c.Repair.EvictRetries == nil`)
	}
	if *c.Repair.EvictRetries != 3 {
		t.Error(`*c.Repair.EvictRetries != 3`)
	}
	if c.Repair.EvictInterval == nil {
		t.Fatal(`c.Repair.EvictInterval == nil`)
	}
	if *c.Repair.EvictInterval != 5 {
		t.Error(`*c.Repair.EvictInterval != 5`)
	}
	if c.Repair.EvictionTimeoutSeconds == nil {
		t.Fatal(`c.Repair.EvictionTimeoutSeconds == nil`)
	}
	if *c.Repair.EvictionTimeoutSeconds != 120 {
		t.Error(`*c.Repair.EvictionTimeoutSeconds != 120`)
	}
	if c.Options.Etcd.VolumeName != "myetcd" {
		t.Error(`c.Options.Etcd.VolumeName != "myetcd"`)
	}
	if !cmp.Equal(c.Options.Etcd.ExtraArguments, []string{"arg1", "arg2"}) {
		t.Error(`!cmp.Equal(c.Options.Etcd.ExtraArguments, []string{"arg1", "arg2"})`)
	}
	if !cmp.Equal(c.Options.APIServer.ExtraBinds, []Mount{{"src1", "target1", true, PropagationShared, LabelShared}}) {
		t.Error(`!cmp.Equal(c.Options.APIServer.ExtraBinds, []Mount{{"src1", "target1", true}})`)
	}
	if c.Options.APIServer.AuditLogEnabled != true {
		t.Error(`c.Options.APIServer.AuditLogEnabled != true`)
	}
	if c.Options.APIServer.AuditLogPolicy != `apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: Metadata
` {
		t.Errorf(`wrong c.Options.APIServer.AuditLogPolicy: %s`, c.Options.APIServer.AuditLogPolicy)
	}
	if c.Options.ControllerManager.ExtraEnvvar["env1"] != "val1" {
		t.Error(`c.Options.ControllerManager.ExtraEnvvar["env1"] != "val1"`)
	}
	kubeSchedulerConfig, err := c.Options.Scheduler.MergeConfig(&schedulerv1.KubeSchedulerConfiguration{
		Parallelism: ptr.To(int32(999)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if kubeSchedulerConfig.Parallelism == nil {
		t.Fatal(`kubeSchedulerConfig.Parallelism == nil`)
	}
	if *kubeSchedulerConfig.Parallelism != 999 {
		t.Fatal(`*kubeSchedulerConfig.Parallelism != 999`)
	}
	if kubeSchedulerConfig.PodMaxBackoffSeconds == nil {
		t.Fatal(`kubeSchedulerConfig.PodMaxBackoffSeconds == nil`)
	}
	if *kubeSchedulerConfig.PodMaxBackoffSeconds != 100 {
		t.Error(`*kubeSchedulerConfig.PodMaxBackoffSeconds != 100`)
	}
	if len(kubeSchedulerConfig.Profiles) != 1 {
		t.Error(`kubeSchedulerConfig.Profiles != 1"`)
	}
	if *kubeSchedulerConfig.Profiles[0].SchedulerName != "default-scheduler" {
		t.Error(`*kubeSchedulerConfig.Profiles != default-scheduler"`)
	}
	if kubeSchedulerConfig.Profiles[0].Plugins.Score.Disabled[0].Name != "PodTopologySpread" {
		t.Error(`kubeSchedulerConfig.Profiles[0].Plugins.Score.Disabled[0].Name != "PodTopologySpread"`)
	}
	if kubeSchedulerConfig.Profiles[0].Plugins.Score.Enabled[0].Name != "PodTopologySpread" {
		t.Error(`kubeSchedulerConfig.Profiles[0].Plugins.Score.Enabled[0].Name != "PodTopologySpread"`)
	}
	if *kubeSchedulerConfig.Profiles[0].Plugins.Score.Enabled[0].Weight != int32(500) {
		t.Error(`*kubeSchedulerConfig.Profiles[0].Plugins.Score.Enabled[0].Weight != int32(500)`)
	}

	proxyConfig, err := c.Options.Proxy.MergeConfig(&proxyv1alpha1.KubeProxyConfiguration{
		Mode:               ProxyModeIPVS,
		HealthzBindAddress: "0.0.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if proxyConfig.Mode != ProxyModeIptables {
		t.Error(`proxyConfig.Mode != ProxyModeIptables`)
	}
	if proxyConfig.HealthzBindAddress != "0.0.0.0" {
		t.Error(`proxyConfig.HealthzBindAddress != 0.0.0.0`)
	}

	if c.Options.Kubelet.CRIEndpoint != "/var/run/k8s-containerd.sock" {
		t.Error(`c.Options.Kubelet.ContainerRuntimeEndpoint != "/var/run/k8s-containerd.sock"`)
	}
	if len(c.Options.Kubelet.BootTaints) != 1 {
		t.Fatal(`len(c.Options.Kubelet.BootTaints) != 1`)
	}
	taint := c.Options.Kubelet.BootTaints[0]
	if taint.Key != "taint1" {
		t.Error(`taint.Key != "taint1"`)
	}
	if taint.Value != "tainted" {
		t.Error(`taint.Value != "tainted"`)
	}
	if taint.Effect != "NoExecute" {
		t.Error(`taint.Effect != "NoExecute"`)
	}
	if !cmp.Equal(c.Options.Kubelet.ExtraArguments, []string{"arg1"}) {
		t.Error(`!cmp.Equal(c.Options.Kubelet.ExtraArguments, []string{"arg1"})`)
	}
	if len(c.Options.Kubelet.CNIConfFile.Name) == 0 {
		t.Error(`len(c.Options.Kubelet.CNIConfFile.Name) == 0`)
	}
	if len(c.Options.Kubelet.CNIConfFile.Content) == 0 {
		t.Error(`len(c.Options.Kubelet.CNIConfFile.Content) == 0`)
	}
	if c.Options.Kubelet.InPlaceUpdate != false {
		t.Error(`c.Options.Kubelet.InPlaceUpdate != false`)
	}
	kubeletConfig, err := c.Options.Kubelet.MergeConfig(&kubeletv1beta1.KubeletConfiguration{
		ClusterDomain: "hoge.com",
		MaxPods:       100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if kubeletConfig.ClusterDomain != "my.domain" {
		t.Error(`kubeletConfig.ClusterDomain != "my.domain"`)
	}
	if kubeletConfig.FailSwapOn == nil || *kubeletConfig.FailSwapOn {
		t.Error(`kubeletConfig.FailSwapOn == nil || *kubeletConfig.FailSwapOn`)
	}
	if kubeletConfig.CgroupDriver != "systemd" {
		t.Error(`kubeletConfig.CgroupDriver != "systemd"`, kubeletConfig.CgroupDriver)
	}
	if kubeletConfig.ContainerLogMaxSize != "10Mi" {
		t.Error(`kubeletConfig.ContainerLogMaxSize != "10Mi"`, kubeletConfig.ContainerLogMaxSize)
	}
	if kubeletConfig.ContainerLogMaxFiles == nil {
		t.Fatal(`kubeletConfig.ContainerLogMaxFiles == nil`)
	}
	if *kubeletConfig.ContainerLogMaxFiles != 10 {
		t.Error(`*kubeletConfig.ContainerLogMaxFiles != 10`)
	}
	if kubeletConfig.MaxPods != 100 {
		t.Error(`kubeletConfig.MaxPods != 100`)
	}
}

func testClusterValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cluster Cluster
		wantErr bool
	}{
		{
			"No cluster name",
			Cluster{
				Name:          "",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid toleration key",
			Cluster{
				Name:          "testcluster",
				CPTolerations: []string{"a_b/c"},
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid toleration key 2",
			Cluster{
				Name:          "testcluster",
				CPTolerations: []string{"a.b/c b"},
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"No service subnet",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "",
				Options: Options{
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid DNS server address",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				DNSServers:    []string{"a.b.c.d"},
				Options: Options{
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid DNS service name",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				DNSService:    "hoge",
				Options: Options{
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"empty policy",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					APIServer: APIServerParams{
						AuditLogEnabled: true,
						AuditLogPolicy:  "",
					},
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid policy",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					APIServer: APIServerParams{
						AuditLogEnabled: true,
						AuditLogPolicy:  "test",
					},
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"valid policy",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					APIServer: APIServerParams{
						AuditLogEnabled: true,
						AuditLogPolicy: `apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: Metadata
`,
					},
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			false,
		},
		{
			"invalid proxy mode",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Proxy: ProxyParams{
						Config: &unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "kubeproxy.config.k8s.io/v1alpha1",
								"kind":       "KubeProxyConfiguration",
								"mode":       "foo",
							},
						},
					},
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid domain",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						Config: &unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion":    "kubelet.config.k8s.io/v1beta1",
								"kind":          "KubeletConfiguration",
								"clusterDomain": "a_b.c",
							},
						},
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid kubelet config",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						Config:      &unstructured.Unstructured{},
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid cri_endpoint",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						CRIEndpoint: "",
					},
				},
			},
			true,
		},
		{
			"invalid boot taint key",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						BootTaints: []corev1.Taint{
							{
								Key:    "a_b/c",
								Value:  "hello",
								Effect: "NoSchedule",
							},
						},
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid boot taint key 2",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						BootTaints: []corev1.Taint{
							{
								Key:    "a.b/c b",
								Value:  "hello",
								Effect: "NoSchedule",
							},
						},
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid boot taint value",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						BootTaints: []corev1.Taint{
							{
								Key:    "a.b/c",
								Value:  "こんにちは",
								Effect: "NoSchedule",
							},
						},
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid boot taint effect",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						BootTaints: []corev1.Taint{
							{
								Key:    "a.b/c",
								Value:  "hello",
								Effect: "NoNoNo",
							},
						},
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"filename is invalid",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						CNIConfFile: CNIConfFile{
							Name:    "aaa&&.txt",
							Content: `{"a":"b"}`,
						},
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"CNI conf file content is not empty, but name is empty",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						CNIConfFile: CNIConfFile{
							Name:    "",
							Content: `{"a":"b"}`,
						},
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"CNI conf file is not JSON",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Kubelet: KubeletParams{
						CNIConfFile: CNIConfFile{
							Name:    "99.loopback.conf",
							Content: "<aaa>",
						},
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid scheduler config",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Options: Options{
					Scheduler: SchedulerParams{
						Config: &unstructured.Unstructured{},
					},
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},

		{
			"duplicate node address",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Nodes: []*Node{
					{
						Address: "10.0.0.1",
						User:    "user",
					},
					{
						Address: "10.0.0.1",
						User:    "another",
					},
				},
				Options: Options{
					Kubelet: KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"valid case",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				DNSService:    "kube-system/dns",
				Options: Options{
					Proxy: ProxyParams{
						Config: &unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "kubeproxy.config.k8s.io/v1alpha1",
								"kind":       "KubeProxyConfiguration",
								"mode":       ProxyModeIptables,
							},
						},
					},
					Kubelet: KubeletParams{
						BootTaints: []corev1.Taint{
							{
								Key:    "a.b/c",
								Value:  "hello",
								Effect: "NoSchedule",
							},
						},
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.cluster
			if err := c.Validate(false); (err != nil) != tt.wantErr {
				t.Errorf("Cluster.Validate(false) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func testClusterValidateNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		node    Node
		isTmpl  bool
		wantErr bool
	}{
		{
			name: "valid case",
			node: Node{
				Address:     "10.0.0.1",
				User:        "testuser",
				Labels:      map[string]string{"cybozu.com/foo": "bar"},
				Annotations: map[string]string{"cybozu.com/fo_o": "こんにちは"},
				Taints: []corev1.Taint{{
					Key:    "cybozu.com/f_oo",
					Value:  "bar",
					Effect: "NoExecute",
				}},
			},
			isTmpl:  false,
			wantErr: false,
		},
		{
			name: "invalid address",
			node: Node{
				Address: "10000",
				User:    "testuser",
			},
			isTmpl:  false,
			wantErr: true,
		},
		{
			name: "no user",
			node: Node{
				Address: "10.0.0.1",
			},
			isTmpl:  false,
			wantErr: true,
		},
		{
			name: "bad label name",
			node: Node{
				Address: "10.0.0.1",
				User:    "testuser",
				Labels:  map[string]string{"a_b/c": "hello"},
			},
			isTmpl:  false,
			wantErr: true,
		},
		{
			name: "bad label value",
			node: Node{
				Address: "10.0.0.1",
				User:    "testuser",
				Labels:  map[string]string{"a_b/c": "こんにちは"},
			},
			isTmpl:  false,
			wantErr: true,
		},
		{
			name: "bad annotation",
			node: Node{
				Address:     "10.0.0.1",
				User:        "testuser",
				Annotations: map[string]string{"a.b/c_": "hello"},
			},
			isTmpl:  false,
			wantErr: true,
		},
		{
			name: "bad taint key",
			node: Node{
				Address: "10.0.0.1",
				User:    "testuser",
				Taints: []corev1.Taint{{
					Key:    "!!!",
					Value:  "hello",
					Effect: "NoSchedule",
				}},
			},
			isTmpl:  false,
			wantErr: true,
		},
		{
			name: "bad taint value",
			node: Node{
				Address: "10.0.0.1",
				User:    "testuser",
				Taints: []corev1.Taint{{
					Key:    "cybozu.com/hello",
					Value:  "こんにちは",
					Effect: "NoSchedule",
				}},
			},
			isTmpl:  false,
			wantErr: true,
		},
		{
			name: "bad taint effect",
			node: Node{
				Address: "10.0.0.1",
				User:    "testuser",
				Taints: []corev1.Taint{{
					Key:    "cybozu.com/hello",
					Value:  "world",
					Effect: "NoNoNo",
				}},
			},
			isTmpl:  false,
			wantErr: true,
		},
		{
			name: "valid template node",
			node: Node{
				User: "testuser",
			},
			isTmpl:  true,
			wantErr: false,
		},
		{
			name: "invalid template node: non-empty address",
			node: Node{
				Address: "10.0.0.1",
				User:    "testuser",
			},
			isTmpl:  true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := tt.node
			if err := validateNode(&n, tt.isTmpl, field.NewPath("node")); (err != nil) != tt.wantErr {
				t.Errorf("validateNode(%t) error = %v, wantErr %v", tt.isTmpl, err, tt.wantErr)
			}
		})
	}
}

func testNodename(t *testing.T) {
	t.Parallel()

	cases := []struct {
		node     *Node
		nodename string
	}{
		{&Node{Address: "172.16.0.1", Hostname: "my-host"}, "my-host"},
		{&Node{Address: "172.16.0.1"}, "172.16.0.1"},
	}
	for _, c := range cases {
		if c.node.Nodename() != c.nodename {
			t.Errorf("%s != %s", c.node.Nodename(), c.nodename)
		}
	}

}

func testClusterValidateReboot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		reboot  Reboot
		wantErr bool
	}{
		{
			name:    "valid case",
			reboot:  Reboot{},
			wantErr: false,
		},
		{
			name: "zero eviction_timeout_seconds",
			reboot: Reboot{
				EvictionTimeoutSeconds: ptr.To(0),
			},
			wantErr: true,
		},
		{
			name: "positive eviction_timeout_seconds",
			reboot: Reboot{
				EvictionTimeoutSeconds: ptr.To(1),
			},
			wantErr: false,
		},
		{
			name: "negative eviction_timeout_seconds",
			reboot: Reboot{
				EvictionTimeoutSeconds: ptr.To(-1),
			},
			wantErr: true,
		},
		{
			name: "zero command_timeout_seconds",
			reboot: Reboot{
				CommandTimeoutSeconds: ptr.To(0),
			},
			wantErr: false,
		},
		{
			name: "positive command_timeout_seconds",
			reboot: Reboot{
				CommandTimeoutSeconds: ptr.To(1),
			},
			wantErr: false,
		},
		{
			name: "negative command_timeout_seconds",
			reboot: Reboot{
				CommandTimeoutSeconds: ptr.To(-1),
			},
			wantErr: true,
		},
		{
			name: "zero max_concurrent_reboots",
			reboot: Reboot{
				MaxConcurrentReboots: ptr.To(0),
			},
			wantErr: true,
		},
		{
			name: "positive max_concurrent_reboots",
			reboot: Reboot{
				MaxConcurrentReboots: ptr.To(1),
			},
			wantErr: false,
		},
		{
			name: "negative max_concurrent_reboots",
			reboot: Reboot{
				MaxConcurrentReboots: ptr.To(-1),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateReboot(tt.reboot); (err != nil) != tt.wantErr {
				t.Errorf("validateReboot() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func testValidateTrustedRESTMappings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mapping TrustedRESTMapping
		wantErr bool
	}{
		{
			name: "valid namespaced",
			mapping: TrustedRESTMapping{
				Group:      "cilium.io",
				Version:    "v2alpha1",
				Kind:       "CiliumNetworkPolicy",
				Resource:   "ciliumnetworkpolicies",
				Namespaced: true,
			},
			wantErr: false,
		},
		{
			name: "valid cluster-scoped",
			mapping: TrustedRESTMapping{
				Group:      "cilium.io",
				Version:    "v2",
				Kind:       "CiliumNode",
				Resource:   "ciliumnodes",
				Namespaced: false,
			},
			wantErr: false,
		},
		{
			name: "empty group (core API)",
			mapping: TrustedRESTMapping{
				Group:      "",
				Version:    "v1",
				Kind:       "ConfigMap",
				Resource:   "configmaps",
				Namespaced: true,
			},
			wantErr: false,
		},
		{
			name: "empty version",
			mapping: TrustedRESTMapping{
				Group:    "cilium.io",
				Version:  "",
				Kind:     "CiliumNetworkPolicy",
				Resource: "ciliumnetworkpolicies",
			},
			wantErr: true,
		},
		{
			name: "empty kind",
			mapping: TrustedRESTMapping{
				Group:    "cilium.io",
				Version:  "v2alpha1",
				Kind:     "",
				Resource: "ciliumnetworkpolicies",
			},
			wantErr: true,
		},
		{
			name: "empty resource",
			mapping: TrustedRESTMapping{
				Group:    "cilium.io",
				Version:  "v2alpha1",
				Kind:     "CiliumNetworkPolicy",
				Resource: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateTrustedRESTMappings([]TrustedRESTMapping{tt.mapping}); (err != nil) != tt.wantErr {
				t.Errorf("validateTrustedRESTMappings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("duplicate mapping", func(t *testing.T) {
		dup := []TrustedRESTMapping{
			{Group: "cilium.io", Version: "v2", Kind: "CiliumNode", Resource: "ciliumnodes"},
			{Group: "cilium.io", Version: "v2", Kind: "CiliumNode", Resource: "ciliumnodes"},
		}
		if err := validateTrustedRESTMappings(dup); err == nil {
			t.Error("expected error for duplicate mappings, got nil")
		}
	})
}

func testLookupTrustedRESTMapping(t *testing.T) {
	t.Parallel()

	mappings := []TrustedRESTMapping{
		{
			Group:      "cilium.io",
			Version:    "v2alpha1",
			Kind:       "CiliumNetworkPolicy",
			Resource:   "ciliumnetworkpolicies",
			Namespaced: true,
		},
		{
			Group:      "cilium.io",
			Version:    "v2",
			Kind:       "CiliumNode",
			Resource:   "ciliumnodes",
			Namespaced: false,
		},
	}

	t.Run("found namespaced", func(t *testing.T) {
		gvk := schema.GroupVersionKind{Group: "cilium.io", Version: "v2alpha1", Kind: "CiliumNetworkPolicy"}
		m, err := LookupTrustedRESTMapping(mappings, gvk)
		if err != nil {
			t.Fatal(err)
		}
		if m.Resource.Resource != "ciliumnetworkpolicies" {
			t.Errorf("unexpected resource: %s", m.Resource.Resource)
		}
		if m.Scope.Name() != meta.RESTScopeNameNamespace {
			t.Errorf("expected namespace scope, got %s", m.Scope.Name())
		}
	})

	t.Run("found cluster-scoped", func(t *testing.T) {
		gvk := schema.GroupVersionKind{Group: "cilium.io", Version: "v2", Kind: "CiliumNode"}
		m, err := LookupTrustedRESTMapping(mappings, gvk)
		if err != nil {
			t.Fatal(err)
		}
		if m.Resource.Resource != "ciliumnodes" {
			t.Errorf("unexpected resource: %s", m.Resource.Resource)
		}
		if m.Scope.Name() != meta.RESTScopeNameRoot {
			t.Errorf("expected root scope, got %s", m.Scope.Name())
		}
	})

	t.Run("not found", func(t *testing.T) {
		gvk := schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"}
		_, err := LookupTrustedRESTMapping(mappings, gvk)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("empty mappings", func(t *testing.T) {
		gvk := schema.GroupVersionKind{Group: "cilium.io", Version: "v2alpha1", Kind: "CiliumNetworkPolicy"}
		_, err := LookupTrustedRESTMapping(nil, gvk)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestCluster(t *testing.T) {
	t.Run("YAML", testClusterYAML)
	t.Run("Validate", testClusterValidate)
	t.Run("ValidateNode", testClusterValidateNode)
	t.Run("Nodename", testNodename)
	t.Run("ValidateReboot", testClusterValidateReboot)
	t.Run("ValidateTrustedRESTMappings", testValidateTrustedRESTMappings)
	t.Run("LookupTrustedRESTMapping", testLookupTrustedRESTMapping)
}
