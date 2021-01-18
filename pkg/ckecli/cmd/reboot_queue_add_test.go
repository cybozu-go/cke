package cmd

import (
	"testing"

	"github.com/cybozu-go/cke"
)

func TestValidateNodes(t *testing.T) {
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
		nodes   []string
		succeed bool
	}{
		{
			name:    "succeed",
			nodes:   []string{"1.1.1.1"},
			succeed: true,
		},
		{
			name:    "non-existing node",
			nodes:   []string{"3.3.3.3"},
			succeed: false,
		},
		{
			name:    "multiple control-plane nodes",
			nodes:   []string{"1.1.1.1", "2.2.2.2"},
			succeed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ret := validateNodes(tc.nodes, cluster)
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
