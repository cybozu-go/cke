Repair Machines
===============

CKE provides the functionality of managing operation steps to repair a machine.
This functionality has similar data structures and configuration parameters to the [reboot functionality](reboot.md).

Unlike other functionalities of CKE, this repair functionality manages a specified machine even if the machine is not a member of the Kubernetes cluster.

Description
-----------

First see the [description of the reboot functionality](reboot.md#description).
The behavior of the repair functionality is almost the same with the reboot.
A significant difference is that the repair functionality issues a series of repair commands instead of one reboot command.

An administrator can request CKE to repair a machine via `ckecli repair-queue add OPERATION MACHINE_TYPE ADDRESS`.
The request is appended to the repair queue.
Each request entry corresponds to a machine.

The command `ckecli repair-queue add` takes extra two arguments in addition to the IP address of the target machine; the operation name and the type of the target machine.

CKE watches the repair queue and handles the repair requests.
CKE processes a repair request in the following manner:

1. determines a sequence of repair steps to apply according to the machine type and the operation name.
2. executes the steps sequentially:
    1. if Pod eviction is required in the step and the machine is used as a Node of the Kubernetes cluster:
        1. cordons the Node to mark it as unschedulable.
        2. checks the existence of Job-managed Pods on the Node. If such Pods exist on the Node, uncordons the Node immediately and processes it again later.
        3. evicts (and/or deletes) non-DaemonSet-managed Pods on the Node.
    2. executes a repair command specified in the step.
    3. watches whether the machine becomes healthy by running a check command specified for the machine type.
    4. if the node becomes healthy, uncordons the node, recovers it, and finishs repairing.
3. if the node is not healthy even after all steps are executed, marks the entry as failed.

Unlike the reboot queue, repair queue entries remain in the queue even after they finished whether they succeeded or failed.
An administrator can delete a finished queue entry by `ckecli repair-queue delete INDEX`.

Data Schema
-----------

### `RepairQueueEntry`

| Name                   | Type      | Description                                                      |
| ---------------------- | --------- | ---------------------------------------------------------------- |
| `index`                | string    | Index number of the entry, formatted as a string.                |
| `address`              | string    | Address of the machine to be repaired.                           |
| `nodename`             | string    | Name of the Kubernetes Node corresponding to the target machine. |
| `machine_type`         | string    | Type name of the target machine.                                 |
| `operation`            | string    | Operation name to be applied for the target machine.             |
| `status`               | string    | One of `queued`, `processing`, `succeeded`, `failed`.            |
| `step`                 | int       | Index number of the current step.                                |
| `step_status`          | string    | One of `waiting`, `draining`, `watching`.                        |
| `last_transition_time` | time.Time | Time of the last transition of `status`+`step`+`step_status`.    |
| `drain_backoff_count`  | int       | Count of drain retries, used for linear backoff algorithm.       |
| `drain_backoff_expire` | time.Time | Expiration time of drain retry wait.                             |

Detailed Behavior and Parameters
--------------------------------

The behavior of the repair functionality is configurable mainly through the [cluster configuration](cluster.md#repair).
The following are the detailed descriptions of the repair functionality and its parameters.

### Repair configuration

`repair_procedures` is a list of [repair procedures](#repairprocedures) for various types of machines.
CKE selects an appropriate repair procedure according to the `TYPE` parameter specified in `ckecli repair-queue add` command line.

At most `max_concurrent_repairs` entries are repaired concurrently.

Other parameters under `repair` are used for [Pod eviction](#podeviction).

#### Pod eviction

If Pod eviction is required in the current repair step of the selected repair procedure, CKE tries to delete Pods running on the target machine before it executes a repair command.

If a Pod to be deleted belongs to one of the Namespaces selected by `protected_namespaces`, CKE tries to delete that Pod gracefully with the Kubernetes Eviction API.
If `protected_namespaces` is not given, all namespaces are protected.

If the Eviction API call has failed, i.e., if CKE fails to start the Pod deletion, CKE retries it for `evict_retries` times with `evict_interval`-second interval.
If CKE finally fails to start the Pod deletion, it interrupts the deletion, uncordons the target machine, waits for a while using a linear backoff algorithm, and retries the deletion.

Once CKE succeeds in starting the Pod deletion for all Pods, it waits for the completion of the deletion.
If the Pod deletion does not finish during `eviction_timeout_seconds`, CKE interrupts the deletion, uncordons the target machine, waits for a while using a linear backoff algorithm, and retries the deletion.
The number of retries of the whole process of the Pod deletion is unlimited.

### Repair procedures

A repair procedure is a set of [repair operations](#repairoperations) for a certain type of machines.

`machine_types` is a list of machine type names.
CKE decides to apply a repair procedure if its `machine_types` contains `TYPE` of a repair queue entry, where `TYPE` is specified in the `ckecli repair-queue add` command line.

`repair_operations` maps operation names to [repair operations](#repairoperations).
More properly speaking, this is not implemented as a mapping but as a list for readability; each element has its name as its property.
CKE decides to execute a repair operation if its name matches `OPERATION` of a repair queue entry, where `OPERATION` is specified in the `ckecli repair-queue add` command line.

### Repair operations

A repair operation is a sequence of [repair steps](#repairsteps) and their parameters.

`operation` is the name of a repair operation.
CKE decides to execute a repair operations if its `operation` matches `OPERATION` of a repair queue entry, where `OPERATION` is specified in the `ckecli repair-queue add` command line.

`repair_steps` is a sequence of [repair steps](#repairsteps).

`health_check_command` and its timeout are used during the execution of the repair steps.
When CKE executes the check command, it appends the IP address of the target machine to the command.
The command should return a string `true` if it evaluates the machine as healthy.

`success_command` and its timeout are used when the machine is evaluated as healthy and the repair operation finishes successfully.
When CKE executes the success command, it appends the IP address of the target machine to the command.
If the repair operation has failed, the command is not executed.
If the `success_command` fails, CKE changes the status of the queue entry to `failed`.
Users can use this command if they want to execute a command as a post-processing of repair operation.

### Repair steps

A repair step is a combination of:
1. (optional) eviction operation
2. repair command
3. (optional) watch duration

If `need_drain` is true and the target machine is used as a Kubernetes Node, CKE tries to [drain the Node](#podeviction) before starting a repair command.

`repair_command` is a command to repair a machine.
When CKE executes the repair command, it appends the IP address of the target machine to the command.
If the command fails, CKE changes the status of the queue entry to `failed` and aborts the repair steps.

After executing `repair_command`, CKE watches whether the machine becomes healthy.
If the health check command returns `true`, CKE finishes repairing and changes the status of the queue entry to `succeeded`.
If the command does not return `true` during `watch_seconds`, CKE proceeds to the next step if exists.
If CKE reaches at the end of the steps, it changes the status of the queue entry to `failed`.

Enabling/Disabling
------------------

An administrator can enable/disable the processing of the repair queue by `ckecli repair-queue enable|disable`.
If the queue is disabled, CKE:
* does not proceed to the [Pod eviction](#podeviction) nor the repair command execution
* abandons ongoing Pod eviction
* still runs health check commands and migrates entries to succeeded/failed
* still dequeues entries if instructed by `ckecli repair-queue delete`
