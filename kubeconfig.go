package cke

import "k8s.io/client-go/tools/clientcmd/api"

func controllerManagerKubeconfig(cluster string, ca, clientCrt, clientKey string) *api.Config {
	return kubeconfig(cluster, "system:kube-controller-manager", ca, clientCrt, clientKey)
}

func schedulerKubeconfig(cluster string, ca, clientCrt, clientKey string) *api.Config {
	return kubeconfig(cluster, "system:kube-scheduler", ca, clientCrt, clientKey)
}

func proxyKubeconfig(cluster string, ca, clientCrt, clientKey string) *api.Config {
	return kubeconfig(cluster, "system:kube-proxy", ca, clientCrt, clientKey)
}

func kubeletKubeconfig(cluster string, n *Node, ca, clientCrt, clientKey string) *api.Config {
	hostname := n.Hostname
	if len(hostname) == 0 {
		hostname = n.Address
	}

	return kubeconfig(cluster, "system:node:"+hostname, ca, clientCrt, clientKey)
}

func kubeconfig(cluster, user, ca, clientCrt, clientKey string) *api.Config {
	cfg := api.NewConfig()
	c := api.NewCluster()
	c.Server = "https://localhost:16443"
	c.CertificateAuthorityData = []byte(ca)
	cfg.Clusters[cluster] = c

	auth := api.NewAuthInfo()
	auth.ClientCertificateData = []byte(clientCrt)
	auth.ClientKeyData = []byte(clientKey)
	cfg.AuthInfos[user] = auth

	ctx := api.NewContext()
	ctx.AuthInfo = user
	ctx.Cluster = cluster
	cfg.Contexts["default"] = ctx
	cfg.CurrentContext = "default"

	return cfg
}
