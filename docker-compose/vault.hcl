# -*- mode: hcl -*-
disable_mlock = true

listener "tcp" {
    address = "0.0.0.0:8200"
    tls_disable = 1
}

storage "etcd" {
  address = "http://etcd:2379"
  etcd_api = "v3"
}

