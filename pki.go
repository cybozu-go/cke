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
type EtcdCA struct{}

func (e EtcdCA) issueServerCert(ctx context.Context, inf Infrastructure, node *Node) (crt, key string, err error) {
	return issueCertificate(inf, CAServer, "system",
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
}

func (e EtcdCA) issuePeerCert(ctx context.Context, inf Infrastructure, node *Node) (crt, key string, err error) {
	return issueCertificate(inf, CAEtcdPeer, "system",
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
}

func (e EtcdCA) issueForAPIServer(ctx context.Context, inf Infrastructure, node *Node) (crt, key string, err error) {
	return issueCertificate(inf, CAEtcdClient, "system",
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
type KubernetesCA struct{}

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

func (k KubernetesCA) issueForAPIServer(ctx context.Context, inf Infrastructure, n *Node) (crt, key string, err error) {
	return issueCertificate(inf, CAKubernetes, "system",
		map[string]interface{}{
			"ttl":               "87600h",
			"max_ttl":           "87600h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
		},
		map[string]interface{}{
			"common_name":          "kubernetes",
			"alt_names":            "localhost,kubernetes.default",
			"ip_sans":              "127.0.0.1," + n.Address,
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
