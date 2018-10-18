etcd
====

CKE bootstraps and maintains an [etcd][] cluster for Kubernetes.

The etcd cluster is not only for Kubernetes, but users can use it
for other applications.

Administration
--------------

First of all, read [Role-based access control][RBAC] for etcd.
Since the etcd cluster has RBAC enabled, you need to be authenticated as `root` to manage it.
The cluster also enables [TLS-based user authentication](https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/authentication.md#using-tls-common-name)

`ckecli etcd root-issue` issues a TLS client certificate for the `root` user that are valid for only 2 hours.  Use it to connect to etcd:

```console
$ CERT=$(ckecli etcd root-issue)
$ echo "$CERT" | jq -r .ca_certificate > /tmp/etcd-ca.crt
$ echo "$CERT" | jq -r .certificate > /tmp/etcd-root.crt
$ echo "$CERT" | jq -r .private_key > /tmp/etcd-root.key
$ export ETCDCTL_API=3
$ export ETCDCTL_CACERT=/tmp/etcd-ca.crt
$ export ETCDCTL_CERT=/tmp/etcd-root.crt
$ export ETCDCTL_KEY=/tmp/etcd-root.key

$ etcdctl --endpoints=CONTROL_PLANE_NODE_IP:2379 member list
```

Application
-----------

### User and key prefix

Kubernetes is authenticated as `kube-apiserver` and its keys are prefixed by `/registry/`.

Other applications should use different user names and prefixes.

To create an etcd user and grant a prefix, use `ckecli` as follows:

```console
$ ckecli etcd user-add USER PREFIX
```

### TLS certificate for a user

Use `ckecli etcd issue` to issue a TLS client certificate for a user.
The command can specify TTL of the certificate long enough for application usage (default is 10 years).

```console
$ CERT=$(ckecli etcd issue -ttl=24h -output=json USER)

$ echo "$CERT" | jq -r .ca_certificate > etcd-ca.crt
$ echo "$CERT" | jq -r .certificate > USER.crt
$ echo "$CERT" | jq -r .private_key > USER.key
```

### Etcd endpoints

Since CKE automatically adds or removes etcd members, applications need
to change the list of etcd endpoints.  To help applications running on
Kubernetes, CKE exports the endpoint list as a Kubernetes resource.

[`Endpoints`][Endpoints] is a Kubernetes resource to list service endpoints.
CKE creates and maintains an `Endpoints` resource named `cke-etcd` in `kube-system` namespace.

To view the contents, use `kubectl` as follows:

```console
$ kubectl -n kube-system get endpoints/cke-etcd -o yaml
```

Backup
------

**TBD**

[etcd]: https://github.com/etcd-io/etcd
[RBAC]: https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/authentication.md
[Endpoints]: https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
