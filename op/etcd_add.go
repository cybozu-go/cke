package op

import (
	"context"
	"fmt"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op/common"
)

type etcdAddMemberOp struct {
	endpoints  []string
	targetNode *cke.Node
	params     cke.EtcdParams
	step       int
	files      *common.FilesBuilder
	domain     string
}

// EtcdAddMemberOp returns an Operator to add member to etcd cluster.
func EtcdAddMemberOp(cp []*cke.Node, targetNode *cke.Node, params cke.EtcdParams, domain string) cke.Operator {
	return &etcdAddMemberOp{
		endpoints:  etcdEndpoints(cp),
		targetNode: targetNode,
		params:     params,
		files:      common.NewFilesBuilder([]*cke.Node{targetNode}),
		domain:     domain,
	}
}

func (o *etcdAddMemberOp) Name() string {
	return "etcd-add-member"
}

func (o *etcdAddMemberOp) NextCommand() cke.Commander {
	volname := etcdVolumeName(o.params)
	extra := o.params.ServiceParams

	nodes := []*cke.Node{o.targetNode}
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(nodes, cke.EtcdImage)
	case 1:
		o.step++
		return common.StopContainerCommand(o.targetNode, EtcdContainerName)
	case 2:
		o.step++
		return common.VolumeRemoveCommand(nodes, volname)
	case 3:
		o.step++
		return common.VolumeCreateCommand(nodes, volname)
	case 4:
		o.step++
		return prepareEtcdCertificatesCommand{o.files, o.domain}
	case 5:
		o.step++
		return o.files
	case 6:
		o.step++
		opts := []string{
			"--mount",
			"type=volume,src=" + volname + ",dst=/var/lib/etcd",
		}
		return addEtcdMemberCommand{o.endpoints, o.targetNode, opts, extra}
	case 7:
		o.step++
		return waitEtcdSyncCommand{etcdEndpoints([]*cke.Node{o.targetNode}), false}
	}
	return nil
}

type addEtcdMemberCommand struct {
	endpoints []string
	node      *cke.Node
	opts      []string
	extra     cke.ServiceParams
}

func (c addEtcdMemberCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cli, err := inf.NewEtcdClient(ctx, c.endpoints)
	if err != nil {
		return err
	}
	defer cli.Close()

	ct, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()
	resp, err := cli.MemberList(ct)
	if err != nil {
		return err
	}
	members := resp.Members

	inMember := false
	for _, m := range members {
		inMember, err = addressInURLs(c.node.Address, m.PeerURLs)
		if err != nil {
			return err
		}
		if inMember {
			break
		}
	}

	if !inMember {
		// wait for several seconds to satisfy etcd server check
		// https://github.com/etcd-io/etcd/blob/fb674833c21e729fe87fff4addcf93b2aa4df9df/etcdserver/server.go#L1562
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}

		ct, cancel := context.WithTimeout(ctx, timeoutDuration)
		defer cancel()
		resp, err := cli.MemberAdd(ct, []string{fmt.Sprintf("https://%s:2380", c.node.Address)})
		if err != nil {
			return err
		}
		members = resp.Members
	}
	// gofail: var etcdAfterMemberAdd struct{}
	ce := inf.Engine(c.node.Address)
	ss, err := ce.Inspect([]string{EtcdContainerName})
	if err != nil {
		return err
	}
	if ss[EtcdContainerName].Running {
		return nil
	}

	var initialCluster []string
	for _, m := range members {
		for _, u := range m.PeerURLs {
			if len(m.Name) == 0 {
				initialCluster = append(initialCluster, c.node.Address+"="+u)
			} else {
				initialCluster = append(initialCluster, m.Name+"="+u)
			}
		}
	}

	return ce.RunSystem(EtcdContainerName, cke.EtcdImage, c.opts, EtcdBuiltInParams(c.node, initialCluster, "existing"), c.extra)
}

func (c addEtcdMemberCommand) Command() cke.Command {
	return cke.Command{
		Name:   "add-etcd-member",
		Target: c.node.Address,
	}
}
