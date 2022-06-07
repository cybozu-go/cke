package cke

import (
	"os"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation/field"
	proxyv1alpha1 "k8s.io/kube-proxy/config/v1alpha1"
	schedulerv1beta1 "k8s.io/kube-scheduler/config/v1beta1"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
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

	if c.ServiceSubnet != "12.34.56.00/24" {
		t.Error(`c.ServiceSubnet != "12.34.56.00/24"`)
	}
	if !reflect.DeepEqual(c.DNSServers, []string{"1.1.1.1", "8.8.8.8"}) {
		t.Error(`!reflect.DeepEqual(c.DNSServers, []string{"1.1.1.1", "8.8.8.8"})`)
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
	if c.Reboot.ProtectedNamespaces == nil {
		t.Fatal(`c.Reboot.ProtectedNamespaces == nil`)
	}
	if c.Reboot.ProtectedNamespaces.MatchLabels == nil {
		t.Fatal(`c.Reboot.ProtectedNamespaces.MatchLabels == nil`)
	}
	if c.Reboot.ProtectedNamespaces.MatchLabels["app"] != "sample" {
		t.Error(`c.Reboot.ProtectedNamespaces.MatchLabels["app"] != "sample"`)
	}
	if c.Options.Etcd.VolumeName != "myetcd" {
		t.Error(`c.Options.Etcd.VolumeName != "myetcd"`)
	}
	if !reflect.DeepEqual(c.Options.Etcd.ExtraArguments, []string{"arg1", "arg2"}) {
		t.Error(`!reflect.DeepEqual(c.Options.Etcd.ExtraArguments, []string{"arg1", "arg2"})`)
	}
	if !reflect.DeepEqual(c.Options.APIServer.ExtraBinds, []Mount{{"src1", "target1", true, PropagationShared, LabelShared}}) {
		t.Error(`!reflect.DeepEqual(c.Options.APIServer.ExtraBinds, []Mount{{"src1", "target1", true}})`)
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
	kubeSchedulerHealthz := "0.0.0.0"
	kubeSchedulerConfig, err := c.Options.Scheduler.MergeConfig(&schedulerv1beta1.KubeSchedulerConfiguration{
		HealthzBindAddress: &kubeSchedulerHealthz,
	})
	if err != nil {
		t.Fatal(err)
	}
	if kubeSchedulerConfig.PodMaxBackoffSeconds == nil {
		t.Fatal(`kubeSchedulerConfig.PodMaxBackoffSeconds == nil`)
	}
	if *kubeSchedulerConfig.PodMaxBackoffSeconds != 100 {
		t.Error(`*kubeSchedulerConfig.PodMaxBackoffSeconds != 100`)
	}
	if kubeSchedulerConfig.HealthzBindAddress == nil {
		t.Fatal(`kubeSchedulerConfig.HealthzBindAddress == nil`)
	}
	if *kubeSchedulerConfig.HealthzBindAddress != "0.0.0.0" {
		t.Error(`*kubeSchedulerConfig.HealthzBindAddress != "0.0.0.0"`)
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
	if !reflect.DeepEqual(c.Options.Kubelet.ExtraArguments, []string{"arg1"}) {
		t.Error(`!reflect.DeepEqual(c.Options.Kubelet.ExtraArguments, []string{"arg1"})`)
	}
	if len(c.Options.Kubelet.CNIConfFile.Name) == 0 {
		t.Error(`len(c.Options.Kubelet.CNIConfFile.Name) == 0`)
	}
	if len(c.Options.Kubelet.CNIConfFile.Content) == 0 {
		t.Error(`len(c.Options.Kubelet.CNIConfFile.Content) == 0`)
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

func intPtr(v int) *int {
	return &v
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
				EvictionTimeoutSeconds: intPtr(0),
			},
			wantErr: true,
		},
		{
			name: "negative eviction_timeout_seconds",
			reboot: Reboot{
				EvictionTimeoutSeconds: intPtr(-1),
			},
			wantErr: true,
		},
		{
			name: "zero command_timeout_seconds",
			reboot: Reboot{
				CommandTimeoutSeconds: intPtr(0),
			},
			wantErr: false,
		},
		{
			name: "negative command_timeout_seconds",
			reboot: Reboot{
				CommandTimeoutSeconds: intPtr(-1),
			},
			wantErr: true,
		},
		{
			name: "zero max_concurrent_reboots",
			reboot: Reboot{
				MaxConcurrentReboots: intPtr(0),
			},
			wantErr: true,
		},
		{
			name: "negative max_concurrent_reboots",
			reboot: Reboot{
				MaxConcurrentReboots: intPtr(-1),
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

func TestCluster(t *testing.T) {
	t.Run("YAML", testClusterYAML)
	t.Run("Validate", testClusterValidate)
	t.Run("ValidateNode", testClusterValidateNode)
	t.Run("Nodename", testNodename)
	t.Run("ValudateReboot", testClusterValidateReboot)
}
