package k8s

import (
	"bytes"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	apiserverv1 "k8s.io/apiserver/pkg/apis/apiserver/v1"
	"k8s.io/client-go/tools/clientcmd/api"
	proxyv1alpha1 "k8s.io/kube-proxy/config/v1alpha1"
	schedulerv1 "k8s.io/kube-scheduler/config/v1"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/ptr"
)

var (
	resourceEncoder runtime.Encoder
	scm             = runtime.NewScheme()
)

func init() {
	if err := apiserverv1.AddToScheme(scm); err != nil {
		panic(err)
	}
	if err := kubeletv1beta1.AddToScheme(scm); err != nil {
		panic(err)
	}
	if err := schedulerv1.AddToScheme(scm); err != nil {
		panic(err)
	}
	if err := proxyv1alpha1.AddToScheme(scm); err != nil {
		panic(err)
	}
	resourceEncoder = k8sjson.NewSerializerWithOptions(k8sjson.DefaultMetaFactory, scm, scm,
		k8sjson.SerializerOptions{Yaml: true})
}

func encodeToYAML(obj runtime.Object) ([]byte, error) {
	unst := &unstructured.Unstructured{}
	if err := scm.Convert(obj, unst, nil); err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err := resourceEncoder.Encode(unst, buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func controllerManagerKubeconfig(cluster string, ca, clientCrt, clientKey string) *api.Config {
	return cke.Kubeconfig(cluster, "system:kube-controller-manager", ca, clientCrt, clientKey)
}

func schedulerKubeconfig(cluster string, ca, clientCrt, clientKey string) *api.Config {
	return cke.Kubeconfig(cluster, "system:kube-scheduler", ca, clientCrt, clientKey)
}

// GenerateSchedulerConfiguration generates scheduler configuration.
// `params` must be validated beforehand.
func GenerateSchedulerConfiguration(params cke.SchedulerParams) *schedulerv1.KubeSchedulerConfiguration {
	// default values
	base := schedulerv1.KubeSchedulerConfiguration{}

	c, err := params.MergeConfig(&base)
	if err != nil {
		panic(err)
	}

	// forced values
	c.ClientConnection.Kubeconfig = op.SchedulerKubeConfigPath
	c.LeaderElection.LeaderElect = ptr.To(true)

	return c
}

// GenerateProxyConfiguration generates proxy configuration.
// `params` must be validated beforehand.
func GenerateProxyConfiguration(params cke.ProxyParams, n *cke.Node) *proxyv1alpha1.KubeProxyConfiguration {
	// default values
	base := proxyv1alpha1.KubeProxyConfiguration{
		HostnameOverride:   n.Nodename(),
		MetricsBindAddress: "0.0.0.0",
		Conntrack: proxyv1alpha1.KubeProxyConntrackConfiguration{
			TCPEstablishedTimeout: &metav1.Duration{Duration: 24 * time.Hour},
			TCPCloseWaitTimeout:   &metav1.Duration{Duration: 1 * time.Hour},
		},
	}

	c, err := params.MergeConfig(&base)
	if err != nil {
		panic(err)
	}

	// forced values
	c.ClientConnection.Kubeconfig = proxyKubeconfigPath

	return c
}

func proxyKubeconfig(cluster, ca, clientCrt, clientKey, server string) *api.Config {
	if server == "" {
		return cke.Kubeconfig(cluster, "system:kube-proxy", ca, clientCrt, clientKey)
	}
	return cke.UserKubeconfig(cluster, "system:kube-proxy", ca, clientCrt, clientKey, server)
}

func kubeletKubeconfig(cluster string, n *cke.Node, caPath, certPath, keyPath string) *api.Config {
	cfg := api.NewConfig()
	c := api.NewCluster()
	c.Server = "https://localhost:16443"
	c.CertificateAuthority = caPath
	cfg.Clusters[cluster] = c

	auth := api.NewAuthInfo()
	auth.ClientCertificate = certPath
	auth.ClientKey = keyPath
	user := "system:node:" + n.Nodename()
	cfg.AuthInfos[user] = auth

	ctx := api.NewContext()
	ctx.AuthInfo = user
	ctx.Cluster = cluster
	cfg.Contexts["default"] = ctx
	cfg.CurrentContext = "default"

	return cfg
}

// GenerateKubeletConfiguration generates kubelet configuration.
// `params` must be validated beforehand.
func GenerateKubeletConfiguration(params cke.KubeletParams, nodeAddress string, running *kubeletv1beta1.KubeletConfiguration) *kubeletv1beta1.KubeletConfiguration {
	caPath := op.K8sPKIPath("ca.crt")
	tlsCertPath := op.K8sPKIPath("kubelet.crt")
	tlsKeyPath := op.K8sPKIPath("kubelet.key")

	// default values
	base := &kubeletv1beta1.KubeletConfiguration{
		ClusterDomain:         "cluster.local",
		RuntimeRequestTimeout: metav1.Duration{Duration: 15 * time.Minute},
		HealthzBindAddress:    "0.0.0.0",
		VolumePluginDir:       "/opt/volume/bin",
	}

	// This won't raise an error because of prior validation
	c, err := params.MergeConfig(base)
	if err != nil {
		panic(err)
	}

	// forced values
	c.TLSCertFile = tlsCertPath
	c.TLSPrivateKeyFile = tlsKeyPath
	c.Authentication = kubeletv1beta1.KubeletAuthentication{
		X509:    kubeletv1beta1.KubeletX509Authentication{ClientCAFile: caPath},
		Webhook: kubeletv1beta1.KubeletWebhookAuthentication{Enabled: ptr.To(true)},
	}
	c.Authorization = kubeletv1beta1.KubeletAuthorization{Mode: kubeletv1beta1.KubeletAuthorizationModeWebhook}
	c.ClusterDNS = []string{nodeAddress}

	taintIndex := map[string]int{}
	for i, t := range c.RegisterWithTaints {
		taintIndex[t.Key] = i
	}
	for _, t := range params.BootTaints {
		if index, ok := taintIndex[t.Key]; ok {
			// Overwrite the user-defined RegisterWithTaints when the same taint exists as boot taints.
			c.RegisterWithTaints[index] = t
		} else {
			c.RegisterWithTaints = append(c.RegisterWithTaints, t)
		}
	}

	if running != nil {
		// Keep the running configurations while the node is running.
		// All these fields are described as:
		//     This field should not be updated without a full node
		//     reboot. It is safest to keep this value the same as the local config.
		//
		// ref:  https://pkg.go.dev/k8s.io/kubelet/config/v1beta1#KubeletConfiguration
		c.KubeletCgroups = running.KubeletCgroups
		c.SystemCgroups = running.SystemCgroups
		c.CgroupRoot = running.CgroupRoot
		c.CgroupsPerQOS = running.CgroupsPerQOS
		c.CgroupDriver = running.CgroupDriver
		c.CPUManagerPolicy = running.CPUManagerPolicy
		c.TopologyManagerPolicy = running.TopologyManagerPolicy
		c.QOSReserved = running.QOSReserved
		c.SystemReservedCgroup = running.SystemReservedCgroup
		c.KubeReservedCgroup = running.KubeReservedCgroup
	}
	return c
}
