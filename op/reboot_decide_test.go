package op

import (
	"context"
	"testing"
	"time"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func cleanNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

func nodeWithVolumes(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			VolumesInUse: []corev1.UniqueVolumeName{"volume1"},
		},
	}
}

func podOnNode(nodeName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "default"},
		Spec:       corev1.PodSpec{NodeName: nodeName},
	}
}

func TestCheckDrainCompletion(t *testing.T) {
	now := time.Now()
	timedOut := now.Add(-time.Duration(cke.DefaultRebootEvictionTimeoutSeconds+1) * time.Second)

	tests := []struct {
		name          string
		clusterNodes  []string
		objects       []runtime.Object
		entries       []*cke.RebootQueueEntry
		wantCompleted int
		wantTimedout  int
	}{
		{
			name:         "non-draining nodes should not be counted",
			clusterNodes: []string{"10.0.0.1", "10.0.0.2"},
			objects:      []runtime.Object{cleanNode("10.0.0.1"), cleanNode("10.0.0.2")},
			entries: []*cke.RebootQueueEntry{
				{Node: "10.0.0.1", Status: cke.RebootStatusQueued, LastTransitionTime: now},
				{Node: "10.0.0.2", Status: cke.RebootStatusRebooting, LastTransitionTime: now},
			},
			wantCompleted: 0,
			wantTimedout:  0,
		},
		{
			name:         "non cluster member should not be counted",
			clusterNodes: []string{},
			objects:      []runtime.Object{cleanNode("10.0.0.1")},
			entries: []*cke.RebootQueueEntry{
				{Node: "10.0.0.1", Status: cke.RebootStatusDraining, LastTransitionTime: now},
			},
			wantCompleted: 0,
			wantTimedout:  0,
		},
		{
			name:         "draining with no pods and no volumes should be counted as completed",
			clusterNodes: []string{"10.0.0.1"},
			objects:      []runtime.Object{cleanNode("10.0.0.1")},
			entries: []*cke.RebootQueueEntry{
				{Node: "10.0.0.1", Status: cke.RebootStatusDraining, LastTransitionTime: now},
			},
			wantCompleted: 1,
			wantTimedout:  0,
		},
		{
			name:         "draining with remaining pods should not be completed",
			clusterNodes: []string{"10.0.0.1"},
			objects:      []runtime.Object{podOnNode("10.0.0.1")},
			entries: []*cke.RebootQueueEntry{
				{Node: "10.0.0.1", Status: cke.RebootStatusDraining, LastTransitionTime: now},
			},
			wantCompleted: 0,
			wantTimedout:  0,
		},
		{
			name:         "draining with timed out pods should be counted as timed out",
			clusterNodes: []string{"10.0.0.1"},
			objects:      []runtime.Object{podOnNode("10.0.0.1")},
			entries: []*cke.RebootQueueEntry{
				{Node: "10.0.0.1", Status: cke.RebootStatusDraining, LastTransitionTime: timedOut},
			},
			wantCompleted: 0,
			wantTimedout:  1,
		},
		{
			name:         "draining with no pods but volumes in use should not be counted as completed",
			clusterNodes: []string{"10.0.0.1"},
			objects:      []runtime.Object{nodeWithVolumes("10.0.0.1")},
			entries: []*cke.RebootQueueEntry{
				{Node: "10.0.0.1", Status: cke.RebootStatusDraining, LastTransitionTime: now},
			},
			wantCompleted: 0,
			wantTimedout:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := fake.NewClientset(tt.objects...)

			clusterNodes := make([]*cke.Node, len(tt.clusterNodes))
			for i, addr := range tt.clusterNodes {
				clusterNodes[i] = &cke.Node{Address: addr}
			}
			cluster := &cke.Cluster{Nodes: clusterNodes}

			completed, timedout, err := CheckDrainCompletion(context.Background(), &fakeInfrastructure{cs: cs}, &cke.Node{}, cluster, tt.entries)
			if err != nil {
				t.Fatalf("CheckDrainCompletion failed: %v", err)
			}
			if len(completed) != tt.wantCompleted {
				t.Errorf("completed: got %d, want %d", len(completed), tt.wantCompleted)
			}
			if len(timedout) != tt.wantTimedout {
				t.Errorf("timedout: got %d, want %d", len(timedout), tt.wantTimedout)
			}
		})
	}
}
