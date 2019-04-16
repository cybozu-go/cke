#!/bin/sh

TARGET="$1"

fin() {
    chmod 600 ./mtest_key
    echo "-------- host1: cke log"
    ./mssh cybozu@${HOST1} sudo journalctl -u cke.service --no-pager
    echo "-------- host2: cke log"
    ./mssh cybozu@${HOST2} sudo journalctl -u cke.service --no-pager
}
trap fin INT TERM HUP 0

$GINKGO -v -focus="${TARGET}" $SUITE_PACKAGE
RET=$?

exit $RET
