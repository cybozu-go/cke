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

type EtcdMemberHealth string
// EtcdStatus is the status of kubelet.
type EtcdStatus struct {
	ServiceStatus
	HasData bool
	IsControlPlane bool
	health EtcdMemberHealth
}

type EtcdClusterStatus struct {
	statuses map[string]*EtcdStatus
	members []*etcdserverpb.Member
	numControlPlane int
}

func (s EtcdClusterStatus) IsInitialized() bool {
	for _, status := range s.statuses {
		if status.HasData {
			return true
		}
	}
	return false
}

func (s EtcdClusterStatus) NumberOfHealthy() int {
	numHealthy := 0
	for _, member := range s.members {
		if s.statuses[member.Name].health == "healthy" && s.statuses[member.Name].IsControlPlane {
			numHealthy++
		}
	}
	return numHealthy
}

func (s EtcdClusterStatus) AvailableCluster() bool{
	return s.NumberOfHealthy() > (s.numControlPlane / 2)
}


type EtcdStrategy struct {
	status EtcdClusterStatus
	cpNodes []*Node
	cluster *Cluster
	agents map[string]Agent
}


func getEtcdNodeStatus(agent Agent, cluster *Cluster) (*EtcdStatus, error) {
	ce := Docker(agent)

	// etcd status
	ss, err := ce.Inspect("etcd")
	if err != nil {
		return nil, err
	}
	ok, err := ce.VolumeExists(etcdVolumeName(cluster))
	if err != nil {
		return nil, err
	}
	return  &EtcdStatus{*ss, ok, true, ""} , nil
}

func (s EtcdStrategy) FetchStatus(ctx context.Context, c *Cluster, agents map[string]Agent) error {


	var cpNodes []*Node
	for _, n := range c.Nodes {
		if n.ControlPlane {
			cpNodes = append(cpNodes, n)
		}
	}
	s.cluster = c
	s.agents = agents
	s.cpNodes = cpNodes
	s.status.numControlPlane = len (cpNodes)

	bootstrap := true
	for _, n := range cpNodes {
		status,err := getEtcdNodeStatus(agents[n.Address],c)
		if err !=nil {
			log.Warn("node status is not available", map[string]interface{}{
				"node": n.Address,
			})
			return err
		}
		s.status.statuses[n.Address] = status
		if status.HasData {
			bootstrap = false
		}
	}
	if bootstrap {
		return nil
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
		return err
	}

	for _, member := range members {
		health:= getEtcdMemberHealth(ctx, member)
		if _, ok := s.status.statuses[member.Name]; ok {
			s.status.statuses[member.Name].health = health
		} else {
			status,err := getEtcdNodeStatus(agents[member.Name],c)
			if err !=nil {
				log.Warn("node status is not available", map[string]interface{}{
					"node": member.Name,
				})
			}
			status.IsControlPlane = false
			status.health = health
			s.status.statuses[member.Name] = status
		}
	}

	return nil
}

func (s EtcdStrategy) DecideToDo() Operator {
	if !s.status.IsInitialized() {
		return EtcdBootOp(s.cpNodes, s.agents, etcdVolumeName(s.cluster), s.cluster.Options.Etcd.ServiceParams)
	}

	if !s.status.AvailableCluster() {
		log.Warn("too few etcd members", map[string]interface{}{
			"num_healthy": s.status.NumberOfHealthy(),
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

func getEtcdMemberHealth(ctx context.Context, member *etcdserverpb.Member) EtcdMemberHealth {
		if len(member.ClientURLs) == 0 {
			return  "notstarted"
		}
		endpoint := member.ClientURLs[0] + "/health"
		client := &cmd.HTTPClient{
			Client: &http.Client{},
		}
		req, _ := http.NewRequest("GET", endpoint, nil)
		req = req.WithContext(ctx)
		resp, err := client.Do(req)
		if err != nil {
			return "dead"
		}
		health := new(etcdhttp.Health)
		err = json.NewDecoder(resp.Body).Decode(health)
		resp.Body.Close()
		if err != nil {
			return "unhealthy"
		}
		switch health.Health {
		case "true":
			return "healthy"
		default:
			return "unhealthy"
		}
}

