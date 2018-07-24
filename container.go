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

// ContainerEngine defines interfaces for a container engine.
type ContainerEngine interface {
	// PullImage pulls the image for the named container.
	PullImage(name string) error
	// Run runs the named container as a foreground process.
	Run(name string, binds []Mount, command string) error
	// RunSystem runs the named container as a system service.
	RunSystem(name string, opts []string, params, extra ServiceParams) error
	// Stop stops the named container.
	Stop(name string) error
	// Remove removes the named container.
	Remove(name string) error
	// Inspect returns ServiceStatus for the named container.
	Inspect(name string) (*ServiceStatus, error)
	// VolumeCreate creates a local volume.
	VolumeCreate(name string) error
	// VolumeRemove creates a local volume.
	VolumeRemove(name string) error
	// VolumeExists returns true if the named volume exists.
	VolumeExists(name string) (bool, error)
}

// Docker is an implementation of ContainerEngine.
func Docker(agent Agent) ContainerEngine {
	return docker{agent}
}

type docker struct {
	agent Agent
}

func (c docker) PullImage(name string) error {
	img := Image(name)
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

func (c docker) Run(name string, binds []Mount, command string) error {
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
	args = append(args, Image(name), command)

	_, _, err := c.agent.Run(strings.Join(args, " "))
	return err
}

func (c docker) RunSystem(name string, opts []string, params, extra ServiceParams) error {
	id, err := c.getID(name)
	if err != nil {
		return err
	}
	if len(id) != 0 {
		_, _, err := c.agent.Run("docker rm " + name)
		if err != nil {
			return err
		}
	}

	args := []string{
		"docker",
		"run",
		"-d",
		"--name=" + name,
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

	args = append(args, Image(name))

	args = append(args, params.ExtraArguments...)
	args = append(args, extra.ExtraArguments...)

	stdout, stderr, err := c.agent.Run(strings.Join(args, " "))
	if err != nil {
		return errors.Wrapf(err, "stdout: %s, stderr: %s", stdout, stderr)
	}
	return nil
}

func (c docker) Stop(name string) error {
	stdout, stderr, err := c.agent.Run("docker container stop " + name)
	if err != nil {
		return errors.Wrapf(err, "stdout: %s, stderr: %s", stdout, stderr)
	}
	return nil
}

func (c docker) Remove(name string) error {
	stdout, stderr, err := c.agent.Run("docker container rm " + name)
	if err != nil {
		return errors.Wrapf(err, "stdout: %s, stderr: %s", stdout, stderr)
	}
	return nil
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

func (c docker) getID(name string) (string, error) {
	dockerPS := "docker ps -a --filter name=%s --format {{.ID}}"
	data, _, err := c.agent.Run(fmt.Sprintf(dockerPS, name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (c docker) Inspect(name string) (*ServiceStatus, error) {
	id, err := c.getID(name)
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
	stdout, stderr, err := c.agent.Run("docker volume create " + name)
	if err != nil {
		return errors.Wrapf(err, "stdout: %s, stderr: %s", stdout, stderr)
	}
	return nil
}

func (c docker) VolumeRemove(name string) error {
	stdout, stderr, err := c.agent.Run("docker volume remove " + name)
	if err != nil {
		return errors.Wrapf(err, "stdout: %s, stderr: %s", stdout, stderr)
	}
	return nil
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
