Constraints on cluster
===================

Constraints
-----------------

Cluster should satisfy these constraints.

| Name                         | Type | Default | Description                                                           |
| ---------------------------- | ---- | ------- | --------------------------------------------------------------------- |
| `control-plane-count`        | int  | 1       | Number of control plane nodes                                         |
| `minimum-workers`            | int  | 1       | The minimum number of worker nodes                                    |
| `maximum-workers`            | int  | 0       | The maximum number of worker nodes. 0 means unlimited.                |
| `reboot-maximum-unreachable` | int  | 0       | The maximum number of unreachable nodes allowed for operating reboot. |
