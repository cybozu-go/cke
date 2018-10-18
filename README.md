[![GitHub release](https://img.shields.io/github/release/cybozu-go/cke.svg?maxAge=60)][releases]
[![CircleCI](https://circleci.com/gh/cybozu-go/cke.svg?style=svg)](https://circleci.com/gh/cybozu-go/cke)
[![GoDoc](https://godoc.org/github.com/cybozu-go/cke?status.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/cke)](https://goreportcard.com/report/github.com/cybozu-go/cke)

Cybozu Kubernetes Engine
========================

**CKE** (Cybozu Kubernetes Engine) is a distributed service that automates
[Kubernetes][] cluster management.

**Project Status**: Initial development.

Requirements
------------

### CKE requirements

* [etcd][]
* [Vault][]

### Node OS Requirements

* Docker

    Data in Docker volumes must persist between reboots.

* A user who belongs to `docker` group
* SSH access for the user

Planned Features
----------------

* Bootstrapping and life-cycle management.

    CKE can bootstrap a Kubernetes and [etcd][] cluster from scratch.
    CKE can also add or remove nodes to/from the Kubernetes and etcd cluster.

* Managed etcd cluster

    CKE manages an etcd cluster for Kubernetes.
    Other applications may also store their data in the same etcd cluster.
    Backups of etcd data are automatically taken by CKE.

    Details are described in [docs/etcd.md](docs/etcd.md).

* Cluster features:

    * HA control plane.
    * [RBAC][].
    * [CNI][] network plugins.
    * [CoreDNS][] add-on.
    * Node-local DNS cache services.

* Sabakan integration

    CKE can be integrated with [sabakan][], a service that automates physical
    server management, to generate cluster configuration automatically.

    Sabakan is not a requirement; cluster configuration can be supplied
    externally by a YAML file.

* High availability

    CKE stores its configurations in [etcd][] to share them among multiple instances.
    [Etcd][etcd] is also used to elect a leader instance that exclusively controls
    the Kubernetes cluster.

* Operation logs

    To track problems and life-cycle events, CKE keeps operation logs in etcd.

Programs
--------

This repository contains these programs:

* `cke`: the service.
* `ckecli`: CLI tool for `cke`.

To see their usage, run them with `-h` option.

Documentation
-------------

[docs](docs/) directory contains tutorials and specifications.

License
-------

CKE is licensed under MIT license.

[releases]: https://github.com/cybozu-go/cke/releases
[godoc]: https://godoc.org/github.com/cybozu-go/cke
[Kubernetes]: https://kubernetes.io/
[etcd]: https://github.com/etcd-io/etcd
[Vault]: https://www.vaultproject.io
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[CNI]: https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/
[CoreDNS]: https://coredns.io/
[sabakan]: https://github.com/cybozu-go/sabakan
