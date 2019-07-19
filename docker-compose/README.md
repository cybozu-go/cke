# Quickstart for CKE

## Overview

This quickstart gets you a single host CKE and Kubernetes cluster.

The CKE deployed by this quickstart is not high availability, and does not use TLS to connect etcd.
You can use this CKE for testing and development.

## Requirements

### CKE host

* git
* Docker
* Docker Compose
* VirtualBox
* Vagrant

## Setup CKE host

### get CKE

```console
$ git clone https://github.com/cybozu-go/cke.git
```

### create directories

```console
$ cd ./cke/docker-compose
$ mkdir bin
$ mkdir etcd-data
```

`bin` is the directory where the cli tools are installed.
`etcd-data` is the directory where the data of etcd is stored.

### start CKE

```console
$ docker-compose up
```

### check status

```console
$ ls ./bin
ckecli  etcdctl  vault

$ ETCDCTL_API=3 ./bin/etcdctl --endpoints=http://localhost:2379 member list
f24244a5c413e9f5, started, etcd0, http://etcd:2380, http://etcd:2379

$ VAULT_ADDR=http://localhost:8200 VAULT_TOKEN=cybozu ./bin/vault status
Key             Value
---             -----
Seal Type       shamir
Initialized     true
Sealed          false
Total Shares    1
Threshold       1
Version         1.1.2
Cluster Name    vault-cluster-265801b6
Cluster ID      4cfb9202-f2d5-ab59-16e4-47a9897a468e
HA Enabled      false

$ ./bin/ckecli --config=./cke.config leader
963f0ac4cdee
```

## Setup nodes

```console
$ vagrant up
```

## Set SSH private-key

```console
$ ./bin/ckecli --config ./cke.config vault ssh-privkey ~/.vagrant.d/insecure_private_key
```

## Deploy Kubernetes cluster

```console
$ ./bin/ckecli --config ./cke.config constraints set minimum-workers 2
$ ./bin/ckecli --config ./cke.config constraints set control-plane-count 1
$ ./bin/ckecli --config=./cke.config cluster set ./cke-cluster.yml
```

## Ckeck the logs

```console
$ docker logs cke -f
```

```console
$ ./bin/ckecli --config ./cke.config history -f
```

## Operate Kubernetes cluster

### setup kubectl

See [Install and Set Up kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

### generate kubectl configuration file

```console
$ ./bin/ckecli --config=./cke.config kubernetes issue > $HOME/.kube/config
```

### use kubectl

```
$ kubectl get node   
NAME            STATUS     ROLES    AGE     VERSION
192.168.1.101   NotReady   <none>   5m11s   v1.14.1
192.168.1.102   NotReady   <none>   5m12s   v1.14.1
192.168.1.103   NotReady   <none>   5m12s   v1.14.1
```

## Setup CNI plugin

See [How to install CNI Network plugin (Calico)](https://github.com/cybozu-go/cke/wiki/How-to-install-CNI-Network-plugin-(Calico)).
