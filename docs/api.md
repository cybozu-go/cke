REST API
========

## `GET /health`

Get health information of this CKE instance.

**Successful response**

- HTTP status code: 200 OK
- HTTP response header: `Content-Type: application/json`
- HTTP response body: Health information of this CKE instance. The response is `{"health":"healthy"}`

**Failure response**

- HTTP status code: 500 Internal Server Error
- HTTP response header: `Content-Type: application/json`
- HTTP response body: Health information of this CKE instance. The response is `{"health":"unhealthy"}`

**Example**

```console
$ curl http://localhost:10180/health
{"health":"healthy"}
```

## `GET /version`

Get current CKE version.

**Successful response**

- HTTP status code: 200 OK
- HTTP response header: `Content-Type: application/json`
- HTTP response body: Current CKE version.

**Example**

```console
$ curl http://localhost:10180/version
{"version":"1.15.5"}
```
