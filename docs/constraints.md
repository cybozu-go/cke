Constraints on cluster
===================

Constraints
-----------------

Cluster should satisfy these constraints.

|                  Name                  | Type | Default |                              Description                              |
| -------------------------------------- | ---- | ------- | --------------------------------------------------------------------- |
| `control-plane-count`                  | int  | 1       | Number of control plane nodes                                         |
| `minimum-workers-rate`                 | int  | 80      | The minimum percentage of workers/machines.                           |
| `maximum-unreachable-nodes-for-reboot` | int  | 0       | The maximum number of unreachable nodes allowed for operating reboot. |
| `maximum-repair-queue-entries`         | int  | 0       | The maximum number of repair queue entries                            |
| `wait-seconds-to-repair-rebooting`     | int  | 0       | The wait time in seconds to repair a rebooting machine                |
