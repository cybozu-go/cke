ckecli
======

```console
$ ckecli [--config FILE] <subcommand> args...
```

Option     | Default value  | Description
---------- | -------------- | -----------
`--config` | `/etc/cke.yml` | config file path

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

Show all constraints on cluster

`ckecli history [-n COUNT]`
---------------------------

Show operation history.
