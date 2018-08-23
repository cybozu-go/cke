package cke

import "github.com/cybozu-go/log"

// DecideToDo returns the next operation to do.
// This returns nil when no operation need to be done.
func DecideToDo(c *Cluster, cs *ClusterStatus) Operator {
	op := etcdDecideToDo(c, cs)
	if op != nil {
		return op
	}
	if len(cs.Etcd.Members) == 0 {
		log.Warn("No members of etcd cluster", nil)
		return nil
	}
	return kubernetesDecideToDo(c, cs)
}
