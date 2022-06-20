package op

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/metrics"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const drainBackOffBaseSeconds = 60

type rebootDrainStartOp struct {
	finished bool

	entries   []*cke.RebootQueueEntry
	config    *cke.Reboot
	apiserver *cke.Node

	mu          sync.Mutex
	failedNodes []string
}

func RebootDrainStartOp(apiserver *cke.Node, entries []*cke.RebootQueueEntry, config *cke.Reboot) cke.InfoOperator {
	return &rebootDrainStartOp{
		entries:   entries,
		config:    config,
		apiserver: apiserver,
	}
}

type rebootDrainStartCommand struct {
	entries             []*cke.RebootQueueEntry
	protectedNamespaces *metav1.LabelSelector
	apiserver           *cke.Node

	notifyFailedNode func(string)
}

func (o *rebootDrainStartOp) Name() string {
	return "reboot-drain-start"
}

func (o *rebootDrainStartOp) notifyFailedNode(node string) {
	o.mu.Lock()
	o.failedNodes = append(o.failedNodes, node)
	o.mu.Unlock()
}

func (o *rebootDrainStartOp) Targets() []string {
	ipAddresses := make([]string, len(o.entries))
	for i, entry := range o.entries {
		ipAddresses[i] = entry.Node
	}
	return ipAddresses
}

func (o *rebootDrainStartOp) Info() string {
	if len(o.failedNodes) == 0 {
		return ""
	}
	return fmt.Sprintf("failed to drain some nodes: %v", o.failedNodes)
}

func (o *rebootDrainStartOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true

	return rebootDrainStartCommand{
		entries:             o.entries,
		protectedNamespaces: o.config.ProtectedNamespaces,
		apiserver:           o.apiserver,
		notifyFailedNode:    o.notifyFailedNode,
	}
}

func (c rebootDrainStartCommand) Command() cke.Command {
	ipAddresses := make([]string, len(c.entries))
	for i, entry := range c.entries {
		ipAddresses[i] = entry.Node
	}
	return cke.Command{
		Name:   "rebootDrainStartCommand",
		Target: strings.Join(ipAddresses, ","),
	}
}

func (c rebootDrainStartCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	nodesAPI := cs.CoreV1().Nodes()

	protected, err := listProtectedNamespaces(ctx, cs, c.protectedNamespaces)
	if err != nil {
		return err
	}

	// Draining should be done sequentially.
	// Parallel draining is relatively prone to deadlock.

	// first, cordon all nodes
	evictNodes := []*cke.RebootQueueEntry{}
	for _, entry := range c.entries {
		err := func() error {
			entry.Status = cke.RebootStatusDraining
			entry.LastTransitionTime = time.Now().Truncate(time.Second).UTC()
			err = inf.Storage().UpdateRebootsEntry(ctx, entry)
			if err != nil {
				return err
			}

			err = checkJobPodNotExist(ctx, cs, entry.Node)
			if err != nil {
				return err
			}

			_, err = nodesAPI.Patch(ctx, entry.Node, types.StrategicMergePatchType, []byte(`
{
	"metadata":{"annotations":{"`+CKEAnnotationReboot+`": "true"}},
	"spec":{"unschedulable": true}
}
`), metav1.PatchOptions{})
			if err != nil {
				return fmt.Errorf("failed to cordon node %s: %v", entry.Node, err)
			}

			return nil
		}()
		if err != nil {
			c.notifyFailedNode(entry.Node)
			err = drainBackOff(ctx, inf, entry, err)
			if err != nil {
				return err
			}
		} else {
			evictNodes = append(evictNodes, entry)
		}
	}

	// next, evict pods on each node
	for _, entry := range evictNodes {
		err := evictOrDeleteNodePod(ctx, cs, entry.Node, protected)
		if err != nil {
			c.notifyFailedNode(entry.Node)
			err = drainBackOff(ctx, inf, entry, err)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//

type rebootRebootOp struct {
	finished bool

	entries []*cke.RebootQueueEntry
	config  *cke.Reboot

	mu          sync.Mutex
	failedNodes []string
}

type rebootRebootCommand struct {
	entries        []*cke.RebootQueueEntry
	command        []string
	timeoutSeconds *int

	notifyFailedNode func(string)
}

func (o *rebootRebootOp) notifyFailedNode(node string) {
	o.mu.Lock()
	o.failedNodes = append(o.failedNodes, node)
	o.mu.Unlock()
}

// RebootOp returns an Operator to reboot nodes.
func RebootRebootOp(apiserver *cke.Node, entries []*cke.RebootQueueEntry, config *cke.Reboot) cke.InfoOperator {
	return &rebootRebootOp{
		entries: entries,
		config:  config,
	}
}

func (o *rebootRebootOp) Name() string {
	return "reboot-reboot"
}

func (o *rebootRebootOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true

	return rebootRebootCommand{
		entries:          o.entries,
		command:          o.config.RebootCommand,
		timeoutSeconds:   o.config.CommandTimeoutSeconds,
		notifyFailedNode: o.notifyFailedNode,
	}
}

func (o *rebootRebootOp) Targets() []string {
	ipAddresses := make([]string, len(o.entries))
	for i, entry := range o.entries {
		ipAddresses[i] = entry.Node
	}
	return ipAddresses
}

func (o *rebootRebootOp) Info() string {
	if len(o.failedNodes) == 0 {
		return ""
	}
	return fmt.Sprintf("failed to reboot some nodes: %v", o.failedNodes)
}

func (c rebootRebootCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	if c.timeoutSeconds != nil && *c.timeoutSeconds != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(*c.timeoutSeconds))
		defer cancel()
	}

	env := well.NewEnvironment(ctx)
	for _, entry := range c.entries {
		entry := entry // save loop variable for goroutine

		env.Go(func(ctx context.Context) error {
			entry.Status = cke.RebootStatusRebooting
			entry.LastTransitionTime = time.Now().Truncate(time.Second).UTC()
			err := inf.Storage().UpdateRebootsEntry(ctx, entry)
			if err != nil {
				return err
			}

			args := append(c.command[1:], entry.Node)
			command := well.CommandContext(ctx, c.command[0], args...)
			err = command.Run()
			if err != nil {
				c.notifyFailedNode(entry.Node)
				log.Warn("failed on rebooting node", map[string]interface{}{
					log.FnError: err,
					"node":      entry.Node,
				})
			}
			return nil
		})
	}
	env.Stop()
	return env.Wait()
}

func (c rebootRebootCommand) Command() cke.Command {
	ipAddresses := make([]string, len(c.entries))
	for i, entry := range c.entries {
		ipAddresses[i] = entry.Node
	}
	return cke.Command{
		Name:   "rebootRebootCommand",
		Target: strings.Join(ipAddresses, ","),
	}
}

//

type rebootDrainTimeoutOp struct {
	finished bool

	entries []*cke.RebootQueueEntry
}

func RebootDrainTimeoutOp(entries []*cke.RebootQueueEntry) cke.Operator {
	return &rebootDrainTimeoutOp{
		entries: entries,
	}
}

type rebootDrainTimeoutCommand struct {
	entries []*cke.RebootQueueEntry
}

func (o *rebootDrainTimeoutOp) Name() string {
	return "reboot-drain-timeout"
}

func (o *rebootDrainTimeoutOp) Targets() []string {
	ipAddresses := make([]string, len(o.entries))
	for i, entry := range o.entries {
		ipAddresses[i] = entry.Node
	}
	return ipAddresses
}

func (o *rebootDrainTimeoutOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true

	return rebootDrainTimeoutCommand{
		entries: o.entries,
	}
}

func (c rebootDrainTimeoutCommand) Command() cke.Command {
	ipAddresses := make([]string, len(c.entries))
	for i, entry := range c.entries {
		ipAddresses[i] = entry.Node
	}
	return cke.Command{
		Name:   "rebootDrainTimeoutCommand",
		Target: strings.Join(ipAddresses, ","),
	}
}

func (c rebootDrainTimeoutCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	for _, entry := range c.entries {
		err := drainBackOff(ctx, inf, entry, fmt.Errorf("drain timed out: %s", entry.Node))
		if err != nil {
			return err
		}
	}

	return nil
}

//

type rebootUncordonOp struct {
	apiserver *cke.Node
	nodeNames []string
	finished  bool
}

// RebootUncordonOp returns an Operator to uncordon nodes.
func RebootUncordonOp(apiserver *cke.Node, nodeNames []string) cke.Operator {
	return &rebootUncordonOp{
		apiserver: apiserver,
		nodeNames: nodeNames,
	}
}

func (o *rebootUncordonOp) Name() string {
	return "reboot-uncordon"
}

func (o *rebootUncordonOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return uncordonCommand{
		apiserver: o.apiserver,
		nodeNames: o.nodeNames,
	}
}

func (o *rebootUncordonOp) Targets() []string {
	return o.nodeNames
}

type uncordonCommand struct {
	apiserver *cke.Node
	nodeNames []string
}

func (c uncordonCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	nodesAPI := cs.CoreV1().Nodes()

	for _, name := range c.nodeNames {
		_, err = nodesAPI.Patch(ctx, name, types.StrategicMergePatchType, []byte(`
{
	"metadata":{"annotations":{"`+CKEAnnotationReboot+`": null}},
	"spec":{"unschedulable": null}
}
`), metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("failed to uncordon node %s: %v", name, err)
		}
	}
	return nil
}

func (c uncordonCommand) Command() cke.Command {
	return cke.Command{
		Name:   "uncordonCommand",
		Target: strings.Join(c.nodeNames, ","),
	}
}

//

type rebootDequeueOp struct {
	finished bool

	entries []*cke.RebootQueueEntry
}

// RebootDequeueOp returns an Operator to dequeue a reboot entry.
func RebootDequeueOp(entries []*cke.RebootQueueEntry) cke.Operator {
	return &rebootDequeueOp{
		entries: entries,
	}
}

func (o *rebootDequeueOp) Name() string {
	return "reboot-dequeue"
}

func (o *rebootDequeueOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return rebootDequeueCommand{
		entries: o.entries,
	}
}

func (o *rebootDequeueOp) Targets() []string {
	ipAddresses := make([]string, len(o.entries))
	for i, entry := range o.entries {
		ipAddresses[i] = entry.Node
	}
	return ipAddresses
}

type rebootDequeueCommand struct {
	entries []*cke.RebootQueueEntry
}

func (c rebootDequeueCommand) Run(ctx context.Context, inf cke.Infrastructure, leaderKey string) error {
	for _, entry := range c.entries {
		err := inf.Storage().DeleteRebootsEntry(ctx, leaderKey, entry.Index)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c rebootDequeueCommand) Command() cke.Command {
	ipAddresses := make([]string, len(c.entries))
	for i, entry := range c.entries {
		ipAddresses[i] = entry.Node
	}
	return cke.Command{
		Name:   "rebootDequeueCommand",
		Target: strings.Join(ipAddresses, ","),
	}
}

//

type rebootRecalcMetricsOp struct {
	finished bool
}

// RebootUncordonOp returns an Operator to uncordon nodes.
func RebootRecalcMetricsOp() cke.Operator {
	return &rebootRecalcMetricsOp{}
}

func (o *rebootRecalcMetricsOp) Name() string {
	return "reboot-recalc-metrics"
}

func (o *rebootRecalcMetricsOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return rebootRecalcMetricsCommand{}
}

func (o *rebootRecalcMetricsOp) Targets() []string {
	return []string{}
}

type rebootRecalcMetricsCommand struct {
}

func (c rebootRecalcMetricsCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	rqEntries, err := inf.Storage().GetRebootsEntries(ctx)
	if err != nil {
		return err
	}
	metrics.UpdateRebootQueueEntries(len(rqEntries))
	rqEntries = cke.DedupRebootQueueEntries(rqEntries)
	itemCounts := cke.CountRebootQueueEntries(rqEntries)
	metrics.UpdateRebootQueueItems(itemCounts)

	return nil
}

func (c rebootRecalcMetricsCommand) Command() cke.Command {
	return cke.Command{
		Name: "rebootRecalcMetricsCommand",
	}
}

//

func listProtectedNamespaces(ctx context.Context, cs *kubernetes.Clientset, ls *metav1.LabelSelector) (map[string]bool, error) {
	selector, err := metav1.LabelSelectorAsSelector(ls)
	if err != nil {
		// ls should have been validated
		panic(err)
	}
	protected, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}

	nss := make(map[string]bool)
	for _, ns := range protected.Items {
		nss[ns.Name] = true
	}

	return nss, nil
}

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
		err := cs.CoreV1().Pods(pod.Namespace).Evict(ctx, &policyv1beta1.Eviction{
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
		case err != nil && !protected[pod.Namespace]:
			log.Warn("failed to evict non-protected pod", map[string]interface{}{
				"namespace": pod.Namespace,
				"name":      pod.Name,
				log.FnError: err,
			})
			err := cs.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
			log.Warn("deleted non-protected pod", map[string]interface{}{
				"namespace": pod.Namespace,
				"name":      pod.Name,
			})
		case err != nil:
			return fmt.Errorf("failed to evict pod %s/%s: %w", pod.Namespace, pod.Name, err)
		}
		return nil
	}, func(pod *corev1.Pod) error {
		return fmt.Errorf("job-managed pod exists: %s/%s, phase=%s", pod.Namespace, pod.Name, pod.Status.Phase)
	})
}

// checkPodDeletion checks whether the evicted or deleted Pods are eventually deleted.
// If those pods still exist, this function returns an error.
func checkPodDeletion(ctx context.Context, cs *kubernetes.Clientset, node string, protected map[string]bool) error {
	return enumeratePods(ctx, cs, node, func(pod *corev1.Pod) error {
		return fmt.Errorf("pod exists: %s/%s, phase=%s", pod.Namespace, pod.Name, pod.Status.Phase)
	}, func(pod *corev1.Pod) error {
		// This should not happen.
		return fmt.Errorf("job-managed pod exists: %s/%s, phase=%s", pod.Namespace, pod.Name, pod.Status.Phase)
	})
}

func drainBackOff(ctx context.Context, inf cke.Infrastructure, entry *cke.RebootQueueEntry, err error) error {
	log.Warn("failed to drain node", map[string]interface{}{
		"name":      entry.Node,
		log.FnError: err,
	})
	entry.Status = cke.RebootStatusQueued
	entry.LastTransitionTime = time.Now().Truncate(time.Second).UTC()
	entry.DrainBackOffCount++
	entry.DrainBackOffExpire = entry.LastTransitionTime.Add(time.Second * time.Duration(drainBackOffBaseSeconds+rand.Int63n(int64(drainBackOffBaseSeconds*entry.DrainBackOffCount))))
	err = inf.Storage().UpdateRebootsEntry(ctx, entry)
	if err != nil {
		return err
	}
	return nil
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

func CheckDrainCompletion(ctx context.Context, inf cke.Infrastructure, apiserver *cke.Node, c *cke.Cluster, rqEntries []*cke.RebootQueueEntry) (completed []*cke.RebootQueueEntry, timedout []*cke.RebootQueueEntry, err error) {
	evictionTimeoutSeconds := cke.DefaultRebootEvictionTimeoutSeconds
	if c.Reboot.EvictionTimeoutSeconds != nil {
		evictionTimeoutSeconds = *c.Reboot.EvictionTimeoutSeconds
	}

	cs, err := inf.K8sClient(ctx, apiserver)
	if err != nil {
		return nil, nil, err
	}

	protected, err := listProtectedNamespaces(ctx, cs, c.Reboot.ProtectedNamespaces)
	if err != nil {
		return nil, nil, err
	}

	t := time.Now().Add(time.Duration(-evictionTimeoutSeconds) * time.Second)

	for _, entry := range rqEntries {
		if !entry.ClusterMember(c) {
			continue
		}
		if entry.Status != cke.RebootStatusDraining {
			continue
		}

		err = checkPodDeletion(ctx, cs, entry.Node, protected)
		if err == nil {
			completed = append(completed, entry)
		} else if entry.LastTransitionTime.Before(t) {
			timedout = append(timedout, entry)
		}
	}

	return
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
