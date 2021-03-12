Logging
=======

CKE logs
--------

`cke` is built on [github.com/cybozu-go/well][well] framework that provides [standard logging options](https://github.com/cybozu-go/well#command-line-options).

In addition, CKE records recent important operations in etcd.  Use [`ckecli history`](ckecli.md) to view them.

Kubernetes programs
-------------------

CKE runs Kubernetes programs such as `apiserver` or `kubelet` by `docker run --log-driver=journald`
to send their logs to `journald`.

To view logs of a program, use `journalctl` as follows:

```console
$ sudo journalctl CONTAINER_NAME=kubelet
```

As kube-apiserver uses both stderr and stdout, filter its logs as follows.

To view general logs of a apiserver, use `journalctl` as follows:

```console
$ sudo journalctl CONTAINER_NAME=kube-apiserver -p 3
```

You can send the audit log of a apiserver to either of journal log or file.

If the log is sent to journal log, you can see it using `journalctl` as follows:

```console
$ sudo journalctl CONTAINER_NAME=kube-apiserver -p 6..6
```

Container names are defined in [op/constants.go](../op/constants.go).

Ref: https://docs.docker.com/config/containers/logging/journald/#retrieve-log-messages-with-journalctl

[well]: https://github.com/cybozu-go/well
