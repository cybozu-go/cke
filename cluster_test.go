package cke

import "testing"

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
	t.Run("Validate", testClusterValidate)
	t.Run("ValidateNode", testClusterValidateNode)
}
