package k8s

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
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
		return moveBlockDeviceFor17(o.nodes)
	default:
		return nil
	}
}

type moveBlockDeviceFor17Command struct {
	nodes []*cke.Node
}

// moveBlockDeviceFor17 move raw block device files.
// This command is used for upgrading to k8s 1.17
func moveBlockDeviceFor17(nodes []*cke.Node) cke.Commander {
	return moveBlockDeviceFor17Command{nodes}
}

func (c moveBlockDeviceFor17Command) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	for _, n := range c.nodes {
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

			// 1. Move device file to tmp
			oldDevicePath := filepath.Join(op.CSIBlockDevicePublishDirectory, pvName)
			tmpDevicePath := oldDevicePath + ".tmp"
			_, stderr, err = agent.Run(fmt.Sprintf("mv %s %s", oldDevicePath, tmpDevicePath))
			if err != nil {
				return fmt.Errorf("unable to mv tmp on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
			}

			// 2. Create directory
			newDirectoryPath := oldDevicePath
			_, stderr, err = agent.Run(fmt.Sprintf("mkdir %s", newDirectoryPath))
			if err != nil {
				return fmt.Errorf("unable to mkdir on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
			}

			// 3. Move device file to the new path
			newDevicePath := filepath.Join(newDirectoryPath, string(po.GetUID()))
			_, stderr, err = agent.Run(fmt.Sprintf("mv %s %s", tmpDevicePath, newDevicePath))
			if err != nil {
				return fmt.Errorf("unable to mv on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
			}

			// 4. Fix symlink
			symlinkSourcePath := filepath.Join(op.CSIBlockDeviceDirectory, pvName, "dev", string(po.GetUID()))
			_, stderr, err = agent.Run(fmt.Sprintf("ln -nfs %s %s", newDevicePath, symlinkSourcePath))
			if err != nil {
				return fmt.Errorf("unable to ln on %s; stderr: %s, err: %v", n.Nodename(), stderr, err)
			}
		}
	}
	return nil
}

func (c moveBlockDeviceFor17Command) Command() cke.Command {
	return cke.Command{Name: "move-block-device-for-1.17"}
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
