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

// Container is the interface to manage a container and the image for it.
type Container interface {
	// PullImage pulls the image for the container.
	PullImage() error
	// RunSystem run container as a system service.
	RunSystem(opts []string, params, extra ServiceParams) error
	// Run runs a container.
	Run(binds []Mount, command string) error
	// Inspect returns ServiceStatus for the container.
	Inspect() (*ServiceStatus, error)
}

// Docker is an implementation of Container using Docker.
func Docker(name string, agent Agent) Container {
	return docker{name, agent}
}

type docker struct {
	name  string
	agent Agent
}

func (c docker) PullImage() error {
	img := Image(c.name)
	data, _, err := c.agent.Run("docker image list --format '{{.Repository}}:{{.Tag}}'")
	if err != nil {
		return err
	}

	for _, i := range strings.Split(string(data), "\n") {
		if img == i {
			return nil
		}
	}

	_, _, err = c.agent.Run("docker image pull " + img)
	return err
}

func (c docker) Run(binds []Mount, command string) error {
	args := []string{
		"docker",
		"run",
		"-d",
		"--rm",
		"--network=host",
		"--uts=host",
	}
	for _, m := range binds {
		o := "rw"
		if m.ReadOnly {
			o = "ro"
		}
		args = append(args, fmt.Sprintf("--volume=%s:%s:%s", m.Source, m.Destination, o))
	}
	args = append(args, Image(c.name), command)

	_, _, err := c.agent.Run(strings.Join(args, " "))
	return err
}

func (c docker) RunSystem(opts []string, params, extra ServiceParams) error {
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

	args = append(args, Image(c.name))

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

func (c docker) getID() (string, error) {
	dockerPS := "docker ps -a --filter name=%s --format {{.ID}}"
	data, _, err := c.agent.Run(fmt.Sprintf(dockerPS, c.name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (c docker) Inspect() (*ServiceStatus, error) {
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
