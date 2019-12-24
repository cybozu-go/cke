ckecli command reference
========================

```console
$ ckecli [--config FILE] <subcommand> args...
```

| Option      | Default value         | Description         |
| ----------- | --------------------- | ------------------- |
| `--config`  | `/etc/cke/config.yml` | config file path    |
| `--version` |                       | show ckecli version |

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
- `minimum-workers`
- `maximum-workers`

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

### `ckecli etcd backup list`

List etcd backup files.

### `ckecli etcd backup get BACKUP_NAME`

Download etcd backup file to current directory.

BACKUP_NAME is the name of backup file.

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

## `ckecli sabakan`

Control [sabakan integration feature](sabakan-integration.md).

### `ckecli sabakan enable|disable`

Enables/Disables sabakan integration.

The integration will run when:
- It is *not* disabled, and
- URL of sabakan is set with `ckecli sabakan set-url`, and
- Cluster configuration template is set with `ckecli sabakan set-template`.

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

Set the query variables to search machines in sabakan.
`FILE` should contain JSON as described in [sabakan integration](sabakan-integration.md#variables).

### `ckecli sabakan get-variables`

Get the query variables to search machines in sabakan.

## `ckecli status`

Report the internal status of the CKE server.

See [schema.md](schema.md#status)

Example:
```json
{"phase":"completed","timestamp":"2009-11-10T23:00:00Z"}
```
