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
	apiserverv1 "k8s.io/apiserver/pkg/apis/config/v1"
	"k8s.io/client-go/tools/clientcmd/api"
	componentv1alpha1 "k8s.io/component-base/config/v1alpha1"
	schedulerv1alpha2 "k8s.io/kube-scheduler/config/v1alpha2"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
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
	resourceEncoder = k8sjson.NewSerializerWithOptions(k8sjson.DefaultMetaFactory, scm, scm, k8sjson.SerializerOptions{Yaml: true})
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
func GenerateSchedulerConfiguration(params cke.SchedulerParams) schedulerv1alpha2.KubeSchedulerConfiguration {
	// default values
	base := schedulerv1alpha2.KubeSchedulerConfiguration{}

	c, err := params.GetConfigV1Alpha2(&base)
	if err != nil {
		panic(err)
	}

	// forced values
	c.ClientConnection = componentv1alpha1.ClientConnectionConfiguration{
		Kubeconfig: op.SchedulerKubeConfigPath,
	}
	c.LeaderElection = schedulerv1alpha2.KubeSchedulerLeaderElectionConfiguration{
		LeaderElectionConfiguration: componentv1alpha1.LeaderElectionConfiguration{
			LeaderElect: boolPointer(true),
		},
	}

	return *c
}

func proxyKubeconfig(cluster string, ca, clientCrt, clientKey string) *api.Config {
	return cke.Kubeconfig(cluster, "system:kube-proxy", ca, clientCrt, clientKey)
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

func newKubeletConfiguration(cert, key, ca string, params cke.KubeletParams) kubeletv1beta1.KubeletConfiguration {
	// default values
	base := &kubeletv1beta1.KubeletConfiguration{
		RuntimeRequestTimeout: metav1.Duration{Duration: 15 * time.Minute},
		HealthzBindAddress:    "0.0.0.0",
		OOMScoreAdj:           int32Pointer(-1000),
	}

	// This won't raise an error because of prior validation
	c, err := params.GetConfigV1Beta1(base)
	if err != nil {
		panic(err)
	}

	// forced values
	c.TLSCertFile = cert
	c.TLSPrivateKeyFile = key
	c.Authentication = kubeletv1beta1.KubeletAuthentication{
		X509:    kubeletv1beta1.KubeletX509Authentication{ClientCAFile: ca},
		Webhook: kubeletv1beta1.KubeletWebhookAuthentication{Enabled: boolPointer(true)},
	}
	c.Authorization = kubeletv1beta1.KubeletAuthorization{Mode: kubeletv1beta1.KubeletAuthorizationModeWebhook}

	return *c
}

// GenerateKubeletConfiguration generates kubelet configuration.
func GenerateKubeletConfiguration(params cke.KubeletParams, nodeAddress string) kubeletv1beta1.KubeletConfiguration {
	caPath := op.K8sPKIPath("ca.crt")
	tlsCertPath := op.K8sPKIPath("kubelet.crt")
	tlsKeyPath := op.K8sPKIPath("kubelet.key")
	cfg := newKubeletConfiguration(tlsCertPath, tlsKeyPath, caPath, params)
	cfg.ClusterDNS = []string{nodeAddress}
	return cfg
}

func int32Pointer(input int32) *int32 {
	return &input
}

func boolPointer(input bool) *bool {
	return &input
}
