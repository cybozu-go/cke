# CKE container
FROM ghcr.io/cybozu/ubuntu:22.04

COPY cke /usr/local/cke/bin/cke
COPY ckecli /usr/local/cke/bin/ckecli
COPY cke-localproxy /usr/local/cke/bin/cke-localproxy
COPY install-tools /usr/local/cke/install-tools

RUN chmod -R +xr /usr/local/cke

ENV PATH=/usr/local/cke/bin:"$PATH"

USER 10000:10000

ENTRYPOINT ["/usr/local/cke/bin/cke"]
