package etcdbackup

import (
	"bytes"
	"context"

	"k8s.io/api/core/v1"

	"github.com/cybozu-go/cke"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

type etcdBackupConfigMapUpdateOp struct {
	apiserver  *cke.Node
	etcdBackup cke.EtcdBackup
	finished   bool
}

// ConfigMapUpdateOp returns an Operator to Update etcdbackup ConfigMap.
func ConfigMapUpdateOp(apiserver *cke.Node, etcdBackup cke.EtcdBackup) cke.Operator {
	return &etcdBackupConfigMapUpdateOp{
		apiserver:  apiserver,
		etcdBackup: etcdBackup,
	}
}

func (o *etcdBackupConfigMapUpdateOp) Name() string {
	return "etcdbackup-configmap-update"
}

func (o *etcdBackupConfigMapUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateEtcdBackupConfigMapCommand{o.apiserver, o.etcdBackup}
}

type updateEtcdBackupConfigMapCommand struct {
	apiserver  *cke.Node
	etcdBackup cke.EtcdBackup
}

func (c updateEtcdBackupConfigMapCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	claims := cs.CoreV1().PersistentVolumeClaims("kube-system")
	_, err = claims.Get(c.etcdBackup.PVCName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = configMapTemplate.Execute(buf, c.etcdBackup)
	if err != nil {
		return err
	}

	ConfigMap := new(v1.ConfigMap)
	err = k8sYaml.NewYAMLToJSONDecoder(buf).Decode(ConfigMap)
	if err != nil {
		return err
	}

	maps := cs.CoreV1().ConfigMaps("kube-system")
	_, err = maps.Update(ConfigMap)
	return err
}

func (c updateEtcdBackupConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "update-etcdbackup-configmap",
		Target: "etcdbackup",
	}
}
