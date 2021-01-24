rivers
======

Rivers is a simple TCP reverse proxy written in GO.

Usage
-----

Launch rivers by docker as follows:

```console
$ ./rivers
    --listen localhost:6443 \
    --upstreams 10.0.0.100:6443,10.0.0.101:6443,10.0.0.102:6443
```

Rivers starts and waits for TCP connections on address and port specified by `--listen`.
Rivers receives TCP packets, and forwards them to upstream servers specified by `--upstreams`.
The upstream server is selected at random.

Available options are following:

- `--listen`: Listen address and port
- `--upstreams`: Comma-separated upstream servers
- `--shutdown-timeout`: Timeout for server shutting-down gracefully, or disable it if `0` is specified
