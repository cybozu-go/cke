package sabakan

import (
	"testing"

	"github.com/cybozu-go/cke"
)

func TestClusterValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tmpl    *cke.Cluster
		wantErr bool
	}{
		{
			"valid case: 1cp, 1worker",
			&cke.Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Nodes: []*cke.Node{
					{
						User:         "user",
						ControlPlane: true,
					},
					{
						User: "another",
					},
				},
				Options: cke.Options{
					Kubelet: cke.KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			false,
		},
		{
			"valid case: 1cp, 2worker (unique roles)",
			&cke.Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Nodes: []*cke.Node{
					{
						User:         "user",
						ControlPlane: true,
					},
					{
						User: "another",
						Labels: map[string]string{
							"cke.cybozu.com/role": "role1",
						},
					},
					{
						User: "another",
						Labels: map[string]string{
							"cke.cybozu.com/role": "role2",
						},
					},
				},
				Options: cke.Options{
					Kubelet: cke.KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			false,
		},
		{
			"invalid case: 1node",
			&cke.Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Nodes: []*cke.Node{
					{
						User:         "user",
						ControlPlane: true,
					},
				},
				Options: cke.Options{
					Kubelet: cke.KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid case: 2cp",
			&cke.Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Nodes: []*cke.Node{
					{
						User:         "user",
						ControlPlane: true,
					},
					{
						User:         "user",
						ControlPlane: true,
					},
					{
						User: "another",
					},
				},
				Options: cke.Options{
					Kubelet: cke.KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid case: no cp",
			&cke.Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Nodes: []*cke.Node{
					{
						User: "another",
						Labels: map[string]string{
							"cke.cybozu.com/role": "role1",
						},
					},
					{
						User: "another",
						Labels: map[string]string{
							"cke.cybozu.com/role": "role2",
						},
					},
				},
				Options: cke.Options{
					Kubelet: cke.KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid case: non-unique worker roles",
			&cke.Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Nodes: []*cke.Node{
					{
						User:         "user",
						ControlPlane: true,
					},
					{
						User: "another",
						Labels: map[string]string{
							"cke.cybozu.com/role": "role1",
						},
					},
					{
						User: "another",
						Labels: map[string]string{
							"cke.cybozu.com/role": "role1",
						},
					},
				},
				Options: cke.Options{
					Kubelet: cke.KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid case: without role and with role",
			&cke.Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Nodes: []*cke.Node{
					{
						User:         "user",
						ControlPlane: true,
					},
					{
						User: "another",
						Labels: map[string]string{
							"cke.cybozu.com/role": "role1",
						},
					},
					{
						User:   "another",
						Labels: map[string]string{},
					},
				},
				Options: cke.Options{
					Kubelet: cke.KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
		{
			"invalid case: non-empty address",
			&cke.Cluster{
				Name:          "testcluster",
				ServiceSubnet: "10.0.0.0/14",
				Nodes: []*cke.Node{
					{
						Address:      "10.0.0.1",
						ControlPlane: true,
						User:         "user",
					},
					{
						Address: "10.0.0.2",
						User:    "another",
					},
				},
				Options: cke.Options{
					Kubelet: cke.KubeletParams{
						CRIEndpoint: "/var/run/k8s-containerd.sock",
					},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateTemplate(tt.tmpl); (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
