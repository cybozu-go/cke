#!/bin/sh

sudo -b sh -c "echo \$\$ >/tmp/placemat_pid$$; exec $PLACEMAT output/cluster.yml" >/dev/null 2>&1
sleep 1
PLACEMAT_PID=$(cat /tmp/placemat_pid$$)
echo "placemat PID: $PLACEMAT_PID"

fin() {
    sudo kill $PLACEMAT_PID
    echo "waiting for placemat to terminate..."
    while true; do
        if [ -d /proc/$PLACEMAT_PID ]; then
            sleep 1
            continue
        fi
        break
    done
}
trap fin INT TERM HUP 0

$GINKGO -v
RET=$?

exit $RET
