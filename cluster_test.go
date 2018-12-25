package cke

import (
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func testClusterYAML(t *testing.T) {
	t.Parallel()

	y := `
name: test
nodes:
  - address: 1.2.3.4
    hostname: host1
    user: cybozu
    control_plane: true
    labels:
      label1: value1
service_subnet: 12.34.56.00/24
pod_subnet: 10.1.0.0/16
dns_servers: ["1.1.1.1", "8.8.8.8"]
dns_service: kube-system/dns
etcd_backup:
  enabled: true
  pvc_name: etcdbackup-pvc
  schedule: "*/1 * * * *"
options:
  etcd:
    volume_name: myetcd
    extra_args:
      - arg1
      - arg2
  kube-api:
    extra_binds:
      - source: src1
        destination: target1
        read_only: true
        propagation: shared
        selinux_label: z
  kube-controller-manager:
    extra_env:
      env1: val1
  kube-scheduler:
    extra_args:
      - arg1
  kube-proxy:
    extra_args:
      - arg1
  kubelet:
    domain: my.domain
    allow_swap: true
    boot_taints:
      - key: taint1
        value: tainted
        effect: NoExecute
    extra_args:
      - arg1
`
	c := new(Cluster)
	err := yaml.Unmarshal([]byte(y), c)
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
	if c.PodSubnet != "10.1.0.0/16" {
		t.Error(`c.PodSubnet != "10.1.0.0/16"`)
	}
	if !reflect.DeepEqual(c.DNSServers, []string{"1.1.1.1", "8.8.8.8"}) {
		t.Error(`!reflect.DeepEqual(c.DNSServers, []string{"1.1.1.1", "8.8.8.8"})`)
	}
	if c.DNSService != "kube-system/dns" {
		t.Error(`c.DNSService != "kube-system/dns"`)
	}
	if !reflect.DeepEqual(c.EtcdBackup, EtcdBackup{Enabled: true, PVCName: "etcdbackup-pvc", Schedule: "*/1 * * * *"}) {
		t.Error(`!reflect.DeepEqual(c.EtcdBackup, EtcdBackup{Enabled:true, PVCName:"etcdbackup-pvc", Schedule:"*/1 * * * *"})`)
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
	if c.Options.ControllerManager.ExtraEnvvar["env1"] != "val1" {
		t.Error(`c.Options.ControllerManager.ExtraEnvvar["env1"] != "val1"`)
	}
	if c.Options.Kubelet.Domain != "my.domain" {
		t.Error(`c.Options.Kubelet.Domain != "my.domain"`)
	}
	if !c.Options.Kubelet.AllowSwap {
		t.Error(`!c.Options.Kubelet.AllowSwap`)
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
				PodSubnet:     "10.1.0.0/16",
			},
			true,
		},
		{
			"No service subnet",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "",
				PodSubnet:     "10.1.0.0/16",
			},
			true,
		},
		{
			"No pod subnet",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.1.0.0/16",
				PodSubnet:     "",
			},
			true,
		},
		{
			"invalid DNS server address",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				PodSubnet:     "10.1.0.0/16",
				DNSServers:    []string{"a.b.c.d"},
			},
			true,
		},
		{
			"invalid DNS service name",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				PodSubnet:     "10.1.0.0/16",
				DNSService:    "hoge",
			},
			true,
		},
		{
			"invalid etcd backup PVC name",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				PodSubnet:     "10.1.0.0/16",
				EtcdBackup: EtcdBackup{
					Enabled:  true,
					PVCName:  "",
					Schedule: "*/1 * * * *",
				},
			},
			true,
		},
		{
			"invalid etcd backup schedule",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				PodSubnet:     "10.1.0.0/16",
				EtcdBackup: EtcdBackup{
					Enabled:  true,
					PVCName:  "etcdbackup-pvc",
					Schedule: "",
				},
			},
			true,
		},
		{
			"invalid domain",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				PodSubnet:     "10.1.0.0/16",
				Options: Options{
					Kubelet: KubeletParams{
						Domain: "a_b.c",
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
				PodSubnet:     "10.1.0.0/16",
				Options: Options{
					Kubelet: KubeletParams{
						BootTaints: []corev1.Taint{
							{
								Key:    "a_b/c",
								Value:  "hello",
								Effect: "NoSchedule",
							},
						},
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
				PodSubnet:     "10.1.0.0/16",
				Options: Options{
					Kubelet: KubeletParams{
						BootTaints: []corev1.Taint{
							{
								Key:    "a.b/c b",
								Value:  "hello",
								Effect: "NoSchedule",
							},
						},
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
				PodSubnet:     "10.1.0.0/16",
				Options: Options{
					Kubelet: KubeletParams{
						BootTaints: []corev1.Taint{
							{
								Key:    "a.b/c",
								Value:  "こんにちは",
								Effect: "NoSchedule",
							},
						},
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
				PodSubnet:     "10.1.0.0/16",
				Options: Options{
					Kubelet: KubeletParams{
						BootTaints: []corev1.Taint{
							{
								Key:    "a.b/c",
								Value:  "hello",
								Effect: "NoNoNo",
							},
						},
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
				PodSubnet:     "10.1.0.0/16",
				DNSService:    "kube-system/dns",
				Options: Options{
					Kubelet: KubeletParams{
						Domain: "cybozu.com",
						BootTaints: []corev1.Taint{
							{
								Key:    "a.b/c",
								Value:  "hello",
								Effect: "NoSchedule",
							},
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.cluster
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Cluster.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func testClusterValidateNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		node    Node
		cluster Cluster
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
			cluster: Cluster{},
			wantErr: false,
		},
		{
			name: "invalid address",
			node: Node{
				Address: "10000",
				User:    "testuser",
			},
			cluster: Cluster{},
			wantErr: true,
		},
		{
			name: "no user",
			node: Node{
				Address: "10.0.0.1",
			},
			cluster: Cluster{},
			wantErr: true,
		},
		{
			name: "bad label name",
			node: Node{
				Address: "10.0.0.1",
				User:    "testuser",
				Labels:  map[string]string{"a_b/c": "hello"},
			},
			cluster: Cluster{},
			wantErr: true,
		},
		{
			name: "bad label value",
			node: Node{
				Address: "10.0.0.1",
				User:    "testuser",
				Labels:  map[string]string{"a_b/c": "こんにちは"},
			},
			cluster: Cluster{},
			wantErr: true,
		},
		{
			name: "bad annotation",
			node: Node{
				Address:     "10.0.0.1",
				User:        "testuser",
				Annotations: map[string]string{"a.b/c_": "hello"},
			},
			cluster: Cluster{},
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
			cluster: Cluster{},
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
			cluster: Cluster{},
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
			cluster: Cluster{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.cluster
			n := tt.node
			if err := c.validateNode(&n, field.NewPath("node")); (err != nil) != tt.wantErr {
				t.Errorf("Cluster.validateNode() error = %v, wantErr %v", err, tt.wantErr)
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

func TestCluster(t *testing.T) {
	t.Run("YAML", testClusterYAML)
	t.Run("Validate", testClusterValidate)
	t.Run("ValidateNode", testClusterValidateNode)
	t.Run("Nodename", testNodename)
}
