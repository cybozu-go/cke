Metrics
=======

CKE exposes the following metrics with the Prometheus format. The listen address can be configured by the CLI flag (see [here](cke.md#Usage)). All these metrics are prefixed with `cke_`

| Name              | Description                                      | Type  | Labels                             |
| ----------------- | ------------------------------------------------ | ----- | ---------------------------------- |
| operation_running | True(=1) if any operations are running.          | Gauge |                                    |
| boot_leader       | True(=1) if the boot server is the leader of CKE | Gauge |                                    |
| node_info         | The Control Plane and Worker info                | Gauge | address, rack, role, control_plane |

Note that cke also exposes the metrics provided by the Prometheus client library which located under `go` and `process` namespaces.
