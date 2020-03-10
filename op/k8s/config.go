package k8s

import (
	"github.com/cybozu-go/cke"
	apiserver "k8s.io/apiserver/pkg/apis/config"
	"k8s.io/client-go/tools/clientcmd/api"
	kubeletev1beta1 "k8s.io/kubelet/config/v1beta1"
)

func controllerManagerKubeconfig(cluster string, ca, clientCrt, clientKey string) *api.Config {
	return cke.Kubeconfig(cluster, "system:kube-controller-manager", ca, clientCrt, clientKey)
}

func schedulerKubeconfig(cluster string, ca, clientCrt, clientKey string) *api.Config {
	return cke.Kubeconfig(cluster, "system:kube-scheduler", ca, clientCrt, clientKey)
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

func newKubeletConfiguration(cert, key, ca, domain, logSize string, logFiles int32, allowSwap bool) kubeletev1beta1.KubeletConfiguration {
	return kubeletev1beta1.KubeletConfiguration{
		APIVersion:            "kubelet.config.k8s.io/v1beta1",
		Kind:                  "KubeletConfiguration",
		ReadOnlyPort:          0,
		TLSCertFile:           cert,
		TLSPrivateKeyFile:     key,
		Authentication:        KubeletAuthentication{ClientCAFile: ca},
		Authorization:         kubeletAuthorization{Mode: "Webhook"},
		HealthzBindAddress:    "0.0.0.0",
		OOMScoreAdj:           -1000,
		ClusterDomain:         domain,
		RuntimeRequestTimeout: "15m",
		FailSwapOn:            !allowSwap,
		ContainerLogMaxSize:   logSize,
		ContainerLogMaxFiles:  logFiles,
	}
}

func newEncryptionConfiguration() apiserver.EncryptionConfiguration {

	return apiserver.EncryptionConfiguration{
		APIVersion: "apiserver.config.k8s.io/v1",
		Kind:       "EncryptionConfiguration",
	}
}
