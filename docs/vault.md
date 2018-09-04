PKI management by HashiCorp [Vault][]
===================================

This document describes how CKE enables TLS connection for etcd/k8s cluster.
All certificates are issued by [Vault][].

Vault configuration
-------------------

CKE requires [Vault][] for issuing certificates with configuration as follows.

### Secret engine `pki`

#### path `cke/ca-server`

Issue certificates for etcd server.

- policy: `cke`
- role: `system`
- CA common name: `server`

#### path `cke/ca-etcd-peer`

Issue certificates for peer connection of the etcd cluster.

- policy: `cke`
- role: `system`
- CA common name: `etcd-peer`

#### path `cke/ca-etcd-client`

Issue certificates for etcd clients such as kube-apiserver.

- policy: `cke`
- role: `system`
- CA common name: `etcd-client`

### approle `cke`

CKE logins to the Vault by approle `cke`. See [ckecli.md][] and [schema.md#vault][].

- policy: `cke`

### policy `cke`

It allows to any operation for path `cke/*`.

```hci
path "cke/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
```

- See more details about PKI secret engine in https://www.vaultproject.io/docs/secrets/pki/index.html

[Vault]: https://www.vaultproject.io/
