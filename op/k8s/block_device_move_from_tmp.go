package k8s

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

type blockDeviceMoveFromTmpOp struct {
	apiServer *cke.Node
	nodes     []*cke.Node
	step      int
}

// BlockDeviceMoveFromTmpOp returns an Operator to restart kubelet
func BlockDeviceMoveFromTmpOp(apiServer *cke.Node, nodes []*cke.Node) cke.Operator {
	return &blockDeviceMoveFromTmpOp{apiServer: apiServer, nodes: nodes}
}

func (o *blockDeviceMoveFromTmpOp) Name() string {
	return "block-device-move-from-tmp"
}

func (o *blockDeviceMoveFromTmpOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

func (o *blockDeviceMoveFromTmpOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return moveBlockDeviceFromTmpForV1_17(o.apiServer, o.nodes)
	default:
		return nil
	}
}

type moveBlockDeviceFromTmpForV1_17Command struct {
	apiServer *cke.Node
	nodes     []*cke.Node
}

// moveBlockDeviceFromTmpForV1_17 move raw block device files.
// This command is used for upgrading to k8s 1.17
func moveBlockDeviceFromTmpForV1_17(apiServer *cke.Node, nodes []*cke.Node) cke.Commander {
	return moveBlockDeviceFromTmpForV1_17Command{apiServer: apiServer, nodes: nodes}
}

func (c moveBlockDeviceFromTmpForV1_17Command) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	begin := time.Now()
	env := well.NewEnvironment(ctx)
	for _, n := range c.nodes {
		n := n
		env.Go(func(ctx context.Context) error {
			clientset, err := inf.K8sClient(ctx, c.apiServer)
			if err != nil {
				return err
			}

			agent := inf.Agent(n.Address)
			if agent == nil {
				return errors.New("unable to prepare agent for " + n.Nodename())
			}

			stdout, stderr, err := agent.Run("sudo find /tmp/ -name pvc-* -type b")
			if err != nil {
				return fmt.Errorf("unable to find block device on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
			}

			deviceFiles := strings.Fields(string(stdout))
			pvNames := getFilesJustUnderTargetDir(deviceFiles, "/tmp/")
			for _, pvName := range pvNames {
				pvcRef, err := getPVCFromPV(clientset, pvName)
				if err != nil {
					return err
				}

				po, err := getPodFromPVC(clientset, pvcRef)
				if err != nil {
					return err
				}

				// Make directory at the path where the device path existed before being moved
				newDirectoryPath := makeOldDevicePath(pvName)
				_, stderr, err = agent.Run(fmt.Sprintf("sudo mkdir -p %s", newDirectoryPath))
				if err != nil {
					return fmt.Errorf("unable to mkdir on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
				}

				tmpDevicePath := makeTmpDevicePath(pvName)
				newDevicePath := makeNewDevicePath(pvName, string(po.GetUID()))
				_, stderr, err = agent.Run(fmt.Sprintf("sudo mv %s %s", tmpDevicePath, newDevicePath))
				if err != nil {
					return fmt.Errorf("unable to mv on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
				}
			}
			return nil
		})
	}
	env.Stop()
	err := env.Wait()
	log.Info("moveBlockDeviceFromTmpForV1_17Command finished", map[string]interface{}{
		"elapsed": time.Now().Sub(begin).Seconds(),
	})
	return err
}

func (c moveBlockDeviceFromTmpForV1_17Command) Command() cke.Command {
	return cke.Command{Name: "move-block-device-from-tmp-for-1.17"}
}
