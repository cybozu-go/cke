package cke

import (
	"context"
	"path/filepath"
)

// CA Keys in Vault
const (
	CAServer     = "cke/ca-server"
	CAEtcdPeer   = "cke/ca-etcd-peer"
	CAEtcdClient = "cke/ca-etcd-client"

	etcdPKIPath = "/etc/etcd/pki"
	k8sPKIPath  = "/etc/kubernetes/pki"
)

// EtcdPKIPath returns a certificate file path for k8s.
func EtcdPKIPath(path string) string {
	return filepath.Join(etcdPKIPath, path)
}

// K8sPKIPath returns a certificate file path for k8s.
func K8sPKIPath(path string) string {
	return filepath.Join(k8sPKIPath, path)
}

// EtcdCA is a certificate authority for etcd cluster.
type EtcdCA struct {
}

func (e EtcdCA) setupNode(ctx context.Context, inf Infrastructure, node *Node) error {
	hostname := node.Hostname
	if len(hostname) == 0 {
		hostname = node.Address
	}
	err := issueCertificate(inf, node, CAServer, EtcdPKIPath("server"), map[string]interface{}{
		"common_name": hostname,
		"alt_names":   "localhost",
		"ip_sans":     "127.0.0.1," + node.Address,
	})
	if err != nil {
		return err
	}
	err = issueCertificate(inf, node, CAEtcdPeer, EtcdPKIPath("peer"), map[string]interface{}{
		"common_name":          hostname,
		"ip_sans":              "127.0.0.1," + node.Address,
		"exclude_cn_from_sans": "true",
	})
	if err != nil {
		return err
	}

	ca, err := inf.Storage().GetCACertificate(ctx, "etcd-peer")
	if err != nil {
		return err
	}
	err = writeFile(inf, node, EtcdPKIPath("ca-peer.crt"), ca)
	if err != nil {
		return err
	}
	ca, err = inf.Storage().GetCACertificate(ctx, "etcd-client")
	if err != nil {
		return err
	}
	return writeFile(inf, node, EtcdPKIPath("ca-client.crt"), ca)
}

func (e EtcdCA) issueForAPIServer(ctx context.Context, inf Infrastructure, node *Node) error {
	hostname := node.Hostname
	if len(hostname) == 0 {
		hostname = node.Address
	}
	err := issueCertificate(inf, node, CAEtcdClient, K8sPKIPath("apiserver-etcd-client"), map[string]interface{}{
		"common_name":          "kube-apiserver",
		"exclude_cn_from_sans": "true",
	})
	if err != nil {
		return err
	}

	ca, err := inf.Storage().GetCACertificate(ctx, "server")
	if err != nil {
		return err
	}
	return writeFile(inf, node, K8sPKIPath("etcd/ca.crt"), ca)
}

func (e EtcdCA) issueRoot(ctx context.Context, inf Infrastructure) (ca, cert, key string, err error) {
	client, err := inf.Vault()
	if err != nil {
		return "", "", "", err
	}

	ca, err = inf.Storage().GetCACertificate(ctx, "server")
	if err != nil {
		return "", "", "", err
	}

	secret, err := client.Logical().Write(CAEtcdClient+"/issue/system", map[string]interface{}{
		"common_name":          "root",
		"exclude_cn_from_sans": "true",
		"ttl":                  "1h",
	})
	if err != nil {
		return "", "", "", err
	}

	cert = secret.Data["certificate"].(string)
	key = secret.Data["private_key"].(string)
	return ca, cert, key, nil
}

// KubernetesCA is a certificate authority for k8s cluster.
type KubernetesCA struct {
}

func (k KubernetesCA) setup() {
}

func (k KubernetesCA) issueForScheduler() {
}

func (k KubernetesCA) issueForControllerManager() {
}

func (k KubernetesCA) issueForKubelet() {
}

func (k KubernetesCA) issueForProxy() {
}

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

func issueCertificate(inf Infrastructure, node *Node, ca, file string, opts map[string]interface{}) error {
	client, err := inf.Vault()
	if err != nil {
		return err
	}
	secret, err := client.Logical().Write(ca+"/issue/system", opts)
	if err != nil {
		return err
	}
	err = writeFile(inf, node, file+".crt", secret.Data["certificate"].(string))
	if err != nil {
		return err
	}
	err = writeFile(inf, node, file+".key", secret.Data["private_key"].(string))
	if err != nil {
		return err
	}
	return nil
}
