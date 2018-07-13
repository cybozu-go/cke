ckecli
=========

Usage
-----

```console
$ ckecli [--config FILE] <subcommand> args...
```

Option     | Default value            | Description
------     | -------------            | -----------
`--config`   | `/etc/cke.yml`           | config file path

`ckecli cluster set FILE`
-------------

Set cluster configuration

`ckecli cluster get`
-------------

Get cluster configuration

`ckecli constraints set NAME VALUE`
-------------

Set constraints on cluster

`NAME` is one of:

- `control-plane-count`
- `minimum-workers`
- `maximum-workers`

`ckecli constraints show`
-------------

Show all constraints on cluster

`ckecli history [COUNT]`
-------------

Show operation history

Option     | Default value  | Description
------     | -------------  | -----------
`COUNT`    | `0`              | Maximum number of the history records
