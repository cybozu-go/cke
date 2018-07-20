package cke

import (
	"context"

	"github.com/cybozu-go/log"
)

type status string

func etcdDecideToDo(ctx context.Context, c *Cluster, cs *ClusterStatus) Operator {
	var cpNodes []*Node
	for _, n := range c.Nodes {
		if n.ControlPlane {
			cpNodes = append(cpNodes, n)
		}
	}

	bootstrap := true
	for _, n := range c.Nodes {
		if cs.NodeStatuses[n.Address].Etcd.HasData {
			bootstrap = false
		}
	}
	if bootstrap {
		return EtcdBootOp(cpNodes, cs.Agents, etcdVolumeName(c), c.Options.Etcd.ServiceParams)
	}

	if len(cs.Etcd.Members) == 0 {
		log.Warn("No members of etcd cluster", nil)
		return nil
	}

	// Remove an unhealthy/unreachable member who is either
	//   (1) not in the defined cluster, or
	//   (2) not a control plane node.
	// Note that unhealthy control plane node will not be removed
	// because it may be a temporary status.

	// Add a non-member control plane node
	// only when it is safe.

	// Remove a healthy member who is either
	//   (1) not in the defined cluster, or
	//   (2) not a control plane node,
	// only when it is safe to remove in point of node count.

	// mem := addedMembers(cpNodes, cs.EtcdCluster.Members, cs.NodeStatuses)
	// if len(mem) > 0 {
	// 	return EtcdAddMemberOp(cpNodes, mem[0], cs.EtcdCluster.Members, cs.Agents, etcdVolumeName(c), c.Options.Etcd.ServiceParams)
	// }

	// removed := removedMembers(c.Nodes, cs.EtcdCluster.Members, cs.NodeStatuses)
	// unknown := unknownMembers(c.Nodes, cs.EtcdCluster.Members)
	// if len(mem) > 0 || len(unknown) > 0 {
	// 	return EtcdRemoveMemberOp(removed, unknown, cs.Agents, etcdVolumeName(c), c.Options.Etcd.ServiceParams)
	// }
	return nil
}

// addedMember := cluster - (member & healthy)
// func addedMembers(cpNodes []*Node, members map[string]*etcdserverpb.Member, statuses map[string]*NodeStatus) []*Node {
// 	var member []*Node
// 	for _, n := range cpNodes {
// 		_, inMember := members[n.Address]
// 		health := statuses[n.Address].Etcd.Health
// 		if health != EtcdNodeHealthy || !inMember {
// 			member = append(member, n)
// 		}
// 	}
// 	return member
// }

// func removedMembers(allNodes []*Node, members map[string]*etcdserverpb.Member, statuses map[string]*NodeStatus) []*Node {
// 	var member []*Node
// 	for _, n := range allNodes {
// 		if n.ControlPlane {
// 			continue
// 		}
// 		_, inMember := members[n.Address]
// 		health := statuses[n.Address].Etcd.Health
// 		if health != EtcdNodeUnreachable || inMember {
// 			member = append(member, n)
// 		}
// 	}
// 	return member
// }

// func unknownMembers(cluster []*Node, members map[string]*etcdserverpb.Member) map[string]*etcdserverpb.Member {
// 	var mem map[string]*etcdserverpb.Member
// 	for k, v := range members {
// 		mem[k] = v
// 	}
// 	for _, n := range cluster {
// 		delete(mem, n.Address)
// 	}
// 	return mem
// }
