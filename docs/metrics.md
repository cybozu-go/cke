Metrics
=======

CKE exposes the following metrics with the Prometheus format at `/metrics` REST API endpoint.  All these metrics are prefixed with `cke_`

| Name                                  | Description                                                                | Type  | Labels   |
| ------------------------------------- | -------------------------------------------------------------------------- | ----- | -------- |
| leader                                | True (=1) if this server is the leader of CKE.                             | Gauge |          |
| operation_phase                       | 1 if CKE is operating in the phase specified by the `phase` label.         | Gauge | `phase`  |
| operation_phase_timestamp_seconds     | The Unix timestamp when `operation_phase` was last updated.                | Gauge |          |
| reboot_queue_entries                  | The number of reboot queue entries remaining.                              | Gauge |          |
| reboot_queue_items                    | The number reboot queue entries remaining per status.                      | Gauge | `status` |
| sabakan_integration_successful        | True (=1) if sabakan-integration satisfies constraints.                    | Gauge |          |
| sabakan_integration_timestamp_seconds | The Unix timestamp when `sabakan_integration_successful` was last updated. | Gauge |          |
| sabakan_workers                       | The number of worker nodes for each role.                                  | Gauge | `role`   |
| sabakan_unused_machines               | The number of unused machines.                                             | Gauge |          |

All metrics but `leader` are available only when the server is the leader of CKE.
`sabakan_*` metrics are available only when [Sabakan integration](sabakan-integration.md) is enabled.

Note that CKE also exposes the metrics for Go runtime (`go_*`) and the process (`process_*`).
