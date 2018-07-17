package cke

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
)

const (
	paramsDir = "/run/cke"
)

type container struct {
	name  string
	agent Agent
}

// runSystem run container as a system service.
func (c container) runSystem(image string, opts []string, params, extra ServiceParams) error {
	err := os.MkdirAll(paramsDir, 0755)
	if err != nil {
		return err
	}

	id, err := c.getID()
	if err != nil {
		return err
	}
	if len(id) != 0 {
		_, _, err := c.agent.Run("docker rm " + c.name)
		if err != nil {
			return err
		}
	}

	args := []string{
		"docker",
		"run",
		"-d",
		"--name=" + c.name,
		"--network=host",
		"--uts=host",
		"--restart=unless-stopped",
	}
	args = append(args, opts...)

	for _, m := range append(params.ExtraBinds, extra.ExtraBinds...) {
		o := "rw"
		if m.ReadOnly {
			o = "ro"
		}
		args = append(args, fmt.Sprintf("--volume=%s:%s:%s", m.Source, m.Destination, o))
	}
	for k, v := range params.ExtraEnvvar {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range extra.ExtraEnvvar {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, image)

	args = append(args, params.ExtraArguments...)
	args = append(args, extra.ExtraArguments...)

	_, _, err = c.agent.Run(strings.Join(args, " "))
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(paramsDir, c.name),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(extra)
}

func (c container) getID() (string, error) {
	dockerPS := "docker ps -a --filter name=%s --format {{.ID}}"
	data, _, err := c.agent.Run(fmt.Sprintf(dockerPS, c.name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (c container) inspect() (*ServiceStatus, error) {
	id, err := c.getID()
	if err != nil {
		return nil, err
	}
	if len(id) == 0 {
		return &ServiceStatus{}, nil
	}

	data, _, err := c.agent.Run("docker container inspect " + id)
	if err != nil {
		return nil, err
	}

	dj := new(types.ContainerJSON)
	err = json.Unmarshal(data, dj)
	if err != nil {
		return nil, err
	}

	params := new(ServiceParams)
	if dj.State.Running {
		data, _, err := c.agent.Run("cat " + filepath.Join(paramsDir, c.name))
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(data, params)
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
