#!/bin/bash -e

/usr/local/etcd/install-tools
/usr/local/etcd/bin/etcd --config-file=/etc/etcd/etcd.conf.yml
