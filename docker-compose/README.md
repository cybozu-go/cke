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

