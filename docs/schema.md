Key structures in etcd
======================

CKE stores its data into etcd.
This document describes how keys are structured.

Prefix
------

Keys are prefixed by a constant string.
The default prefix is `/cke/`.

`cluster`
---------

`cluster` key stores JSON formatted [Cluster](cluster.md) data.

`constraints`
-------------

`constraints` key stores JSON formatted [Constraints](constraints.md) data.

`records`
----------

The next ID of the record formatted as a decimal string.

`records/<16-digit HEX string>`
-------------------------------

Each entry of audit log is stored with this type of key.

The value is JSON defined in [Record](record.md).
