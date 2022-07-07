Notes on reboot design
======================

cf. [reboot.md](reboot.md)

## Design decisions

### Special care for API servers

CKE processes API servers as:

- API servers are rebooted one by one.
  - In order to minimize API server degrade.
- API Servers are rebooted with higher priority than worker nodes.
  - No explicit reason.
- API Servers are not rebooted simultaneously with worker nodes.
  - It might be better not to reboot worker nodes during API server degrade.

### Reboot command failure and timeout

CKE does not handle them. There is no status indicates those failures -- it is not worth adding. Reboot queue entries are leaved `rebooting` state.

If many nodes encounter reboot command failure or timeout, overall reboot operation will slow down and get stuck eventually. Administrators should detect them by checking metrics.

## Considered but not adopted

### Checking drainability before deciding which node to drain

If we check a node can be drained without eviction failure beforehand, draining nodes will proceed (more) smoothly. It may be a future work.

### Explicit same-rack prioritize strategy

In most cases, Pods are distributed between racks; i.e. multiple Pods covered by single PDB are not running in single rack. If CKE choose nodes in the same rack in which the nodes already processed, draining will progress smoothly. However, this control requires much implementation but achieve few advantages.

In current implementation, CKE basically processes reboot queue entires from front to back. Administrators should simply add nodes to reboot queue grouped by rack. By doing so, nodes are processed almost per-rack basis.

### Race condition between CKE and `ckecli rq cancel`

There are a race condition between them. In this case, reboot queue entries might not be cancelled actually although administrators run `ckecli rq cancel`. This issue has not been addressed because there is little harm.

Administrators should check the entry is successfully cancelled by using `ckecli rq list`.

## Unknown

The following decisions have been buried in history and the reason is not known today.

- Why maximum-unreachable-nodes-for-reboot is in [constraints](constraints.md), not in [cluster.yml](cluster.md)
