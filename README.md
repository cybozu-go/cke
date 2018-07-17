[![GitHub release](https://img.shields.io/github/release/cybozu-go/cke.svg?maxAge=60)][releases]
[![CircleCI](https://circleci.com/gh/cybozu-go/cke.svg?style=svg)](https://circleci.com/gh/cybozu-go/cke)
[![GoDoc](https://godoc.org/github.com/cybozu-go/cke?status.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/cke)](https://goreportcard.com/report/github.com/cybozu-go/cke)

Cybozu Kubernetes Engine
========================

**CKE** (Cybozu Kubernetes Engine) is a distributed service that automates
[Kubernetes][] cluster management.

**Project Status**: Initial development.

Node OS Requirements
--------------------

* Docker
* A user who belongs to `docker` group

Planned Features
----------------

* Bootstrapping and life-cycle management.

    CKE can bootstrap a Kubernetes and [etcd][] cluster from scratch.
    CKE can also add or remove nodes to/from the Kubernetes and etcd cluster.

* Automatic backup for etcd data.

* Cluster features:

    * HA control plane.
    * Self-hosting (except for etcd, which is managed by CKE).
    * CoreDNS add-on.
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
[etcd]: https://github.com/coreos/etcd
[CRI]: https://github.com/kubernetes/community/blob/master/contributors/devel/container-runtime-interface.md
[sabakan]: https://github.com/cybozu-go/sabakan
