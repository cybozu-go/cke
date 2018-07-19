package cke

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/etcdhttp"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/cmd"
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

	members, err := getEtcdMembers(ctx, cpNodes)
	if err != nil {
		hostnames := make([]string, len(cpNodes))
		for i, n := range cpNodes {
			hostnames[i] = n.Hostname
		}
		log.Warn("failed to get etcd members", map[string]interface{}{
			"hosts":     hostnames,
			log.FnError: err,
		})
		return nil
	}

	membersStatus := getEtcdMembersStatus(ctx, members)

	numHealthy := 0
	for _, member := range members {
		if membersStatus[member.Name] == "healthy" && containsMember(cpNodes, member) {
			numHealthy++
		}
	}

	if numHealthy < len(cpNodes)/2+1 {
		log.Warn("too few etcd members", map[string]interface{}{
			"num_healthy": numHealthy,
			log.FnError:   err,
		})
		return nil
	}

	return nil
}

func getEtcdMembers(ctx context.Context, nodes []*Node) ([]*etcdserverpb.Member, error) {
	endpoints := make([]string, len(nodes))
	for i, n := range nodes {
		endpoints[i] = fmt.Sprintf("http://%s:2379", n.Address)
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	resp, err := cli.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Members, nil
}

func getEtcdMembersStatus(ctx context.Context, members []*etcdserverpb.Member) map[string]status {
	memberStatus := make(map[string]status)
	for _, member := range members {
		if len(member.ClientURLs) == 0 {
			memberStatus[member.Name] = "notstarted"
			continue
		}
		endpoint := member.ClientURLs[0] + "/health"
		client := &cmd.HTTPClient{
			Client: &http.Client{},
		}
		req, _ := http.NewRequest("GET", endpoint, nil)
		req = req.WithContext(ctx)
		resp, err := client.Do(req)
		if err != nil {
			memberStatus[member.Name] = "dead"
			continue
		}
		health := new(etcdhttp.Health)
		err = json.NewDecoder(resp.Body).Decode(health)
		resp.Body.Close()
		if err != nil {
			memberStatus[member.Name] = "unhealthy"
			continue
		}
		switch health.Health {
		case "true":
			memberStatus[member.Name] = "healthy"
		default:
			memberStatus[member.Name] = "unhealthy"
		}
	}

	return memberStatus
}

func containsMember(cpNodes []*Node, member *etcdserverpb.Member) bool {
	for _, n := range cpNodes {
		if n.Address == member.Name {
			return true
		}
	}
	return false
}
