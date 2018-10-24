#!/bin/bash
set -e

export VAULT_ADDR=http://vault:8200
export VAULT_TOKEN=cybozu

res=$(vault operator init -format=json -key-shares=1 -key-threshold=1)
unseal_key=$(echo ${res} | jq .unseal_keys_b64[0])
root_token=$(echo ${res} | jq -r .root_token)

curl -XPUT http://vault:8200/v1/sys/unseal -d "{\"key\": ${unseal_key}}"

export VAULT_TOKEN=${root_token}

vault audit enable file file_path=stdout

vault policy write admin /opt/setup/admin-policy.hcl
vault policy write cke /opt/setup/cke-policy.hcl
vault auth enable approle

vault write auth/approle/role/cke policies=cke period=1h
r=$(vault read -format=json auth/approle/role/cke/role-id)
s=$(vault write -f -format=json auth/approle/role/cke/secret-id)
role_id=$(echo ${r} | jq -r .data.role_id)
secret_id=$(echo ${s} | jq -r .data.secret_id)

echo "{\"endpoint\": \"http://vault:8200\", \"role-id\": \"${role_id}\", \"secret-id\": \"${secret_id}\"}" | ckecli vault config -
a=$(vault write -f -format=json auth/approle/login role_id=${role_id} secret_id=${secret_id})
approle_token=$(echo ${a} | jq -r .auth.client_token)

function create_ca(){
  ca=$1
  common_name=$2
  key=$3

  vault secrets enable -path ${ca} -max-lease-ttl=876000h -default-lease-ttl=87600h pki
  s=$(VAULT_TOKEN=${approle_token} vault write -format=json ${ca}/root/generate/internal common_name=${common_name} ttl=876000h format=pem)
  echo ${s} | jq -r .data.certificate > /tmp/${key}
  ckecli ca set ${key} /tmp/${key}
}

create_ca "cke/ca-server" "server-CA" "server"
create_ca "cke/ca-etcd-peer" "etcd-peer-CA" "etcd-peer"
create_ca "cke/ca-etcd-client" "etcd-client-CA" "etcd-client"
create_ca "cke/ca-kubernetes" "kubernetes-CA" "kubernetes"

#export ETCDCTL_API=3
#etcdctl --endpoints=etcd:2379 put boot/vault-unseal-key ${unseal_key}
#etcdctl --endpoints=etcd:2379 put boot/vault-root-token ${root_token}


sleep infinity
