#!/bin/sh -e

ETCDCTL_API=3 etcdctl --endpoints=http://172.30.0.14:2379 member list
VAULT_ADDR=http://172.30.0.13:8200 VAULT_TOKEN=cybozu vault status
ckecli leader
