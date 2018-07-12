package cke

// Node represents a node in Kubernetes.
type Node struct {
	Address      string            `json:"address"       yaml:"address"`
	Hostname     string            `json:"hostname"      yaml:"hostname"`
	User         string            `json:"user"          yaml:"user"`
	SSHKey       string            `json:"ssh_key"       yaml:"ssh_key"`
	ControlPlane bool              `json:"control_plane" yaml:"control_plane"`
	Labels       map[string]string `json:"labels"        yaml:"labels"`
}

// ServiceParams is a common set of extra parameters for k8s components.
type ServiceParams struct {
	ExtraArguments map[string]string `json:"extra_args"  yaml:"extra_args"`
	ExtraBinds     map[string]string `json:"extra_binds" yaml:"extra_binds"`
	ExtraEnvvar    map[string]string `json:"extra_env"   yaml:"extra_env"`
}

// KubeletParams is a set of extra parameters for kubelet.
type KubeletParams struct {
	ServiceParams
	Domain    string `json:"cluster_domain" yaml:"cluster_domain"`
	AllowSwap bool   `json:"allow_swap"     yaml:"allow_swap"`
}

// Options is a set of optional parameters for k8s components.
type Options struct {
	Etcd       ServiceParams `json:"etcd"            yaml:"etcd"`
	APIServer  ServiceParams `json:"kube-api"        yaml:"kube-api"`
	Controller ServiceParams `json:"kube-controller" yaml:"kube-controller"`
	Scheduler  ServiceParams `json:"kube-scheduler"  yaml:"kube-scheduler"`
	Proxy      ServiceParams `json:"kube-proxy"      yaml:"kube-proxy"`
	Kubelet    KubeletParams `json:"kubelet"         yaml:"kubelet"`
}

// Cluster is a set of configurations for a etcd/Kubernetes cluster.
type Cluster struct {
	Name          string  `json:"name"           yaml:"name"`
	Nodes         []*Node `json:"nodes"          yaml:"nodes"`
	SSHKey        string  `json:"ssh_key"        yaml:"ssh_key"`
	ServiceSubnet string  `json:"service_subnet" yaml:"service_subnet"`
	Options       Options `json:"options"        yaml:"options"`
	RBAC          bool    `json:"rbac"           yaml:"rbac"`
}

// Validate validates the cluster definition.
func (c *Cluster) Validate() error {
	return nil
}
