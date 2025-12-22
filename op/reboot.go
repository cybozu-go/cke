package op

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const drainBackOffBaseSeconds = 300
const drainBackOffMaxSeconds = 1200

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
	evictAttempts       int
	evictInterval       time.Duration

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

	attempts := 1
	if o.config.EvictRetries != nil {
		attempts = *o.config.EvictRetries + 1
	}
	interval := 0 * time.Second
	if o.config.EvictInterval != nil {
		interval = time.Second * time.Duration(*o.config.EvictInterval)
	}

	return rebootDrainStartCommand{
		entries:             o.entries,
		protectedNamespaces: o.config.ProtectedNamespaces,
		apiserver:           o.apiserver,
		notifyFailedNode:    o.notifyFailedNode,
		evictAttempts:       attempts,
		evictInterval:       interval,
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
			_, err = nodesAPI.Patch(ctx, entry.Node, types.StrategicMergePatchType, []byte(`
{
	"metadata":{"annotations":{"`+CKEAnnotationReboot+`": "true"}},
	"spec":{"unschedulable": true}
}
`), metav1.PatchOptions{})
			if err != nil {
				return fmt.Errorf("failed to cordon node %s: %v", entry.Node, err)
			}
			log.Info("start eviction dry-run", map[string]interface{}{
				"name": entry.Node,
			})
			err = dryRunEvictOrDeleteNodePod(ctx, cs, entry.Node, protected)
			if err != nil {
				log.Warn("eviction dry-run failed", map[string]interface{}{
					"name":      entry.Node,
					log.FnError: err,
				})
				return err
			}
			log.Info("eviction dry-run succeeded", map[string]interface{}{
				"name": entry.Node,
			})
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
		log.Info("start eviction", map[string]interface{}{
			"name": entry.Node,
		})
		err := evictOrDeleteNodePod(ctx, cs, entry.Node, protected, c.evictAttempts, c.evictInterval)
		if err != nil {
			log.Warn("eviction failed", map[string]interface{}{
				"name":      entry.Node,
				log.FnError: err,
			})
			c.notifyFailedNode(entry.Node)
			err = drainBackOff(ctx, inf, entry, err)
			if err != nil {
				return err
			}
			log.Info("eviction succeeded", map[string]interface{}{
				"name": entry.Node,
			})
		}
	}

	return nil
}

//

type rebootDeleteDaemonSetPodOp struct {
	finished bool

	entries   []*cke.RebootQueueEntry
	config    *cke.Reboot
	apiserver *cke.Node

	mu          sync.Mutex
	failedNodes []string
}

func RebootDeleteDaemonSetPodOp(apiserver *cke.Node, entries []*cke.RebootQueueEntry, config *cke.Reboot) cke.InfoOperator {
	return &rebootDeleteDaemonSetPodOp{
		entries:   entries,
		config:    config,
		apiserver: apiserver,
	}
}

type rebootDeleteDaemonSetPodCommand struct {
	entries   []*cke.RebootQueueEntry
	apiserver *cke.Node

	notifyFailedNode func(string)
}

func (o *rebootDeleteDaemonSetPodOp) Name() string {
	return "reboot-delete-daemonset-pod"
}

func (o *rebootDeleteDaemonSetPodOp) notifyFailedNode(node string) {
	o.mu.Lock()
	o.failedNodes = append(o.failedNodes, node)
	o.mu.Unlock()
}

func (o *rebootDeleteDaemonSetPodOp) Targets() []string {
	ipAddresses := make([]string, len(o.entries))
	for i, entry := range o.entries {
		ipAddresses[i] = entry.Node
	}
	return ipAddresses
}

func (o *rebootDeleteDaemonSetPodOp) Info() string {
	if len(o.failedNodes) == 0 {
		return ""
	}
	return fmt.Sprintf("failed to delete DaemonSet pods on some nodes: %v", o.failedNodes)
}

func (o *rebootDeleteDaemonSetPodOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true

	return rebootDeleteDaemonSetPodCommand{
		entries:          o.entries,
		apiserver:        o.apiserver,
		notifyFailedNode: o.notifyFailedNode,
	}
}

func (c rebootDeleteDaemonSetPodCommand) Command() cke.Command {
	ipAddresses := make([]string, len(c.entries))
	for i, entry := range c.entries {
		ipAddresses[i] = entry.Node
	}
	return cke.Command{
		Name:   "rebootDeleteDaemonSetPodCommand",
		Target: strings.Join(ipAddresses, ","),
	}
}

func (c rebootDeleteDaemonSetPodCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// delete DaemonSet pod on each node
	for _, entry := range c.entries {
		// keep entry.Status as RebootStatusDraining and don't update it here.

		log.Info("start deletion of DaemonSet pod", map[string]interface{}{
			"name": entry.Node,
		})
		err := deleteOnDeleteDaemonSetPod(ctx, cs, entry.Node)
		if err != nil {
			log.Warn("deletion of DaemonSet pod failed", map[string]interface{}{
				"name":      entry.Node,
				log.FnError: err,
			})
			c.notifyFailedNode(entry.Node)
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
	retries        *int
	interval       *int

	notifyFailedNode func(string)
}

func (o *rebootRebootOp) notifyFailedNode(node string) {
	o.mu.Lock()
	o.failedNodes = append(o.failedNodes, node)
	o.mu.Unlock()
}

// RebootRebootOp returns an Operator to reboot nodes.
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
		retries:          o.config.CommandRetries,
		interval:         o.config.CommandInterval,
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
	var mu sync.Mutex

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

			mu.Lock()
			inf.ReleaseAgent(entry.Node)
			mu.Unlock()

			var attempts int = 1
			if c.retries != nil {
				attempts = *c.retries + 1
			}
		RETRY:
			for i := 0; i < attempts; i++ {
				err := func() error {
					ctx := ctx
					if c.timeoutSeconds != nil && *c.timeoutSeconds != 0 {
						var cancel context.CancelFunc
						ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(*c.timeoutSeconds))
						defer cancel()
					}

					args := append(c.command[1:], entry.Node)
					command := well.CommandContext(ctx, c.command[0], args...)
					return command.Run()
				}()
				if err == nil {
					return nil
				}

				log.Warn("failed on rebooting node", map[string]interface{}{
					log.FnError: err,
					"node":      entry.Node,
					"attempts":  i,
				})
				if c.interval != nil && *c.interval != 0 {
					select {
					case <-time.After(time.Second * time.Duration(*c.interval)):
					case <-ctx.Done():
						break RETRY
					}
				}
			}
			c.notifyFailedNode(entry.Node)
			log.Warn("given up rebooting node", map[string]interface{}{
				"node": entry.Node,
			})
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
	return rebootUncordonCommand{
		apiserver: o.apiserver,
		nodeNames: o.nodeNames,
	}
}

func (o *rebootUncordonOp) Targets() []string {
	return o.nodeNames
}

type rebootUncordonCommand struct {
	apiserver *cke.Node
	nodeNames []string
}

func (c rebootUncordonCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
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

func (c rebootUncordonCommand) Command() cke.Command {
	return cke.Command{
		Name:   "rebootUncordonCommand",
		Target: strings.Join(c.nodeNames, ","),
	}
}

//

type rebootDequeueOp struct {
	finished bool

	entries []*cke.RebootQueueEntry
}

// RebootDequeueOp returns an Operator to dequeue reboot entries.
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

type rebootCancelOp struct {
	rebootDequeueOp
}

// RebootCancelOp returns an Operator to dequeue cancelled reboot entries.
func RebootCancelOp(entries []*cke.RebootQueueEntry) cke.Operator {
	return &rebootCancelOp{
		rebootDequeueOp{
			entries: entries,
		},
	}
}

func (o *rebootCancelOp) Name() string {
	return "reboot-cancel"
}

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

func drainBackOff(ctx context.Context, inf cke.Infrastructure, entry *cke.RebootQueueEntry, err error) error {
	log.Warn("failed to drain node", map[string]interface{}{
		"name":      entry.Node,
		log.FnError: err,
	})
	etcdEntry, err := inf.Storage().GetRebootsEntry(ctx, entry.Index)
	if err != nil {
		return err
	}
	if etcdEntry.Status == cke.RebootStatusCancelled {
		return nil
	}
	entry.Status = cke.RebootStatusQueued
	entry.LastTransitionTime = time.Now().Truncate(time.Second).UTC()
	entry.DrainBackOffCount++
	var backoffSeconds int64 = (1 << (entry.DrainBackOffCount - 1)) * drainBackOffBaseSeconds
	if backoffSeconds > drainBackOffMaxSeconds {
		backoffSeconds = drainBackOffMaxSeconds
	}
	entry.DrainBackOffExpire = entry.LastTransitionTime.Add(time.Second * time.Duration(backoffSeconds+rand.Int63n(drainBackOffBaseSeconds)))

	err = inf.Storage().UpdateRebootsEntry(ctx, entry)
	if err != nil {
		return err
	}
	return nil
}
