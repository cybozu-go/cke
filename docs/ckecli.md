ckecli command reference
========================

```console
$ ckecli [--config FILE] <subcommand> args...
```

| Option      | Default value         | Description         |
| ----------- | --------------------- | ------------------- |
| `--config`  | `/etc/cke/config.yml` | config file path    |
| `--version` |                       | show ckecli version |

- [`ckecli cluster`](#ckecli-cluster)
  - [`ckecli cluster set FILE`](#ckecli-cluster-set-file)
  - [`ckecli cluster get`](#ckecli-cluster-get)
- [`ckecli constraints`](#ckecli-constraints)
  - [`ckecli constraints set NAME VALUE`](#ckecli-constraints-set-name-value)
  - [`ckecli constraints show`](#ckecli-constraints-show)
- [`ckecli vault`](#ckecli-vault)
  - [`ckecli vault init`](#ckecli-vault-init)
  - [`ckecli vault config JSON`](#ckecli-vault-config-json)
  - [`ckecli vault ssh-privkey [--host=HOST] FILE`](#ckecli-vault-ssh-privkey---hosthost-file)
  - [`ckecli vault enckey`](#ckecli-vault-enckey)
- [`ckecli ca`](#ckecli-ca)
  - [`ckecli ca set NAME PEM`](#ckecli-ca-set-name-pem)
  - [`ckecli ca get NAME`](#ckecli-ca-get-name)
- [`ckecli leader`](#ckecli-leader)
- [`ckecli history [OPTION]...`](#ckecli-history-option)
- [`ckecli images`](#ckecli-images)
- [`ckecli etcd`](#ckecli-etcd)
  - [`ckecli etcd user-add NAME PREFIX`](#ckecli-etcd-user-add-name-prefix)
  - [`ckecli etcd issue [--ttl=TTL] [--output=FORMAT] NAME`](#ckecli-etcd-issue---ttlttl---outputformat-name)
  - [`ckecli etcd root-issue [--output=FORMAT]`](#ckecli-etcd-root-issue---outputformat)
  - [`ckecli etcd local-backup`](#ckecli-etcd-local-backup)
- [`ckecli kubernetes`](#ckecli-kubernetes)
  - [`ckecli kubernetes issue [--ttl=TTL] [--group=GROUPNAME] [--user=USERNAME]`](#ckecli-kubernetes-issue---ttlttl---groupgroupname---userusername)
- [`ckecli resource`](#ckecli-resource)
  - [`ckecli resource list`](#ckecli-resource-list)
  - [`ckecli resource set FILE`](#ckecli-resource-set-file)
  - [`ckecli resource get KEY`](#ckecli-resource-get-key)
  - [`ckecli resource delete FILE`](#ckecli-resource-delete-file)
- [`ckecli ssh [user@]NODE [COMMAND...]`](#ckecli-ssh-usernode-command)
- [`ckecli scp [-r] [[user@]NODE1:]FILE1 ... [[user@]NODE2:]FILE2`](#ckecli-scp--r-usernode1file1--usernode2file2)
- [`ckecli reboot-queue`, `ckecli rq`](#ckecli-reboot-queue-ckecli-rq)
  - [`ckecli reboot-queue enable|disable`](#ckecli-reboot-queue-enabledisable)
  - [`ckecli reboot-queue is-enabled`](#ckecli-reboot-queue-is-enabled)
  - [`ckecli reboot-queue add FILE`](#ckecli-reboot-queue-add-file)
  - [`ckecli reboot-queue list`](#ckecli-reboot-queue-list)
  - [`ckecli reboot-queue cancel INDEX`](#ckecli-reboot-queue-cancel-index)
  - [`ckecli reboot-queue cancel-all`](#ckecli-reboot-queue-cancel-all)
  - [`ckecli reboot-queue reset-backoff`](#ckecli-reboot-queue-reset-backoff)
- [`ckecli repair-queue`](#ckecli-repair-queue)
  - [`ckecli repair-queue enable|disable`](#ckecli-repair-queue-enabledisable)
  - [`ckecli repair-queue is-enabled`](#ckecli-repair-queue-is-enabled)
  - [`ckecli repair-queue add OPERATION MACHINE_TYPE ADDRESS [SERIAL]`](#ckecli-repair-queue-add-operation-machine_type-address-serial)
  - [`ckecli repair-queue list`](#ckecli-repair-queue-list)
  - [`ckecli repair-queue delete INDEX`](#ckecli-repair-queue-delete-index)
  - [`ckecli repair-queue delete-finished`](#ckecli-repair-queue-delete-finished)
  - [`ckecli repair-queue delete-unfinished`](#ckecli-repair-queue-delete-unfinished)
  - [`ckecli repair-queue reset-backoff`](#ckecli-repair-queue-reset-backoff)
- [`ckecli sabakan`](#ckecli-sabakan)
  - [`ckecli sabakan enable|disable`](#ckecli-sabakan-enabledisable)
  - [`ckecli sabakan is-enabled`](#ckecli-sabakan-is-enabled)
  - [`ckecli sabakan set-url URL`](#ckecli-sabakan-set-url-url)
  - [`ckecli sabakan get-url`](#ckecli-sabakan-get-url)
  - [`ckecli sabakan set-template FILE`](#ckecli-sabakan-set-template-file)
  - [`ckecli sabakan get-template`](#ckecli-sabakan-get-template)
  - [`ckecli sabakan set-variables FILE`](#ckecli-sabakan-set-variables-file)
  - [`ckecli sabakan get-variables`](#ckecli-sabakan-get-variables)
- [`ckecli auto-repair`](#ckecli-auto-repair)
  - [`ckecli auto-repair enable|disable`](#ckecli-auto-repair-enabledisable)
  - [`ckecli auto-repair is-enabled`](#ckecli-auto-repair-is-enabled)
  - [`ckecli auto-repair set-variables FILE`](#ckecli-auto-repair-set-variables-file)
  - [`ckecli auto-repair get-variables`](#ckecli-auto-repair-get-variables)
- [`ckecli status`](#ckecli-status)

## `ckecli cluster`

### `ckecli cluster set FILE`

Set the cluster configuration.

### `ckecli cluster get`

Get the cluster configuration.

## `ckecli constraints`

### `ckecli constraints set NAME VALUE`

Set a constraint on the cluster configuration.

`NAME` is one of:

- `control-plane-count`
- `minimum-workers-rate`
- `maximum-unreachable-nodes-for-reboot`
- `maximum-repair-queue-entries`
- `wait-seconds-to-repair-rebooting`

### `ckecli constraints show`

Show all constraints on the cluster.

## `ckecli vault`

Vault related commands.

### `ckecli vault init`

Initialize vault configuration for CKE as described in [vault.md](vault.md).

### `ckecli vault config JSON`

Set vault configuration for CKE.
`JSON` is a filename whose body is a JSON object described in [schema.md](schema.md#vault).

If `JSON` is "-", `ckecli` reads from stdin.

### `ckecli vault ssh-privkey [--host=HOST] FILE`

Store SSH private key for a host into Vault.  If no HOST is specified, the key will be
used as the default key.

FILE should be a SSH private key file.  If FILE is `-`, the contents are read from stdin.

### `ckecli vault enckey`

Generate a new cipher key to encrypt Kubernetes [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/).

The current key, if any, is retained for key rotation.  Old keys are removed.

**WARNING**

Key rotation is not automated in the current version.
You need to restart API servers manually and replace all secrets as follows:

```console
$ kubectl get secrets --all-namespaces -o json | kubectl replace -f -
```

## `ckecli ca`

### `ckecli ca set NAME PEM`

`NAME` is one of `server`, `etcd-peer`, `etcd-client`, `kubernetes`.

`PEM` is a filename of a x509 certificate.

### `ckecli ca get NAME`

`NAME` is one of `server`, `etcd-peer`, `etcd-client`, `kubernetes`.

## `ckecli leader`

Show the host name of the current leader.

## `ckecli history [OPTION]...`

Show operation history.

| Option           | Default value | Description                                                               |
| ---------------- | ------------- | ------------------------------------------------------------------------- |
| `-n`, `--count`  | `0`           | The number of the history to show. If `0` is specified, show all history. |
| `-f`, `--follow` | `false`       | Show the history in a new order, and continuously print new entries.      |

## `ckecli images`

List container image names used by `cke`.

## `ckecli etcd`

Control CKE managed etcd.

### `ckecli etcd user-add NAME PREFIX`

This subcommand is for programs to operate etcd server.

Add `NAME` user/role to etcd.

The user can only access under `PREFIX`.

### `ckecli etcd issue [--ttl=TTL] [--output=FORMAT] NAME`

This subcommand is for programs to operate etcd server.

Create a client certificate for user `NAME`.

| Option     | Default value | Description                   |
| ---------- | ------------- | ----------------------------- |
| `--ttl`    | `87600h`      | TTL for client certificate    |
| `--output` | `json`        | output format (`json`,`file`) |

### `ckecli etcd root-issue [--output=FORMAT]`

Create client certificate for `root`.

TTL for this certificate is fixed to 2h.

This subcommand is for human to operate etcd server.

| Option     | Default value | Description                   |
| ---------- | ------------- | ----------------------------- |
| `--output` | `json`        | output format (`json`,`file`) |

### `ckecli etcd local-backup`

This command takes a snapshot of CKE-managed etcd that stores Kubernetes data.

The snapshots are saved in a directory specified with `--dir` flag
with this format: `etcd-YYYYMMDD-hhmmss.backup`

The date and time is UTC.

Old backups are automatically removed when the number of backup files
exceed the maximum specified with `--max-backups` flag.

```
Usage:
  ckecli etcd local-backup [flags]

Flags:
      --dir string        the directory to keep the backup files (default "/var/cke/etcd-backups")
  -h, --help              help for local-backup
      --max-backups int   the maximum number of backups to keep (default 10)
```

## `ckecli kubernetes`

Control CKE managed kubernetes.

### `ckecli kubernetes issue [--ttl=TTL] [--group=GROUPNAME] [--user=USERNAME]`

Write kubeconfig to stdout.

This config file embeds client certificate and can be used with `kubectl` to connect Kubernetes cluster.

| Option    | Default value    | Description                                 |
| --------- | ---------------- | ------------------------------------------- |
| `--ttl`   | `2h`             | TTL of the client certificate               |
| `--group` | `system:masters` | organization name of the client certificate |
| `--user`  | `admin`          | user name of the client certificate         |

## `ckecli resource`

Edit user-defined resources in Kubernetes.
See [User-defined resources](user-resources.md) for details.

### `ckecli resource list`

List registered resources.

### `ckecli resource set FILE`

Register user-defined resources listed in `FILE`.
If `FILE` is "-", then resources are read from stdin.

The registered resources will be synchronized with Kubernetes by CKE.

### `ckecli resource get KEY`

Get a user-defined resource by `KEY`.
The list of the resources is available using `ckecli resource list`.

### `ckecli resource delete FILE`

Remove user-defined resources listed in `FILE` from etcd.
If `FILE` is "-", then resources are read from stdin.

Note that Kubernetes resources will not be removed automatically.

## `ckecli ssh [user@]NODE [COMMAND...]`

Connect to the node via ssh.

`NODE` is IP address or hostname of the node to be connected.

If `COMMAND` is specified, it will be executed on the node.

## `ckecli scp [-r] [[user@]NODE1:]FILE1 ... [[user@]NODE2:]FILE2`

Copy files between hosts via scp.

`NODE` is IP address or hostname of the node.

| Option | Default value | Description                          |
| ------ | ------------- | ------------------------------------ |
| `-r`   | `false`       | Recursively copy entire directories. |

## `ckecli reboot-queue`, `ckecli rq`

`rq` is an alias of `reboot-queue`.

### `ckecli reboot-queue enable|disable`

Enable/Disable processing reboot queue entries.

### `ckecli reboot-queue is-enabled`

Show reboot queue is enabled or disabled.
It displays `true` or `false`.

### `ckecli reboot-queue add FILE`

Append the nodes written in `FILE` to the reboot queue.
The nodes should be specified with their IP addresses.
If `FILE` is `-`, the contents are read from stdin.

For safety, multiple control plane nodes cannot be enqueued in one entry.

### `ckecli reboot-queue list`

List the entries in the reboot queue.
The output is a list of [entries](reboot.md#rebootqueueentry) formatted in JSON.

### `ckecli reboot-queue cancel INDEX`

Cancel the specified reboot queue entry.

### `ckecli reboot-queue cancel-all`

Cancel all the reboot queue entries.

### `ckecli reboot-queue reset-backoff`

Reset `drain_backoff_count` and `drain_backoff_expire` of the entries in reboot queue.
Resetting these values makes CKE try to reboot nodes again immediately.

## `ckecli repair-queue`

Control a queue of repair requests.

### `ckecli repair-queue enable|disable`

Enable/Disable processing repair queue entries.

### `ckecli repair-queue is-enabled`

Show repair queue is enabled or disabled.
This displays `true` or `false`.

### `ckecli repair-queue add OPERATION MACHINE_TYPE ADDRESS [SERIAL]`

Append a repair request to the repair queue.
The repair target is a machine with an IP address `ADDRESS` and a machine type `MACHINE_TYPE`.
The machine should be processed with an operation `OPERATION`.
Optionally, you can specify the machine's serial number `SERIAL` as the fourth argument.

### `ckecli repair-queue list`

List the entries in the repair queue.

### `ckecli repair-queue delete INDEX`

Delete the specified repair queue entry.
This has two meanings: this clears up an old entry if the specified entry has finished and cancels an ongoing entry otherwise.

Unlike the reboot queue, repair queue entries remain in the queue even after they finish.

### `ckecli repair-queue delete-finished`

Delete all finished repair queue entries.
Entries in `succeeded` or `failed` status are deleted.
This displays the index numbers of deleted entries, one per line.

### `ckecli repair-queue delete-unfinished`

Delete all unfinished repair queue entries.
Entries not in `succeeded` or `failed` status are deleted.
This displays the index numbers of deleted entries, one per line.

### `ckecli repair-queue reset-backoff`

Reset `drain_backoff_count` and `drain_backoff_expire` of the entries in repair queue.
Resetting these values makes CKE try to drain machines again immediately.

## `ckecli sabakan`

Control [sabakan integration feature](sabakan-integration.md).

### `ckecli sabakan enable|disable`

Enables/Disables sabakan integration.

The integration will run when:
- It is *not* disabled, and
- URL of sabakan is set with `ckecli sabakan set-url`, and
- Cluster configuration template is set with `ckecli sabakan set-template`.

### `ckecli sabakan is-enabled`

Show sabakan integration is enabled or disabled.
It displays `true` or `false`.

### `ckecli sabakan set-url URL`

Set URL of sabakan.

### `ckecli sabakan get-url`

Show stored URL of sabakan.

### `ckecli sabakan set-template FILE`

Set the cluster configuration template.

The template format is the same as defined in [cluster.md](cluster.md).
The template must have one control-plane node and one non-control-plane node.

Node addresses are ignored.

### `ckecli sabakan get-template`

Get the cluster configuration template.

### `ckecli sabakan set-variables FILE`

Set the query variables to search available machines in sabakan.
`FILE` should contain JSON as described in [sabakan integration](sabakan-integration.md#variables).

### `ckecli sabakan get-variables`

Get the query variables to search available machines in sabakan.

## `ckecli auto-repair`

### `ckecli auto-repair enable|disable`

Enable/Disable [sabakan-triggered automatic repair](sabakan-triggered-repair.md).

### `ckecli auto-repair is-enabled`

Show sabakan-triggered automatic repair is enabled or disabled.
It displays `true` or `false`.

### `ckecli auto-repair set-variables FILE`

Set the query variables to search non-healthy machines in sabakan.
`FILE` should contain JSON as described in [sabakan-triggered automatic repair](sabakan-triggered-repair.md#query).

### `ckecli auto-repair get-variables`

Get the query variables to search non-healthy machines in sabakan.

## `ckecli status`

Report the internal status of the CKE server.

See [schema.md](schema.md#status)

Example:
```json
{"phase":"completed","timestamp":"2009-11-10T23:00:00Z"}
```
