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

const auditPolicyBasePath = "/etc/kubernetes/apiserver/audit-policy-%x.yaml"

var (
	// admissionPlugins is our recommended list of admission plugins in addition to the default ones.
	// https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use
	admissionPlugins = []string{
		// NodeRestriction restricts kubelet privilege.
		"NodeRestriction",
		// DenyServiceExternalIPs prohibits use of .spec.externalIPs for Services.
		// LoadBalancer-type Services are still usable.
		"DenyServiceExternalIPs",
	}
)

type apiServerRestartOp struct {
	nodes []*cke.Node
	cps   []*cke.Node

	serviceSubnet string
	params        cke.APIServerParams
	clusterDomain string

	step  int
	files *common.FilesBuilder
}

// APIServerRestartOp returns an Operator to restart kube-apiserver
func APIServerRestartOp(nodes, cps []*cke.Node, serviceSubnet string, params cke.APIServerParams, clusterDomain string) cke.Operator {
	return &apiServerRestartOp{
		nodes:         nodes,
		cps:           cps,
		serviceSubnet: serviceSubnet,
		clusterDomain: clusterDomain,
		params:        params,
		files:         common.NewFilesBuilder(nodes),
	}
}

func (o *apiServerRestartOp) Name() string {
	return "kube-apiserver-restart"
}

func (o *apiServerRestartOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.KubernetesImage)
	case 1:
		o.step++
		return common.MakeDirsCommandWithMode(o.nodes, []string{encryptionConfigDir}, "700")
	case 2:
		o.step++
		return prepareAPIServerFilesCommand{o.files, o.serviceSubnet, o.clusterDomain, o.params}
	case 3:
		o.step++
		return o.files
	case 4:
		o.step++
		return common.StopContainersCommand(o.nodes, op.KubeAPIServerContainerName)
	case 5:
		o.step++
		opts := []string{
			"--mount", "type=tmpfs,dst=/run/kubernetes",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			paramsMap[n.Address] = APIServerParams(n.Address, o.serviceSubnet, o.params.AuditLogEnabled, o.params.AuditLogPolicy, o.params.AuditLogPath, o.clusterDomain)
		}
		return common.RunContainerCommand(o.nodes,
			op.KubeAPIServerContainerName, cke.KubernetesImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params.ServiceParams))
	default:
		return nil
	}
}

func (o *apiServerRestartOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

type prepareAPIServerFilesCommand struct {
	files         *common.FilesBuilder
	serviceSubnet string
	clusterDomain string
	params        cke.APIServerParams
}

func (c prepareAPIServerFilesCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	storage := inf.Storage()

	// server (and client) certs of API server.
	f := func(ctx context.Context, n *cke.Node) (cert, key []byte, err error) {
		c, k, e := cke.KubernetesCA{}.IssueForAPIServer(ctx, inf, n, c.serviceSubnet, c.clusterDomain)
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
	ca, err := storage.GetCACertificate(ctx, cke.CAKubernetes)
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
	etcdCA, err := storage.GetCACertificate(ctx, cke.CAServer)
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
	saKey, err := storage.GetServiceAccountKey(ctx)
	if err != nil {
		return err
	}
	f = func(ctx context.Context, n *cke.Node) (cert, key []byte, err error) {
		return []byte(saCert), []byte(saKey), nil
	}
	err = c.files.AddKeyPair(ctx, op.K8sPKIPath("service-account"), f)
	if err != nil {
		return err
	}

	// Aggregation cert.
	agCert, err := storage.GetCACertificate(ctx, cke.CAKubernetesAggregation)
	if err != nil {
		return err
	}
	agCertData := []byte(agCert)
	g = func(ctx context.Context, n *cke.Node) ([]byte, error) {
		return agCertData, nil
	}
	err = c.files.AddFile(ctx, op.K8sPKIPath("aggregation-ca.crt"), g)
	if err != nil {
		return err
	}

	// client certs for Aggregation
	f = func(ctx context.Context, n *cke.Node) (cert, key []byte, err error) {
		c, k, e := cke.AggregationCA{}.IssueClientCertificate(ctx, inf)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err = c.files.AddKeyPair(ctx, op.K8sPKIPath("aggregation"), f)
	if err != nil {
		return err
	}

	// EncryptionConfiguration
	enccfg, err := getEncryptionConfiguration(ctx, inf)
	if err != nil {
		return err
	}
	enccfgData, err := encodeToYAML(enccfg)
	if err != nil {
		return err
	}
	err = c.files.AddFile(ctx, encryptionConfigFile, func(ctx context.Context, node *cke.Node) ([]byte, error) {
		return enccfgData, nil
	})
	if err != nil {
		return err
	}

	// audit log policy
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
func APIServerParams(advertiseAddress, serviceSubnet string, auditLogEnabled bool, auditLogPolicy, auditLogPath string, clusterDomain string) cke.ServiceParams {
	args := []string{
		"kube-apiserver",
		"--allow-privileged",
		"--etcd-servers=https://127.0.0.1:12379",
		"--etcd-cafile=" + op.K8sPKIPath("etcd-ca.crt"),
		"--etcd-certfile=" + op.K8sPKIPath("apiserver-etcd-client.crt"),
		"--etcd-keyfile=" + op.K8sPKIPath("apiserver-etcd-client.key"),
		// disable compaction by apisever as it cannot do it.
		"--etcd-compaction-interval=0",

		"--bind-address=0.0.0.0",
		"--client-ca-file=" + op.K8sPKIPath("ca.crt"),
		"--tls-cert-file=" + op.K8sPKIPath("apiserver.crt"),
		"--tls-private-key-file=" + op.K8sPKIPath("apiserver.key"),
		"--kubelet-certificate-authority=" + op.K8sPKIPath("ca.crt"),
		"--kubelet-client-certificate=" + op.K8sPKIPath("apiserver.crt"),
		"--kubelet-client-key=" + op.K8sPKIPath("apiserver.key"),

		"--enable-admission-plugins=" + strings.Join(admissionPlugins, ","),

		// for service accounts
		"--service-account-issuer=https://kubernetes.default.svc." + clusterDomain,
		"--service-account-key-file=" + op.K8sPKIPath("service-account.crt"),
		"--service-account-signing-key-file=" + op.K8sPKIPath("service-account.key"),
		"--service-account-lookup",

		// for aggregation
		"--requestheader-client-ca-file=" + op.K8sPKIPath("aggregation-ca.crt"),
		"--requestheader-allowed-names=" + cke.CNAPIServer,
		"--requestheader-extra-headers-prefix=X-Remote-Extra-",
		"--requestheader-group-headers=X-Remote-Group",
		"--requestheader-username-headers=X-Remote-User",
		"--proxy-client-cert-file=" + op.K8sPKIPath("aggregation.crt"),
		"--proxy-client-key-file=" + op.K8sPKIPath("aggregation.key"),

		"--authorization-mode=Node,RBAC",

		"--advertise-address=" + advertiseAddress,

		// See https://github.com/cybozu-go/neco/issues/397
		"--endpoint-reconciler-type=none",

		"--service-cluster-ip-range=" + serviceSubnet,
		"--encryption-provider-config=" + encryptionConfigFile,
	}
	if auditLogEnabled {
		logPath := "-"
		if auditLogPath != "" {
			logPath = auditLogPath
		}
		args = append(args, "--audit-log-path="+logPath)
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
