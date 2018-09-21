package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/google/subcommands"
)

type etcd struct{}

func (v etcd) SetFlags(f *flag.FlagSet) {}

func (v etcd) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	newc := NewCommander(f, "etcd")
	newc.Register(etcdIssueCommand(), "")
	return newc.Execute(ctx)
}

// EtcdCommand implements "etcd" subcommand
func EtcdCommand() subcommands.Command {
	return subcmd{
		etcd{},
		"etcd",
		"control CKE managed etcd",
		"etcd ACTION ...",
	}
}

type etcdIssue struct {
	ttl string
}

func (c etcdIssue) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ttl, "ttl", "87600h", "TTL for client certificate")
}

func (c etcdIssue) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	if f.NArg() != 1 {
		f.Usage()
		return subcommands.ExitUsageError
	}

	commonName := f.Arg(0)

	if len(commonName) == 0 {
		return handleError(errors.New("common_name is empty"))
	}

	_, err := time.ParseDuration(c.ttl)
	if err != nil {
		return handleError(err)
	}

	cfg, err := storage.GetCluster(ctx)
	if err != nil {
		return handleError(err)
	}

	inf, err := cke.NewInfrastructure(ctx, cfg, storage)
	if err != nil {
		return handleError(err)
	}

	vaultCfg, err := storage.GetVaultConfig(ctx)
	if err != nil {
		return handleError(err)
	}
	data, err := json.Marshal(vaultCfg)
	if err != nil {
		return handleError(err)

	}

	err = cke.ConnectVault(ctx, data)
	if err != nil {
		return handleError(err)
	}

	// Issue certificate
	roleOpts := map[string]interface{}{
		"ttl":               c.ttl,
		"max_ttl":           "87600h",
		"enforce_hostnames": "false",
		"allow_any_name":    "true",
	}
	certOpts := map[string]interface{}{
		"common_name":          commonName,
		"exclude_cn_from_sans": "true",
	}
	crt, key, err := cke.IssueCertificate(inf, cke.CAEtcdClient, "system", roleOpts, certOpts)
	if err != nil {
		return handleError(err)
	}
	fmt.Println(crt)
	fmt.Println(key)

	// Add user/role to managed etcd
	//cpNodes := cke.ControlPlanes(cfg.Nodes)
	//endpoints := make([]string, len(cpNodes))
	//for i, n := range cpNodes {
	//	endpoints[i] = "https://" + n.Address + ":2379"
	//}

	//err := etcdutil.NewClient(&etcdutil.Config{
	//	Endpoints: endpoints,
	//	Timeout:   etcdutil.DefaultTimeout,
	//	TLSCA:     i.serverCA,
	//	TLSCert:   i.etcdCert,
	//	TLSKey:    i.etcdKey,
	//})

	//e := json.NewEncoder(os.Stdout)
	//e.SetIndent("", "  ")
	//e.Encode(secret.Data)
	return handleError(nil)
}

func etcdIssueCommand() subcommands.Command {
	return subcmd{
		etcdIssue{},
		"issue",
		"Issue client certificate and add user/role for CKE managed etcd",
		"vault issue COMMON_NAME [-ttl TTL]",
	}
}
