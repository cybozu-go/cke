package cke

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
)

const paramsDir = "/run/cke"

type container struct {
	name  string
	agent Agent
}

func (c container) run(image string, params, extra ServiceParams) error {
	return nil
}

func (c container) inspect() (*ServiceStatus, error) {
	id, _, err := c.agent.Run(fmt.Sprintf("docker ps --filter name=%s --format {{.ID}}", c.name))
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if len(id) == 0 {
		return &ServiceStatus{}, nil
	}

	data, _, err := c.agent.Run("docker container inspect " + id)
	if err != nil {
		return nil, err
	}

	dj := new(types.ContainerJSON)
	err = json.Unmarshal([]byte(data), dj)
	if err != nil {
		return nil, err
	}

	params := new(ServiceParams)
	if dj.State.Running {
		data, _, err := c.agent.Run("cat " + filepath.Join(paramsDir, c.name))
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(data), params)
		if err != nil {
			return nil, err
		}
	}

	return &ServiceStatus{
		Running:        dj.State.Running,
		Image:          dj.Image,
		ExtraArguments: params.ExtraArguments,
		ExtraBinds:     params.ExtraBinds,
		ExtraEnvvar:    params.ExtraEnvvar,
	}, nil
}
