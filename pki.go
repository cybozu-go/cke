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

func issueEtcdCertificates(ctx context.Context, inf Infrastructure, node *Node) error {
	hostname := node.Hostname
	if len(hostname) == 0 {
		hostname = node.Address
	}
	err := issueCertificate(inf, node, CAServer, filepath.Join(etcdPKIPath, "server"), map[string]interface{}{
		"common_name": hostname,
		"alt_names":   "localhost",
		"ip_sans":     "127.0.0.1," + node.Address,
	})
	if err != nil {
		return err
	}
	err = issueCertificate(inf, node, CAEtcdPeer, filepath.Join(etcdPKIPath, "peer"), map[string]interface{}{
		"common_name":          hostname,
		"ip_sans":              "127.0.0.1," + node.Address,
		"exclude_cn_from_sans": "true",
	})
	if err != nil {
		return err
	}
	err = issueCertificate(inf, node, CAEtcdClient, filepath.Join(etcdPKIPath, "etcdctl"), map[string]interface{}{
		"common_name":          "etcdctl",
		"exclude_cn_from_sans": "true",
	})
	if err != nil {
		return err
	}

	ca, err := inf.Storage().GetCACertificate(ctx, "etcd-peer")
	if err != nil {
		return err
	}
	err = writeFile(inf, node, filepath.Join(etcdPKIPath, "ca-peer.crt"), ca)
	if err != nil {
		return err
	}
	ca, err = inf.Storage().GetCACertificate(ctx, "etcd-client")
	if err != nil {
		return err
	}
	return writeFile(inf, node, filepath.Join(etcdPKIPath, "ca-client.crt"), ca)
}

func issueAPIServerCertificates(ctx context.Context, inf Infrastructure, node *Node) error {
	hostname := node.Hostname
	if len(hostname) == 0 {
		hostname = node.Address
	}
	err := issueCertificate(inf, node, CAEtcdClient, "/etc/kubernetes/apiserver/apiserver", map[string]interface{}{
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
	return writeFile(inf, node, "/etc/kubernetes/apiserver/ca-server.crt", ca)
}

func issueEtcdClientCertificates(ctx context.Context, inf Infrastructure) (ca, cert, key string, err error) {
	client, e := inf.Vault()
	if e != nil {
		err = e
		return
	}

	ca, err = inf.Storage().GetCACertificate(ctx, "server")
	if err != nil {
		return
	}
	secret, err := client.Logical().Write(CAEtcdClient+"/issue/system", map[string]interface{}{
		"common_name":          "root",
		"exclude_cn_from_sans": "true",
	})
	if err != nil {
		return
	}
	cert = secret.Data["certificate"].(string)
	key = secret.Data["private_key"].(string)
	return
}
