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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type blockDeviceMoveOp struct {
	nodes []*cke.Node
	step  int
}

// BlockDeviceMoveOp returns an Operator to restart kubelet
func BlockDeviceMoveOp(nodes []*cke.Node) cke.Operator {
	return &blockDeviceMoveOp{nodes: nodes}
}

func (o *blockDeviceMoveOp) Name() string {
	return "block-device-move"
}

func (o *blockDeviceMoveOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

func (o *blockDeviceMoveOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return moveBlockDeviceForV1_17(o.nodes)
	default:
		return nil
	}
}

type moveBlockDeviceForV1_17Command struct {
	nodes []*cke.Node
}

// moveBlockDeviceForV1_17 move raw block device files.
// This command is used for upgrading to k8s 1.17
func moveBlockDeviceForV1_17(nodes []*cke.Node) cke.Commander {
	return moveBlockDeviceForV1_17Command{nodes}
}

func (c moveBlockDeviceForV1_17Command) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	begin := time.Now()
	env := well.NewEnvironment(ctx)
	for _, n := range c.nodes {
		n := n
		env.Go(func(ctx context.Context) error {
			agent := inf.Agent(n.Address)
			if agent == nil {
				return errors.New("unable to prepare agent for " + n.Nodename())
			}

			stdout, stderr, err := agent.Run(fmt.Sprintf("sudo find %s -type b", op.CSIBlockDevicePublishDirectory))
			if err != nil {
				return fmt.Errorf("unable to find block device on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
			}

			deviceFiles := strings.Fields(string(stdout))
			pvNames := getFilesJustUnderTargetDir(deviceFiles, op.CSIBlockDevicePublishDirectory)
			for _, pvName := range pvNames {
				oldDevicePath := makeOldDevicePath(pvName)
				tmpDevicePath := makeTmpDevicePath(pvName)
				_, stderr, err = agent.Run(fmt.Sprintf("sudo mv %s %s", oldDevicePath, tmpDevicePath))
				if err != nil {
					return fmt.Errorf("unable to mv %s %s on %s; stderr: %s, err: %v", oldDevicePath, tmpDevicePath, n.Nodename(), stderr, err)
				}
			}
			return nil
		})
	}
	env.Stop()
	err := env.Wait()
	log.Info("moveBlockDeviceForV1_17Command finished", map[string]interface{}{
		"elapsed": time.Now().Sub(begin).Seconds(),
	})
	return err
}

func (c moveBlockDeviceForV1_17Command) Command() cke.Command {
	return cke.Command{Name: "move-block-device-for-1.17"}
}

func makeOldDevicePath(pvName string) string {
	return filepath.Join(op.CSIBlockDevicePublishDirectory, pvName)
}

func makeNewDevicePath(pvName, podUID string) string {
	return filepath.Join(op.CSIBlockDevicePublishDirectory, pvName, podUID)
}

func makeTmpDevicePath(pvName string) string {
	return filepath.Join("/tmp", pvName)
}

func makeSymlinkSourcePath(pvName, podUID string) string {
	return filepath.Join(op.CSIBlockDeviceDirectory, pvName, "dev", podUID)
}

func getPodFromPVC(clientset *kubernetes.Clientset, pvcRef *corev1.ObjectReference) (*corev1.Pod, error) {
	pods, err := clientset.CoreV1().Pods(pvcRef.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, p := range pods.Items {
		for _, v := range p.Spec.Volumes {
			if v.PersistentVolumeClaim == nil {
				continue
			}
			if v.PersistentVolumeClaim.ClaimName == pvcRef.Name {
				return &p, nil
			}
		}
	}
	return nil, errors.New("pod not found from PVC " + pvcRef.String())
}

func getPVCFromPV(clientset *kubernetes.Clientset, pvName string) (*corev1.ObjectReference, error) {
	pvs, err := clientset.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, pv := range pvs.Items {
		if pv.Name == pvName && pv.Spec.ClaimRef != nil {
			return pv.Spec.ClaimRef, nil
		}
	}
	return nil, errors.New("pvc not found from PV " + pvName)
}

func getFilesJustUnderTargetDir(files []string, targetDir string) (res []string) {
	for _, f := range files {
		if f == filepath.Join(targetDir, filepath.Base(f)) {
			res = append(res, filepath.Base(f))
		}
	}
	return res
}
