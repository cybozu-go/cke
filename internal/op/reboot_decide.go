package op

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/cybozu-go/cke"
)

// enumeratePods enumerates Pods on a specified node.
// It calls podHandler for each Pods not owned by Job nor DaemonSet and calls jobPodHandler for each running Pods owned by a Job.
// If those handlers returns error, this function returns the error immediately.
// Note: This function does not distinguish API errors and state evaluation returned from subfunction.
func enumeratePods(ctx context.Context, cs kubernetes.Interface, node string,
	podHandler func(pod *corev1.Pod) error, jobPodHandler func(pod *corev1.Pod) error,
) error {
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

// enumerateOnDeleteDaemonSetPods enumerates Pods on a specified node that are owned by "updateStrategy:OnDelete" DaemonSets.
// It calls podHandler for each target pods.
// If the handler returns error, this function returns the error immediately.
// Note: This function does not distinguish API errors and state evaluation returned from subfunction.
func enumerateOnDeleteDaemonSetPods(ctx context.Context, cs kubernetes.Interface, node string,
	podHandler func(pod *corev1.Pod) error,
) error {
	daemonSets, err := cs.AppsV1().DaemonSets(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ds := range daemonSets.Items {
		if ds.Spec.UpdateStrategy.Type == appsv1.OnDeleteDaemonSetStrategyType {
			labelSelector := metav1.FormatLabelSelector(ds.Spec.Selector)
			pods, err := cs.CoreV1().Pods(ds.Namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				return err
			}

			for _, pod := range pods.Items {
				if pod.Spec.NodeName == node {
					err = podHandler(&pod)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// dryRunEvictOrDeleteNodePod checks eviction or deletion of Pods on the specified Node can proceed.
// It returns an error if a running Pod exists or an eviction of the Pod in protected namespace failed.
func dryRunEvictOrDeleteNodePod(ctx context.Context, cs kubernetes.Interface, node string, protected map[string]bool, protectedJobPods *metav1.LabelSelector) error {
	return doEvictOrDeleteNodePod(ctx, cs, node, protected, protectedJobPods, 0, 0, true)
}

// evictOrDeleteNodePod evicts or delete Pods on the specified Node.
// If a running Job Pod exists and its labels match protectedJobPods, this function returns an error.
func evictOrDeleteNodePod(ctx context.Context, cs kubernetes.Interface, node string, protected map[string]bool, protectedJobPods *metav1.LabelSelector, attempts int, interval time.Duration) error {
	return doEvictOrDeleteNodePod(ctx, cs, node, protected, protectedJobPods, attempts, interval, false)
}

// deleteOnDeleteDaemonSetPod evicts or delete Pods on the specified Node that are owned by "updateStrategy:OnDelete" DaemonSets.
func deleteOnDeleteDaemonSetPod(ctx context.Context, cs kubernetes.Interface, node string) error {
	return doDeleteOnDeleteDaemonSetPod(ctx, cs, node)
}

// doEvictOrDeleteNodePod evicts or delete Pods on the specified Node.
// It first tries eviction.
// If the eviction failed and the Pod's namespace is not protected, it deletes the Pod.
// If the eviction failed and the Pod's namespace is protected, it retries after `interval` interval at most `attempts` times.
// If a running Job Pod exists and its labels do not match protectedJobPods, it deletes the Pod.
// If a running Job Pod exists and its labels match protectedJobPods, this function returns an error.
// If `dry` is true, it performs dry run and `attempts` and `interval` are ignored.
func doEvictOrDeleteNodePod(ctx context.Context, cs kubernetes.Interface, node string, protected map[string]bool, protectedJobPodsSpec *metav1.LabelSelector, attempts int, interval time.Duration, dry bool) error {
	var deleteOptions *metav1.DeleteOptions
	if dry {
		deleteOptions = &metav1.DeleteOptions{
			DryRun: []string{"All"},
		}
	}

	// When protected_job_pods is nil, protects all Job-managed Pods.
	// LabelSelectorAsSelector(nil) returns labels.Nothing(), so defult to labels.Everything() here.
	protectedJobPodSelector := labels.Everything()
	if protectedJobPodsSpec != nil {
		var err error
		protectedJobPodSelector, err = metav1.LabelSelectorAsSelector(protectedJobPodsSpec)
		if err != nil {
			return err
		}
	}

	return enumeratePods(ctx, cs, node, func(pod *corev1.Pod) error {
		if dry && !protected[pod.Namespace] {
			// in case of dry-run for Pods in non-protected namespace,
			// return immediately because its "eviction or deletion" never fails
			log.Info("skip evicting pod because its namespace is not protected", map[string]any{
				"namespace": pod.Namespace,
				"name":      pod.Name,
				"dry":       dry, // for consistency
			})
			return nil
		}

		evictCount := 0
	EVICT:
		log.Info("start evicting pod", map[string]any{
			"namespace": pod.Namespace,
			"name":      pod.Name,
			"dry":       dry,
		})
		err := cs.CoreV1().Pods(pod.Namespace).EvictV1(ctx, &policyv1.Eviction{
			ObjectMeta:    metav1.ObjectMeta{Name: pod.Name, Namespace: pod.Namespace},
			DeleteOptions: deleteOptions,
		})
		evictCount++
		switch {
		case err == nil:
			log.Info("evicted pod", map[string]any{
				"namespace": pod.Namespace,
				"name":      pod.Name,
				"dry":       dry,
			})
		case apierrors.IsNotFound(err):
			// already evicted or deleted.
		case !apierrors.IsTooManyRequests(err):
			// not a PDB related error
			return fmt.Errorf("failed to evict pod %s/%s: %w", pod.Namespace, pod.Name, err)
		case !protected[pod.Namespace]:
			// not dry here
			log.Warn("failed to evict non-protected pod due to PDB", map[string]any{
				"namespace": pod.Namespace,
				"name":      pod.Name,
				"dry":       dry, // for consistency (same for below)
				log.FnError: err,
			})
			err := cs.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			log.Warn("deleted non-protected pod", map[string]any{
				"namespace": pod.Namespace,
				"name":      pod.Name,
				"dry":       dry,
			})
		default:
			// in case of dry-run, do not retry.
			if !dry && evictCount < attempts {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(interval):
				}
				log.Info("retry eviction of pod", map[string]any{
					"namespace": pod.Namespace,
					"name":      pod.Name,
					"dry":       dry,
				})
				goto EVICT
			}
			return fmt.Errorf("failed to evict pod %s/%s due to PDB: %w", pod.Namespace, pod.Name, err)
		}
		return nil
	}, func(pod *corev1.Pod) error {
		if protectedJobPodSelector.Matches(labels.Set(pod.Labels)) {
			return fmt.Errorf("protected job-managed pod exists: %s/%s, phase=%s", pod.Namespace, pod.Name, pod.Status.Phase)
		}
		if dry {
			log.Info("skip actually deleting job-managed pod in dry-run (labels do not match protected_job_pods)", map[string]any{
				"namespace": pod.Namespace,
				"name":      pod.Name,
				"dry":       dry,
			})
			return nil
		}
		log.Info("start deleting job-managed pod", map[string]any{
			"namespace": pod.Namespace,
			"name":      pod.Name,
		})
		err := cs.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete job-managed pod %s/%s: %w", pod.Namespace, pod.Name, err)
		}
		log.Info("deleted job-managed pod", map[string]any{
			"namespace": pod.Namespace,
			"name":      pod.Name,
		})
		return nil
	})
}

// doDeleteOnDeleteDaemonSetPod deletes 'OnDelete' DaemonSet pods on the specified Node.
func doDeleteOnDeleteDaemonSetPod(ctx context.Context, cs kubernetes.Interface, node string) error {
	return enumerateOnDeleteDaemonSetPods(ctx, cs, node, func(pod *corev1.Pod) error {
		err := cs.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		log.Info("deleted daemonset pod", map[string]any{
			"namespace": pod.Namespace,
			"name":      pod.Name,
		})
		return nil
	})
}

// checkPodDeletion checks whether the evicted or deleted Pods are eventually deleted.
// If those pods still exist, this function returns an error.
func checkPodDeletion(ctx context.Context, cs kubernetes.Interface, node string) error {
	return enumeratePods(ctx, cs, node, func(pod *corev1.Pod) error {
		return fmt.Errorf("pod exists: %s/%s, phase=%s", pod.Namespace, pod.Name, pod.Status.Phase)
	}, func(pod *corev1.Pod) error {
		// This should not happen... or rare case?
		return fmt.Errorf("job-managed pod exists: %s/%s, phase=%s", pod.Namespace, pod.Name, pod.Status.Phase)
	})
}

// chooseRebootCandidates chooses next targets to drain attempt.
// For now, this function does not check "drainability".
func ChooseRebootCandidates(c *cke.Cluster, apiServers map[string]bool, rqEntries []*cke.RebootQueueEntry) []*cke.RebootQueueEntry {
	maxConcurrentReboots := cke.DefaultMaxConcurrentReboots
	if c.Reboot.MaxConcurrentReboots != nil {
		maxConcurrentReboots = *c.Reboot.MaxConcurrentReboots
	}
	now := time.Now()

	apiServerInProgress := false
	var apiServerDrainable *cke.RebootQueueEntry
	workerInProgress := []*cke.RebootQueueEntry{}
	workerDrainable := []*cke.RebootQueueEntry{}
	for _, entry := range rqEntries {
		if !entry.ClusterMember(c) {
			continue
		}
		switch entry.Status {
		case cke.RebootStatusDraining, cke.RebootStatusRebooting:
			if apiServers[entry.Node] {
				apiServerInProgress = true
			} else {
				workerInProgress = append(workerInProgress, entry)
			}
		case cke.RebootStatusQueued:
			if entry.DrainBackOffExpire.After(now) {
				continue
			}
			if apiServers[entry.Node] {
				if apiServerDrainable == nil {
					apiServerDrainable = entry
				}
			} else {
				workerDrainable = append(workerDrainable, entry)
			}
		}
	}

	// rules:
	//   - API Servers are rebooted one by one.
	//       - It is VERY important.
	//   - API Servers are rebooted with lower priority than worker nodes.
	//   - API Servers are not rebooted simultaneously with worker nodes.
	if apiServerInProgress {
		return nil
	}
	if len(workerInProgress) == 0 && len(workerDrainable) == 0 {
		if apiServerDrainable != nil {
			return []*cke.RebootQueueEntry{apiServerDrainable}
		} else {
			return nil
		}
	}
	if len(workerInProgress) < maxConcurrentReboots && len(workerDrainable) > 0 {
		return workerDrainable[:1]
	} else {
		return nil
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

		errPodDeletion := checkPodDeletion(ctx, cs, entry.Node)
		if errPodDeletion == nil {
			volumesInUse, errVolumeInUse := checkVolumesInUse(ctx, cs, entry.Node)
			if errVolumeInUse != nil {
				return nil, nil, errVolumeInUse
			}
			if !volumesInUse {
				completed = append(completed, entry)
				continue
			}
		}
		if entry.LastTransitionTime.Before(t) {
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
		case entry.Status == cke.RebootStatusRebooting && rebootCompleted(ctx, c, entry):
		default:
			continue
		}

		dequeued = append(dequeued, entry)
	}

	return dequeued
}

func CheckRebootCancelled(ctx context.Context, c *cke.Cluster, rqEntries []*cke.RebootQueueEntry) []*cke.RebootQueueEntry {
	cancelled := []*cke.RebootQueueEntry{}

	for _, entry := range rqEntries {
		if entry.Status == cke.RebootStatusCancelled {
			cancelled = append(cancelled, entry)
		}
	}

	return cancelled
}

func rebootCompleted(ctx context.Context, c *cke.Cluster, entry *cke.RebootQueueEntry) bool {
	var timeout int
	if c.Reboot.CommandTimeoutSeconds != nil {
		timeout = *c.Reboot.CommandTimeoutSeconds
	}

	stdout, err := runCommand(ctx, timeout, fnTypeReboot, append(c.Reboot.BootCheckCommand, entry.Node, strconv.FormatInt(entry.LastTransitionTime.Unix(), 10)))
	if err != nil {
		log.Warn("failed to check boot", map[string]any{
			log.FnType:  fnTypeReboot,
			log.FnError: err,
			"index":     entry.Index,
			"node":      entry.Node,
		})
		return false
	}
	return strings.TrimSpace(stdout) == "true"
}

func checkVolumesInUse(ctx context.Context, cs kubernetes.Interface, node string) (bool, error) {
	nodeObj, err := cs.CoreV1().Nodes().Get(ctx, node, metav1.GetOptions{})
	if err != nil {
		return true, err
	}
	if len(nodeObj.Status.VolumesInUse) > 0 {
		return true, nil
	}
	return false, nil
}
