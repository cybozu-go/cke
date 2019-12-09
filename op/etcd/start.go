package etcd

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

type etcdStartOp struct {
	endpoints []string
	nodes     []*cke.Node
	params    cke.EtcdParams
	step      int
	files     *common.FilesBuilder
	domain    string
}

// StartOp returns an Operator to start etcd containers.
func StartOp(cp []*cke.Node, nodes []*cke.Node, params cke.EtcdParams, domain string) cke.Operator {
	return &etcdStartOp{
		endpoints: etcdEndpoints(cp),
		nodes:     nodes,
		params:    params,
		files:     common.NewFilesBuilder(nodes),
		domain:    domain,
	}
}

func (o *etcdStartOp) Name() string {
	return "etcd-start"
}

func (o *etcdStartOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return prepareEtcdCertificatesCommand{o.files, o.domain}
	case 1:
		o.step++
		return o.files
	case 2:
		o.step++
		opts := []string{
			"--mount",
			"type=volume,src=" + op.EtcdVolumeName(o.params) + ",dst=/var/lib/etcd",
		}
		return startEtcdCommand{o.endpoints, o.nodes, opts, o.params.ServiceParams}
	case 3:
		o.step++
		return waitEtcdSyncCommand{etcdEndpoints(o.nodes), false}
	default:
		return nil
	}
}

func (o *etcdStartOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

type startEtcdCommand struct {
	endpoints []string
	nodes     []*cke.Node
	opts      []string
	extra     cke.ServiceParams
}

func (c startEtcdCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cli, err := inf.NewEtcdClient(ctx, c.endpoints)
	if err != nil {
		return err
	}
	defer cli.Close()

	ct, cancel := context.WithTimeout(ctx, op.TimeoutDuration)
	defer cancel()
	resp, err := cli.MemberList(ct)
	if err != nil {
		return err
	}
	members := resp.Members
	log.Debug("members in MemberList response", map[string]interface{}{
		"members": members,
	})

	env := well.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := inf.Engine(n.Address)
		n := n

		env.Go(func(ctx context.Context) error {
			ss, err := ce.Inspect([]string{op.EtcdContainerName})
			if err != nil {
				return err
			}
			if ss[op.EtcdContainerName].Running {
				return nil
			}

			var initialCluster []string
			for _, m := range members {
				for _, u := range m.PeerURLs {
					if len(m.Name) == 0 {
						initialCluster = append(initialCluster, n.Address+"="+u)
					} else {
						initialCluster = append(initialCluster, m.Name+"="+u)
					}
				}
			}

			exists, err := ce.Exists(op.EtcdContainerName)
			if err != nil {
				return err
			}
			if exists {
				err = ce.Remove(op.EtcdContainerName)
				if err != nil {
					return err
				}
			}
			log.Debug("initial-cluster", map[string]interface{}{
				"node":            n.Nodename(),
				"initial-cluster": initialCluster,
			})
			return ce.RunSystem(op.EtcdContainerName, cke.EtcdImage, c.opts, BuiltInParams(n, initialCluster, "existing"), c.extra)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c startEtcdCommand) Command() cke.Command {
	return cke.Command{
		Name: "start-etcd",
	}
}
