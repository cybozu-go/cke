package cke

import (
	"context"
	"errors"
	"io/ioutil"
	"path/filepath"
)

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
	client := inf.Vault()
	if client == nil {
		return errors.New("can not connect to vault")
	}
	secret, err := client.Logical().Write(ca+"/issue/system", opts)
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
	hostname := node.Hostname
	if len(hostname) == 0 {
		hostname = node.Address
	}
	err := issueCertificate(inf, node, CAServer, "server", map[string]interface{}{
		"common_name": hostname,
		"alt_names":   "localhost",
		"ip_sans":     "127.0.0.1," + node.Address,
	})
	if err != nil {
		return err
	}
	err = issueCertificate(inf, node, CAEtcdPeer, "peer", map[string]interface{}{
		"common_name":          hostname,
		"ip_sans":              "127.0.0.1," + node.Address,
		"exclude_cn_from_sans": "true",
	})
	if err != nil {
		return err
	}
	err = issueCertificate(inf, node, CAEtcdClient, "etcdctl", map[string]interface{}{
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
	err = writeFile(inf, node, "/etc/etcd/pki/ca-peer.crt", ca)
	if err != nil {
		return err
	}
	ca, err = inf.Storage().GetCACertificate(ctx, "etcd-client")
	if err != nil {
		return err
	}
	return writeFile(inf, node, "/etc/etcd/pki/ca-client.crt", ca)
}

func issueEtcdClientCertificates(ctx context.Context, inf Infrastructure, dir string) error {
	client := inf.Vault()
	if client == nil {
		return errors.New("can not connect to vault")
	}

	secret, err := client.Logical().Write(CAEtcdClient+"/issue/system", map[string]interface{}{
		"common_name":          "cke",
		"exclude_cn_from_sans": "true",
	})
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(dir, "cke.crt"), []byte(secret.Data["certificate"].(string)), 0644)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(dir, "cke.key"), []byte(secret.Data["private_key"].(string)), 0644)
	if err != nil {
		return err
	}
	ca, err := inf.Storage().GetCACertificate(ctx, "server")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(dir, "ca-server.crt"), []byte(ca), 0644)
	if err != nil {
		return err
	}
	return nil
}
