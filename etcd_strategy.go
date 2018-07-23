package cke

import (
	"context"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
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
		return EtcdBootOp(cpNodes, cs.Agents, c.Options.Etcd)
	}

	if len(cs.Etcd.Members) == 0 {
		log.Warn("No members of etcd cluster", nil)
		return nil
	}

	endpoints := make([]string, len(cpNodes))
	for i, n := range cpNodes {
		endpoints[i] = "http://" + n.Address + ":2379"
	}

	members := unhealthyNonClusterMember(c.Nodes, cs.Etcd)
	if len(members) > 0 {
		return EtcdRemoveMemberOp(endpoints, members)
	}
	nodes := unhealthyNonControlPlaneMember(c.Nodes, cs.Etcd)
	if len(nodes) > 0 {
		return EtcdDestroyMemberOp(endpoints, nodes, cs.Agents, cs.Etcd.Members)
	}
	nodes = newMemberControlPlane(cpNodes, cs.Etcd)
	if len(nodes) > 0 {
		return EtcdAddMemberOp(endpoints, nodes, cs.Agents, c.Options.Etcd)
	}
	members = healthyNonClusterMember(c.Nodes, cs.Etcd)
	if len(members) > 0 {
		return EtcdRemoveMemberOp(endpoints, members)
	}
	nodes = healthyNonControlPlaneMember(c.Nodes, cs.Etcd)
	if len(nodes) > 0 {
		return EtcdDestroyMemberOp(endpoints, nodes, cs.Agents, cs.Etcd.Members)
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
	return nil
}

func unhealthyNonClusterMember(allNodes []*Node, cs EtcdClusterStatus) map[string]*etcdserverpb.Member {
	var mem map[string]*etcdserverpb.Member
	for k, v := range cs.Members {
		mem[k] = v
	}
	for _, n := range allNodes {
		delete(mem, n.Address)
	}
	for k := range mem {
		if cs.MemberHealth[k] == EtcdNodeHealthy {
			delete(mem, k)
		}
	}
	return mem
}

func unhealthyNonControlPlaneMember(nodes []*Node, cs EtcdClusterStatus) []*Node {
	var targets []*Node
	for _, n := range nodes {
		if n.ControlPlane {
			continue
		}
		_, inMember := cs.Members[n.Address]
		health := cs.MemberHealth[n.Address]
		if health != EtcdNodeHealthy && inMember {
			targets = append(targets, n)
		}

	}
	return targets
}

func newMemberControlPlane(cpNodes []*Node, cs EtcdClusterStatus) []*Node {
	var targets []*Node
	for _, n := range cpNodes {
		_, inMember := cs.Members[n.Address]
		if !inMember {
			targets = append(targets, n)
		}
	}
	return targets
}

func healthyNonClusterMember(allNodes []*Node, cs EtcdClusterStatus) map[string]*etcdserverpb.Member {
	var mem map[string]*etcdserverpb.Member
	for k, v := range cs.Members {
		mem[k] = v
	}
	for _, n := range allNodes {
		delete(mem, n.Address)
	}
	for k := range mem {
		if cs.MemberHealth[k] != EtcdNodeHealthy {
			delete(mem, k)
		}
	}
	return mem
}

func healthyNonControlPlaneMember(allNodes []*Node, cs EtcdClusterStatus) []*Node {
	var targets []*Node
	for _, n := range allNodes {
		if n.ControlPlane {
			continue
		}
		_, inMember := cs.Members[n.Address]
		health := cs.MemberHealth[n.Address]
		if health == EtcdNodeHealthy && inMember {
			targets = append(targets, n)
		}

	}
	return targets
}
