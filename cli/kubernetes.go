package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/cybozu-go/cke"
	"github.com/google/subcommands"
	"k8s.io/client-go/tools/clientcmd"
)

type kubernetes struct{}

func (k kubernetes) SetFlags(f *flag.FlagSet) {}

func (k kubernetes) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	newc := NewCommander(f, "kubernetes")
	newc.Register(kubernetesIssueCommand(), "")
	return newc.Execute(ctx)
}

// KubernetesCommand implements "kubernetes" subcommand
func KubernetesCommand() subcommands.Command {
	return subcmd{
		kubernetes{},
		"kubernetes",
		"control CKE managed kubernetes",
		"kubernetes ACTION ...",
	}
}

type kubernetesIssue struct {
	ttl string
}

func (c *kubernetesIssue) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ttl, "ttl", "2h", "TTL for client certificate")
}

func (c *kubernetesIssue) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	cluster, inf, err := prepareInfrastructure(ctx)
	if err != nil {
		return handleError(err)
	}

	caCrt, err := inf.Storage().GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return handleError(err)
	}

	crt, key, err := cke.KubernetesCA{}.IssueAdminCert(ctx, inf, c.ttl)
	if err != nil {
		return handleError(err)
	}

	cpNodes := cke.ControlPlanes(cluster.Nodes)
	apiServerPort := ":6443"
	// TODO: Replace `server` by Ingress address. Since there is no Ingress yet, set the node IP
	server := "https://" + cpNodes[0].Address + apiServerPort
	cfg := cke.AdminKubeconfig(cluster.Name, caCrt, crt, key, server)
	src, err := clientcmd.Write(*cfg)
	if err != nil {
		return handleError(err)
	}
	_, err = fmt.Println(string(src))
	return handleError(err)
}

func kubernetesIssueCommand() subcommands.Command {
	return subcmd{
		&kubernetesIssue{},
		"issue",
		"Issue client certificate to connect kube-apiserver",
		"kubernetes issue",
	}
}
