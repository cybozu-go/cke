#!/bin/sh -ex

. $(dirname $0)/env-sonobuoy

RET=0
delete_instance() {
  if [ $RET -ne 0 ]; then
    # do not delete GCP instance upon test failure to help debugging.
    return
  fi
  $GCLOUD compute instances delete --zone ${ZONE} \
    ${INSTANCE_NAME}-0 \
    ${INSTANCE_NAME}-1 \
    ${INSTANCE_NAME}-2 \
    ${INSTANCE_NAME}-3 || true
}

delete_instance
$GCLOUD compute instances create ${INSTANCE_NAME}-0 \
  --zone ${ZONE} \
  --machine-type ${MACHINE_TYPE_SONOBUOY} \
  --image vmx-enabled \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE} \
  --metadata shutdown-at=$(date -Iseconds -d+4hours)

ssh-keygen -t rsa -f gcp_rsa -C cybozu -N ''
sed -e "s#PUBLIC_KEY#$(cat gcp_rsa.pub)#g" $(dirname $0)/../sonobuoy/worker.cfg > /tmp/worker.cfg

for i in $(seq 3); do
  $GCLOUD compute instances create ${INSTANCE_NAME}-${i} \
    --zone ${ZONE} \
    --machine-type ${MACHINE_TYPE_WORKER} \
    --image-project cos-cloud \
    --image-family cos-stable \
    --boot-disk-type ${DISK_TYPE} \
    --boot-disk-size ${BOOT_DISK_SIZE} \
    --metadata-from-file user-data=/tmp/worker.cfg \
    --metadata shutdown-at=$(date -Iseconds -d+4hours),cos-update-strategy=update_disabled
done

trap delete_instance INT QUIT TERM 0

for i in $(seq 0 3); do
  for j in $(seq 300); do
    if $GCLOUD compute ssh --zone=${ZONE} cke@${INSTANCE_NAME}-${i} --ssh-key-file=gcp_rsa --command=date 2>/dev/null; then
      break
    fi
    sleep 1
  done
done

# Register SSH key
$GCLOUD compute scp --zone=${ZONE} ./gcp_rsa cybozu@${INSTANCE_NAME}-0:

cat >run.sh <<EOF
#!/bin/sh -ex

# Run sonobuoy
GOPATH=\$HOME/go
export GOPATH
GO111MODULE=on
export GO111MODULE
PATH=/usr/local/go/bin:\$GOPATH/bin:\$PATH
export PATH

git clone https://github.com/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME} \
    \$HOME/go/src/github.com/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}
cd \$HOME/go/src/github.com/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}
git checkout -qf ${CIRCLE_SHA1}

cd sonobuoy
make setup
make run
make sonobuoy
mv sonobuoy.tar.gz /tmp
EOF
chmod +x run.sh

WORKER1_ADDRESS=$($GCLOUD compute instances describe ${INSTANCE_NAME}-1 --zone ${ZONE} --format='get(networkInterfaces[0].networkIP)')
WORKER2_ADDRESS=$($GCLOUD compute instances describe ${INSTANCE_NAME}-2 --zone ${ZONE} --format='get(networkInterfaces[0].networkIP)')
WORKER3_ADDRESS=$($GCLOUD compute instances describe ${INSTANCE_NAME}-3 --zone ${ZONE} --format='get(networkInterfaces[0].networkIP)')

sed -e "s|@WORKER1_ADDRESS@|${WORKER1_ADDRESS}|" \
  -e "s|@WORKER2_ADDRESS@|${WORKER2_ADDRESS}|" \
  -e "s|@WORKER3_ADDRESS@|${WORKER3_ADDRESS}|" $(dirname $0)/../sonobuoy/cke-cluster.yml.template > cke-cluster.yml

$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}-0:
$GCLOUD compute scp --zone=${ZONE} cke-cluster.yml cybozu@${INSTANCE_NAME}-0:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME}-0 --command='sudo -H /home/cybozu/run.sh'
RET=$?
if [ "$RET" -eq 0 ]; then
  $GCLOUD compute scp --zone=${ZONE} cybozu@${INSTANCE_NAME}-0:/tmp/sonobuoy.tar.gz /tmp/sonobuoy.tar.gz
fi

exit $RET
