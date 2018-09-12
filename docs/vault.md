PKI management by HashiCorp Vault
=================================

CKE requires [Vault][] to issue certificates for etcd and k8s.

This document describes how to configure Vault for CKE.

## Secret engines

Create following `pki` secret engines and root certificates.
Root certificates need to be registered with `ckecli`.

* `cke/ca-server`: issues etcd server certificates.
* `cke/ca-etcd-peer`: issues certificates for etcd peer connection.
* `cke/ca-etcd-client`: issues client authentication certificates for etcd.
* `cke/ca-kubernetes`: issues Kubernetes certificates.

Example:
```console
$ vault secrets enable -path cke/ca-server \
    -max-lease-ttl=876000h -default-lease-ttl=87600h pki

$ vault write -format=json cke/ca-server/root/generate/internal \
    common_name='CKE server CA' ttl=876000h | \
    jq -r .data.certificate > ca-server.crt

$ ckecli ca set server ca-server.crt
```

## Policy

Create `cke` policy as follows to allow CKE to manage CAs.

```hcl
path "cke/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
```

## AppRole

Create `cke` AppRole to login to Vault.

```console
$ vault auth enable approle
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

[Vault]: https://www.vaultproject.io/
