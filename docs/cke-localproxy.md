# cke-localproxy command reference

`cke-localproxy` is an optional service that runs on the same host as CKE.

It runs `kube-proxy` and a node local DNS service to allow programs running
on the same host to access Kubernetes Services.

## Prerequisites

`cke-localproxy` depends on Docker.
The user account that runs `cke-localproxy` therefore should be granted to use Docker.

In order to run the local DNS service, you may need to disable `systemd-resolved.service`.

To access Kubernetes Services, the host needs to be able to communicate with Kubernetes Pods.

## Configuration

To resolve Service DNS names, configure `/etc/resolv.conf` like this:

```
nameserver 127.0.0.1
search cluster.local
options ndots:3
```

## Synopsis

```
Usage of cke-localproxy:
      --config string       configuration file path (default "/etc/cke/config.yml")
      --interval duration   check interval (default 1m0s)
      --logfile string      Log filename
      --logformat string    Log format [plain,logfmt,json]
      --loglevel string     Log level [critical,error,warning,info,debug]
```
