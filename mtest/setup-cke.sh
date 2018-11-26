#!/bin/sh -e

install_apps() {
  sudo cp /data/{vault,cke,ckecli,kubectl,etcd,etcdctl} /opt/bin
  PATH=/opt/bin:$PATH
  export PATH
}

run_etcd() {
    sudo systemd-run --unit=my-etcd.service /data/etcd --listen-client-urls=http://0.0.0.0:2379 --advertise-client-urls=http://10.0.0.11:2379 --data-dir /home/cybozu/default.etcd
}

run_vault() {
    sudo systemd-run --unit=my-vault.service /data/vault server -dev -dev-listen-address=0.0.0.0:8200 -dev-root-token-id=cybozu

    VAULT_TOKEN=cybozu
    export VAULT_TOKEN
    VAULT_ADDR=http://10.0.0.11:8200
    export VAULT_ADDR

    for i in $(seq 10); do
        sleep 1
        if vault status >/dev/null 2>&1; then
            break
        fi
    done

    ckecli vault init

    # admin role need to be created here to generate .kube/config
    vault write cke/ca-kubernetes/roles/admin ttl=2h max_ttl=24h \
           enforce_hostnames=false allow_any_name=true organization=system:masters
}

install_cke_configs() {
  sudo tee /etc/cke/config.yml >/dev/null <<EOF
endpoints: ["http://10.0.0.11:2379"]
EOF
}

install_kubectl_config() {
  vault write -format=json \
    cke/ca-kubernetes/issue/admin common_name=admin exclude_cn_from_sans=true \
    >/tmp/admin.json

  ca="$(jq -r .data.issuing_ca /tmp/admin.json | base64  -w0)"
  crt="$(jq -r .data.certificate /tmp/admin.json | base64  -w0)"
  key="$(jq -r .data.private_key /tmp/admin.json | base64  -w0)"

  mkdir -p $HOME/.kube
  cat >$HOME/.kube/config <<EOF
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ${ca}
    server: https://10.0.0.101:6443
  name: "mtest"
contexts:
- context:
    cluster: "mtest"
    user: admin
  name: default
current-context: default
kind: Config
users:
- name: admin
  user:
    client-certificate-data: ${crt}
    client-key-data: ${key}
EOF
}

install_apps
install_cke_configs

if [ $(hostname) = 'host1' ]; then
    if [ ! -f $HOME/.kube/config ]; then
        run_etcd
        sleep 1
        run_vault
        install_kubectl_config
    fi
fi

cat <<EOF

CKE configuration has been initialized.  Run CKE by the following:

    $ /data/cke [-interval <interval>]

Then, use kubectl to manage a kubernetes cluster as:

    $ /data/kubectl api-resources

EOF
