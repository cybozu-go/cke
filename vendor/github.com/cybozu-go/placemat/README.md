[![GitHub release](https://img.shields.io/github/release/cybozu-go/placemat.svg?maxAge=60)][releases]
[![CircleCI](https://circleci.com/gh/cybozu-go/placemat.svg?style=svg)](https://circleci.com/gh/cybozu-go/placemat)
[![GoDoc](https://godoc.org/github.com/cybozu-go/placemat?status.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/placemat)](https://goreportcard.com/report/github.com/cybozu-go/placemat)

Placemat
========

Placemat is a tool to simulate data center networks and servers using [rkt][] Pods,
QEMU/KVM virtual machines, and Linux networking stacks.  Placemat can simulate
virtually *any* kind of network topologies to help tests and experiments for software
usually used in data centers.

Features
--------

* No daemons

    Placemat is a single binary executable.  It just builds networks and
    virtual machines when it starts, and destroys them when it terminates.
    This simplicity makes placemat great for a continuous testing tool.

* Declarative YAML

    Networks, virtual machines, and other kind of resources are defined
    in YAML files in a declarative fashion.  Users need not mind the order
    of creation and/or destruction of resources.

* Virtual BMC for IPMI power management

    Power on/off/reset of VMs can be done by [IPMI][] commands.
    See [virtual BMC](docs/virtual_bmc.md) for details.

* Automation

    Placemat supports [cloud-init][] and [ignition][] to automate
    virtual machine initialization.  Files on the host machine can be
    exported to guests as a [VVFAT drive](https://en.wikibooks.org/wiki/QEMU/Devices/Storage).
    QEMU disk images can be downloaded from remote HTTP servers.

    All of these help implementation of fully-automated tests.

* UEFI

    Not only traditional BIOS, but placemat VMs can be booted in UEFI
    mode if [OVMF][] is available.

Usage
-----

This project provides these commands:

* `placemat` is the main tool to build networks and virtual machines.
* `placemat-connect` is a utility to connect to VM serial console.

### placemat command

`placemat` reads all YAML files specified in command-line arguments,
then creates resources defined in YAML.  To destroy, just kill the
process (by sending a signal or Control-C).

```console
$ placemat [OPTIONS] YAML [YAML ...]

Options:
  -graphic
        run QEMU with graphical console
  -run-dir string
        run directory (default "/tmp")
  -cache-dir string
        directory for cache data.
  -data-dir string
        directory to store data (default "/var/scratch/placemat")
  -debug
        show QEMU's and Pod's stdout and stderr        
  -force
        force run with removal of garbage
```

If `-cache-dir` is not specified, the default will be `/home/${SUDO_USER}/placemat_data`
if `sudo` is used for `placemat`.  If `sudo` is not used, cache directory will be
the same as `-data-dir`.
`-force` is used for forced run. Remaining garbage, for example virtual networks, mounts, socket files will be removed.

### placemat-connect command

If placemat starts without `-graphic` option, VMs will have no graphic console.
Instead, they have serial consoles exposed via UNIX domain sockets.

`placemat-connect` is a tool to connect to the serial console.

```console
$ placemat-connect [-run-dir=/tmp] your-vm-name

Options:
  -run-dir
        the directory specified for placemat by -run-dir.
```

**To exit** from the console, press Ctrl-Q, Ctrl-X in this order.

Getting started
---------------

### Prerequisites

- [QEMU][]
- [OVMF][] for UEFI.
- [picocom](https://github.com/npat-efault/picocom) for `placemat-connect`
- [rkt][] for `Pod` resource.

For Ubuntu or Debian, you can install them as follows:

```console
$ sudo apt-get update
$ sudo apt-get install qemu-system-x86 qemu-utils ovmf picocom
```

As to rkt, obtain a deb (or rpm) package then install it as follows:

```console
$ wget https://github.com/rkt/rkt/releases/download/v1.30.0/rkt_1.30.0-1_amd64.deb
$ sudo dpkg -i rkt_1.30.0-1_amd64.deb
```

### Install placemat

Install `placemat` and `placemat-connect`:

```console
$ go get -u github.com/cybozu-go/placemat/cmd/placemat
$ go get -u github.com/cybozu-go/placemat/cmd/placemat-connect
```

### Run examples

See [examples](examples) how to write YAML files.

To launch placemat from YAML files, run it with `sudo` as follows:

```console
$ sudo $GOPATH/bin/placemat cluster.yml
```

To connect to a serial console of a VM, use `placemat-connect`:

```console
$ sudo $GOPATH/bin/placemat-connect VM
```

This will launch `picocom`.  To exit, type `Ctrl-Q`, then `Ctrl-X`.

Specification
-------------

See specifications under [docs directory](docs/).

License
-------

MIT

[releases]: https://github.com/cybozu-go/placemat/releases
[godoc]: https://godoc.org/github.com/cybozu-go/placemat
[cloud-init]: http://cloudinit.readthedocs.io/en/latest/index.html
[ignition]: https://coreos.com/ignition/docs/latest/
[QEMU]: https://www.qemu.org/
[OVMF]: https://github.com/tianocore/tianocore.github.io/wiki/OVMF
[rkt]: https://coreos.com/rkt/
[IPMI]: https://en.wikipedia.org/wiki/Intelligent_Platform_Management_Interface
