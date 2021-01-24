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

Furthermore, these endpoints address records are registered at CoreDNS.

The domain name is `cke-etcd.kube-system.svc.<cluster-domain>`.

Backup
------

There are two ways to take backups of CKE-managed etcd.

1. Use `ckecli etcd local-backup` to take and save a snapshot of etcd locally.
2. Configure CronJob to save snapshots in a filesystem given through a PVC.

The former is described in [ckecli.md](ckecli.md##ckecli-etcd-local-backup).

The rest are the descriptions of the latter.

When the etcd backup is enabled on cluster configuration, [etcdbackup](../tools/etcdbackup) `Pod` and `CronJob` are deployed on Kubernetes cluster to manage compressed etcd snapshot.
Before enable etcd backup, you need to create `PersistentVolume` and `PersistentVolumeClaim` to store the backup data.

1. Deploy `PersistentVolume` and `PersistentVolumeClaim`. This is example of using local persistent volume in particular node.

    ```yaml
    ---
    kind: StorageClass
    apiVersion: storage.k8s.io/v1
    metadata:
      name: local-storage
    provisioner: kubernetes.io/no-provisioner
    volumeBindingMode: WaitForFirstConsumer
    ---
    apiVersion: v1
    kind: PersistentVolume
    metadata:
      name: etcdbackup-pv
    spec:
      capacity:
        storage: 2Gi
      accessModes:
      - ReadWriteOnce
      persistentVolumeReclaimPolicy: Retain
      storageClassName: local-storage
      local:
        path: /mnt/disks/etcdbackup
      nodeAffinity:
        required:
          nodeSelectorTerms:
          - matchExpressions:
            - key: kubernetes.io/hostname
              operator: In
              values:
              - 10.0.0.101
    ---
    kind: PersistentVolumeClaim
    apiVersion: v1
    metadata:
      name: etcdbackup-pvc
      namespace: kube-system
    spec:
      accessModes:
      - ReadWriteOnce
      resources:
        requests:
          storage: 2Gi
      storageClassName: local-storage
    ```

2. Enable etcd backup in `cluster.yml` and set cron schedule. For example:

    ```yaml
    options:
      kubelet:
        extra_binds:            # Bind mounts local directory for local persistent volume
        - source: /mnt/disks
          destination: /mnt/disks
          read_only: false
    etcd_backup:
      enabled: enable           # Enable etcd backup
      pvc_name: etcdbackup-pvc  # Make sure this name is same as `PersistentVolumeClaim` name.
      schedule: "0 * * * *"     # Cron job format
      rotate: 14                # Keep a number of backups
    ```

3. Run `ckecli cluster set cluster.yml` to deploy etcd backup `CronJob`.
4. You can find etcd backups in persistent volume after etcd backup `Job` is completed.

    ```console
    $ kubectl get job -n kube-system
    NAME                    COMPLETIONS   DURATION   AGE
    etcdbackup-1545803760   1/1           7s         2m23s
    
    $ ckecli etcd backup list
    ["snapshot-20181226_054710.db.gz"]
    
    $ ls -l /mnt/disks/etcdbackup/
    -rw-r--r--. 1 root root 506231 Dec 26 05:47 snapshot-20181226_054710.db.gz
    ...
    ```

5. You can download it too.

    ```console
    $ ckecli etcd backup get snapshot-20181226_054710.db.gz
    ```

[etcd]: https://github.com/etcd-io/etcd
[RBAC]: https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/authentication.md
[Endpoints]: https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
[PersistentVolume]: https://kubernetes.io/docs/concepts/storage/persistent-volumes
