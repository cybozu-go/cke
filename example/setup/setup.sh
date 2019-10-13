#!/bin/bash
set -e

function retry() {
  for i in {1..10}; do
    sleep 1
    if "$@"; then
      return 0
    fi
    echo "retry"
  done
  return $?
}

export VAULT_ADDR=http://172.30.0.13:8200
export VAULT_TOKEN=cybozu
export ETCDCTL_API=3

# wait for preparation of vault
retry curl ${VAULT_ADDR}/v1/sys/health

res=$(curl ${VAULT_ADDR}/v1/sys/health)
initialized=$(echo ${res} | jq -r .initialized)
sealed=$(echo ${res} | jq -r .sealed)

if [ ${initialized} = "true" ]; then
  if [ ${sealed} = "true" ]; then
    # if vault is initialized and sealed, only unseal
    unseal_key=$(etcdctl --endpoints=http://172.30.0.14:2379 get --print-value-only boot/vault-unseal-key)
    curl -XPUT http://172.30.0.13:8200/v1/sys/unseal -d "{\"key\": ${unseal_key}}"
  fi
  exit 0
fi

# initialize vault
res=$(vault operator init -format=json -key-shares=1 -key-threshold=1)
unseal_key=$(echo ${res} | jq .unseal_keys_b64[0])
root_token=$(echo ${res} | jq -r .root_token)
export VAULT_TOKEN=${root_token}

# store unseal key and root token to etcd
etcdctl --endpoints=http://172.30.0.14:2379 put boot/vault-unseal-key ${unseal_key}
etcdctl --endpoints=http://172.30.0.14:2379 put boot/vault-root-token ${root_token}

# unseal vault
curl -XPUT http://172.30.0.13:8200/v1/sys/unseal -d "{\"key\": ${unseal_key}}"

# setup vault
vault audit enable file file_path=stdout
vault policy write admin /opt/setup/admin-policy.hcl
vault policy write cke /opt/setup/cke-policy.hcl
vault auth enable approle

# setup approle for cke
vault write auth/approle/role/cke policies=cke period=1h
r=$(vault read -format=json auth/approle/role/cke/role-id)
s=$(vault write -f -format=json auth/approle/role/cke/secret-id)
role_id=$(echo ${r} | jq -r .data.role_id)
secret_id=$(echo ${s} | jq -r .data.secret_id)
a=$(vault write -f -format=json auth/approle/login role_id=${role_id} secret_id=${secret_id})
approle_token=$(echo ${a} | jq -r .auth.client_token)

# register information for connecting to vault to CKE
echo "{\"endpoint\": \"http://172.30.0.13:8200\", \"role-id\": \"${role_id}\", \"secret-id\": \"${secret_id}\"}" | ckecli vault config -

ckecli vault init
