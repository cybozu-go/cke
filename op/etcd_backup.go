package op

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"text/template"

	"github.com/cybozu-go/cke"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

var configMapText = `
metadata:
  name: etcd-backup-scripts
  namespace: kube-system
data:
  etcd-backup.sh: |
    #!/bin/sh -e

    SNAPSHOT=snapshot-$(date '+%Y%m%d_%H%M%S');

    env ETCDCTL_API=3 /usr/local/etcd/bin/etcdctl \
        --endpoints https://cke-etcd:2379 \
        --cacert=/etcd-certs/ca \
        --cert=/etcd-certs/cert \
        --key=/etcd-certs/key \
        snapshot save /etcd-backup/${SNAPSHOT}.db;
    tar --remove-files -cvzf /etcd-backup/${SNAPSHOT}.tar.gz /etcd-backup/${SNAPSHOT}.db;
    find /etcd-backup/ -mtime +14 -exec rm -f {} \;
`

var secretTemplate = template.Must(template.New("").Parse(`
metadata:
  name: etcd-backup-secret
  namespace: kube-system
data:
  cert: "{{ .Cert }}"
  key: "{{ .Key }}"
  ca: "{{ .CA }}"
`))

var cronJobTemplate = template.Must(template.New("").Parse(`
metadata:
  name: etcd-backup
  namespace: kube-system
spec:
  schedule: "{{ .Schedule }}"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: etcd-backup
            image: ` + cke.EtcdImage.Name() + `
            command:
              - /etcd-scripts/etcd-backup.sh
            volumeMounts:
              - mountPath: /etcd-scripts
                name: etcd-backup-scripts
              - mountPath: /etcd-certs
                name: etcd-certs
              - mountPath: /etcd-backup
                name: etcd-backup
          volumes:
          - name: etcd-backup-scripts
            configMap:
              name: etcd-backup-scripts
              items:
              - key: etcd-backup.sh
                path: etcd-backup.sh
              defaultMode: 0555
          - name: etcd-certs
            secret:
              secretName: etcd-backup-secret
              defaultMode: 0444
          - name: etcd-backup
            persistentVolumeClaim:
              claimName: {{ .PVCName }}
          restartPolicy: Never
`))

type etcdBackupConfigMapCreateOp struct {
	apiserver *cke.Node
	finished  bool
}

// EtcdBackupConfigMapCreateOp returns an Operator to create etcd-backup config.
func EtcdBackupConfigMapCreateOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupConfigMapCreateOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupConfigMapCreateOp) Name() string {
	return "etcd-backup-configmap-create"
}

func (o *etcdBackupConfigMapCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createEtcdBackupConfigMapCommand{o.apiserver}
}

type createEtcdBackupConfigMapCommand struct {
	apiserver *cke.Node
}

func (c createEtcdBackupConfigMapCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	configs := cs.CoreV1().ConfigMaps("kube-system")
	_, err = configs.Get(etcdBackupConfigMapName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		config := new(corev1.ConfigMap)
		err = k8sYaml.NewYAMLToJSONDecoder(strings.NewReader(configMapText)).Decode(config)
		if err != nil {
			return err
		}
		_, err = configs.Create(config)
		if err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

func (c createEtcdBackupConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "create-etcd-backup-configmap",
		Target: "etcd-backup",
	}
}

type etcdBackupSecretCreateOp struct {
	apiserver *cke.Node
	finished  bool
}

// EtcdBackupSecretCreateOp returns an Operator to create etcd-backup certificates.
func EtcdBackupSecretCreateOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupSecretCreateOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupSecretCreateOp) Name() string {
	return "etcd-backup-secret-create"
}

func (o *etcdBackupSecretCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createEtcdBackupSecretCommand{o.apiserver}
}

type createEtcdBackupSecretCommand struct {
	apiserver *cke.Node
}

func (c createEtcdBackupSecretCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	secrets := cs.CoreV1().Secrets("kube-system")
	_, err = secrets.Get(etcdBackupSecretName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		crt, key, err := cke.EtcdCA{}.IssueForBackup(ctx, inf)
		if err != nil {
			return err
		}
		ca, err := inf.Storage().GetCACertificate(ctx, "server")
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		err = secretTemplate.Execute(buf, struct {
			Cert string
			Key  string
			CA   string
		}{
			Cert: base64.StdEncoding.EncodeToString([]byte(crt)),
			Key:  base64.StdEncoding.EncodeToString([]byte(key)),
			CA:   base64.StdEncoding.EncodeToString([]byte(ca)),
		})
		if err != nil {
			return err
		}

		secret := new(corev1.Secret)
		err = k8sYaml.NewYAMLToJSONDecoder(buf).Decode(secret)
		if err != nil {
			return err
		}
		_, err = secrets.Create(secret)
		if err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

func (c createEtcdBackupSecretCommand) Command() cke.Command {
	return cke.Command{
		Name:   "create-etcd-backup-secret",
		Target: "etcd-backup",
	}
}

type etcdBackupConfigMapRemoveOp struct {
	apiserver *cke.Node
	finished  bool
}

// EtcdBackupConfigMapRemoveOp returns an Operator to Remove etcd-backup config.
func EtcdBackupConfigMapRemoveOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupConfigMapRemoveOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupConfigMapRemoveOp) Name() string {
	return "etcd-backup-configmap-remove"
}

func (o *etcdBackupConfigMapRemoveOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return removeEtcdBackupConfigMapCommand{o.apiserver}
}

type removeEtcdBackupConfigMapCommand struct {
	apiserver *cke.Node
}

func (c removeEtcdBackupConfigMapCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	return cs.CoreV1().ConfigMaps("kube-system").Delete(etcdBackupConfigMapName, metav1.NewDeleteOptions(0))
}

func (c removeEtcdBackupConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "remove-etcd-backup-configmap",
		Target: "etcd-backup-configmap",
	}
}

type etcdBackupSecretRemoveOp struct {
	apiserver *cke.Node
	finished  bool
}

// EtcdBackupSecretRemoveOp returns an Operator to Remove etcd-backup certificates.
func EtcdBackupSecretRemoveOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupSecretRemoveOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupSecretRemoveOp) Name() string {
	return "etcd-backup-secret-remove"
}

func (o *etcdBackupSecretRemoveOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return removeEtcdBackupSecretCommand{o.apiserver}
}

type removeEtcdBackupSecretCommand struct {
	apiserver *cke.Node
}

func (c removeEtcdBackupSecretCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	return cs.CoreV1().Secrets("kube-system").Delete(etcdBackupSecretName, metav1.NewDeleteOptions(0))
}

func (c removeEtcdBackupSecretCommand) Command() cke.Command {
	return cke.Command{
		Name:   "remove-etcd-backup-secret",
		Target: "etcd-backup-secret",
	}
}

type etcdBackupCronJobCreateOp struct {
	apiserver  *cke.Node
	etcdBackup cke.EtcdBackup
	finished   bool
}

// EtcdBackupCronJobCreateOp returns an Operator to create etcd-backup cron job.
func EtcdBackupCronJobCreateOp(apiserver *cke.Node, etcdBackup cke.EtcdBackup) cke.Operator {
	return &etcdBackupCronJobCreateOp{
		apiserver:  apiserver,
		etcdBackup: etcdBackup,
	}
}

func (o *etcdBackupCronJobCreateOp) Name() string {
	return "etcd-backup-job-create"
}

func (o *etcdBackupCronJobCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createEtcdBackupCronJobCommand{o.apiserver, o.etcdBackup}
}

type createEtcdBackupCronJobCommand struct {
	apiserver  *cke.Node
	etcdBackup cke.EtcdBackup
}

func (c createEtcdBackupCronJobCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	claims := cs.CoreV1().PersistentVolumeClaims("kube-system")
	_, err = claims.Get(c.etcdBackup.PVCName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	jobs := cs.BatchV1beta1().CronJobs("kube-system")
	_, err = jobs.Get(etcdBackupJobName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		buf := new(bytes.Buffer)
		err := cronJobTemplate.Execute(buf, c.etcdBackup)
		if err != nil {
			return err
		}

		cronJob := new(batchv1beta1.CronJob)
		err = k8sYaml.NewYAMLToJSONDecoder(buf).Decode(cronJob)
		if err != nil {
			return err
		}
		_, err = jobs.Create(cronJob)
		if err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

func (c createEtcdBackupCronJobCommand) Command() cke.Command {
	return cke.Command{
		Name:   "create-etcd-backup-job",
		Target: "etcd-backup",
	}
}

type etcdBackupCronJobUpdateOp struct {
	apiserver  *cke.Node
	etcdBackup cke.EtcdBackup
	finished   bool
}

// EtcdBackupCronJobUpdateOp returns an Operator to Update etcd-backup cron job.
func EtcdBackupCronJobUpdateOp(apiserver *cke.Node, etcdBackup cke.EtcdBackup) cke.Operator {
	return &etcdBackupCronJobUpdateOp{
		apiserver:  apiserver,
		etcdBackup: etcdBackup,
	}
}

func (o *etcdBackupCronJobUpdateOp) Name() string {
	return "etcd-backup-job-update"
}

func (o *etcdBackupCronJobUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateEtcdBackupCronJobCommand{o.apiserver, o.etcdBackup}
}

type updateEtcdBackupCronJobCommand struct {
	apiserver  *cke.Node
	etcdBackup cke.EtcdBackup
}

func (c updateEtcdBackupCronJobCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
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
	err = cronJobTemplate.Execute(buf, c.etcdBackup)
	if err != nil {
		return err
	}

	cronJob := new(batchv1beta1.CronJob)
	err = k8sYaml.NewYAMLToJSONDecoder(buf).Decode(cronJob)
	if err != nil {
		return err
	}

	jobs := cs.BatchV1beta1().CronJobs("kube-system")
	_, err = jobs.Update(cronJob)
	return err
}

func (c updateEtcdBackupCronJobCommand) Command() cke.Command {
	return cke.Command{
		Name:   "update-etcd-backup-job",
		Target: "etcd-backup",
	}
}

type etcdBackupCronJobRemoveOp struct {
	apiserver *cke.Node
	finished  bool
}

// EtcdBackupCronJobRemoveOp returns an Operator to Remove etcd-backup cron job.
func EtcdBackupCronJobRemoveOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupCronJobRemoveOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupCronJobRemoveOp) Name() string {
	return "etcd-backup-job-remove"
}

func (o *etcdBackupCronJobRemoveOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return removeEtcdBackupCronJobCommand{o.apiserver}
}

type removeEtcdBackupCronJobCommand struct {
	apiserver *cke.Node
}

func (c removeEtcdBackupCronJobCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	return cs.BatchV1beta1().CronJobs("kube-system").Delete(etcdBackupJobName, metav1.NewDeleteOptions(60))
}

func (c removeEtcdBackupCronJobCommand) Command() cke.Command {
	return cke.Command{
		Name:   "remove-etcd-backup-job",
		Target: "etcd-backup",
	}
}
