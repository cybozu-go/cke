package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	watchtools "k8s.io/client-go/tools/watch"
)

var snapshotListTemplate = template.Must(template.New("").Parse(`
metadata:
  name: ckecli-etcd-snapshot-list
  namespace: kube-system
spec:
  template:
    metadata:
      labels:
        job-name: ckecli-etcd-snapshot-list
    spec:
      containers:
        - command:
            - ls
            - /etcd-backup
          image: quay.io/cybozu/etcd:3.3.9-4
          imagePullPolicy: IfNotPresent
          name: ckecli-etcd-snapshot-list
          volumeMounts:
            - mountPath: /etcd-backup
              name: etcd-backup
      restartPolicy: Never
      volumes:
        - name: etcd-backup
          persistentVolumeClaim:
            claimName: {{ .PVCName }}
`))

var etcdSnapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "List etcd snapshots.",
	Long: `List etcd snapshot file names.
The filenames are usually contained created date string.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			cfg, err := storage.GetCluster(ctx)
			if err != nil {
				return err
			}
			if !cfg.EtcdBackup.Enabled {
				return errors.New("etcd backup is disabled")
			}
			var controlPlane *cke.Node
			for _, node := range cfg.Nodes {
				if node.ControlPlane {
					controlPlane = node
					break
				}
			}
			if controlPlane == nil {
				return errors.New("control plane not found")
			}
			cs, err := inf.K8sClient(ctx, controlPlane)
			if err != nil {
				return err
			}
			jobs := cs.BatchV1().Jobs("kube-system")

			buf := new(bytes.Buffer)
			err = snapshotListTemplate.Execute(buf, struct {
				PVCName string
			}{
				PVCName: cfg.EtcdBackup.PVCName,
			})
			if err != nil {
				return err
			}

			job := new(batchv1.Job)
			err = k8sYaml.NewYAMLToJSONDecoder(strings.NewReader(buf.String())).Decode(job)
			if err != nil {
				return err
			}
			job, err = jobs.Create(job)
			if err != nil {
				return err
			}
			defer jobs.Delete("ckecli-etcd-snapshot-list", &metav1.DeleteOptions{})

			pods := cs.CoreV1().Pods("kube-system")
			w, err := pods.Watch(metav1.ListOptions{
				LabelSelector:   "job-name=ckecli-etcd-snapshot-list",
				ResourceVersion: job.ResourceVersion,
			})
			if err != nil {
				return err
			}
			var result *corev1.Pod
			ev, err := watchtools.UntilWithoutRetry(ctx, w, func(ev watch.Event) (bool, error) {
				return podCompleted(ev)
			})
			if ev != nil {
				result = ev.Object.(*corev1.Pod)
			}
			if err != nil {
				return err
			}
			defer pods.Delete(result.Name, &metav1.DeleteOptions{})

			req := cs.CoreV1().RESTClient().Get().
				Namespace("kube-system").
				Name(result.Name).
				Resource("pods").
				SubResource("log")
			readCloser, err := req.Stream()
			if err != nil {
				return err
			}
			defer readCloser.Close()
			_, err = io.Copy(os.Stdout, readCloser)
			return err
		})
		well.Stop()
		return well.Wait()
	},
}

func podCompleted(event watch.Event) (bool, error) {
	switch event.Type {
	case watch.Deleted:
		return false, errors.New("target pod is deleted")
	}
	switch t := event.Object.(type) {
	case *corev1.Pod:
		switch t.Status.Phase {
		case corev1.PodFailed, corev1.PodSucceeded:
			return true, nil
		}
	}
	return false, nil
}

func init() {
	etcdSnapshotCmd.AddCommand(etcdSnapshotListCmd)
}
