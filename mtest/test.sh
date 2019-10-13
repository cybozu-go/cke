#!/bin/sh

TARGET="$1"

$GINKGO -v -focus="${TARGET}" $SUITE_PACKAGE
RET=$?

exit $RET
