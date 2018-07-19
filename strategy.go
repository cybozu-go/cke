package cke

import "github.com/cybozu-go/log"

// DecideToDo returns the next operation to do.
// This returns nil when no operation need to be done.
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
		return EtcdBootOp(cpNodes, cs.Agents, etcdVolumeName(c), c.Options.Etcd.ServiceParams)
	}

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
