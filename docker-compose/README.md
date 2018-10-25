# Setup scripts for CKE with Docker Compose

## Requirements

* Docker
* Docker Compose

## Usage

* create data directories
```console
mkdir bin
sudo chown 10000:10000 bin
mkdir etcd-data
sudo chown 10000:10000 etcd-data
```

* start containers
```console
docker-compose up
```

* use cli tools
```console
ETCDCTL_API=3 ./bin/etcdctl --endpoints=http://localhost:2379 member list
VAULT_ADDR=http://localhost:8200 VAULT_TOKEN=cybozu ./bin/vault status
./bin/ckecli --config=./cke.config history
```
