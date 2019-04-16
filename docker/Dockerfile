# CKE container
FROM quay.io/cybozu/ubuntu:18.04

COPY cke /usr/local/cke/bin/
COPY ckecli /usr/local/cke/bin/
COPY install-tools /usr/local/cke/install-tools

ENV PATH=/usr/local/cke/bin:"$PATH"

USER 10000:10000

ENTRYPOINT ["/usr/local/cke/bin/cke"]
