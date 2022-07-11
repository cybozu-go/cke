Reboot Nodes Gracefully
======================

Description
-----------

An administrator can request CKE to gracefully reboot a set of nodes via `ckecli`.
The requests are appended to the reboot queue.
Each request entry corresponds to a list of nodes.

CKE watches the reboot queue and handles the reboot requests.
CKE processes reboot requests in the following manner:

1. cordons the nodes to mark them as unschedulable.
2. checks the existence of Job-managed Pods on the nodes. If such Pods exist on the nodes, uncordons the node immediately and process it again later.
3. evicts (and/or deletes) non-DaemonSet-managed pods on the nodes.
4. reboot the node by running hardware reboot command for the node.
5. waits for boot by running boot check command for the node.
6. uncordons the nodes and recovers them.

The behavior of the reboot functionality is configurable through the [cluster configuration](cluster.md#reboot).


Data schema
-----------

### `RebootQueueEntry`

| Name                   | Type      | Description                                            |
| ---------------------- | --------- | ------------------------------------------------------ |
| `index`                | string    | Index number of entry, formatted as a string.          |
| `node`                 | string    | An addresses of a node to reboot.                      |
| `status`               | string    | One of `queued`, `draining`, `rebooting`, `cancelled`. |
| `last_transition_time` | time.Time | The time last transition of `status`                   |
| `drain_backoff_count`  | int       | The number of drain backoff                            |
| `drain_backoff_expire` | time.Time | The time drain backoff expires                         |

Detailed behavior
-----------------

An administrator issues a reboot request using `ckecli reboot-queue add`.
The command writes reboot queue entry(s) and increments `reboots/write-index` atomically.

The queue is processed by CKE as follows:

1. If `reboots/disabled` is `true`, it doesn't process the queue.
2. Check the reboot queue to find an entry.
   - If the number of nodes under processing is less than maximum concurrent reboots and the number of unreachable nodes that are not under this reboot process is not more than `maximum-unreachable-nodes-for-reboot` in the constraints, pick several nodes from front of the queue and start draining them.
     1. Cordon the node.
     2. If there are Job-managed Pods, backoff the draining. i.e.:
       - update the entry status back to `queued`.
       - mark the entry not to be drained again immediately
     3. evict non-DaemonSet-managed Pods. If the eviction is failed due to PDBs and the namespace of the Pod is not protected by `.reboot.protected_namespaces`, delete the Pods. If the deletion is also failed, backoff the draining.
   - If draining is timed out, backoff the draining.
   - If draining is completed, run hardware reboot command specified by `.reboot.reboot_command` for the node and update the entry status to `rebooting`.
   - remove entries if:
     - the node is confirmed booted by boot check command specified by `.reboot.boot_check_command` or
     - the entry status is `cancelled`
   - If a node is cordoned by reboot operation and its entry status is not `draining` or `rebooting`, uncordon it.

There are several rules for API server nodes.

- API servers are processed one by one.
  - Multiple API servers are never processed simultaneously.
- API servers are processed with higher priority.
  - If API servers and non-API servers are in reboot queue, API servers are processed first.
- API servers are not processed simultaneously with non-API servers.

[LabelSelector]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
