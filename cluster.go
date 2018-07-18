package cke

import (
	"errors"
	"net"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// Node represents a node in Kubernetes.
type Node struct {
	Address      string            `json:"address"       yaml:"address"`
	Hostname     string            `json:"hostname"      yaml:"hostname"`
	User         string            `json:"user"          yaml:"user"`
	SSHKey       string            `json:"ssh_key"       yaml:"ssh_key"`
	ControlPlane bool              `json:"control_plane" yaml:"control_plane"`
	Labels       map[string]string `json:"labels"        yaml:"labels"`

	signer ssh.Signer
}

// Mount is volume mount information
type Mount struct {
	Source      string `json:"source"      yaml:"source"`
	Destination string `json:"destination" yaml:"destination"`
	ReadOnly    bool   `json:"read_only"   yaml:"read_only"`
}

// ServiceParams is a common set of extra parameters for k8s components.
type ServiceParams struct {
	ExtraArguments []string          `json:"extra_args"  yaml:"extra_args"`
	ExtraBinds     []Mount           `json:"extra_binds" yaml:"extra_binds"`
	ExtraEnvvar    map[string]string `json:"extra_env"   yaml:"extra_env"`
}

// EtcdParams is a set of extra parameters for etcd.
type EtcdParams struct {
	ServiceParams `yaml:",inline"`
	VolumeName    string `json:"volume_name" yaml:"volume_name"`
}

// KubeletParams is a set of extra parameters for kubelet.
type KubeletParams struct {
	ServiceParams `yaml:",inline"`
	Domain        string `json:"domain"      yaml:"domain"`
	AllowSwap     bool   `json:"allow_swap"  yaml:"allow_swap"`
}

// Options is a set of optional parameters for k8s components.
type Options struct {
	Etcd       EtcdParams    `json:"etcd"            yaml:"etcd"`
	APIServer  ServiceParams `json:"kube-api"        yaml:"kube-api"`
	Controller ServiceParams `json:"kube-controller" yaml:"kube-controller"`
	Scheduler  ServiceParams `json:"kube-scheduler"  yaml:"kube-scheduler"`
	Proxy      ServiceParams `json:"kube-proxy"      yaml:"kube-proxy"`
	Kubelet    KubeletParams `json:"kubelet"         yaml:"kubelet"`
}

// Cluster is a set of configurations for a etcd/Kubernetes cluster.
type Cluster struct {
	Name          string   `json:"name"           yaml:"name"`
	Nodes         []*Node  `json:"nodes"          yaml:"nodes"`
	SSHKey        string   `json:"ssh_key"        yaml:"ssh_key"`
	ServiceSubnet string   `json:"service_subnet" yaml:"service_subnet"`
	DNSServers    []string `json:"dns_servers"    yaml:"dns_servers"`
	Options       Options  `json:"options"        yaml:"options"`
	RBAC          bool     `json:"rbac"           yaml:"rbac"`
}

// Validate validates the cluster definition.
func (c *Cluster) Validate() error {
	if len(c.Name) == 0 {
		return errors.New("cluster name is empty")
	}

	_, _, err := net.ParseCIDR(c.ServiceSubnet)
	if err != nil {
		return err
	}

	for _, n := range c.Nodes {
		err := c.validateNode(n)
		if err != nil {
			return err
		}
	}

	for _, a := range c.DNSServers {
		if net.ParseIP(a) == nil {
			return errors.New("invalid IP address: " + a)
		}
	}

	err = validateOptions(c.Options)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) validateNode(n *Node) error {
	if net.ParseIP(n.Address) == nil {
		return errors.New("invalid IP address: " + n.Address)
	}
	if len(n.User) == 0 {
		return errors.New("user name is empty")
	}
	if len(c.SSHKey) == 0 && len(n.SSHKey) == 0 {
		return errors.New("no SSH private key")
	}
	key := n.SSHKey
	if len(key) == 0 {
		key = c.SSHKey
	}

	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		return err
	}
	n.signer = signer
	return nil
}

func validateOptions(opts Options) error {
	v := func(binds []Mount) error {
		for _, m := range binds {
			if !filepath.IsAbs(m.Source) {
				return errors.New("source path must be absolute: " + m.Source)
			}
			if !filepath.IsAbs(m.Destination) {
				return errors.New("destination path must be absolute: " + m.Destination)
			}
		}
		return nil
	}

	err := v(opts.Etcd.ExtraBinds)
	if err != nil {
		return err
	}
	err = v(opts.APIServer.ExtraBinds)
	if err != nil {
		return err
	}
	err = v(opts.Controller.ExtraBinds)
	if err != nil {
		return err
	}
	err = v(opts.Scheduler.ExtraBinds)
	if err != nil {
		return err
	}
	err = v(opts.Proxy.ExtraBinds)
	if err != nil {
		return err
	}
	err = v(opts.Kubelet.ExtraBinds)
	if err != nil {
		return err
	}

	return nil
}
