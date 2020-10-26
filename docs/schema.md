Key structures in etcd
======================

CKE stores its data into etcd.
This document describes how keys are structured.

Prefix
------

Keys are prefixed by a constant string.
The default prefix is `/cke/`.

`config-version`
----------------

This represents the configuration version of the constructed
Kubernetes cluster.  If this key does not exist, the version
is considered as "1".

See [cluster_overview.md](cluster_overview.md#config-version) for details.

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

| Name        | Required | Type   | Description                                        |
| ----------- | -------- | ------ | -------------------------------------------------- |
| `endpoint`  | true     | string | URL of the Vault server.                           |
| `ca-cert`   | false    | string | x509 certificate in PEM format of the endpoint CA. |
| `role-id`   | true     | string | AppRole ID to login to Vault.                      |
| `secret-id` | true     | string | AppRole secret to login to Vault.                  |

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
---------

The next ID of the record formatted as a decimal string.

`records/<16-digit HEX string>`
-------------------------------

Each entry of audit log is stored with this type of key.

The value is JSON defined in [Record](record.md).

`resource/`
-----------

### `resource/<KIND>[/<NAMESPACE>]/<NAME>`

User defined resource definitions in JSON format.

Non-namespace resources omit `/<NAMESPACE>` part.

`sabakan/`
----------

Configurations for [sabakan integration](sabakan-integration.md).

### `sabakan/disabled`

If this key exists and its value is `true`, sabakan integration is disabled.

### `sabakan/query-variables`

User-specified variables for the GraphQL query.

### `sabakan/template`

This key stores cluster template from which `cluster` will be generated.

The template is JSON formatted [Cluster](cluster.md) data.

### `sabakan/last-revision`

Record the ModRevision of the template used to generate the cluster
configuration.

### `sabakan/url`

Sabakan URL.

`reboots/`
----------

The reboot queue.

### `reboots/write-index`

The next index to write reboot queue entry formatted as a decimal string.

### `reboots/data/<16-digit HEX string>`

Each entry of reboot queue is stored with this type of key.

The value is JSON formatted [RebootQueueEntry](reboot.md#rebootqueueentry).

<a name="status"></a>
`status`
--------

JSON object that has the following fields:

| Name        | Type   | Description                                                                    |
| ----------- | ------ | ------------------------------------------------------------------------------ |
| `phase`     | string | CKE server processing phase represented as a string.                           |
| `timestamp` | string | RFC3339 formatted string of the time when CKE reads the cluster configuration. |
