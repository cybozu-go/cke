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

<a name="vault"></a>
`vault`
-------

JSON object that has the following fields:

Name        | Required | Type | Description
----------- | -------- | ---- | -----------
`endpoint`  | true  | string | URL of the Vault server.
`ca-cert`   | false | string | x509 certificate in PEM format of the endpoint CA.
`role-id`   | true  | string | AppRole ID to login to Vault.
`secret-id` | true  | string | AppRole secret to login to Vault.

CA certificates
---------------

The following keys store x509 certificates in PEM format.

### `ca/server`

CA that issues TLS server certificates for docker containers.

### `ca/etcd-peer`

CA that issues certificates for client and server authentication between etcd peers.

### `ca/etcd-client`

CA that issues client authentication certificates for etcd clients.

`records`
----------

The next ID of the record formatted as a decimal string.

`records/<16-digit HEX string>`
-------------------------------

Each entry of audit log is stored with this type of key.

The value is JSON defined in [Record](record.md).
