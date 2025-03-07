#!/usr/bin/env bash

coil_version=${COIL_VERSION:-2.8.0}

KUSTOMIZE=${KUSTOMIZE:-kustomize}

EXAMPLE_DIR=$(cd $(dirname $0) && pwd)

rm -rf /tmp/work-coil
mkdir -p /tmp/work-coil/
curl -sSfL https://github.com/cybozu-go/coil/archive/v"${coil_version}".tar.gz | tar -C /tmp/work-coil -xzf - --strip-components=1
cd /tmp/work-coil/v2
make certs
${KUSTOMIZE} build > ${EXAMPLE_DIR}/manifests/cni.yaml
rm -rf /tmp/work-coil
