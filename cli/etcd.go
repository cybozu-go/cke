package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/cybozu-go/cke"
	"github.com/google/subcommands"
)

type etcd struct{}

type IssueResponse struct {
	Crt   string `json:"certificate"`
	Key   string `json:"private_key"`
	CACrt string `json:"ca_certificate"`
}

func (v etcd) SetFlags(f *flag.FlagSet) {}

func (v etcd) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	newc := NewCommander(f, "etcd")
	newc.Register(etcdUserAddCommand(), "")
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

type etcdUserAdd struct {
	ttl    string
	prefix string
}

func (c *etcdUserAdd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ttl, "ttl", "87600h", "TTL for client certificate")
	f.StringVar(&c.prefix, "prefix", "/", "PREFIX to grant permission of etcd key path")
}

func (c *etcdUserAdd) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	if f.NArg() != 1 {
		f.Usage()
		return subcommands.ExitUsageError
	}

	userName := f.Arg(0)
	if len(userName) == 0 {
		return handleError(errors.New("username is empty"))
	}
	_, err := time.ParseDuration(c.ttl)
	if err != nil {
		return handleError(err)
	}

	cfg, err := storage.GetCluster(ctx)
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
	inf, err := cke.NewInfrastructureWithoutSSH(ctx, cfg, storage)
	if err != nil {
		return handleError(err)
	}

	// Issue certificate
	crt, key, err := cke.EtcdCA{}.IssueRoot(ctx, inf)
	if err != nil {
		return handleError(err)
	}

	// Get server CA certificate
	caCrt, err := storage.GetCACertificate(ctx, "server")
	if err != nil {
		return handleError(err)
	}

	// Add user/role to managed etcd
	cpNodes := cke.ControlPlanes(cfg.Nodes)
	endpoints := make([]string, len(cpNodes))
	for i, n := range cpNodes {
		endpoints[i] = "https://" + n.Address + ":2379"
	}
	etcdClient, err := inf.NewEtcdClient(endpoints)
	if err != nil {
		return handleError(err)
	}
	err = cke.AddUserRole(ctx, etcdClient, userName, c.prefix)
	// accept if user and role already exist
	if err != nil && err != rpctypes.ErrUserAlreadyExist {
		return handleError(err)
	}

	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	err = e.Encode(IssueResponse{crt, key, caCrt})
	return handleError(err)
}

func etcdUserAddCommand() subcommands.Command {
	return subcmd{
		&etcdUserAdd{},
		"user-add",
		"Issue client certificate and add user/role for CKE managed etcd",
		"etcd user-add COMMON_NAME [-ttl TTL] [-prefix PREFIX]",
	}
}
