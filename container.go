package cke

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"crypto/rand"
	"encoding/hex"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
)

const (
	ckeLabelName = "com.cybozu.cke"
)

// Container is the interface to manage a container and the image for it.
type Container interface {
	// PullImage pulls the image for the container.
	PullImage() error
	// Run runs the container as a foreground process.
	Run(binds []Mount, command string) error
	// RunSystem runs the container as a system service.
	RunSystem(opts []string, params, extra ServiceParams) error
	// Inspect returns ServiceStatus for the container.
	Inspect() (*ServiceStatus, error)
	// VolumeCreate creates a local volume.
	VolumeCreate(name string) error
	// VolumeExists returns true if the named volume exists.
	VolumeExists(name string) (bool, error)
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
		"--read-only",
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
	data, err := json.Marshal(extra)
	if err != nil {
		return err
	}
	labelFile, err := c.putData(ckeLabelName + "=" + string(data))
	if err != nil {
		return err
	}
	args = append(args, "--label-file="+labelFile)

	args = append(args, Image(c.name))

	args = append(args, params.ExtraArguments...)
	args = append(args, extra.ExtraArguments...)

	_, _, err = c.agent.Run(strings.Join(args, " "))
	return err
}

func (c docker) putData(data string) (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	fileName := filepath.Join("/tmp", hex.EncodeToString(b))
	err = c.agent.RunWithInput("tee "+fileName, data)
	if err != nil {
		return "", err
	}
	return fileName, nil
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

	var djs []types.ContainerJSON
	err = json.Unmarshal(data, &djs)
	if err != nil {
		return nil, err
	}
	if len(djs) != 1 {
		return nil, errors.New("unexpected docker inspect result")
	}
	dj := djs[0]

	params := new(ServiceParams)
	label := dj.Config.Labels[ckeLabelName]

	err = json.Unmarshal([]byte(label), params)
	if err != nil {
		return nil, err
	}

	return &ServiceStatus{
		Running:        dj.State.Running,
		Image:          dj.Image,
		ExtraArguments: params.ExtraArguments,
		ExtraBinds:     params.ExtraBinds,
		ExtraEnvvar:    params.ExtraEnvvar,
	}, nil
}

func (c docker) VolumeCreate(name string) error {
	_, _, err := c.agent.Run("docker volume create " + name)
	return err
}

func (c docker) VolumeExists(name string) (bool, error) {
	data, _, err := c.agent.Run("docker volume list -q")
	if err != nil {
		return false, err
	}

	for _, n := range strings.Split(string(data), "\n") {
		if n == name {
			return true, nil
		}
	}
	return false, nil
}
