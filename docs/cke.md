CKE (Cybozu Kubernetes Engine)
==============================

Usage
-----

`cke [OPTIONS]`

| Option          | Default value            | Description             |
| --------------- | ------------------------ | ----------------------- |
| `-http`         | 0.0.0.0:10180            | Listen IP:Port number   |
| `-config`       | /etc/cke/config.yml      | configuration file path |
| `-interval`     | 1m                       | check interval          |
| `-session-ttl`  | 60s                      | leader session's TTL    |

Configuration file
------------------

CKE read etcd configurations from a YAML file.
Parameters are defined by [cybozu-go/etcdutil](https://github.com/cybozu-go/etcdutil), and not shown below will use default values of the etcdutil.

| Name       | Type           | Required | Description                                         |
| ---------- | -------------- | -------- | --------------------------------------------------- |
| `prefix`   | string         | No       | Key prefix of etcd objects.  Default is `/cke/`.    |
