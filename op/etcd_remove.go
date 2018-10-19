package op

import (
	"context"
	"strconv"
	"strings"

	"github.com/cybozu-go/cke"
)

type etcdRemoveMemberOp struct {
	endpoints []string
	ids       []uint64
	executed  bool
}

// EtcdRemoveMemberOp returns an Operator to remove member from etcd cluster.
func EtcdRemoveMemberOp(cp []*cke.Node, ids []uint64) cke.Operator {
	return &etcdRemoveMemberOp{
		endpoints: etcdEndpoints(cp),
		ids:       ids,
	}
}

func (o *etcdRemoveMemberOp) Name() string {
	return "etcd-remove-member"
}

func (o *etcdRemoveMemberOp) NextCommand() cke.Commander {
	if o.executed {
		return nil
	}
	o.executed = true

	return removeEtcdMemberCommand{o.endpoints, o.ids}
}

type removeEtcdMemberCommand struct {
	endpoints []string
	ids       []uint64
}

func (c removeEtcdMemberCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cli, err := inf.NewEtcdClient(ctx, c.endpoints)
	if err != nil {
		return err
	}
	defer cli.Close()

	for _, id := range c.ids {
		ct, cancel := context.WithTimeout(ctx, timeoutDuration)
		_, err := cli.MemberRemove(ct, id)
		cancel()
		if err != nil {
			return err
		}
	}
	// gofail: var etcdAfterMemberRemove struct{}
	return nil
}

func (c removeEtcdMemberCommand) Command() cke.Command {
	idStrs := make([]string, len(c.ids))
	for i, id := range c.ids {
		idStrs[i] = strconv.FormatUint(id, 10)
	}
	return cke.Command{
		Name:   "remove-etcd-member",
		Target: strings.Join(idStrs, ","),
	}
}
