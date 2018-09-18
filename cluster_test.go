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
    volume_name: myetcd
    extra_args:
      - arg1
      - arg2
  kube-api:
    extra_binds:
      - source: src1
        destination: target1
        read_only: true
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

	if c.Options.Etcd.VolumeName != "myetcd" {
		t.Error(`c.Options.Etcd.VolumeName != "myetcd"`)
	}
	if !reflect.DeepEqual(c.Options.Etcd.ExtraArguments, []string{"arg1", "arg2"}) {
		t.Error(`!reflect.DeepEqual(c.Options.Etcd.ExtraArguments, []string{"arg1", "arg2"})`)
	}
	if !reflect.DeepEqual(c.Options.APIServer.ExtraBinds, []Mount{{"src1", "target1", true}}) {
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
				SSHKey: `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAqGDTa8Fw0qWuQ/7xvsTdFfP0oqgpXUxl8FpOFWpJWTUM9ZhJ
NidBfglbnAAQ14/7JfvTUHAQgQ7X0ih8IJ2Z6VDGGZ9kr1N42iN5/17yMCZwgGgR
bwtvj+/X4lFUa0zVHe/dnLxkBQyV2hOvlLO/E7fUeVy70Mt/1NdlZtI15l1JX/+p
JJVSW+YaL7QF4X4aYad+PSQU5jRQB5KMfhTSayuAMDJc/FCYcArqhHvC/eh6lbyu
Qy3WokI1p+XaRb1giybfFW35ymfVT3EsGBZjkpsgZGCMfnQ5H+xr9JiItvLE7yAd
nBZevx5DyKmrC95tvjKiDGSULd8j4IiAvlsALwIDAQABAoIBACQJJPZo3gaXIua2
h3J2m4J5RaASMVggY6i/CvsWVkBbVDyzrOeEG0YoJo0KjpAz5mJItP8AHOgiDxqR
Q4+Pa0M94EfXjyreyHyXHyMCZP7dGzLAEwsa/XNmt2NeWJzmQq43icxjnVxfRyr3
D5rZpUlJDJY0vJWBGAirWK5ayuJUN9SFfsJWqEk4CDNQvONWNK1gvxazbppdCu93
FuuQvNkutosx8tmyl9eCev6sIugB6pp/YRf57JLRKJ0BwG7qn3gRNpyQOhGrF1MX
+0I9Ldi42OluLKP1X7n6MOux7Alxh5KuIq28d4mrE0iKUGU3yBt9R61UUGgynWc/
98QUQ/ECgYEA11Oj2fizzNnvEWn8nO1apYohtG+fjga8Tt472EpDjwwvhFVAX59f
2VoTJZct/oCkgffeut+bTB9FIYMRPoO1OH7Vd5lqsa+GCO+vTDM2mezFdfItxPoe
8h8u4brBy+x0aPyiNLEuYIjUh0ymUoviFGB4jP/J2QNzJvhM1nu12BsCgYEAyC7w
nHiMmkfPEslG1DyKsD2rmPiVHLldjVzYSOgBcL8bPGU2SYQedRdQBpzK6OF9TqXv
QsvO6HVgq8bmZVr2e0zhZhCak+NyxczObOdP2i+M2QUIXGBXG7ivCBexSiUH0DUd
xV2LEWkXA+3WuJ9gKY9GBBBdTOD+jqssiLZvIX0CgYEAtlHgo9g8TZCeJy2Jskoa
/Z2nCkOVYsl7OoBbRbkj2QRlW3RfzFeC7eOh4KtQS3UbVdzN34cj1GGJxGVY/YjB
sfNaxijFuWu4XuqrkCaw7cYYL9T+QhHSkAotRP4/x24P5zE6GsmHTj+tTF5vWeeN
ZtmEWUbf3vtXzkBhtx4Ki88CgYAaliFepqQF2YOm+xRtG51PyuD/cARdzECghbQz
+pw2XStA2jBbkzB4XKBEQI6yX0BFMcSVGnxgYzZzmfb/fxU9SviklY/yFEMqAglo
bVAtqiMKr6BspF7tT5nveTYSothmzqclj0bpCQwFeZEK9B/RZTXnVEUP8NHeIN3J
SnF4AQKBgCXupLs3AqbEWg2iUs+Eqeru0rEWopuTUiLJOvoT6X5NQlUIlpv5Ye+Z
tsChz55NjCxNEpn4NvGyeGgJrBEGwAPbx/X2v2BWFxWPNWh6byHi9ZxELa0Utlc8
B29lX8k9dqD0HitCL6ibsw0DqsU6FC3fd179rH8Bik83FuukuxvD
-----END RSA PRIVATE KEY-----
`,
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
			cluster: Cluster{SSHKey: `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAqGDTa8Fw0qWuQ/7xvsTdFfP0oqgpXUxl8FpOFWpJWTUM9ZhJ
NidBfglbnAAQ14/7JfvTUHAQgQ7X0ih8IJ2Z6VDGGZ9kr1N42iN5/17yMCZwgGgR
bwtvj+/X4lFUa0zVHe/dnLxkBQyV2hOvlLO/E7fUeVy70Mt/1NdlZtI15l1JX/+p
JJVSW+YaL7QF4X4aYad+PSQU5jRQB5KMfhTSayuAMDJc/FCYcArqhHvC/eh6lbyu
Qy3WokI1p+XaRb1giybfFW35ymfVT3EsGBZjkpsgZGCMfnQ5H+xr9JiItvLE7yAd
nBZevx5DyKmrC95tvjKiDGSULd8j4IiAvlsALwIDAQABAoIBACQJJPZo3gaXIua2
h3J2m4J5RaASMVggY6i/CvsWVkBbVDyzrOeEG0YoJo0KjpAz5mJItP8AHOgiDxqR
Q4+Pa0M94EfXjyreyHyXHyMCZP7dGzLAEwsa/XNmt2NeWJzmQq43icxjnVxfRyr3
D5rZpUlJDJY0vJWBGAirWK5ayuJUN9SFfsJWqEk4CDNQvONWNK1gvxazbppdCu93
FuuQvNkutosx8tmyl9eCev6sIugB6pp/YRf57JLRKJ0BwG7qn3gRNpyQOhGrF1MX
+0I9Ldi42OluLKP1X7n6MOux7Alxh5KuIq28d4mrE0iKUGU3yBt9R61UUGgynWc/
98QUQ/ECgYEA11Oj2fizzNnvEWn8nO1apYohtG+fjga8Tt472EpDjwwvhFVAX59f
2VoTJZct/oCkgffeut+bTB9FIYMRPoO1OH7Vd5lqsa+GCO+vTDM2mezFdfItxPoe
8h8u4brBy+x0aPyiNLEuYIjUh0ymUoviFGB4jP/J2QNzJvhM1nu12BsCgYEAyC7w
nHiMmkfPEslG1DyKsD2rmPiVHLldjVzYSOgBcL8bPGU2SYQedRdQBpzK6OF9TqXv
QsvO6HVgq8bmZVr2e0zhZhCak+NyxczObOdP2i+M2QUIXGBXG7ivCBexSiUH0DUd
xV2LEWkXA+3WuJ9gKY9GBBBdTOD+jqssiLZvIX0CgYEAtlHgo9g8TZCeJy2Jskoa
/Z2nCkOVYsl7OoBbRbkj2QRlW3RfzFeC7eOh4KtQS3UbVdzN34cj1GGJxGVY/YjB
sfNaxijFuWu4XuqrkCaw7cYYL9T+QhHSkAotRP4/x24P5zE6GsmHTj+tTF5vWeeN
ZtmEWUbf3vtXzkBhtx4Ki88CgYAaliFepqQF2YOm+xRtG51PyuD/cARdzECghbQz
+pw2XStA2jBbkzB4XKBEQI6yX0BFMcSVGnxgYzZzmfb/fxU9SviklY/yFEMqAglo
bVAtqiMKr6BspF7tT5nveTYSothmzqclj0bpCQwFeZEK9B/RZTXnVEUP8NHeIN3J
SnF4AQKBgCXupLs3AqbEWg2iUs+Eqeru0rEWopuTUiLJOvoT6X5NQlUIlpv5Ye+Z
tsChz55NjCxNEpn4NvGyeGgJrBEGwAPbx/X2v2BWFxWPNWh6byHi9ZxELa0Utlc8
B29lX8k9dqD0HitCL6ibsw0DqsU6FC3fd179rH8Bik83FuukuxvD
-----END RSA PRIVATE KEY-----
`},
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
