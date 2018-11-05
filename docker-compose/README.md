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
* can connect to the internet

### Kubernetes nodes
* Docker
* can be connected from CKE host via SSH
* can connect to the internet

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
Sealed          false
Total Shares    1
Threshold       1
Version         0.11.0
Cluster Name    vault-cluster-f201e8a4
Cluster ID      9fcb601f-2e94-31b1-c715-45a3d8164c63
HA Enabled      false

$ ./bin/ckecli --config=./cke.config leader
963f0ac4cdee
```

## Prepare cluster configuration

Prepare the following file: cluster.yml

```yaml
name: quickstart
nodes:
  - name: node1
    address: <YOUR NODE1 ADDRESS>
    user: <SSH USERNAME>
    control_plane: true
  - name: node2
    address: <YOUR NODE2 ADDRESS>
    user: <SSH USERNAME>
  - name: node3
    address: <YOUR NODE3 ADDRESS>
    user: <SSH USERNAME>
ssh_key: |-
  -----BEGIN RSA PRIVATE KEY-----
  <PUT YOUR SSH PRIVATE KEY TO CONNECT NODES>
  -----END RSA PRIVATE KEY-----
service_subnet: 172.16.0.0/16
pod_subnet: 192.168.0.0/16
dns_servers: ["8.8.8.8"]
options:
  kubelet:
    allow_swap: true
```

See [CKE docs: Cluster configuration](https://github.com/cybozu-go/cke/blob/master/docs/cluster.md).

## Deploy Kubernetes cluster

```console
$ ./bin/ckecli --config=./cke.config cluster set ./cluster.yml
```

## Operate Kubernetes cluster

### setup kubectl

See [Install and Set Up kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

### generate kubectl configuration file

```console
$ ./bin/ckecli --config=./cke.config kubernetes issue > $HOME/.kube
```

### use kubectl

```
$ kubectl get nodes
NAME     STATUS    AGE
node1    Ready     1m
node2    Ready     1m
node3    Ready     1m
```

## Setup CNI plugin

See [How to install CNI Network plugin (Calico)](https://github.com/cybozu-go/cke/wiki/How-to-install-CNI-Network-plugin-(Calico)).
