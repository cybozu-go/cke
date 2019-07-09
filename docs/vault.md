PKI management by HashiCorp Vault
=================================

CKE depends on [Vault][] to issue certificates for etcd and k8s.

This document describes how `ckecli vault init` configures Vault.

## Bootstrapping

### Prerequisites

* `approle` auth method need to be enabled as follows.

```console
$ vault auth enable approle
```

### Secret engines

Create following `pki` secret engines and root certificates.
Root certificates need to be registered with `ckecli`.

* `cke/ca-server`: issues etcd server certificates.
* `cke/ca-etcd-peer`: issues certificates for etcd peer connection.
* `cke/ca-etcd-client`: issues client authentication certificates for etcd.
* `cke/ca-kubernetes`: issues Kubernetes certificates.
* `cke/ca-kubernetes-aggregation`: issues certificates used for aggregated API servers.

Additionally, `kv` secret engine version 1 is mounted at `cke/secrets`.

### Secrets in `cke/secrets`

Currently, there are two secrets in `cke/secrets`.

One is `ssh` that holds SSH private keys to logging in to nodes.
Another is `k8s` that holds cipher keys to [encrypt data at rest](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/).

A secret in Vault can keep arbitrary number of key-value pairs.

Keys in `ssh` are node addresses.  Empty key holds the default SSH
private key used if matching key for the host is not found.

Keys in `k8s` are provider names such as `aescbc` or `secretbox`.
Values are JSON data of cipher keys.

### Policy

Create `cke` policy as follows to allow CKE to manage CAs.

```hcl
path "cke/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
```

### AppRole

Create `cke` AppRole to login to Vault as follows:

```console
$ vault write auth/approle/role/cke policies=cke period=1h
```

Read `role-id` and `secret-id` of the `cke` role and configure CKE as follows:

```console
$ VAULT_URL=https://aa.bb.cc.dd:8200
$ role_id=$(vault read -format=json auth/approle/role/cke/role-id | jq -r .data.role_id)
$ secret_id=$(vault write -f -format=json auth/approle/role/cke/secret-id | jq -r .data.secret_id)

$ ckecli vault config - <<EOF
{
    "endpoint": "$VAULT_URL",
    "role-id": "$role_id",
    "secret-id": "$secret_id"
}
EOF
```

## Lifecycle

### Tidy up expired certificates

Expired certificates in cert_store and revoked_certs should be cleaned up by following command:

```console
vault write <target> tidy_cert_store=true tidy_revoked_certs=true
```

CKE executes this command for all pki secret engines periodically.


[Vault]: https://www.vaultproject.io/
