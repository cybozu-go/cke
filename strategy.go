package cke

// DecideToDo returns the next operation to do.
// This returns nil when no operation need to be done.
func DecideToDo(c *Cluster, cs *ClusterStatus) (Operator, error) {
	op, err := etcdDecideToDo(c, cs)
	if err != nil {
		return nil, err
	}
	if op != nil {
		return op, nil
	}
	return kubernetesDecideToDo(c, cs)
}
