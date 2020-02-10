Metrics
=======

CKE exposes the following metrics with the Prometheus format. The listen address can be configured by the CLI flag (see [here](cke.md#Usage)). All these metrics are prefixed with `cke_`

| Name                          | Description                                           | Type  | Labels                             |
| -----------------             | ------------------------------------------------      | ----- | ---------------------------------- |
| progressing                   | True(=1) if any operations are progressing.           | Gauge |                                    |
| leader                        | True(=1) if this server is the leader of CKE          | Gauge |                                    |
| node_info                     | The Control Plane and Worker info                     | Gauge | address, rack, role, control_plane |
| sabakan_integration_available | True(=1) if sabakan-integration satisfies constraints | Gauge |                                    |
| sabakan_workers               | The number of worker nodes.                           | Gauge | role                               |
| sabakan_unused_machines       | The number of unused machines.                        | Gauge |                                    |

Note that cke also exposes the metrics provided by the Prometheus client library which located under `go` and `process` namespaces.
