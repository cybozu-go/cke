package cke

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/containernetworking/cni/libcni"
	corev1 "k8s.io/api/core/v1"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	v1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	proxyv1alpha1 "k8s.io/kube-proxy/config/v1alpha1"
	schedulerv1 "k8s.io/kube-scheduler/config/v1"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
	"sigs.k8s.io/yaml"
)

// Node represents a node in Kubernetes.
type Node struct {
	Address      string            `json:"address"`
	Hostname     string            `json:"hostname"`
	User         string            `json:"user"`
	ControlPlane bool              `json:"control_plane"`
	Annotations  map[string]string `json:"annotations"`
	Labels       map[string]string `json:"labels"`
	Taints       []corev1.Taint    `json:"taints"`
}

// Nodename returns a hostname or address if hostname is empty
func (n *Node) Nodename() string {
	if len(n.Hostname) == 0 {
		return n.Address
	}
	return n.Hostname
}

// BindPropagation is bind propagation option for Docker
// https://docs.docker.com/storage/bind-mounts/#configure-bind-propagation
type BindPropagation string

// Bind propagation definitions
const (
	PropagationShared   = BindPropagation("shared")
	PropagationSlave    = BindPropagation("slave")
	PropagationPrivate  = BindPropagation("private")
	PropagationRShared  = BindPropagation("rshared")
	PropagationRSlave   = BindPropagation("rslave")
	PropagationRPrivate = BindPropagation("rprivate")
)

func (p BindPropagation) String() string {
	return string(p)
}

// SELinuxLabel is selinux label of the host file or directory
// https://docs.docker.com/storage/bind-mounts/#configure-the-selinux-label
type SELinuxLabel string

// SELinux Label definitions
const (
	LabelShared  = SELinuxLabel("z")
	LabelPrivate = SELinuxLabel("Z")
)

func (l SELinuxLabel) String() string {
	return string(l)
}

// Mount is volume mount information
type Mount struct {
	Source      string          `json:"source"`
	Destination string          `json:"destination"`
	ReadOnly    bool            `json:"read_only"`
	Propagation BindPropagation `json:"propagation"`
	Label       SELinuxLabel    `json:"selinux_label"`
}

// Equal returns true if the mount is equals to other one, otherwise return false
func (m Mount) Equal(o Mount) bool {
	return m.Source == o.Source && m.Destination == o.Destination && m.ReadOnly == o.ReadOnly
}

// ServiceParams is a common set of extra parameters for k8s components.
type ServiceParams struct {
	ExtraArguments []string          `json:"extra_args"`
	ExtraBinds     []Mount           `json:"extra_binds"`
	ExtraEnvvar    map[string]string `json:"extra_env"`
}

// Equal returns true if the services params is equals to other one, otherwise return false
func (s ServiceParams) Equal(o ServiceParams) bool {
	return compareStrings(s.ExtraArguments, o.ExtraArguments) &&
		compareMounts(s.ExtraBinds, o.ExtraBinds) &&
		compareStringMap(s.ExtraEnvvar, o.ExtraEnvvar)
}

// EtcdParams is a set of extra parameters for etcd.
type EtcdParams struct {
	ServiceParams `json:",inline"`
	VolumeName    string `json:"volume_name"`
}

// APIServerParams is a set of extra parameters for kube-apiserver.
type APIServerParams struct {
	ServiceParams   `json:",inline"`
	AuditLogEnabled bool   `json:"audit_log_enabled"`
	AuditLogPolicy  string `json:"audit_log_policy"`
	AuditLogPath    string `json:"audit_log_path"`
}

// CNIConfFile is a config file for CNI plugin deployed on worker nodes by CKE.
type CNIConfFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// SchedulerParams is a set of extra parameters for kube-scheduler.
type SchedulerParams struct {
	ServiceParams `json:",inline"`
	Config        *unstructured.Unstructured `json:"config,omitempty"`
}

// MergeConfig merges the input struct `base`.
func (p SchedulerParams) MergeConfig(base *schedulerv1.KubeSchedulerConfiguration) (*schedulerv1.KubeSchedulerConfiguration, error) {
	// FOR IMPLEMENTORS.
	// DO NOT SUPPORT MORE THAN ONE ComponentConfig VERSIONS.
	// When we need to upgrade the component config version, users will
	// stop CKE, update cluster.yml in etcd, then start the new CKE.
	// So, CKE should only support the latest config version.
	cfg := *base.DeepCopy()
	if p.Config == nil {
		return &cfg, nil
	}

	if p.Config.GetAPIVersion() != schedulerv1.SchemeGroupVersion.String() {
		return nil, fmt.Errorf("unexpected kube-scheduler API version: %s", p.Config.GetAPIVersion())
	}
	if p.Config.GetKind() != "KubeSchedulerConfiguration" {
		return nil, fmt.Errorf("wrong kind for kube-scheduler config: %s", p.Config.GetKind())
	}

	data, err := json.Marshal(p.Config)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	cfg.TypeMeta = metav1.TypeMeta{}
	return &cfg, nil
}

// ProxyParams is a set of extra parameters for kube-proxy.
type ProxyParams struct {
	ServiceParams `json:",inline"`
	Disable       bool                       `json:"disable,omitempty"`
	Config        *unstructured.Unstructured `json:"config,omitempty"`
}

// GetMode returns the proxy mode.
func (p ProxyParams) GetMode() string {
	mode := p.Config.UnstructuredContent()["Mode"].(string)
	if len(mode) == 0 {
		return string(ProxyModeIPVS)
	}
	return mode
}

// MergeConfig merges the input struct with `base`.
func (p ProxyParams) MergeConfig(base *proxyv1alpha1.KubeProxyConfiguration) (*proxyv1alpha1.KubeProxyConfiguration, error) {
	// FOR IMPLEMENTORS.
	// DO NOT SUPPORT MORE THAN ONE ComponentConfig VERSIONS.
	// When we need to upgrade the component config version, users will
	// stop CKE, update cluster.yml in etcd, then start the new CKE.
	// So, CKE should only support the latest config version.
	cfg := *base.DeepCopy()
	if p.Config == nil {
		return &cfg, nil
	}

	if p.Config.GetAPIVersion() != proxyv1alpha1.SchemeGroupVersion.String() {
		return nil, fmt.Errorf("unexpected kube-proxy API version: %s", p.Config.GetAPIVersion())
	}
	if p.Config.GetKind() != "KubeProxyConfiguration" {
		return nil, fmt.Errorf("wrong kind for kube-proxy config: %s", p.Config.GetKind())
	}

	data, err := json.Marshal(p.Config)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	cfg.TypeMeta = metav1.TypeMeta{}
	return &cfg, nil
}

// ProxyMode is a type for kube-proxy's --proxy-mode argument.
type ProxyMode string

const (
	ProxyModeUserspace proxyv1alpha1.ProxyMode = "userspace"
	ProxyModeIptables  proxyv1alpha1.ProxyMode = "iptables"
	ProxyModeIPVS      proxyv1alpha1.ProxyMode = "ipvs"
)

// ValidateProxyMode validates ProxyMode
func ValidateProxyMode(mode proxyv1alpha1.ProxyMode) error {
	switch mode {
	case ProxyModeUserspace, ProxyModeIptables, ProxyModeIPVS:
		return nil
	}

	return errors.New("invalid proxy mode " + string(mode))
}

// KubeletParams is a set of extra parameters for kubelet.
type KubeletParams struct {
	ServiceParams `json:",inline"`
	BootTaints    []corev1.Taint             `json:"boot_taints"`
	CNIConfFile   CNIConfFile                `json:"cni_conf_file"`
	Config        *unstructured.Unstructured `json:"config,omitempty"`
	CRIEndpoint   string                     `json:"cri_endpoint"`
}

// MergeConfig merges the input struct with `base`.
func (p KubeletParams) MergeConfig(base *kubeletv1beta1.KubeletConfiguration) (*kubeletv1beta1.KubeletConfiguration, error) {
	// FOR IMPLEMENTORS.
	// DO NOT SUPPORT MORE THAN ONE ComponentConfig VERSIONS.
	// When we need to upgrade the component config version, users will
	// stop CKE, update cluster.yml in etcd, then start the new CKE.
	// So, CKE should only support the latest config version.
	cfg := *base.DeepCopy()
	if p.Config == nil {
		return &cfg, nil
	}

	if p.Config.GetAPIVersion() != kubeletv1beta1.SchemeGroupVersion.String() {
		return nil, fmt.Errorf("unexpected kubelet API version: %s", p.Config.GetAPIVersion())
	}
	if p.Config.GetKind() != "KubeletConfiguration" {
		return nil, fmt.Errorf("wrong kind for kubelet config: %s", p.Config.GetKind())
	}

	data, err := json.Marshal(p.Config)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	cfg.TypeMeta = metav1.TypeMeta{}
	return &cfg, nil
}

// Reboot is a set of configurations for reboot.
type Reboot struct {
	RebootCommand          []string              `json:"reboot_command"`
	BootCheckCommand       []string              `json:"boot_check_command"`
	MaxConcurrentReboots   *int                  `json:"max_concurrent_reboots,omitempty"`
	EvictionTimeoutSeconds *int                  `json:"eviction_timeout_seconds,omitempty"`
	CommandTimeoutSeconds  *int                  `json:"command_timeout_seconds,omitempty"`
	CommandRetries         *int                  `json:"command_retries"`
	CommandInterval        *int                  `json:"command_interval"`
	EvictRetries           *int                  `json:"evict_retries"`
	EvictInterval          *int                  `json:"evict_interval"`
	ProtectedNamespaces    *metav1.LabelSelector `json:"protected_namespaces,omitempty"`
}

const DefaultRebootEvictionTimeoutSeconds = 600
const DefaultMaxConcurrentReboots = 1

type Repair struct {
	RepairProcedures       []RepairProcedure     `json:"repair_procedures"`
	MaxConcurrentRepairs   *int                  `json:"max_concurrent_repairs,omitempty"`
	ProtectedNamespaces    *metav1.LabelSelector `json:"protected_namespaces,omitempty"`
	EvictRetries           *int                  `json:"evict_retries,omitempty"`
	EvictInterval          *int                  `json:"evict_interval,omitempty"`
	EvictionTimeoutSeconds *int                  `json:"eviction_timeout_seconds,omitempty"`
}

type RepairProcedure struct {
	MachineTypes     []string          `json:"machine_types"`
	RepairOperations []RepairOperation `json:"repair_operations"`
}

type RepairOperation struct {
	Operation             string       `json:"operation"`
	RepairSteps           []RepairStep `json:"repair_steps"`
	HealthCheckCommand    []string     `json:"health_check_command"`
	CommandTimeoutSeconds *int         `json:"command_timeout_seconds,omitempty"`
}

type RepairStep struct {
	RepairCommand         []string `json:"repair_command"`
	CommandTimeoutSeconds *int     `json:"command_timeout_seconds,omitempty"`
	CommandRetries        *int     `json:"command_retries,omitempty"`
	CommandInterval       *int     `json:"command_interval,omitempty"`
	NeedDrain             bool     `json:"need_drain,omitempty"`
	WatchSeconds          *int     `json:"watch_seconds,omitempty"`
}

const DefaultMaxConcurrentRepairs = 1
const DefaultRepairEvictionTimeoutSeconds = 600
const DefaultRepairHealthCheckCommandTimeoutSeconds = 30
const DefaultRepairCommandTimeoutSeconds = 30

type Retire struct {
	ShutdownCommand       []string `json:"shutdown_command"`
	CheckCommand          []string `json:"check_command"`
	CommandTimeoutSeconds *int     `json:"command_timeout_seconds,omitempty"`
	CheckTimeoutSeconds   *int     `json:"check_timeout_seconds,omitempty"`
}

const DefaultRetireCommandTimeoutSeconds = 30
const DefaultRetireCheckTimeoutSeconds = 300

// Options is a set of optional parameters for k8s components.
type Options struct {
	Etcd              EtcdParams      `json:"etcd"`
	Rivers            ServiceParams   `json:"rivers"`
	EtcdRivers        ServiceParams   `json:"etcd-rivers"`
	APIServer         APIServerParams `json:"kube-api"`
	ControllerManager ServiceParams   `json:"kube-controller-manager"`
	Scheduler         SchedulerParams `json:"kube-scheduler"`
	Proxy             ProxyParams     `json:"kube-proxy"`
	Kubelet           KubeletParams   `json:"kubelet"`
}

// Cluster is a set of configurations for a etcd/Kubernetes cluster.
type Cluster struct {
	Name          string   `json:"name"`
	Nodes         []*Node  `json:"nodes"`
	TaintCP       bool     `json:"taint_control_plane"`
	CPTolerations []string `json:"control_plane_tolerations"`
	ServiceSubnet string   `json:"service_subnet"`
	DNSServers    []string `json:"dns_servers"`
	DNSService    string   `json:"dns_service"`
	Reboot        Reboot   `json:"reboot"`
	Repair        Repair   `json:"repair"`
	Retire        Retire   `json:"retire"`
	Options       Options  `json:"options"`
}

// Validate validates the cluster definition.
func (c *Cluster) Validate(isTmpl bool) error {
	if len(c.Name) == 0 {
		return errors.New("cluster name is empty")
	}

	_, _, err := net.ParseCIDR(c.ServiceSubnet)
	if err != nil {
		return err
	}

	fldPath := field.NewPath("nodes")
	nodeAddressSet := make(map[string]struct{})
	for i, n := range c.Nodes {
		err := validateNode(n, isTmpl, fldPath.Index(i))
		if err != nil {
			return err
		}
		if _, ok := nodeAddressSet[n.Address]; ok {
			return errors.New("duplicate node address: " + n.Address)
		}
		if !isTmpl {
			nodeAddressSet[n.Address] = struct{}{}
		}
	}

	fldPath = field.NewPath("control_plane_tolerations")
	err = validateTolerationKeys(c.CPTolerations, fldPath)
	if err != nil {
		return err
	}

	for _, a := range c.DNSServers {
		if net.ParseIP(a) == nil {
			return errors.New("invalid IP address: " + a)
		}
	}

	if len(c.DNSService) > 0 {
		fields := strings.Split(c.DNSService, "/")
		if len(fields) != 2 {
			return errors.New("invalid DNS service (no namespace?): " + c.DNSService)
		}
	}

	err = validateReboot(c.Reboot)
	if err != nil {
		return err
	}

	err = validateOptions(c.Options)
	if err != nil {
		return err
	}

	return nil
}

func validateNode(n *Node, isTmpl bool, fldPath *field.Path) error {
	if isTmpl {
		if len(n.Address) != 0 {
			return errors.New("address is not empty: " + n.Address)
		}
	} else {
		if net.ParseIP(n.Address) == nil {
			return errors.New("invalid IP address: " + n.Address)
		}
	}

	if len(n.User) == 0 {
		return errors.New("user name is empty")
	}

	if err := validateNodeLabels(n, fldPath.Child("labels")); err != nil {
		return err
	}
	if err := validateNodeAnnotations(n, fldPath.Child("annotations")); err != nil {
		return err
	}
	if err := validateNodeTaints(n, fldPath.Child("taints")); err != nil {
		return err
	}
	return nil
}

// validateNodeLabels validates label names and values with
// rules described in:
// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
func validateNodeLabels(n *Node, fldPath *field.Path) error {
	el := v1validation.ValidateLabels(n.Labels, fldPath)
	if len(el) == 0 {
		return nil
	}
	return el.ToAggregate()
}

// validateNodeAnnotations validates annotation names.
// The validation logic references:
// https://github.com/kubernetes/apimachinery/blob/v0.21.7/pkg/api/validation/objectmeta.go#L186
func validateNodeAnnotations(n *Node, fldPath *field.Path) error {
	el := apivalidation.ValidateAnnotations(n.Annotations, fldPath)
	if len(el) == 0 {
		return nil
	}
	return el.ToAggregate()
}

// validateNodeTaints validates taint names, values, and effects.
func validateNodeTaints(n *Node, fldPath *field.Path) error {
	for i, taint := range n.Taints {
		err := validateTaint(taint, fldPath.Index(i))
		if err != nil {
			return err
		}
	}
	return nil
}

// validateTaint validates a taint name, value, and effect.
// The validation logic references:
// https://github.com/kubernetes/kubernetes/blob/7cbb9995189c5ecc8182da29cd0e30188c911401/pkg/apis/core/validation/validation.go#L4105
func validateTaint(taint corev1.Taint, fldPath *field.Path) error {
	el := v1validation.ValidateLabelName(taint.Key, fldPath.Child("key"))
	if msgs := validation.IsValidLabelValue(taint.Value); len(msgs) > 0 {
		el = append(el, field.Invalid(fldPath.Child("value"), taint.Value, strings.Join(msgs, ";")))
	}
	switch taint.Effect {
	case corev1.TaintEffectNoSchedule:
	case corev1.TaintEffectPreferNoSchedule:
	case corev1.TaintEffectNoExecute:
	default:
		el = append(el, field.Invalid(fldPath.Child("effect"), string(taint.Effect), "invalid effect"))
	}
	if len(el) > 0 {
		return el.ToAggregate()
	}
	return nil
}

func validateTolerationKeys(keys []string, fldPath *field.Path) error {
	var el field.ErrorList
	for i, key := range keys {
		el = append(el, v1validation.ValidateLabelName(key, fldPath.Index(i))...)
	}
	if len(el) > 0 {
		return el.ToAggregate()
	}
	return nil
}

// ControlPlanes returns control planes []*Node
func ControlPlanes(nodes []*Node) []*Node {
	return filterNodes(nodes, func(n *Node) bool {
		return n.ControlPlane
	})
}

// Workers returns workers []*Node
func Workers(nodes []*Node) []*Node {
	return filterNodes(nodes, func(n *Node) bool {
		return !n.ControlPlane
	})
}

func filterNodes(nodes []*Node, f func(n *Node) bool) []*Node {
	var filtered []*Node
	for _, n := range nodes {
		if f(n) {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

func validateReboot(reboot Reboot) error {
	if reboot.EvictionTimeoutSeconds != nil && *reboot.EvictionTimeoutSeconds <= 0 {
		return errors.New("eviction_timeout_seconds must be positive")
	}
	if reboot.CommandTimeoutSeconds != nil && *reboot.CommandTimeoutSeconds < 0 {
		return errors.New("command_timeout_seconds must not be negative")
	}
	if reboot.CommandRetries != nil && *reboot.CommandRetries < 0 {
		return errors.New("command_retries must not be negative")
	}
	if reboot.CommandInterval != nil && *reboot.CommandInterval < 0 {
		return errors.New("command_interval must not be negative")
	}
	if reboot.EvictRetries != nil && *reboot.EvictRetries < 0 {
		return errors.New("evict_retries must not be negative")
	}
	if reboot.EvictInterval != nil && *reboot.EvictInterval < 0 {
		return errors.New("evict_interval must not be negative")
	}
	if reboot.MaxConcurrentReboots != nil && *reboot.MaxConcurrentReboots <= 0 {
		return errors.New("max_concurrent_reboots must be positive")
	}
	// nil is safe for LabelSelectorAsSelector
	_, err := metav1.LabelSelectorAsSelector(reboot.ProtectedNamespaces)
	if err != nil {
		return fmt.Errorf("invalid label selector: %w", err)
	}
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
	err = v(opts.ControllerManager.ExtraBinds)
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

	base := &kubeletv1beta1.KubeletConfiguration{}
	kubeletConfig, err := opts.Kubelet.MergeConfig(base)
	if err != nil {
		return err
	}

	fldPath := field.NewPath("options", "kubelet")
	if len(kubeletConfig.ClusterDomain) > 0 {
		msgs := validation.IsDNS1123Subdomain(kubeletConfig.ClusterDomain)
		if len(msgs) > 0 {
			return field.Invalid(fldPath.Child("domain"),
				kubeletConfig.ClusterDomain, strings.Join(msgs, ";"))
		}
	}
	if len(opts.Kubelet.CRIEndpoint) == 0 {
		return errors.New("kubelet.cri_endpoint should not be empty")
	}
	if len(opts.Kubelet.CNIConfFile.Content) != 0 && len(opts.Kubelet.CNIConfFile.Name) == 0 {
		return fmt.Errorf("kubelet.cni_conf_file.name should not be empty when kubelet.cni_conf_file.content is not empty")
	}
	if filename := opts.Kubelet.CNIConfFile.Name; len(filename) != 0 {
		matched, err := regexp.Match(`^[0-9A-Za-z_.-]+$`, []byte(filename))
		if err != nil {
			return err
		}
		if !matched {
			return errors.New(filename + " is invalid as file name")
		}

		if filepath.Ext(opts.Kubelet.CNIConfFile.Name) == ".conflist" {
			_, err = libcni.ConfListFromBytes([]byte(opts.Kubelet.CNIConfFile.Content))
			if err != nil {
				return err
			}
		} else {
			_, err = libcni.ConfFromBytes([]byte(opts.Kubelet.CNIConfFile.Content))
			if err != nil {
				return err
			}
		}
	}

	fldPath = fldPath.Child("boot_taints")
	for i, taint := range opts.Kubelet.BootTaints {
		err := validateTaint(taint, fldPath.Index(i))
		if err != nil {
			return err
		}
	}

	if opts.APIServer.AuditLogEnabled && len(opts.APIServer.AuditLogPolicy) == 0 {
		return errors.New("audit_log_policy should not be empty")
	}

	if len(opts.APIServer.AuditLogPolicy) != 0 {
		policy := make(map[string]interface{})
		err = yaml.Unmarshal([]byte(opts.APIServer.AuditLogPolicy), &policy)
		if err != nil {
			return err
		}
	}

	if _, err := opts.Scheduler.MergeConfig(&schedulerv1.KubeSchedulerConfiguration{}); err != nil {
		return err
	}

	p, err := opts.Proxy.MergeConfig(&proxyv1alpha1.KubeProxyConfiguration{})
	if err != nil {
		return err
	}
	if len(p.Mode) != 0 {
		if err := ValidateProxyMode(p.Mode); err != nil {
			return err
		}
	}

	return nil
}
