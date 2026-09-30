package cmd

import (
	"testing"

	"github.com/cybozu-go/cke"
)

func TestValidateNode(t *testing.T) {
	cluster := &cke.Cluster{
		Nodes: []*cke.Node{
			{
				Address:      "1.1.1.1",
				ControlPlane: true,
			},
			{
				Address:      "2.2.2.2",
				ControlPlane: true,
			},
		},
	}

	testCases := []struct {
		name    string
		node    string
		succeed bool
	}{
		{
			name:    "succeed",
			node:    "1.1.1.1",
			succeed: true,
		},
		{
			name:    "non-existing node",
			node:    "3.3.3.3",
			succeed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ret := validateNode(tc.node, cluster)
			if tc.succeed {
				if ret != nil {
					t.Errorf("validateNodes() failed unexpectedly: %v", ret)
				}
			} else {
				if ret == nil {
					t.Error("validateNodes() succeeded unexpectedly")
				}
			}
		})
	}
}
