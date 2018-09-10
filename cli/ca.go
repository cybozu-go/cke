package cli

import (
	"context"
	"flag"
	"io/ioutil"
	"os"

	"github.com/google/subcommands"
)

type ca struct{}

func (c ca) SetFlags(f *flag.FlagSet) {}

func (c ca) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
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
		"ca set/get ARGS...",
	}
}

func isValidCAName(name string) bool {
	switch name {
	case "server", "etcd-peer", "etcd-client", "kubernetes":
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

	pemfile := f.Arg(1)
	g, err := os.Open(pemfile)
	if err != nil {
		return handleError(err)
	}
	defer g.Close()

	pem, err := ioutil.ReadAll(g)
	if err != nil {
		return handleError(err)
	}

	err = storage.PutCACertificate(ctx, name, string(pem))

	return handleError(err)
}

func caSetCommand() subcommands.Command {
	return subcmd{
		caSet{},
		"set",
		"set CA certificate",
		`ca set NAME PEMFILE

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

	_, err = os.Stdout.WriteString(pem)
	return handleError(err)
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
