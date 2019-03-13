package op

import "github.com/cybozu-go/cke"

// ResourceCreateOp creates a new resource.
func ResourceCreateOp(apiServer *cke.Node, resource cke.ResourceDefinition) cke.Operator {
	return nil
}

// ResourcePatchOp patches a resource using 3-way strategic merge patch.
func ResourcePatchOp(apiServer *cke.Node, resource cke.ResourceDefinition) cke.Operator {
	return nil
}
