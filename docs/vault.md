PKI management by HashiCorp [Vault][]
===================================

CKE requires [Vault][] for issuing certificates with configuration as follows.

This document describes how CKE enables TLS connection for etcd/k8s cluster.
All certificates are issued by [Vault][].

## Secret engine `pki`

Create following `pki` secret engines.

### path `cke/ca-server`

Issue certificates for etcd server.

Create `system` role with these options:

- `allow_any_name=true`
- `client_flag=false`

### path `cke/ca-etcd-peer`

Issue certificates for peer connection of the etcd cluster.

Create `system` role with these options:

- `allow_any_name=true`

### path `cke/ca-etcd-client`

Issue certificates for etcd clients such as kube-apiserver.

Create `system` role with these options:

- `allow_any_name=true`
- `server_flag=false`

## policy `cke`

Create `cke` policy as follows to allow CKE to issue certificates:

```hcl
path "cke/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
```

## approle `cke`

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
