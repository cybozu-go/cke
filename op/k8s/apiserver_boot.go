package k8s

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

var (
	// admissionPlugins is the recommended list of admission plugins.
	// https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use
	admissionPlugins = []string{
		"NamespaceLifecycle",
		"LimitRanger",
		"ServiceAccount",
		"Priority",
		"DefaultTolerationSeconds",
		"DefaultStorageClass",
		"PersistentVolumeClaimResize",
		"MutatingAdmissionWebhook",
		"ValidatingAdmissionWebhook",
		"ResourceQuota",
	}
)

const auditPolicyBasePath = "/etc/kubernetes/apiserver/audit-policy-%x.yaml"

type apiServerBootOp struct {
	nodes []*cke.Node
	cps   []*cke.Node

	serviceSubnet string
	domain        string
	params        cke.APIServerParams

	step  int
	files *common.FilesBuilder
}

// APIServerBootOp returns an Operator to bootstrap kube-apiserver
func APIServerBootOp(nodes, cps []*cke.Node, serviceSubnet, domain string, params cke.APIServerParams) cke.Operator {
	return &apiServerBootOp{
		nodes:         nodes,
		cps:           cps,
		serviceSubnet: serviceSubnet,
		domain:        domain,
		params:        params,
		files:         common.NewFilesBuilder(nodes),
	}
}

func (o *apiServerBootOp) Name() string {
	return "kube-apiserver-bootstrap"
}

func (o *apiServerBootOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.HyperkubeImage)
	case 1:
		o.step++
		return prepareAPIServerFilesCommand{o.files, o.serviceSubnet, o.domain, o.params}
	case 2:
		o.step++
		return o.files
	case 3:
		o.step++
		opts := []string{
			"--mount", "type=tmpfs,dst=/run/kubernetes",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			paramsMap[n.Address] = APIServerParams(o.cps, n.Address, o.serviceSubnet, o.params.AuditLogEnabled, o.params.AuditLogPolicy)
		}
		return common.RunContainerCommand(o.nodes, op.KubeAPIServerContainerName, cke.HyperkubeImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params.ServiceParams))
	default:
		return nil
	}
}

type prepareAPIServerFilesCommand struct {
	files         *common.FilesBuilder
	serviceSubnet string
	domain        string
	params        cke.APIServerParams
}

func (c prepareAPIServerFilesCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	storage := inf.Storage()

	// server (and client) certs of API server.
	f := func(ctx context.Context, n *cke.Node) (cert, key []byte, err error) {
		c, k, e := cke.KubernetesCA{}.IssueForAPIServer(ctx, inf, n, c.serviceSubnet, c.domain)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err := c.files.AddKeyPair(ctx, op.K8sPKIPath("apiserver"), f)
	if err != nil {
		return err
	}

	// client certs for etcd auth.
	f = func(ctx context.Context, n *cke.Node) (cert, key []byte, err error) {
		c, k, e := cke.EtcdCA{}.IssueForAPIServer(ctx, inf, n)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err = c.files.AddKeyPair(ctx, op.K8sPKIPath("apiserver-etcd-client"), f)
	if err != nil {
		return err
	}

	// CA of k8s cluster.
	ca, err := storage.GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	caData := []byte(ca)
	g := func(ctx context.Context, n *cke.Node) ([]byte, error) {
		return caData, nil
	}
	err = c.files.AddFile(ctx, op.K8sPKIPath("ca.crt"), g)
	if err != nil {
		return err
	}

	// CA of etcd server.
	etcdCA, err := storage.GetCACertificate(ctx, "server")
	if err != nil {
		return err
	}
	etcdCAData := []byte(etcdCA)
	g = func(ctx context.Context, n *cke.Node) ([]byte, error) {
		return etcdCAData, nil
	}
	err = c.files.AddFile(ctx, op.K8sPKIPath("etcd-ca.crt"), g)
	if err != nil {
		return err
	}

	// ServiceAccount cert.
	saCert, err := storage.GetServiceAccountCert(ctx)
	if err != nil {
		return err
	}
	saCertData := []byte(saCert)
	g = func(ctx context.Context, n *cke.Node) ([]byte, error) {
		return saCertData, nil
	}
	err = c.files.AddFile(ctx, op.K8sPKIPath("service-account.crt"), g)
	if err != nil {
		return err
	}
	if c.params.AuditLogEnabled {
		return c.files.AddFile(ctx, auditPolicyFilePath(c.params.AuditLogPolicy), func(context.Context, *cke.Node) ([]byte, error) {
			return []byte(c.params.AuditLogPolicy), nil
		})
	}
	return nil
}

func (c prepareAPIServerFilesCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-apiserver-files",
	}
}

func auditPolicyFilePath(policy string) string {
	return fmt.Sprintf(auditPolicyBasePath, md5.Sum([]byte(policy)))
}

// APIServerParams returns parameters for API server.
func APIServerParams(controlPlanes []*cke.Node, advertiseAddress, serviceSubnet string, auditLogEnabeled bool, auditLogPolicy string) cke.ServiceParams {
	var etcdServers []string
	for _, n := range controlPlanes {
		etcdServers = append(etcdServers, "https://"+n.Address+":2379")
	}

	args := []string{
		"apiserver",
		"--allow-privileged",
		"--etcd-servers=" + strings.Join(etcdServers, ","),
		"--etcd-cafile=" + op.K8sPKIPath("etcd-ca.crt"),
		"--etcd-certfile=" + op.K8sPKIPath("apiserver-etcd-client.crt"),
		"--etcd-keyfile=" + op.K8sPKIPath("apiserver-etcd-client.key"),

		"--bind-address=0.0.0.0",
		"--insecure-port=0",
		"--client-ca-file=" + op.K8sPKIPath("ca.crt"),
		"--tls-cert-file=" + op.K8sPKIPath("apiserver.crt"),
		"--tls-private-key-file=" + op.K8sPKIPath("apiserver.key"),
		"--kubelet-certificate-authority=" + op.K8sPKIPath("ca.crt"),
		"--kubelet-client-certificate=" + op.K8sPKIPath("apiserver.crt"),
		"--kubelet-client-key=" + op.K8sPKIPath("apiserver.key"),
		"--kubelet-https=true",

		"--enable-admission-plugins=" + strings.Join(admissionPlugins, ","),

		// for service accounts
		"--service-account-key-file=" + op.K8sPKIPath("service-account.crt"),
		"--service-account-lookup",

		"--authorization-mode=Node,RBAC",

		"--advertise-address=" + advertiseAddress,
		"--service-cluster-ip-range=" + serviceSubnet,
		"--machine-id-file=/etc/machine-id",
	}
	if auditLogEnabeled {
		args = append(args, "--audit-log-path=-")
		args = append(args, "--audit-policy-file="+auditPolicyFilePath(auditLogPolicy))
	}

	return cke.ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []cke.Mount{
			{
				Source:      "/etc/machine-id",
				Destination: "/etc/machine-id",
				ReadOnly:    true,
				Propagation: "",
				Label:       "",
			},
			{
				Source:      "/etc/kubernetes",
				Destination: "/etc/kubernetes",
				ReadOnly:    true,
				Propagation: "",
				Label:       cke.LabelShared,
			},
		},
	}
}
