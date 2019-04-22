package etcd

import (
	"context"
	"strconv"
	"strings"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
)

type removeMemberOp struct {
	endpoints []string
	ids       []uint64
	members   []*etcdserverpb.Member
	executed  bool
}

// RemoveMemberOp returns an Operator to remove member from etcd cluster.
func RemoveMemberOp(cp []*cke.Node, members []*etcdserverpb.Member) cke.Operator {
	ids := []uint64{}
	for _, m := range members {
		ids = append(ids, m.ID)
	}
	return &removeMemberOp{
		endpoints: etcdEndpoints(cp),
		ids:       ids,
		members:   members,
	}
}

func (o *removeMemberOp) Name() string {
	return "etcd-remove-member"
}

func (o *removeMemberOp) NextCommand() cke.Commander {
	if o.executed {
		return nil
	}
	o.executed = true

	return removeMemberCommand{o.endpoints, o.ids}
}

func (o *removeMemberOp) Nodes() []string {
	return []string{}
}

type removeMemberCommand struct {
	endpoints []string
	ids       []uint64
}

func (c removeMemberCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cli, err := inf.NewEtcdClient(ctx, c.endpoints)
	if err != nil {
		return err
	}
	defer cli.Close()

	for _, id := range c.ids {
		ct, cancel := context.WithTimeout(ctx, op.TimeoutDuration)
		_, err := cli.MemberRemove(ct, id)
		cancel()
		if err != nil {
			return err
		}
	}
	// gofail: var etcdAfterMemberRemove struct{}
	return nil
}

func (c removeMemberCommand) Command() cke.Command {
	idStrs := make([]string, len(c.ids))
	for i, id := range c.ids {
		idStrs[i] = strconv.FormatUint(id, 10)
	}
	return cke.Command{
		Name:   "remove-etcd-member",
		Target: strings.Join(idStrs, ","),
	}
}
