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

1. Check the reboot queue to find an entry. If the entry's status is `cancelled`, remove it and check the queue again. If there is no entry, return from this routine.
2. Check the number of unreachable nodes. If it exceeds `reboot-maximum-unreachable` in the constraints, return from this routine.
3. Update the entry status to `rebooting`.
4. For each node in `nodes`, do the following in parallel:
   1. Cordon the node.
   2. Drain the Pods on the node using Kubernetes eviction API.  DaemonSet-managed Pods are ignored. The Pods not in `protected-namespaces` are deleted immediately.
   3. Wait for the deletion of the Pods.
   4. Reboot and wait for the node using `.reboot.command` in the cluster configuration.
   5. Uncordon the node.
   6. Record the success/failure status in the history record.
   7. If any of the above steps fails or times out, it skips the succeeding steps.
5. Remove the entry.


[LabelSelector]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
