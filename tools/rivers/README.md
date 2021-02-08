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

### Command-line options

```
  -check-interval string
        Interval for health check (default "20s")
  -dial-keep-alive string
        Interval between keep-alive probes (default "15s")
  -dial-timeout string
        Timeout for dial to an upstream server (default "10s")
  -listen string
        Listen address and port (address:port)
  -logfile string
        Log filename
  -logformat string
        Log format [plain,logfmt,json]
  -loglevel string
        Log level [critical,error,warning,info,debug]
  -shutdown-timeout string
        Timeout for server shutting-down gracefully (disabled if specified "0") (default "10s")
  -upstreams string
        Comma-separated upstream servers (addr1:port1,addr2:port2)
```
