#!/bin/bash -e

function retry() {
  for i in {1..10}; do
    sleep 1
    if "$@"; then
      return 0
    fi
    echo "retry connecting to etcd"
  done
  return $?
}
 
retry curl http://etcd:2379/health

/usr/local/vault/install-tools
/usr/local/vault/bin/vault server -config=/etc/vault/config.hcl
