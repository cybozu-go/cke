package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
)

type ca struct{}

func (v ca) SetFlags(f *flag.FlagSet) {}

func (v ca) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	newc := NewCommander(f, "ca")
	newc.Register(caSetCommand(), "")
	newc.Register(caGetCommand(), "")
	return newc.Execute(ctx)
}

// CACommand implements "ca" subcommand
func CACommand() subcommands.Command {
	return subcmd{
		ca{},
		"ca",
		"set the ca configuration",
		"ca config JSON",
	}
}

func isValidCAName(name string) bool {
	if name == "server" || name == "etcd-peer" || name == "etcd-client" {
		return true
	}
	return false
}

type caSet struct{}

func (c caSet) SetFlags(f *flag.FlagSet) {}

func (c caSet) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	if f.NArg() != 2 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	name := f.Arg(0)
	if !isValidCAName(name) {
		f.Usage()
		return subcommands.ExitUsageError
	}
	pem := f.Arg(1)
	err := storage.PutCACertificate(ctx, name, pem)

	return handleError(err)
}

func caSetCommand() subcommands.Command {
	return subcmd{
		caSet{},
		"set",
		"set CA certificate",
		`ca set NAME PEM

NAME is one of:
		- server
		- etcd-peer
		- etcd-client
`,
	}
}

type caGet struct{}

func (c caGet) SetFlags(f *flag.FlagSet) {}

func (c caGet) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	if f.NArg() != 1 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	name := f.Arg(0)
	if !isValidCAName(name) {
		f.Usage()
		return subcommands.ExitUsageError
	}

	pem, err := storage.GetCACertificate(ctx, name)
	if err != nil {
		return handleError(err)
	}

	fmt.Println(pem)
	return handleError(nil)
}

func caGetCommand() subcommands.Command {
	return subcmd{
		caGet{},
		"get",
		"get CA certificate",
		`ca get NAME PEM

NAME is one of:
		- server
		- etcd-peer
		- etcd-client
`,
	}
}
