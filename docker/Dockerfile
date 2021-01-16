# CKE container
FROM quay.io/cybozu/ubuntu:20.04

COPY cke /usr/local/cke/bin/cke
COPY ckecli /usr/local/cke/bin/ckecli
COPY install-tools /usr/local/cke/install-tools

RUN chmod -R +xr /usr/local/cke

ENV PATH=/usr/local/cke/bin:"$PATH"

USER 10000:10000

ENTRYPOINT ["/usr/local/cke/bin/cke"]
