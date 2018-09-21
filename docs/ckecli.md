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

### `ckecli etcd issue COMMON_NAME [-ttl=TTL]`

Add user and role using `COMMON_NAME`, and issue client certificate to stdout.

`COMMON_NAME` is `common_name` for client certificate and user/role for etcd.

`-ttl` is `TTL` for client certificate, default is `87600h`.

`ckecli kubernetes`
-------------------

Control CKE managed kubernetes.

### `ckecli kubernetes issue COMMON_NAME [-ttl=TTL]`

Issue client certificate to stdout.

`COMMON_NAME` is used for `common_name` and `organization`for client certificate to access kube-apiserver.

`-ttl` is `TTL` for client certificate, default is `87600h`.
