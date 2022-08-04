package k8s

import (
	"reflect"
	"testing"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	schedulerv1beta3 "k8s.io/kube-scheduler/config/v1beta3"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/pointer"
)

func TestGenerateSchedulerConfiguration(t *testing.T) {
	t.Parallel()

	cfg := &unstructured.Unstructured{}
	cfg.SetGroupVersionKind(schedulerv1beta3.SchemeGroupVersion.WithKind("KubeSchedulerConfiguration"))
	cfg.Object["leaderElection"] = map[string]interface{}{
		"leaderElect": false,
	}
	cfg.Object["podMaxBackoffSeconds"] = 100

	input := cke.SchedulerParams{
		Config: cfg,
	}

	expected := &schedulerv1beta3.KubeSchedulerConfiguration{}
	expected.LeaderElection.LeaderElect = pointer.BoolPtr(true)
	expected.ClientConnection.Kubeconfig = "/etc/kubernetes/scheduler/kubeconfig"
	expected.PodMaxBackoffSeconds = pointer.Int64Ptr(100)

	conf := GenerateSchedulerConfiguration(input)
	if !reflect.DeepEqual(conf, expected) {
		t.Errorf("GenerateSchedulerConfiguration() generated unexpected result:\n%s", cmp.Diff(conf, expected))
	}
}

func TestGenerateKubeletConfiguration(t *testing.T) {
	t.Parallel()

	baseExpected := &kubeletv1beta1.KubeletConfiguration{
		ClusterDomain:         "cluster.local",
		RuntimeRequestTimeout: metav1.Duration{Duration: 15 * time.Minute},
		HealthzBindAddress:    "0.0.0.0",
		VolumePluginDir:       "/opt/volume/bin",
		TLSCertFile:           "/etc/kubernetes/pki/kubelet.crt",
		TLSPrivateKeyFile:     "/etc/kubernetes/pki/kubelet.key",
		Authentication: kubeletv1beta1.KubeletAuthentication{
			X509:    kubeletv1beta1.KubeletX509Authentication{ClientCAFile: "/etc/kubernetes/pki/ca.crt"},
			Webhook: kubeletv1beta1.KubeletWebhookAuthentication{Enabled: pointer.BoolPtr(true)},
		},
		Authorization: kubeletv1beta1.KubeletAuthorization{
			Mode: kubeletv1beta1.KubeletAuthorizationModeWebhook,
		},
		ClusterDNS: []string{"1.2.3.4"},
	}

	expected := baseExpected.DeepCopy()
	expected.FailSwapOn = pointer.BoolPtr(false)
	expected.ContainerLogMaxSize = "100Mi"
	expected.CgroupDriver = "systemd"

	expected2 := expected.DeepCopy()
	expected2.CgroupDriver = ""

	cfg := &unstructured.Unstructured{}
	cfg.SetGroupVersionKind(kubeletv1beta1.SchemeGroupVersion.WithKind("KubeletConfiguration"))
	cfg.Object["failSwapOn"] = false
	cfg.Object["containerLogMaxSize"] = "100Mi"
	cfg.Object["cgroupDriver"] = "systemd"

	cases := []struct {
		Name     string
		Input    cke.KubeletParams
		Running  *kubeletv1beta1.KubeletConfiguration
		Expected *kubeletv1beta1.KubeletConfiguration
	}{
		{
			Name:     "base",
			Input:    cke.KubeletParams{},
			Expected: baseExpected,
		},
		{
			Name: "with config",
			Input: cke.KubeletParams{
				Config: cfg,
			},
			Expected: expected,
		},
		{
			Name: "with running config",
			Input: cke.KubeletParams{
				Config: cfg,
			},
			Running:  &kubeletv1beta1.KubeletConfiguration{},
			Expected: expected2,
		},
	}

	for _, c := range cases {
		conf := GenerateKubeletConfiguration(c.Input, "1.2.3.4", c.Running)
		if !cmp.Equal(conf, c.Expected) {
			t.Errorf("case %q: GenerateKubeletConfiguration() generated unexpected result:\n%s", c.Name, cmp.Diff(conf, c.Expected))
		}
	}
}
