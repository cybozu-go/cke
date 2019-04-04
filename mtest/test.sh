#!/bin/sh

TARGET="$1"

sudo -b sh -c "echo \$\$ >/tmp/placemat_pid$$; exec $PLACEMAT output/cluster.yml" >/dev/null 2>&1
sleep 1
PLACEMAT_PID=$(cat /tmp/placemat_pid$$)
echo "placemat PID: $PLACEMAT_PID"

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
