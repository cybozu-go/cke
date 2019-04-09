# cke container

# Stage1: build from source
FROM quay.io/cybozu/golang:1.12-bionic AS build

ARG CKE_VERSION=1.13.17

WORKDIR /work

RUN curl -fsSL -o cke.tar.gz https://github.com/cybozu-go/cke/archive/v${CKE_VERSION}.tar.gz \
    && tar -x -z --strip-components 1 -f cke.tar.gz \
    && rm -f cke.tar.gz \
    && go install -mod=vendor ./pkg/cke ./pkg/ckecli

# Stage2: setup runtime container
FROM quay.io/cybozu/ubuntu:18.04

COPY --from=build /go/bin /usr/local/cke/bin
COPY --from=build /work/LICENSE /usr/local/cke/LICENSE
COPY install-tools /usr/local/cke/install-tools

ENV PATH=/usr/local/cke/bin:"$PATH"

USER 10000:10000

ENTRYPOINT ["usr/local/cke/bin/cke"]

