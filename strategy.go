package cke

import "github.com/cybozu-go/log"

// DecideToDo return next operation to do.
// This returns nil if nothing to do.
func DecideToDo(c *Cluster, cs *ClusterStatus) Operator {
	var cpNodes []*Node
	for _, n := range c.Nodes {
		if n.ControlPlane {
			cpNodes = append(cpNodes, n)
		}
	}

	for _, n := range cpNodes {
		if _, ok := cs.NodeStatuses[n.Address]; !ok {
			log.Warn("node status is not available", map[string]interface{}{
				"node": n.Address,
			})
			return nil
		}
	}

	if allTrue(func(n *Node) bool { return !cs.NodeStatuses[n.Address].Etcd.HasData }, cpNodes) {
		return NewEtcdBootOperator(cpNodes, cs.Agents, etcdDataDir(c))
	}

	// TODO
	return nil
}

func allTrue(cond func(node *Node) bool, nodes []*Node) bool {
	for _, n := range nodes {
		if !cond(n) {
			return false
		}
	}
	return true
}
