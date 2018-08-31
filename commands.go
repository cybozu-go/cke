package cke

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/cmd"
)

// Command represents some command
type Command struct {
	Name   string `json:"name"`
	Target string `json:"target"`
	Detail string `json:"detail"`
}

// String implements fmt.Stringer
func (c Command) String() string {
	if len(c.Detail) > 0 {
		return fmt.Sprintf("%s %s: %s", c.Name, c.Target, c.Detail)
	}
	return fmt.Sprintf("%s %s", c.Name, c.Target)
}

// Commander is a single step to proceed an operation
type Commander interface {
	// Run executes the command
	Run(ctx context.Context, inf Infrastructure) error
	// Command returns the command information
	Command() Command
}

type makeDirCommand struct {
	nodes     []*Node
	targetDir string
}

func (c makeDirCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	binds := []Mount{{
		Source:      filepath.Dir(c.targetDir),
		Destination: filepath.Join("/mnt", filepath.Dir(c.targetDir)),
	}}
	mkdirCommand := "mkdir -p " + filepath.Join("/mnt", c.targetDir)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.Run("tools", binds, mkdirCommand)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c makeDirCommand) Command() Command {
	return Command{
		Name:   "mkdir",
		Target: c.targetDir,
	}
}

type makeFileCommand struct {
	nodes  []*Node
	source string
	target string
}

func (c makeFileCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	targetDir := filepath.Dir(c.target)
	binds := []Mount{{
		Source:      targetDir,
		Destination: filepath.Join("/mnt", targetDir),
	}}
	mkdirCommand := "mkdir -p " + filepath.Join("/mnt", targetDir)
	ddCommand := "dd of=" + filepath.Join("/mnt", c.target)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			err := ce.Run("tools", binds, mkdirCommand)
			if err != nil {
				return err
			}
			return ce.RunWithInput("tools", binds, ddCommand, c.source)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c makeFileCommand) Command() Command {
	return Command{
		Name:   "make-file",
		Target: c.target,
	}
}

type removeFileCommand struct {
	nodes  []*Node
	target string
}

func (c removeFileCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	dir := filepath.Dir(c.target)
	binds := []Mount{{
		Source:      dir,
		Destination: filepath.Join("/mnt", dir),
	}}
	command := "rm -f " + filepath.Join("/mnt", c.target)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.Run("tools", binds, command)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c removeFileCommand) Command() Command {
	return Command{
		Name:   "rm",
		Target: c.target,
	}
}

type imagePullCommand struct {
	nodes []*Node
	name  string
}

func (c imagePullCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.PullImage(c.name)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c imagePullCommand) Command() Command {
	return Command{
		Name:   "image-pull",
		Target: c.name,
		Detail: Image(c.name),
	}
}

type volumeCreateCommand struct {
	nodes   []*Node
	volname string
}

func (c volumeCreateCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.VolumeCreate(c.volname)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c volumeCreateCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "volume-create",
		Target: strings.Join(targets, ","),
		Detail: c.volname,
	}
}

type volumeRemoveCommand struct {
	nodes   []*Node
	volname string
}

func (c volumeRemoveCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			exists, err := ce.VolumeExists(c.volname)
			if err != nil {
				return err
			}
			if exists {
				return ce.VolumeRemove(c.volname)
			}
			return nil
		})
	}
	env.Stop()
	return env.Wait()
}

func (c volumeRemoveCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "volume-remove",
		Target: strings.Join(targets, ","),
		Detail: c.volname,
	}
}

type runContainerCommand struct {
	nodes  []*Node
	name   string
	opts   []string
	params ServiceParams
	extra  ServiceParams
}

func (c runContainerCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.RunSystem(c.name, c.opts, c.params, c.extra)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c runContainerCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "run-container",
		Target: strings.Join(targets, ","),
		Detail: c.name,
	}
}

type stopContainerCommand struct {
	node *Node
	name string
}

func (c stopContainerCommand) Run(ctx context.Context, inf Infrastructure) error {
	ce := Docker(inf.Agent(c.node.Address))
	exists, err := ce.Exists(c.name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	err = ce.Stop(c.name)
	if err != nil {
		return err
	}
	// gofail: var dockerAfterContainerStop struct{}
	return ce.Remove(c.name)
}

func (c stopContainerCommand) Command() Command {
	return Command{
		Name:   "stop-container",
		Target: c.node.Address,
		Detail: c.name,
	}
}

type issueEtcdCertificatesCommand struct {
	nodes []*Node
}

// CA Keys in Vault
const (
	CAServer     = "cke/ca-server"
	CAEtcdPeer   = "cke/ca-etcd-peer"
	CAEtcdClient = "cke/ca-etcd-client"
)

func writeFile(inf Infrastructure, node *Node, target string, source string) error {
	targetDir := filepath.Dir(target)
	binds := []Mount{{
		Source:      targetDir,
		Destination: filepath.Join("/mnt", targetDir),
	}}
	mkdirCommand := "mkdir -p " + filepath.Join("/mnt", targetDir)
	ddCommand := "dd of=" + filepath.Join("/mnt", target)
	ce := Docker(inf.Agent(node.Address))
	err := ce.Run("tools", binds, mkdirCommand)
	if err != nil {
		return err
	}
	return ce.RunWithInput("tools", binds, ddCommand, source)
}

func issueCertificate(inf Infrastructure, node *Node, ca, name string, opts map[string]interface{}) error {
	hostname := node.Hostname
	if len(hostname) == 0 {
		hostname = node.Address
	}
	data := map[string]interface{}{
		"common_name": hostname,
		"ip_sans":     "127.0.0.1," + node.Address,
	}
	for k, v := range opts {
		data[k] = v
	}
	fmt.Println(data)
	client := inf.Vault()
	if client == nil {
		return errors.New("can not connect to vault")
	}
	secret, err := client.Logical().Write(ca+"/issue/system", data)
	if err != nil {
		return err
	}
	err = writeFile(inf, node, "/etc/etcd/pki/"+name+".crt", secret.Data["certificate"].(string))
	if err != nil {
		return err
	}
	err = writeFile(inf, node, "/etc/etcd/pki/"+name+".key", secret.Data["private_key"].(string))
	if err != nil {
		return err
	}
	return nil
}

func issueEtcdCertificates(ctx context.Context, inf Infrastructure, node *Node) error {
	err := issueCertificate(inf, node, CAServer, "server", map[string]interface{}{
		"alt_names": "localhost",
	})
	if err != nil {
		return err
	}
	err = issueCertificate(inf, node, CAEtcdPeer, "peer", map[string]interface{}{
		"exclude_cn_from_sans": "true",
	})
	if err != nil {
		return err
	}
	err = issueCertificate(inf, node, CAEtcdClient, "etcdctl", map[string]interface{}{
		"exclude_cn_from_sans": "true",
	})
	if err != nil {
		return err
	}

	ca, err := inf.Storage().GetCACertificate(ctx, "etcd-peer")
	if err != nil {
		return err
	}
	err = writeFile(inf, node, "/etc/etcd/pki/peer.crt", ca)
	if err != nil {
		return err
	}
	ca, err = inf.Storage().GetCACertificate(ctx, "etcd-client")
	if err != nil {
		return err
	}
	return writeFile(inf, node, "/etc/etcd/pki/client.crt", ca)
}

func (c issueEtcdCertificatesCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, node := range c.nodes {
		env.Go(func(ctx context.Context) error {
			return issueEtcdCertificates(ctx, inf, node)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c issueEtcdCertificatesCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "issue-etcd-certificate",
		Target: strings.Join(targets, ","),
	}
}
