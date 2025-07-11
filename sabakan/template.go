package sabakan

import (
	"errors"

	"github.com/cybozu-go/cke"
)

// ValidateTemplate validates cluster template.
func ValidateTemplate(tmpl *cke.Cluster) error {
	if len(tmpl.Nodes) < 2 {
		return errors.New("template must contain at least two nodes")
	}

	roles := make(map[string]bool)
	var cpCount, ncpCount int
	for _, n := range tmpl.Nodes {
		if n.ControlPlane {
			cpCount++
			continue
		}

		ncpCount++
		if n.Labels[CKELabelRole] == "" {
			continue
		}
		roles[n.Labels[CKELabelRole]] = true
	}

	if cpCount != 1 {
		return errors.New("template must contain only one control plane node")
	}
	if ncpCount >= 2 && ncpCount != len(roles) {
		return errors.New("non-control plane nodes must be associated with unique roles")
	}

	return tmpl.Validate(true)
}
