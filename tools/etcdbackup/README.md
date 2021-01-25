etcdbackup
==========

etcdbackup is REST API based etcd backup service.

Usage
-----

Launch etcdbackup by docker as follows:

```console
$ ./etcdbackup --config string
```

etcdbackup listen HTTP protocol to receive request from HTTP clients.
When it receives a request, action for etcd backup and reply the result. 

Configuration file
------------------

etcdbackup reads etcd configurations from a YAML file.


Name         | Type            | Required | Description
----         | ----            | -------- | -----------
`backup-dir` | string          | No       | etcd backup saved directory.  Default: `/etcd-backup/`.
`listen`     | string          | No       | REST API listen address and port.  Default: `0.0.0.0:8080`.
`rotate`     | int             | No       | Keep a number of backup files.  Default: `14`.
`etcd`       | etcdutil.Config | Yes      | etcd parameter defined by [cybozu-go/etcdutil](https://github.com/cybozu-go/etcdutil).

APIs
----

## `GET /api/v1/backup`

List etcd backup files in JSON format.

**Successful response**

- HTTP status code: 200 OK
- HTTP response header: `Content-Type: application/json`
- HTTP response body: etcd backup file list

**Failure responses**

**Example**

```console
$ curl -s -XGET 'localhost:8080/api/v1/backup'
[
    "snapshot-20181220_024105.tar.gz",
    "snapshot-20181220_024204.tar.gz"
]
```

## `GET /api/v1/backup/<FILENAME>`

Retrieve an etcd backup file.

**Successful response**

- HTTP status code: 200 OK
- HTTP response header: `Content-Type: application/gzip`
- HTTP response body: An etcd backup data

**Failure responses**

- FILENAME does not match "snapshot-*.db.gz"

    - HTTP status code: 400 Bad Request

- FILENAME is not found.

    - HTTP status code: 404 Not Found
    
**Example**

```console
$ curl -O -s -XGET 'localhost:8080/api/v1/backup/snapshot-20181220_024105.tar.gz'
```

## `POST /api/v1/backup`

Access etcd server to save snapshot as an etcd backup, then compress with tar archive.

**Successful response**

- HTTP status code: 200 OK
- HTTP response header: `Content-Type: application/json`
- HTTP response body: Status, file name and file size

**Failure responses**

- Fail to request for etcd server, Fail to save an etcd backup into `backup-dir`.

    - HTTP status code: 500 Internal Server Error
    - HTTP response header: `Content-Type: application/json`
    - HTTP response body: Error message

**Example**

```console
$ curl -s -XPOST 'localhost:8080/api/v1/backup'
{
    "message": "backup successfully",
    "filename": "snapshot-20181220_024105.tar.gz",
    "filesize": 125927
}
```

