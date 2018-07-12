Operation Record
================

CKE stores operation records in etcd.
A record is an object with these fields:

| Name        | Type      | Description                                       |
| ----------- | --------- | ------------------------------------------------- |
| `id`        | string    | ID of the operation                               |
| `status`    | string    | One of `new`, `running`, `cancelled`, `completed` |
| `operation` | string    | The operation name                                |
| `command`   | `Command` | See `Command` spec.                               |
| `error`     | string    | Command error message if operation failed.        |
| `start-at`  | string    | RFC3339 formatted time                            |
| `end-at`    | string    | RFC3339 formatted time                            |

`Command` is an object with these fields:

| Name     | Type    | Description               |
| -------- | ------- | -----------------------   |
| `name`   | string  | The name of the command   |
| `target` | string  | The target of the command |
| `detail` | string  | The detail of the command |

