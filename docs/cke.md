cke command reference
=====================

Usage
-----

`cke [OPTIONS]`

```console
$ cke -h
      --config string        configuration file path (default "/etc/cke/config.yml")
      --debug-sabakan        debug sabakan integration
      --http string          <Listen IP>:<Port number> (default "0.0.0.0:10180")
      --interval string      check interval (default "1m")
      --logfile string       Log filename
      --logformat string     Log format [plain,logfmt,json]
      --loglevel string      Log level [critical,error,warning,info,debug]
      --session-ttl string   leader session's TTL (default "60s")
```

Configuration file
------------------

CKE read etcd configurations from a YAML file.
Parameters are defined by [cybozu-go/etcdutil](https://github.com/cybozu-go/etcdutil), and not shown below will use default values of the etcdutil.

Name       | Type    | Required | Description
---------- | ------- | -------- | -----------
`prefix`   | string  | No       | Key prefix of etcd objects.  Default is `/cke/`.
