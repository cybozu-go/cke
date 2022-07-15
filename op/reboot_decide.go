package op

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

// enumeratePods enumerates Pods on a specified node.
// It calls podHandler for each Pods not owned by Job nor DaemonSet and calls jobPodHandler for each running Pods owned by a Job.
// If those handlers returns error, this function returns the error immediately.
func enumeratePods(ctx context.Context, cs *kubernetes.Clientset, node string,
	podHandler func(pod *corev1.Pod) error, jobPodHandler func(pod *corev1.Pod) error) error {

	podList, err := cs.CoreV1().Pods(corev1.NamespaceAll).List(ctx, metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node}).String(),
	})
	if err != nil {
		return err
	}

	for i := range podList.Items {
		pod := &podList.Items[i]
		owner := metav1.GetControllerOf(pod)
		if owner != nil {
			switch owner.Kind {
			case "DaemonSet":
				continue
			case "Job":
				switch pod.Status.Phase {
				case corev1.PodPending:
				case corev1.PodSucceeded:
				case corev1.PodFailed:
				default:
					err = jobPodHandler(pod)
					if err != nil {
						return err
					}
				}
				continue
			}
		}
		err = podHandler(pod)
		if err != nil {
			return err
		}
	}

	return nil
}

// checkJobPodNotExist checks running Pods on the specified Node.
// It returns an error if a running Pod exists.
func checkJobPodNotExist(ctx context.Context, cs *kubernetes.Clientset, node string) error {
	return enumeratePods(ctx, cs, node, func(_ *corev1.Pod) error {
		return nil
	}, func(pod *corev1.Pod) error {
		return fmt.Errorf("job-managed pod exists: %s/%s, phase=%s", pod.Namespace, pod.Name, pod.Status.Phase)
	})
}

// evictOrDeleteNodePod evicts or delete Pods on the specified Node.
// It first tries eviction. If the eviction failed and the Pod's namespace is not protected, it deletes the Pod.
// If a running Job Pod exists, this function returns an error.
func evictOrDeleteNodePod(ctx context.Context, cs *kubernetes.Clientset, node string, protected map[string]bool) error {
	return enumeratePods(ctx, cs, node, func(pod *corev1.Pod) error {
		err := cs.CoreV1().Pods(pod.Namespace).EvictV1(ctx, &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{Name: pod.Name, Namespace: pod.Namespace},
		})
		switch {
		case err == nil:
			log.Info("start evicting pod", map[string]interface{}{
				"namespace": pod.Namespace,
				"name":      pod.Name,
			})
		case apierrors.IsNotFound(err):
			// already evicted or deleted.
		case !apierrors.IsTooManyRequests(err):
			// not a PDB related error
			return fmt.Errorf("failed to evict pod %s/%s: %w", pod.Namespace, pod.Name, err)
		case !protected[pod.Namespace]:
			log.Warn("failed to evict non-protected pod due to PDB", map[string]interface{}{
				"namespace": pod.Namespace,
				"name":      pod.Name,
				log.FnError: err,
			})
			err := cs.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			log.Warn("deleted non-protected pod", map[string]interface{}{
				"namespace": pod.Namespace,
				"name":      pod.Name,
			})
		default:
			return fmt.Errorf("failed to evict pod %s/%s due to PDB: %w", pod.Namespace, pod.Name, err)
		}
		return nil
	}, func(pod *corev1.Pod) error {
		return fmt.Errorf("job-managed pod exists: %s/%s, phase=%s", pod.Namespace, pod.Name, pod.Status.Phase)
	})
}

// checkPodDeletion checks whether the evicted or deleted Pods are eventually deleted.
// If those pods still exist, this function returns an error.
func checkPodDeletion(ctx context.Context, cs *kubernetes.Clientset, node string) error {
	return enumeratePods(ctx, cs, node, func(pod *corev1.Pod) error {
		return fmt.Errorf("pod exists: %s/%s, phase=%s", pod.Namespace, pod.Name, pod.Status.Phase)
	}, func(pod *corev1.Pod) error {
		// This should not happen... or rare case?
		return fmt.Errorf("job-managed pod exists: %s/%s, phase=%s", pod.Namespace, pod.Name, pod.Status.Phase)
	})
}

// chooseDrainedNodes chooses nodes to be newly drained.
// For now, this function does not check "drainability".
func ChooseDrainedNodes(c *cke.Cluster, apiServers map[string]bool, rqEntries []*cke.RebootQueueEntry) []*cke.RebootQueueEntry {
	maxConcurrentReboots := cke.DefaultMaxConcurrentReboots
	if c.Reboot.MaxConcurrentReboots != nil {
		maxConcurrentReboots = *c.Reboot.MaxConcurrentReboots
	}
	now := time.Now()

	alreadyDrained := []*cke.RebootQueueEntry{}
	apiServerAlreadyDrained := false
	canBeDrained := []*cke.RebootQueueEntry{}
	var apiServerCanBeDrained *cke.RebootQueueEntry
	for _, entry := range rqEntries {
		if !entry.ClusterMember(c) {
			continue
		}
		switch entry.Status {
		case cke.RebootStatusDraining, cke.RebootStatusRebooting:
			alreadyDrained = append(alreadyDrained, entry)
			if apiServers[entry.Node] {
				apiServerAlreadyDrained = true
			}
		case cke.RebootStatusQueued:
			if entry.DrainBackOffExpire.After(now) {
				continue
			}
			canBeDrained = append(canBeDrained, entry)
			if apiServerCanBeDrained == nil && apiServers[entry.Node] {
				apiServerCanBeDrained = entry
			}
		}
	}

	// rules:
	//   - API Servers are rebooted one by one.
	//       - It is VERY important.
	//   - API Servers are rebooted with higher priority than worker nodes.
	//   - API Servers are not rebooted simultaneously with worker nodes.
	if apiServerCanBeDrained != nil {
		if len(alreadyDrained) == 0 {
			return []*cke.RebootQueueEntry{apiServerCanBeDrained}
		} else {
			return nil
		}
	}
	if apiServerAlreadyDrained {
		return nil
	}
	if len(alreadyDrained) >= maxConcurrentReboots {
		return nil
	} else if len(alreadyDrained)+len(canBeDrained) <= maxConcurrentReboots {
		return canBeDrained
	} else {
		return canBeDrained[:maxConcurrentReboots-len(alreadyDrained)]
	}
}

func CheckDrainCompletion(ctx context.Context, inf cke.Infrastructure, apiserver *cke.Node, c *cke.Cluster, rqEntries []*cke.RebootQueueEntry) ([]*cke.RebootQueueEntry, []*cke.RebootQueueEntry, error) {
	evictionTimeoutSeconds := cke.DefaultRebootEvictionTimeoutSeconds
	if c.Reboot.EvictionTimeoutSeconds != nil {
		evictionTimeoutSeconds = *c.Reboot.EvictionTimeoutSeconds
	}

	cs, err := inf.K8sClient(ctx, apiserver)
	if err != nil {
		return nil, nil, err
	}

	t := time.Now().Add(time.Duration(-evictionTimeoutSeconds) * time.Second)

	var completed []*cke.RebootQueueEntry
	var timedout []*cke.RebootQueueEntry
	for _, entry := range rqEntries {
		if !entry.ClusterMember(c) {
			continue
		}
		if entry.Status != cke.RebootStatusDraining {
			continue
		}

		err := checkPodDeletion(ctx, cs, entry.Node)
		if err == nil {
			completed = append(completed, entry)
		} else if entry.LastTransitionTime.Before(t) {
			timedout = append(timedout, entry)
		}
	}

	return completed, timedout, nil
}

func CheckRebootDequeue(ctx context.Context, c *cke.Cluster, rqEntries []*cke.RebootQueueEntry) []*cke.RebootQueueEntry {
	dequeued := []*cke.RebootQueueEntry{}

	for _, entry := range rqEntries {
		switch {
		case !entry.ClusterMember(c):
		case entry.Status == cke.RebootStatusCancelled:
		case entry.Status == cke.RebootStatusRebooting && rebootCompleted(ctx, c, entry):
		default:
			continue
		}

		dequeued = append(dequeued, entry)
	}

	return dequeued
}

func rebootCompleted(ctx context.Context, c *cke.Cluster, entry *cke.RebootQueueEntry) bool {
	if c.Reboot.CommandTimeoutSeconds != nil && *c.Reboot.CommandTimeoutSeconds != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(*c.Reboot.CommandTimeoutSeconds))
		defer cancel()
	}

	result := false

	env := well.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		args := append(c.Reboot.BootCheckCommand[1:], entry.Node, fmt.Sprintf("%d", entry.LastTransitionTime.Unix()))
		command := well.CommandContext(ctx, c.Reboot.BootCheckCommand[0], args...)
		stdout, err := command.Output()
		if err != nil {
			return err
		}

		if strings.TrimSuffix(string(stdout), "\n") == "true" {
			result = true
		}
		return nil
	})
	env.Stop()
	err := env.Wait()
	if err != nil {
		log.Warn("failed to check boot", map[string]interface{}{
			"name": entry.Node,
		})
		return false
	}
	return result
}
