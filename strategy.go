package cke

import "context"

type Strategy interface {
	FetchStatus(ctx context.Context, c *Cluster, agents map[string]Agent) error
	DecideToDo() Operator
}

// DecideToDo returns the next operation to do.
// This returns nil when no operation need to be done.

func allTrue(cond func(node *Node) bool, nodes []*Node) bool {
	for _, n := range nodes {
		if !cond(n) {
			return false
		}
	}
	return true
}
