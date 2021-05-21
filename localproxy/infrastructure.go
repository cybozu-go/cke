package localproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	vault "github.com/hashicorp/vault/api"
	"go.etcd.io/etcd/clientv3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var kubeHTTP cke.KubeHTTP

type localInfra struct {
	storage cke.Storage
	vc      *vault.Client
}

var _ cke.Infrastructure = &localInfra{}

func newInfrastructure(storage cke.Storage) cke.Infrastructure {
	return &localInfra{storage: storage}
}

func (i *localInfra) Close() {}

func (i *localInfra) Agent(addr string) cke.Agent {
	panic("not implemented") // TODO: Implement
}

func (i *localInfra) Engine(addr string) cke.ContainerEngine {
	return localDocker{}
}

func (i *localInfra) Vault() (*vault.Client, error) {
	if i.vc != nil {
		return i.vc, nil
	}

	cfg, err := i.Storage().GetVaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	vc, _, err := cke.VaultClient(cfg)
	if err != nil {
		return nil, err
	}

	i.vc = vc
	return vc, nil
}

func (i *localInfra) Storage() cke.Storage {
	return i.storage
}

func (i *localInfra) NewEtcdClient(ctx context.Context, endpoints []string) (*clientv3.Client, error) {
	panic("not implemented") // TODO: Implement
}

func (i *localInfra) K8sConfig(ctx context.Context, n *cke.Node) (*rest.Config, error) {
	panic("not implemented") // TODO: Implement
}

func (i *localInfra) K8sClient(ctx context.Context, n *cke.Node) (*kubernetes.Clientset, error) {
	if err := kubeHTTP.Init(ctx, i); err != nil {
		return nil, err
	}

	c, k, err := kubeHTTP.GetCert(ctx, i)
	if err != nil {
		return nil, err
	}
	tlsCfg := rest.TLSClientConfig{
		CertData: []byte(c),
		KeyData:  []byte(k),
		CAData:   []byte(kubeHTTP.CACert()),
	}
	cfg := &rest.Config{
		Host:            "https://" + n.Address + ":6443",
		TLSClientConfig: tlsCfg,
		Timeout:         5 * time.Second,
	}
	return kubernetes.NewForConfig(cfg)
}

func (i *localInfra) HTTPClient() *well.HTTPClient {
	panic("not implemented") // TODO: Implement
}

func (i *localInfra) HTTPSClient(ctx context.Context) (*well.HTTPClient, error) {
	panic("not implemented") // TODO: Implement
}

type localDocker struct{}

var _ cke.ContainerEngine = localDocker{}

// PullImage pulls an image.
func (l localDocker) PullImage(img cke.Image) error {
	cmd := exec.Command("docker", "image", "list", "--format={{.Repository}}:{{.Tag}}")
	stdout, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute docker image list: %w", err)
	}

	for _, i := range strings.Fields(string(stdout)) {
		if img.Name() == i {
			return nil
		}
	}

	return exec.Command("docker", "image", "pull", img.Name()).Run()
}

// Run runs a container as a foreground process.
func (l localDocker) Run(img cke.Image, binds []cke.Mount, command string, args ...string) error {
	runArgs := []string{
		"run",
		"--log-driver=journald",
		"--rm",
		"--network=host",
		"--uts=host",
		"--read-only",
	}
	for _, m := range binds {
		o := "rw"
		if m.ReadOnly {
			o = "ro"
		}
		runArgs = append(runArgs, fmt.Sprintf("--volume=%s:%s:%s", m.Source, m.Destination, o))
	}
	runArgs = append(runArgs, img.Name(), command)
	runArgs = append(runArgs, args...)

	out, err := exec.Command("docker", runArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run %s: %s: %w", img.Name(), out, err)
	}
	return nil
}

// RunWithInput runs a container as a foreground process with stdin as a string.
func (l localDocker) RunWithInput(img cke.Image, binds []cke.Mount, command, input string, args ...string) error {
	runArgs := []string{
		"run",
		"--log-driver=journald",
		"--rm",
		"-i",
		"--network=host",
		"--uts=host",
		"--read-only",
	}
	for _, m := range binds {
		o := "rw"
		if m.ReadOnly {
			o = "ro"
		}
		runArgs = append(runArgs, fmt.Sprintf("--volume=%s:%s:%s", m.Source, m.Destination, o))
	}
	runArgs = append(runArgs, img.Name(), command)
	runArgs = append(runArgs, args...)

	cmd := exec.Command("docker", runArgs...)
	cmd.Stdin = bytes.NewReader([]byte(input))

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run %s: %s: %w", img.Name(), out, err)
	}
	return nil
}

/// RunWithOutput runs a container as a foreground process and get stdout and stderr.
func (l localDocker) RunWithOutput(img cke.Image, binds []cke.Mount, command string, args ...string) ([]byte, []byte, error) {
	runArgs := []string{
		"run",
		"--log-driver=journald",
		"--rm",
		"--network=host",
		"--uts=host",
		"--read-only",
	}
	for _, m := range binds {
		o := "rw"
		if m.ReadOnly {
			o = "ro"
		}
		runArgs = append(runArgs, fmt.Sprintf("--volume=%s:%s:%s", m.Source, m.Destination, o))
	}
	runArgs = append(runArgs, img.Name(), command)
	runArgs = append(runArgs, args...)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := exec.Command("docker", runArgs...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// RunSystem runs the named container as a system service.
func (l localDocker) RunSystem(name string, img cke.Image, opts []string, params cke.ServiceParams, extra cke.ServiceParams) error {
	args := []string{
		"run",
		"--rm",
		"--log-driver=journald",
		"-d",
		"--name=" + name,
		"--read-only",
		"--network=host",
		"--uts=host",
	}
	args = append(args, opts...)

	for _, m := range append(params.ExtraBinds, extra.ExtraBinds...) {
		var opts []string
		if m.ReadOnly {
			opts = append(opts, "ro")
		}
		if len(m.Propagation) > 0 {
			opts = append(opts, m.Propagation.String())
		}
		if len(m.Label) > 0 {
			opts = append(opts, m.Label.String())
		}
		args = append(args, fmt.Sprintf("--volume=%s:%s:%s", m.Source, m.Destination, strings.Join(opts, ",")))
	}
	for k, v := range params.ExtraEnvvar {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range extra.ExtraEnvvar {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	type ckeLabel struct {
		BuiltInParams cke.ServiceParams `json:"builtin"`
		ExtraParams   cke.ServiceParams `json:"extra"`
	}

	label := ckeLabel{
		BuiltInParams: params,
		ExtraParams:   extra,
	}
	data, err := json.Marshal(label)
	if err != nil {
		return err
	}
	labelFile, err := os.CreateTemp("", "cke-")
	if err != nil {
		return err
	}
	defer func() {
		labelFile.Close()
		os.Remove(labelFile.Name())
	}()
	if _, err := io.WriteString(labelFile, cke.CKELabelName+"="+string(data)); err != nil {
		return err
	}
	args = append(args, "--label-file="+labelFile.Name())

	args = append(args, img.Name())

	args = append(args, params.ExtraArguments...)
	args = append(args, extra.ExtraArguments...)

	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to docker run %s: %s: %w", img.Name(), out, err)
	}
	return nil
}

// Exists returns if named system container exists.
func (l localDocker) Exists(name string) (bool, error) {
	args := []string{
		"ps", "-a", "--no-trunc", "--filter=name=^/" + name + "$", "--format={{.ID}}",
	}
	stdout, err := exec.Command("docker", args...).Output()
	if err != nil {
		return false, fmt.Errorf("failed to run docker ps: %w", err)
	}
	return len(bytes.TrimSpace(stdout)) > 0, nil
}

// Stop stops the named system container.
func (l localDocker) Stop(name string) error {
	return exec.Command("docker", "container", "stop", name).Run()
}

// Kill kills the named system container.
func (l localDocker) Kill(name string) error {
	return exec.Command("docker", "container", "kill", name).Run()
}

// Remove removes the named system container.
func (l localDocker) Remove(name string) error {
	return exec.Command("docker", "container", "rm", name).Run()
}

// Inspect returns ServiceStatus for the named container.
func (l localDocker) Inspect(name []string) (map[string]cke.ServiceStatus, error) {
	panic("not implemented") // TODO: Implement
}

// VolumeCreate creates a local volume.
func (l localDocker) VolumeCreate(name string) error {
	panic("not implemented") // TODO: Implement
}

// VolumeRemove creates a local volume.
func (l localDocker) VolumeRemove(name string) error {
	panic("not implemented") // TODO: Implement
}

// VolumeExists returns true if the named volume exists.
func (l localDocker) VolumeExists(name string) (bool, error) {
	panic("not implemented") // TODO: Implement
}
