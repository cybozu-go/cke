CKE (Cybozu Kubernetes Engine)
==============================

Usage
-----

`cke [OPTIONS]`

| Option          | Default value            | Description             |
| --------------- | ------------------------ | ----------------------- |
| `-config`       | /etc/cke.yml             | configuration file path |
| `-interval`     | 10m                      | check interval          |
| `-session-ttl`  | 60s                      | leader session's TTL    |

Configuration file
------------------

CKE read etcd configurations from a YAML file.
Parameters are defined by [cybozu-go/etcdutil](https://github.com/cybozu-go/etcdutil), and not shown below will use default values of the etcdutil.

| Name       | Type           | Required | Description                                         |
| ---------- | -------------- | -------- | --------------------------------------------------- |
| `prefix`   | string         | No       | Key prefix of etcd objects.  Default is `/cke/`.    |
