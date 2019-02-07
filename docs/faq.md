Frequently Asked Questions
==========================

## kubelet dies saying: failed to get device for dir "/var/lib/kubelet": could not find device ... in cached partitions map

This error happens when the filesystem of `/var/lib/kubelet` is `tmpfs`.

[kubelet][] uses `/var/lib/kubelet` directory to prepare Pod volumes and [local ephemeral storages][les].

To limit and request usage of local ephemeral storages, kubelet has a feature gate called [`LocalStorageCapacityIsolation`][LSI].  When `LocalStorageCapacityIsolation` is enabled (default since Kubernetes 1.10), kubelet tries to identify the underlying block device of `/var/lib/kubelet`.  If `tmpfs` is used for `/var/lib/kubelet`, kubelet dies because there is no underlying block device.

You have several options to workaround the problem:

1. Use filesystem other than `tmpfs`.
2. Specify another directory by `--root-dir` option.
3. Disable the feature gate by adding the following `extra_args` to kubelet.

    ```console
    --feature-gates=LocalStorageCapacityIsolation=disable
    ```

[kubelet]: https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/#options
[les]: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#local-ephemeral-storage
[LSI]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
