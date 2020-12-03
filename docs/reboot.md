Reboot Nodes Gracefully
======================

Description
-----------

An administrator can request CKE to gracefully reboot a set of nodes via `ckecli`.
The requests are appended to the reboot queue.
Each request entry corresponds to a list of nodes.

CKE watches the reboot queue and handles the reboot request one by one.
CKE first cordons the nodes to mark them as unschedulable.
Then CKE calls the Kubernetes eviction API to delete the Pods on the target nodes while respecting the PodDisruptionBudget.
It proceeds without deleting DaemonSet-managed Pods.
After waiting for the deletion of the Pods, CKE reboots and waits for the nodes by invoking an external command specified in the [cluster configuration](cluster.md#reboot).
Finally, CKE uncordons the nodes and recovers them, e.g. resumes kubelet.

The behavior of the reboot functionality is configurable through the [cluster configuration](cluster.md#reboot).


Data schema
-----------

### `RebootQueueEntry`

| Name     | Type     | Description                                   |
| -------- | -------- | --------------------------------------------- |
| `index`  | string   | Index number of entry, formatted as a string. |
| `nodes`  | []string | A list of IP addresses of nodes to reboot.    |
| `status` | string   | One of `queued`, `rebooting`, `cancelled`.    |


Detailed behavior
-----------------

An administrator issues a reboot request using `ckecli reboot-queue add`.
The command writes a reboot queue entry and increments `reboots/write-index` atomically.

The queue is processed by CKE as follows:

1. If `reboots/disabled` is `true`, it doesn't process the queue.
2. Check the number of unreachable nodes. If it exceeds `maximum-unreachable-nodes-for-reboot` in the constraints, it doesn't process the queue.
3. Check the reboot queue to find an entry. If the entry's status is `cancelled`, remove it and check the queue again. If there is no entry, CKE stops the processing.
4. For the first entry in the reboot queue, do the following steps.
   1. Update the entry status to `rebooting`.
   2. Cordon the nodes in the entry.
   3. Call the eviction API for Pods running on the target nodes.  DaemonSet-managed Pods are ignored.  If pods not in the `protected_namespaces` fail to be evicted, they are deleted instead.
   4. Wait for the deletion of the Pods.  If this step exceeds a deadline specified in the cluster configuration, the operation is aborted and the queue entry is left as is.
   5. Reboot the nodes using `.reboot.command` in the cluster configuration. In this step, all the nodes are rebooted simultaneously. If some of the nodes won't get back ready within the deadline specified in the cluster configuration, CKE gives up waiting for them (no error).
   6. Record the status in the history record.  It includes the list of nodes that failed to reboot.
   7. Remove the entry.
   8. Uncordon the nodes.


[LabelSelector]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
