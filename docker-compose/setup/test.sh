#!/bin/sh -e

ETCDCTL_API=3 etcdctl --endpoints=http://etcd:2379 member list
VAULT_ADDR=http://vault:8200 VAULT_TOKEN=cybozu vault status
ckecli leader
