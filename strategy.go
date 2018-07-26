package cke

// DecideToDo returns the next operation to do.
// This returns nil when no operation need to be done.
func DecideToDo(c *Cluster, cs *ClusterStatus) Operator {
	return etcdDecideToDo(c, cs)
}
