package cke

import "context"

// DecideToDo returns the next operation to do.
// This returns nil when no operation need to be done.
func DecideToDo(ctx context.Context, c *Cluster, cs *ClusterStatus) Operator {
	return etcdDecideToDo(ctx, c, cs)
}

func allTrue(cond func(node *Node) bool, nodes []*Node) bool {
	for _, n := range nodes {
		if !cond(n) {
			return false
		}
	}
	return true
}
