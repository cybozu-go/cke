#!/bin/sh

TARGET="$1"

$GINKGO -v -focus="${TARGET}" .
RET=$?

exit $RET
