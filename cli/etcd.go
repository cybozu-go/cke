package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/cybozu-go/cke"
	"github.com/google/subcommands"
)

type etcd struct{}

// IssueResponse is cli output format
type IssueResponse struct {
	Crt   string `json:"certificate"`
	Key   string `json:"private_key"`
	CACrt string `json:"ca_certificate"`
}

func (v etcd) SetFlags(f *flag.FlagSet) {}

func (v etcd) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	newc := NewCommander(f, "etcd")
	newc.Register(etcdUserAddCommand(), "")
	newc.Register(etcdIssueCommand(), "")
	newc.Register(etcdRootIssueCommand(), "")
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
	userName string
	prefix   string
}

func (c *etcdUserAdd) SetFlags(f *flag.FlagSet) {
}

func (c *etcdUserAdd) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	if f.NArg() != 2 {
		f.Usage()
		return subcommands.ExitUsageError
	}

	c.userName = f.Arg(0)
	if len(c.userName) == 0 {
		return handleError(errors.New("COMMON_NAME is empty"))
	}

	c.prefix = f.Arg(1)
	if len(c.prefix) == 0 {
		return handleError(errors.New("PREFIX is empty"))
	}

	err := c.validation()
	if err != nil {
		return handleError(err)
	}

	cfg, inf, err := prepareInfrastructure(ctx)
	if err != nil {
		return handleError(err)
	}

	// Add user/role to managed etcd
	endpoints := endpoints(cfg)
	etcdClient, err := inf.NewEtcdClient(ctx, endpoints)
	if err != nil {
		return handleError(err)
	}
	err = cke.AddUserRole(ctx, etcdClient, c.userName, c.prefix)
	// accept if user and role already exist
	if err != nil && err != rpctypes.ErrUserAlreadyExist {
		return handleError(err)
	} else if err == rpctypes.ErrUserAlreadyExist {
		fmt.Println(c.userName + " already exists.")
	} else {
		fmt.Println(c.userName + " created.")
	}
	return handleError(nil)
}

func (c *etcdUserAdd) validation() error {
	if strings.HasPrefix(c.userName, "system:") {
		return errors.New("COMMON_NAME should not have `system:` prefix")
	}
	return nil
}

func etcdUserAddCommand() subcommands.Command {
	return subcmd{
		&etcdUserAdd{},
		"user-add",
		"Add user/role for CKE managed etcd",
		"etcd user-add COMMON_NAME PREFIX",
	}
}

type etcdIssue struct {
	ttl    string
	output string
}

func (c *etcdIssue) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ttl, "ttl", "87600h", "TTL for client certificate")
	f.StringVar(&c.output, "output", "json", "output format (json|file)")
}

func (c *etcdIssue) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	userName := f.Arg(0)
	if len(userName) == 0 {
		return handleError(errors.New("COMMON_NAME is empty"))
	}
	if !(c.output == "json" || c.output == "file") {
		return handleError(errors.New("invalid option: output=" + c.output))
	}

	cfg, inf, err := prepareInfrastructure(ctx)
	if err != nil {
		return handleError(err)
	}
	endpoints := endpoints(cfg)
	etcdClient, err := inf.NewEtcdClient(ctx, endpoints)
	if err != nil {
		return handleError(err)
	}
	roles, err := cke.GetUserRoles(ctx, etcdClient, userName)
	if err != nil {
		return handleError(err)
	}
	role := role(roles, userName)
	if len(role) == 0 {
		return handleError(errors.New(userName + " does not have " + userName + " role"))
	}

	// Issue certificate
	crt, key, err := cke.IssueEtcdClientCertificate(inf, role, userName, c.ttl)
	if err != nil {
		return handleError(err)
	}

	// Get server CA certificate
	caCrt, err := storage.GetCACertificate(ctx, "server")
	if err != nil {
		return handleError(err)
	}

	switch c.output {
	case "file":
		random := random()
		crtFile := userName + "-" + random + ".crt"
		caCrtFile := userName + "-" + random + ".ca"
		keyFile := userName + "-" + random + ".key"
		err = ioutil.WriteFile(crtFile, []byte(crt), 0600)
		if err != nil {
			return handleError(err)
		}
		err = ioutil.WriteFile(caCrtFile, []byte(caCrt), 0600)
		if err != nil {
			return handleError(err)
		}
		err = ioutil.WriteFile(keyFile, []byte(key), 0600)
		if err != nil {
			return handleError(err)
		}
		_, err = fmt.Println("write files: ", crtFile, caCrtFile, keyFile)
		if err != nil {
			return handleError(err)
		}
	default:
		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", "  ")
		err = e.Encode(IssueResponse{crt, key, caCrt})
		if err != nil {
			return handleError(err)
		}
	}
	return subcommands.ExitSuccess
}

func etcdIssueCommand() subcommands.Command {
	return subcmd{
		&etcdIssue{},
		"issue",
		"Issue client certificate",
		"etcd issue [-ttl TTL] [-output FORMAT] COMMON_NAME",
	}
}

type etcdRootIssue struct {
	output string
}

func (c *etcdRootIssue) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.output, "output", "json", "output format (json|file)")
}

func (c *etcdRootIssue) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	_, inf, err := prepareInfrastructure(ctx)
	if err != nil {
		return handleError(err)
	}
	if !(c.output == "json" || c.output == "file") {
		return handleError(errors.New("invalid option: output=" + c.output))
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
	switch c.output {
	case "file":
		random := random()
		crtFile := "root-" + random + ".crt"
		caCrtFile := "root-" + random + ".ca"
		keyFile := "root-" + random + ".key"
		err = ioutil.WriteFile(crtFile, []byte(crt), 0600)
		if err != nil {
			return handleError(err)
		}
		err = ioutil.WriteFile(caCrtFile, []byte(caCrt), 0600)
		if err != nil {
			return handleError(err)
		}
		err = ioutil.WriteFile(keyFile, []byte(key), 0600)
		if err != nil {
			return handleError(err)
		}
		_, err = fmt.Println("write files: ", crtFile, caCrtFile, keyFile)
		if err != nil {
			return handleError(err)
		}
	default:
		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", "  ")
		err = e.Encode(IssueResponse{crt, key, caCrt})
		if err != nil {
			return handleError(err)
		}
	}
	return subcommands.ExitSuccess
}

func etcdRootIssueCommand() subcommands.Command {
	return subcmd{
		&etcdRootIssue{},
		"root-issue",
		"Issue root client certificate for CKE managed etcd",
		"etcd root-issue [-output FORMAT]",
	}
}

func endpoints(cfg *cke.Cluster) []string {
	cpNodes := cke.ControlPlanes(cfg.Nodes)
	endpoints := make([]string, len(cpNodes))
	for i, n := range cpNodes {
		endpoints[i] = "https://" + n.Address + ":2379"
	}
	return endpoints
}

func prepareInfrastructure(ctx context.Context) (*cke.Cluster, cke.Infrastructure, error) {
	cfg, err := storage.GetCluster(ctx)
	if err != nil {
		return nil, nil, err
	}
	vaultCfg, err := storage.GetVaultConfig(ctx)
	if err != nil {
		return nil, nil, err
	}
	data, err := json.Marshal(vaultCfg)
	if err != nil {
		return nil, nil, err
	}
	err = cke.ConnectVault(ctx, data)
	if err != nil {
		return nil, nil, err
	}
	inf := cke.NewInfrastructureWithoutSSH(storage)
	return cfg, inf, nil
}

func random() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func role(roles []string, userName string) string {
	for _, r := range roles {
		if r == userName {
			return r
		}
	}
	return ""
}
