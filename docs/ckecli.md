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
