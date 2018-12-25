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
  containers:
    - command:
        - ls
        - /etcdbackup
      image: quay.io/cybozu/etcd:3.3.9-4
      imagePullPolicy: IfNotPresent
      name: ckecli-etcd-snapshot-list
      volumeMounts:
        - mountPath: /etcdbackup
          name: etcdbackup
  restartPolicy: Never
  volumes:
    - name: etcdbackup
      persistentVolumeClaim:
        claimName: {{ .PVCName }}
`))

func snapshotList(ctx context.Context) error {
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
	pods := cs.CoreV1().Pods("kube-system")

	buf := new(bytes.Buffer)
	err = snapshotListTemplate.Execute(buf, struct {
		PVCName string
	}{
		PVCName: cfg.EtcdBackup.PVCName,
	})
	if err != nil {
		return err
	}

	pod := new(corev1.Pod)
	err = k8sYaml.NewYAMLToJSONDecoder(strings.NewReader(buf.String())).Decode(pod)
	if err != nil {
		return err
	}
	pod, err = pods.Create(pod)
	if err != nil {
		return err
	}
	defer pods.Delete(pod.Name, &metav1.DeleteOptions{})

	w, err := pods.Watch(metav1.ListOptions{
		FieldSelector:   "metadata.name=ckecli-etcd-snapshot-list",
		ResourceVersion: pod.ResourceVersion,
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
}

var etcdSnapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "List etcd snapshots",
	Long: `List etcd snapshot file names.

The filenames are usually contained created date string.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(snapshotList)
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
