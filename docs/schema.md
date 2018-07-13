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

`records/`
----------

TBD
