# Demonstration with `docker compose` and Vagrant

## Overview

This demonstration gets you a three-node Kubernetes cluster installed by CKE.

**Be warned that `etcd` and `vault` deployed by this example is not durable nor secure.**
Use this only for testing and development.

## Requirements

### CKE host

* git
* Docker
* Docker Compose
* VirtualBox
* Vagrant

## Setup CKE

Follow the steps to setup CKE with `docker compose`.

```console
$ git clone https://github.com/cybozu-go/cke.git
$ cd ./cke/example/
$ mkdir bin
$ mkdir etcd-data
$ docker compose up -d
```

`bin` is the directory where the cli tools are installed.
`etcd-data` is the directory where the data of etcd is stored.

You will be able to see that the following containers are running.

```console
$ docker ps
CONTAINER ID        IMAGE                      COMMAND                  CREATED             STATUS              PORTS                                  NAMES
844ea90ab7b5        quay.io/cybozu/cke:1.15    "/entrypoint.sh"         12 seconds ago      Up 10 seconds                                              cke
9617f2dc36c5        quay.io/cybozu/vault:1.1   "/entrypoint.sh"         14 seconds ago      Up 12 seconds       0.0.0.0:8200-8201->8200-8201/tcp       vault
7140fa308dc3        quay.io/cybozu/etcd:3.4    "/entrypoint.sh"         16 seconds ago      Up 14 seconds       0.0.0.0:2379-2380->2379-2380/tcp       etcd
```

## Setup node VMs

In this demonstration, Kubernetes Cluster is deployed on 3 Virtual Machines.

Follow the steps to setup the VMs with Vagrant.

```console
$ vagrant up
```

After a few minutes you will be able to log in to the VM via ssh.

```console
$ vagrant ssh worker-1
```

## Deploying Kubernetes Cluster

### Registering SSH private-key

Register SSH private-key to log in to the VMs.

```console
$ ./bin/ckecli --config=./cke.config vault ssh-privkey ~/.vagrant.d/insecure_private_key
```

## Declare Kubernetes Cluster Configuration

Declares the number of control planes of Kubernetes cluster and configuration.

```console
$ ./bin/ckecli --config=./cke.config constraints set control-plane-count 1
$ ./bin/ckecli --config=./cke.config cluster set ./cke-cluster.yml
```

## Checking the logs

Once the cluster configuration is set, CKE will soon install Kubernetes.

You can see the operation history with the following command.

```console
$ ./bin/ckecli --config=./cke.config history -f
```

You can also see the logs of CKE.

```console
$ docker logs cke -f
```

CKE will finish installation of Kubernetes components in a few minutes.

## Operating Kubernetes cluster

### setup kubectl

See [Install and Set Up kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

### Issuing configuration file of kubectl

You can get a configuration file of kubectl to access Kubernetes cluster with the following command.

```console
$ ./bin/ckecli --config=./cke.config kubernetes issue > .kubeconfig
$ KUBECONFIG=$(pwd)/.kubeconfig
$ export KUBECONFIG
```

## Setup CNI plugin

CKE itself does not install any network plugins.
To implement the [Kubernetes networking model](https://kubernetes.io/docs/concepts/cluster-administration/networking/), you have to install [a plugin](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/).

You can use Cilium as one of the CNI plugins.

See [Cilium Documentation](https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/) for details.

After a few minutes, Kubernetes cluster will become ready.

```console
$ kubectl get nodes
NAME            STATUS   ROLES    AGE     VERSION
192.168.1.101   Ready    <none>   7h29m   v1.15.3
192.168.1.102   Ready    <none>   7h29m   v1.15.3
192.168.1.103   Ready    <none>   7h29m   v1.15.3
```
