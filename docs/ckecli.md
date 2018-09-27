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

`ckecli vault config JSON`
--------------------------

`JSON` is a filename whose body is a JSON object described in [schema.md](schema.md#vault).

If `JSON` is "-", `ckecli` reads from stdin.

`ckecli ca set NAME PEM`
------------------------

`NAME` is one of `server`, `etcd-peer`, `etcd-client`.

`PEM` is a filename of a x509 certificate.

`ckecli ca get NAME`
--------------------

`NAME` is one of `server`, `etcd-peer`, `etcd-client`.

`ckecli leader`
-------------------------

Show the host name of the current leader.

`ckecli history [-n COUNT]`
---------------------------

Show operation history.

`ckecli etcd`
-------------

Control CKE managed etcd.

### `ckecli etcd user-add COMMON_NAME PREFIX`

This subcommand is for programs to operate etcd server.

Add `COMMON_NAME` user/role to etcd.

The user can only access under `PREFIX`.

`COMMON_NAME` must not have prefix `system:`.

### `ckecli etcd issue [-ttl=TTL] [-output=FORMAT] COMMON_NAME`

This subcommand is for programs to operate etcd server.

Create client certificate for `COMMON_NAME`.

If `COMMON_NAME` user does not exist, execute `$ ckecli etcd user-add COMMON_NAME PREFIX`.

Option      | Default value         | Description
----------  | --------------------- | -----------
`-ttl`      | `87600h`              | TTL for client certificate
`-output`   | `json`                | output format (`json`,`file`)

### `ckecli etcd root-issue [-output=FORMAT]`

Create client certificate for `root`.

TTL for this certificate is fixed to 2h.

This subcommand is for human to operate etcd server.

Option      | Default value         | Description
----------  | --------------------- | -----------
`-output`   | `json`                | output format (`json`,`file`)


`ckecli kubernetes`
-------------------

Control CKE managed kubernetes.

### `ckecli kubernetes issue [-ttl=TTL]`

Write kubeconfig to stdout.

This config file embeds client certificate and can be used with `kubectl` to connect Kubernetes cluster.

Option      | Default value         | Description
----------  | --------------------- | -----------
`-ttl`      | `2h`                  | TTL for client certificate
