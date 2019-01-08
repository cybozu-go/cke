ckecli
======

```console
$ ckecli [--config FILE] <subcommand> args...
```

Option      | Default value         | Description
----------  | --------------------- | -----------
`--config`  | `/etc/cke/config.yml` | config file path
`--version` |                       | show ckecli version

`ckecli cluster set FILE`
-------------------------

Set the cluster configuration.

`ckecli cluster get`
--------------------

Get the cluster configuration.

`ckecli constraints set NAME VALUE`
-----------------------------------

Set a constraint on the cluster configuration.

`NAME` is one of:

- `control-plane-count`
- `minimum-workers`
- `maximum-workers`

`ckecli constraints show`
-------------------------

Show all constraints on the cluster.

`ckecli vault init`
--------------------------

Initialize vault configuration for CKE as described in [vault.md](vault.md).

`ckecli vault config JSON`
--------------------------

Set vault configuration for CKE.
`JSON` is a filename whose body is a JSON object described in [schema.md](schema.md#vault).

If `JSON` is "-", `ckecli` reads from stdin.

`ckecli vault ssh-privkey [--host=HOST] FILE`
---------------------------------------------

Store SSH private key for a host into Vault.  If no HOST is specified, the key will be
used as the default key.

FILE should be a SSH private key file.  If FILE is `-`, the contents are read from stdin.

`ckecli ca set NAME PEM`
------------------------

`NAME` is one of `server`, `etcd-peer`, `etcd-client`, `kubernetes`.

`PEM` is a filename of a x509 certificate.

`ckecli ca get NAME`
--------------------

`NAME` is one of `server`, `etcd-peer`, `etcd-client`, `kubernetes`.

`ckecli leader`
-------------------------

Show the host name of the current leader.

`ckecli history [-n COUNT]`
---------------------------

Show operation history.

`ckecli images`
---------------

List container image names used by `cke`.

`ckecli etcd`
-------------

Control CKE managed etcd.

### `ckecli etcd user-add NAME PREFIX`

This subcommand is for programs to operate etcd server.

Add `NAME` user/role to etcd.

The user can only access under `PREFIX`.

### `ckecli etcd issue [--ttl=TTL] [--output=FORMAT] NAME`

This subcommand is for programs to operate etcd server.

Create a client certificate for user `NAME`.

Option     | Default value | Description
---------- | ------------- | -----------
`--ttl`    | `87600h`      | TTL for client certificate
`--output` | `json`        | output format (`json`,`file`)

### `ckecli etcd root-issue [--output=FORMAT]`

Create client certificate for `root`.

TTL for this certificate is fixed to 2h.

This subcommand is for human to operate etcd server.

Option     | Default value | Description
---------- | ------------- | -----------
`--output` | `json`        | output format (`json`,`file`)

### `ckecli etcd backup list`

List etcd backup files.

### `ckecli etcd backup get BACKUP_NAME`

Download etcd backup file to current directory.

BACKUP_NAME is the name of backup file.

`ckecli kubernetes`
-------------------

Control CKE managed kubernetes.

### `ckecli kubernetes issue [--ttl=TTL]`

Write kubeconfig to stdout.

This config file embeds client certificate and can be used with `kubectl` to connect Kubernetes cluster.

Option  | Default value | Description
------- | ------------- | -----------
`--ttl` | `2h`          | TTL of the client certificate

`ckecli ssh NODE [COMMAND...]`
------------------------------

Connect to the node via ssh.

`NODE` is IP address or hostname of the node to be connected.
The node should be defined in the cluster definition.
The user name defined in the cluster will be used.

If `COMMAND` is specified, it will be executed on the node.

`ckecli scp [-r] [NODE1:]FILE1 ... [NODE2:]FILE2`
-------------------------------------------------

Copy files between hosts via scp.

`NODE` is IP address or hostname of the node.
The node should be defined in the cluster definition.
The user name defined in the cluster will be used.

Option  | Default value | Description
------- | ------------- | -----------
`-r`    | `false`       | Recursively copy entire directories.

`ckecli sabakan`
----------------

Control [sabakan integration feature](sabakan-integration.md).

### `ckecli sabakan set-url URL`

Set URL of sabakan.  This enables sabakan integration.

### `ckecli sabakan get-url`

Show stored URL of sabakan.

### `ckecli sabakan disable`

Disables sabakan integration and removes sabakan URL.

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
