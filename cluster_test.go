package cke

import (
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func testClusterYAML(t *testing.T) {
	t.Parallel()

	y := `
name: test
nodes:
  - address: 1.2.3.4
    hostname: host1
    user: cybozu
    ssh_key: abc
    control_plane: true
    labels:
      label1: value1
ssh_key: clusterkey
service_subnet: 12.34.56.00/24
dns_servers: ["1.1.1.1", "8.8.8.8"]
options:
  etcd:
    extra_args:
      arg1: val1
  kube-api:
    extra_binds:
      src1: target1
  kube-controller:
    extra_env:
      env1: val1
  kube-scheduler:
    extra_args:
      arg1: val1
  kube-proxy:
    extra_args:
      arg1: val1
  kubelet:
    domain: my.domain
    allow_swap: true
    extra_args:
      arg1: val1
rbac: true
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
	if node.SSHKey != "abc" {
		t.Error(`node.SSHKey != "abc"`)
	}
	if !node.ControlPlane {
		t.Error(`!node.ControlPlane`)
	}
	if node.Labels["label1"] != "value1" {
		t.Error(`node.Labels["label1"] != "value1"`)
	}

	if c.SSHKey != "clusterkey" {
		t.Error(`c.SSHKey != "clusterkey"`)
	}
	if c.ServiceSubnet != "12.34.56.00/24" {
		t.Error(`c.ServiceSubnet != "12.34.56.00/24"`)
	}
	if !reflect.DeepEqual(c.DNSServers, []string{"1.1.1.1", "8.8.8.8"}) {
		t.Error(`!reflect.DeepEqual(c.DNSServers, []string{"1.1.1.1", "8.8.8.8"})`)
	}

	if c.Options.Etcd.ExtraArguments["arg1"] != "val1" {
		t.Error(`c.Options.Etcd.ExtraArguments["arg1"] != "val1"`)
	}
	if c.Options.APIServer.ExtraBinds["src1"] != "target1" {
		t.Error(`c.Options.APIServer.ExtraBinds["src1"] != "target1"`)
	}
	if c.Options.Controller.ExtraEnvvar["env1"] != "val1" {
		t.Error(`c.Options.Controller.ExtraEnvvar["env1"] != "val1"`)
	}
	if c.Options.Scheduler.ExtraArguments["arg1"] != "val1" {
		t.Error(`c.Options.Scheduler.ExtraArguments["arg1"] != "val1"`)
	}
	if c.Options.Proxy.ExtraArguments["arg1"] != "val1" {
		t.Error(`c.Options.Proxy.ExtraArguments["arg1"] != "val1"`)
	}
	if c.Options.Kubelet.Domain != "my.domain" {
		t.Error(`c.Options.Kubelet.Domain != "my.domain"`)
	}
	if !c.Options.Kubelet.AllowSwap {
		t.Error(`!c.Options.Kubelet.AllowSwap`)
	}
	if c.Options.Kubelet.ExtraArguments["arg1"] != "val1" {
		t.Error(`c.Options.Kubelet.ExtraArguments["arg1"] != "val1"`)
	}

	if !c.RBAC {
		t.Error(`!c.RBAC`)
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
			},
			true,
		},
		{
			"No service subnet",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "",
			},
			true,
		},
		{
			"invalid DNS server address",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				DNSServers:    []string{"a.b.c.d"},
			},
			true,
		},
		{
			"valid case",
			Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
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
				Address: "10.0.0.1",
				User:    "testuser",
				SSHKey:  "aaa",
			},
			cluster: Cluster{SSHKey: ""},
			wantErr: false,
		},
		{
			name: "valid case: node has no ssh key, but cluster has global one",
			node: Node{
				Address: "10.0.0.1",
				User:    "testuser",
				SSHKey:  "",
			},
			cluster: Cluster{SSHKey: "aaa"},
			wantErr: false,
		},
		{
			name: "invalid address",
			node: Node{
				Address: "10000",
				User:    "testuser",
				SSHKey:  "aaa",
			},
			cluster: Cluster{SSHKey: "aaa"},
			wantErr: true,
		},
		{
			name: "no user",
			node: Node{
				Address: "10.0.0.1",
				User:    "",
				SSHKey:  "aaa",
			},
			cluster: Cluster{SSHKey: "aaa"},
			wantErr: true,
		},
		{
			name: "no SSH key",
			node: Node{
				Address: "10.0.0.1",
				User:    "testuser",
				SSHKey:  "",
			},
			cluster: Cluster{SSHKey: ""},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.cluster
			n := tt.node
			if err := c.validateNode(&n); (err != nil) != tt.wantErr {
				t.Errorf("Cluster.validateNode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCluster(t *testing.T) {
	t.Run("YAML", testClusterYAML)
	t.Run("Validate", testClusterValidate)
	t.Run("ValidateNode", testClusterValidateNode)
}
