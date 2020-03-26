package k8s

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

type blockDeviceMoveToTmpOp struct {
	nodes []*cke.Node
	step  int
}

// BlockDeviceMoveToTmpOp returns an Operator to restart kubelet
func BlockDeviceMoveToTmpOp(nodes []*cke.Node) cke.Operator {
	return &blockDeviceMoveToTmpOp{nodes: nodes}
}

func (o *blockDeviceMoveToTmpOp) Name() string {
	return "block-device-move-to-tmp"
}

func (o *blockDeviceMoveToTmpOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

func (o *blockDeviceMoveToTmpOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return moveBlockDeviceToTmpFor17(o.nodes)
	default:
		return nil
	}
}

type moveBlockDeviceToTmpFor17Command struct {
	nodes []*cke.Node
}

// moveBlockDeviceToTmpFor17 move raw block device files.
// This command is used for upgrading to k8s 1.17
func moveBlockDeviceToTmpFor17(nodes []*cke.Node) cke.Commander {
	return moveBlockDeviceToTmpFor17Command{nodes}
}

func (c moveBlockDeviceToTmpFor17Command) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
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

				newDirectoryPath := filepath.Join(op.CSIBlockDevicePublishDirectory, pvName)
				_, stderr, err = agent.Run(fmt.Sprintf("mkdir -p %s", newDirectoryPath))
				if err != nil {
					return fmt.Errorf("unable to mkdir on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
				}

				tmpDevicePath := filepath.Join("/tmp", pvName)
				newDevicePath := filepath.Join(newDirectoryPath, string(po.GetUID()))
				_, stderr, err = agent.Run(fmt.Sprintf("mv %s %s", tmpDevicePath, newDevicePath))
				if err != nil {
					return fmt.Errorf("unable to mv on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
				}
			}
			return nil
		})
	}
	env.Stop()
	err := env.Wait()
	log.Info("moveBlockDeviceToTmpFor17Command finished", map[string]interface{}{
		"elapsed": time.Now().Sub(begin).Seconds(),
	})
	return err
}

func (c moveBlockDeviceToTmpFor17Command) Command() cke.Command {
	return cke.Command{Name: "move-block-device-to-tmp-for-1.17"}
}
