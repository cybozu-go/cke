package cke

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/cybozu-go/log"
	vault "github.com/hashicorp/vault/api"
)

// CA Keys in Vault
const (
	CAServer     = "cke/ca-server"
	CAEtcdPeer   = "cke/ca-etcd-peer"
	CAEtcdClient = "cke/ca-etcd-client"
	CAKubernetes = "cke/ca-kubernetes"

	etcdPKIPath = "/etc/etcd/pki"
	k8sPKIPath  = "/etc/kubernetes/pki"
)

// addRole adds a role to CA if not exists.
func addRole(client *vault.Client, ca, role string, data map[string]interface{}) error {
	l := client.Logical()
	rpath := path.Join(ca, "roles", role)
	secret, err := l.Read(rpath)
	if err != nil {
		return err
	}
	if secret != nil {
		// already exists
		return nil
	}

	_, err = l.Write(rpath, data)
	if err != nil {
		log.Error("failed to create vault role", map[string]interface{}{
			log.FnError: err,
			"ca":        ca,
			"role":      role,
		})
	}
	return err
}

// EtcdPKIPath returns a certificate file path for k8s.
func EtcdPKIPath(p string) string {
	return filepath.Join(etcdPKIPath, p)
}

// K8sPKIPath returns a certificate file path for k8s.
func K8sPKIPath(p string) string {
	return filepath.Join(k8sPKIPath, p)
}

// EtcdCA is a certificate authority for etcd cluster.
type EtcdCA struct {
}

func (e EtcdCA) setupNode(ctx context.Context, inf Infrastructure, node *Node) error {
	err := writeCertificate(inf, node, CAServer, EtcdPKIPath("server"),
		map[string]interface{}{
			"ttl":            "87600h",
			"max_ttl":        "87600h",
			"client_flag":    "false",
			"allow_any_name": "true",
		},
		map[string]interface{}{
			"common_name": node.Nodename(),
			"alt_names":   "localhost",
			"ip_sans":     "127.0.0.1," + node.Address,
		})
	if err != nil {
		return err
	}
	err = writeCertificate(inf, node, CAEtcdPeer, EtcdPKIPath("peer"),
		map[string]interface{}{
			"ttl":            "87600h",
			"max_ttl":        "87600h",
			"allow_any_name": "true",
		},
		map[string]interface{}{
			"common_name":          node.Nodename(),
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
	err := writeCertificate(inf, node, CAEtcdClient, K8sPKIPath("apiserver-etcd-client"),
		map[string]interface{}{
			"ttl":            "87600h",
			"max_ttl":        "87600h",
			"server_flag":    "false",
			"allow_any_name": "true",
		},
		map[string]interface{}{
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

func (e EtcdCA) issueRoot(ctx context.Context, inf Infrastructure) (cert, key string, err error) {
	return issueCertificate(inf, CAEtcdClient, "admin",
		map[string]interface{}{
			"ttl":            "2h",
			"max_ttl":        "24h",
			"server_flag":    "false",
			"allow_any_name": "true",
		},
		map[string]interface{}{
			"common_name":          "root",
			"exclude_cn_from_sans": "true",
			"ttl":                  "1h",
		})
}

// KubernetesCA is a certificate authority for k8s cluster.
type KubernetesCA struct {
}

// setup generates and installs certificates for API server.
func (k KubernetesCA) setup(ctx context.Context, inf Infrastructure, node *Node) error {
	sakey, err := inf.Storage().GetServiceAccountKey(ctx)
	if err != nil {
		return err
	}
	err = writeFile(inf, node, K8sPKIPath("service-account.key"), sakey)
	if err != nil {
		return err
	}

	sacert, err := inf.Storage().GetServiceAccountCert(ctx)
	if err != nil {
		return err
	}
	err = writeFile(inf, node, K8sPKIPath("service-account.crt"), sacert)
	if err != nil {
		return err
	}

	err = writeCertificate(inf, node, CAKubernetes, K8sPKIPath("apiserver"),
		map[string]interface{}{
			"ttl":               "87600h",
			"max_ttl":           "87600h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
		},
		map[string]interface{}{
			"common_name":          node.Nodename(),
			"alt_names":            "localhost,kubernetes.default",
			"ip_sans":              "127.0.0.1," + node.Address,
			"exclude_cn_from_sans": "true",
		})
	if err != nil {
		return err
	}

	ca, err := inf.Storage().GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	return writeFile(inf, node, K8sPKIPath("ca.crt"), ca)
}

// issueAdminCert issues client certificates for cluster admin.
func (k KubernetesCA) issueAdminCert(ctx context.Context, inf Infrastructure, hours uint) (crt, key string, err error) {
	return issueCertificate(inf, CAKubernetes, "admin",
		map[string]interface{}{
			"ttl":               "2h",
			"max_ttl":           "48h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
			"organization":      "system:masters",
		},
		map[string]interface{}{
			"ttl":                  fmt.Sprintf("%dh", hours),
			"common_name":          "admin",
			"exclude_cn_from_sans": "true",
		})
}

func (k KubernetesCA) issueForScheduler(ctx context.Context, inf Infrastructure) (crt, key string, err error) {
	return issueCertificate(inf, CAKubernetes, "kube-scheduler",
		map[string]interface{}{
			"ttl":               "87600h",
			"max_ttl":           "87600h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
			"organization":      "system:kube-scheduler",
		},
		map[string]interface{}{
			"common_name":          "system:kube-scheduler",
			"exclude_cn_from_sans": "true",
		})
}

func (k KubernetesCA) issueForControllerManager(ctx context.Context, inf Infrastructure) (crt, key string, err error) {
	return issueCertificate(inf, CAKubernetes, "kube-controller-manager",
		map[string]interface{}{
			"ttl":               "87600h",
			"max_ttl":           "87600h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
			"organization":      "system:kube-controller-manager",
		},
		map[string]interface{}{
			"common_name":          "system:kube-controller-manager",
			"exclude_cn_from_sans": "true",
		})
}

func (k KubernetesCA) issueForKubelet(ctx context.Context, inf Infrastructure, node *Node) (crt, key string, err error) {
	nodename := node.Nodename()
	altNames := "localhost"
	if nodename != node.Address {
		altNames = "localhost," + nodename
	}

	return issueCertificate(inf, CAKubernetes, "kubelet",
		map[string]interface{}{
			"ttl":               "87600h",
			"max_ttl":           "87600h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
			"organization":      "system:nodes",
		},
		map[string]interface{}{
			"common_name":          "system:node:" + nodename,
			"alt_names":            altNames,
			"ip_sans":              "127.0.0.1," + node.Address,
			"exclude_cn_from_sans": "true",
		})
}

func (k KubernetesCA) issueForProxy(ctx context.Context, inf Infrastructure) (crt, key string, err error) {
	return issueCertificate(inf, CAKubernetes, "kube-proxy",
		map[string]interface{}{
			"ttl":               "87600h",
			"max_ttl":           "87600h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
			"organization":      "system:node-proxier",
		},
		map[string]interface{}{
			"common_name":          "system:kube-proxy",
			"exclude_cn_from_sans": "true",
		})
}

func (k KubernetesCA) issueForServiceAccount(ctx context.Context, inf Infrastructure) (crt, key string, err error) {
	return issueCertificate(inf, CAKubernetes, "service-account",
		map[string]interface{}{
			"ttl":            "87600h",
			"max_ttl":        "87600h",
			"allow_any_name": "true",
			"client_flag":    "false",
			"server_flag":    "false",
			"key_usage":      "DigitalSignature,CertSign",
			"no_store":       "true",
		},
		map[string]interface{}{
			"common_name":          "service-account",
			"exclude_cn_from_sans": "true",
		})
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

func writeCertificate(inf Infrastructure, node *Node, ca, file string, roleOpts, certOpts map[string]interface{}) error {
	crt, key, err := issueCertificate(inf, ca, "system", roleOpts, certOpts)
	if err != nil {
		return err
	}
	err = writeFile(inf, node, file+".crt", crt)
	if err != nil {
		return err
	}
	err = writeFile(inf, node, file+".key", key)
	if err != nil {
		return err
	}
	return nil
}

func issueCertificate(inf Infrastructure, ca, role string, roleOpts, certOpts map[string]interface{}) (crt, key string, err error) {
	client, err := inf.Vault()
	if err != nil {
		return "", "", err
	}

	err = addRole(client, ca, role, roleOpts)
	if err != nil {
		return "", "", err
	}

	secret, err := client.Logical().Write(path.Join(ca, "issue", role), certOpts)
	if err != nil {
		return "", "", err
	}
	crt = secret.Data["certificate"].(string)
	key = secret.Data["private_key"].(string)
	return crt, key, err
}
