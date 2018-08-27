package cke

import (
	"github.com/pkg/errors"
)

// Infrastructure presents an interface for infrastructure on CKE
type Infrastructure interface {
	Close()
	Agent(addr string) Agent
}

// NewInfrastructure creates a new Infrastructure instance
func NewInfrastructure(c *Cluster) (Infrastructure, error) {
	agents := make(map[string]Agent)
	defer func() {
		for _, a := range agents {
			a.Close()
		}
	}()

	for _, n := range c.Nodes {
		a, err := SSHAgent(n)
		if err != nil {
			return nil, errors.Wrap(err, n.Address)
		}
		agents[n.Address] = a
	}

	// These assignments should be placed last.

	inf := &ckeInfrastructure{agents: agents}
	agents = nil
	return inf, nil

}

type ckeInfrastructure struct {
	agents map[string]Agent
}

func (i ckeInfrastructure) Agent(addr string) Agent {
	return i.agents[addr]
}

func (i ckeInfrastructure) Close() {
	for _, a := range i.agents {
		a.Close()
	}
	i.agents = nil
}
