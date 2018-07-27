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

This file provides following parameters to connect to etcd cluster.

| Name       | Type           | Required | Description                                         |
| ---------- | -------------- | -------- | --------------------------------------------------- |
| `servers`  | list of string | Yes      | List of etcd end point URLs.                        |
| `prefix`   | string         | No       | Key prefix of etcd objects.  Default is `/cke/`.    |
| `username` | string         | No       | Username for etcd authentication.                   |
| `password` | string         | No       | Password for etcd authentication.                   |

