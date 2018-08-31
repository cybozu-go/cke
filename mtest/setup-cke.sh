#!/bin/sh -e

VAULT=/data/vault
CKECLI=/data/ckecli

if [ ! -f /usr/bin/jq ]; then
    echo "please wait; cloud-init will install jq."
    exit 1
fi

run_etcd() {
    sudo systemd-run --unit=my-etcd.service /data/etcd --listen-client-urls=http://0.0.0.0:2379 --advertise-client-urls=http://10.0.0.11:2379 --data-dir /home/cybozu/default.etcd
}

create_ca() {
    ca="$1"
    common_name="$2"
    key="$3"

    $VAULT secrets enable -path $ca -max-lease-ttl=876000h -default-lease-ttl=87600h pki
    $VAULT write -format=json "$ca/root/generate/internal" \
           common_name="$common_name" \
           ttl=876000h format=pem | jq -r .data.certificate > /tmp/ca.pem
    $CKECLI ca set $key /tmp/ca.pem
}

run_vault() {
    sudo systemd-run --unit=my-vault.service /data/vault server -dev -dev-listen-address=0.0.0.0:8200 -dev-root-token-id=cybozu
    sleep 1

    VAULT_TOKEN=cybozu
    export VAULT_TOKEN
    VAULT_ADDR=http://127.0.0.1:8200
    export VAULT_ADDR

    $VAULT auth enable approle
    cat > /home/cybozu/cke-policy.hcl <<'EOF'
path "cke/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
EOF
    $VAULT policy write cke /home/cybozu/cke-policy.hcl
    $VAULT write auth/approle/role/cke policies=cke period=5s
    role_id=$($VAULT read -format=json auth/approle/role/cke/role-id | jq -r .data.role_id)
    secret_id=$($VAULT write -f -format=json auth/approle/role/cke/secret-id | jq -r .data.secret_id)
    cat >/tmp/vault.json <<EOF
{
    "endpoint": "http://10.0.0.11:8200",
    "role-id": "$role_id",
    "secret-id": "$secret_id"
}
EOF
    $CKECLI vault config /tmp/vault.json

    create_ca cke/ca-server "server CA" server
    create_ca cke/ca-etcd-peer "etcd peer CA" etcd-peer
    create_ca cke/ca-etcd-client "etcd client CA" etcd-client
}

install_configs() {
  sudo tee /etc/cke/config.yml >/dev/null <<EOF
endpoints: ["http://10.0.0.11:2379"]
EOF

  mkdir -p $HOME/.kube
  cat >$HOME/.kube/config <<EOF
apiVersion: v1
clusters:
- name: local
  cluster:
    server: http://10.0.0.101:8080
users:
- name: admin
contexts:
- context:
    cluster: local
    user: admin
EOF


}

install_configs

if [ $(hostname) = 'host1' ]; then
    run_etcd
    sleep 1
    run_vault
fi

cat <<EOF

CKE configuration has been initialized.  Run CKE by the following:

    $ /data/cke [-interval <interval>]

Then, use kubectl to manage a kubernetes cluster as:

    $ /data/kubectl api-resources

EOF
