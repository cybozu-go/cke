package etcdbackup

import (
	"bytes"
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

type etcdBackupPodCreateOp struct {
	apiserver  *cke.Node
	etcdBackup cke.EtcdBackup
	finished   bool
}

// PodCreateOp returns an Operator to create etcdbackup cron job.
func PodCreateOp(apiserver *cke.Node, etcdBackup cke.EtcdBackup) cke.Operator {
	return &etcdBackupPodCreateOp{
		apiserver:  apiserver,
		etcdBackup: etcdBackup,
	}
}

func (o *etcdBackupPodCreateOp) Name() string {
	return "etcdbackup-job-create"
}

func (o *etcdBackupPodCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createEtcdBackupPodCommand{o.apiserver, o.etcdBackup}
}

type createEtcdBackupPodCommand struct {
	apiserver  *cke.Node
	etcdBackup cke.EtcdBackup
}

func (c createEtcdBackupPodCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	claims := cs.CoreV1().PersistentVolumeClaims("kube-system")
	_, err = claims.Get(c.etcdBackup.PVCName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	pods := cs.CoreV1().Pods("kube-system")
	_, err = pods.Get(op.EtcdBackupPodName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		buf := new(bytes.Buffer)
		err := podTemplate.Execute(buf, c.etcdBackup)
		if err != nil {
			return err
		}

		pod := new(corev1.Pod)
		err = k8sYaml.NewYAMLToJSONDecoder(buf).Decode(pod)
		if err != nil {
			return err
		}
		_, err = pods.Create(pod)
		if err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

func (c createEtcdBackupPodCommand) Command() cke.Command {
	return cke.Command{
		Name:   "create-etcdbackup-pod",
		Target: "etcdbackup",
	}
}
