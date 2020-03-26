package k8s

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

type blockDeviceLinkUpdateOp struct {
	nodes []*cke.Node
	step  int
}

// BlockDeviceLinkUpdateOp returns an Operator to restart kubelet
func BlockDeviceLinkUpdateOp(nodes []*cke.Node) cke.Operator {
	return &blockDeviceLinkUpdateOp{nodes: nodes}
}

func (o *blockDeviceLinkUpdateOp) Name() string {
	return "block-device-link-update"
}

func (o *blockDeviceLinkUpdateOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

func (o *blockDeviceLinkUpdateOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return updateBlockDeviceLinkFor17(o.nodes)
	default:
		return nil
	}
}

type updateBlockDeviceLinkFor17Command struct {
	nodes []*cke.Node
}

// updateBlockDeviceLinkFor17 move raw block device files.
// This command is used for upgrading to k8s 1.17
func updateBlockDeviceLinkFor17(nodes []*cke.Node) cke.Commander {
	return updateBlockDeviceLinkFor17Command{nodes}
}

func (c updateBlockDeviceLinkFor17Command) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	begin := time.Now()
	env := well.NewEnvironment(ctx)
	for _, n := range c.nodes {
		n := n
		env.Go(func(ctx context.Context) error {
			clientset, err := inf.K8sClient(ctx, n)
			if err != nil {
				return err
			}

			agent := inf.Agent(n.Address)
			if agent == nil {
				return errors.New("unable to prepare agent for " + n.Nodename())
			}

			stdout, stderr, err := agent.Run(fmt.Sprintf("find %s -type b", op.CSIBlockDevicePublishDirectory))
			if err != nil {
				return fmt.Errorf("unable to ls on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
			}

			deviceFiles := strings.Fields(string(stdout))
			pvNames := getFilesJustUnderTargetDir(deviceFiles, op.CSIBlockDevicePublishDirectory)
			for _, pvName := range pvNames {
				pvcRef, err := getPVCFromPV(clientset, pvName)
				if err != nil {
					return err
				}

				po, err := getPodFromPVC(clientset, pvcRef)
				if err != nil {
					return err
				}

				podUID := string(po.GetUID())
				newDevicePath := makeNewDevicePath(pvName, podUID)
				symlinkSourcePath := makeSymlinkSourcePath(pvName, podUID)
				_, stderr, err = agent.Run(fmt.Sprintf("ln -nfs %s %s", newDevicePath, symlinkSourcePath))
				if err != nil {
					return fmt.Errorf("unable to ln on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
				}
			}
			return nil
		})
	}
	env.Stop()
	err := env.Wait()
	log.Info("updateBlockDeviceLinkFor17Command finished", map[string]interface{}{
		"elapsed": time.Now().Sub(begin).Seconds(),
	})
	return err
}

func (c updateBlockDeviceLinkFor17Command) Command() cke.Command {
	return cke.Command{Name: "update-block-device-link-for-1.17"}
}
