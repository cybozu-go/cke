#!/bin/sh -ex

. $(dirname $0)/env-sonobuoy

delete_instance() {
  if [ $RET -ne 0 ]; then
    # do not delete GCP instance upon test failure to help debugging.
    return
  fi
  for i in $(seq 0 3); do
    $GCLOUD compute instances delete ${INSTANCE_NAME}-${i} --zone ${ZONE} || true
  done
}

$GCLOUD compute instances delete ${INSTANCE_NAME}-0 --zone ${ZONE} || true
$GCLOUD compute instances create ${INSTANCE_NAME}-0 \
  --zone ${ZONE} \
  --machine-type ${MACHINE_TYPE} \
  --image vmx-enabled \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE}

for i in $(seq 3); do
  $GCLOUD compute instances delete ${INSTANCE_NAME}-${i} --zone ${ZONE} || true
  $GCLOUD compute instances create ${INSTANCE_NAME}-${i} \
    --zone ${ZONE} \
    --machine-type ${MACHINE_TYPE} \
    --image-project coreos-cloud \
    --image-family coreos-stable \
    --boot-disk-type ${DISK_TYPE} \
    --boot-disk-size ${BOOT_DISK_SIZE}
done

RET=0
trap delete_instance INT QUIT TERM 0

for i in $(seq 0 3); do
  for i in $(seq 300); do
    if $GCLOUD compute ssh --zone=${ZONE} core@${INSTANCE_NAME}-${i} --command=date 2>/dev/null; then
      break
    fi
    sleep 1
  done
done

# Extend instance life to complete sonobuoy test
for i in $(seq 0 3); do
  $GCLOUD compute instances add-metadata ${INSTANCE_NAME}-${i} --zone ${ZONE} \
    --metadata extended=$(date -Iseconds -d+4hours)
done

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

WORKER1_ADDRESS=$($GCLOUD compute instances describe ${INSTANCE_NAME}-1 --zone ${ZONE} --format='get(networkInterfaces[0].accessConfigs[0].natIP)')
WORKER2_ADDRESS=$($GCLOUD compute instances describe ${INSTANCE_NAME}-2 --zone ${ZONE} --format='get(networkInterfaces[0].accessConfigs[0].natIP)')
WORKER3_ADDRESS=$($GCLOUD compute instances describe ${INSTANCE_NAME}-3 --zone ${ZONE} --format='get(networkInterfaces[0].accessConfigs[0].natIP)')

sed -e "s|@WORKER1_ADDRESS@|${WORKER1_ADDRESS}|" \
  -e "s|@WORKER2_ADDRESS@|${WORKER2_ADDRESS}|" \
  -e "s|@WORKER3_ADDRESS@|${WORKER3_ADDRESS}|" $(dirname $0)/../sonobuoy/cke-cluster.yml.template > cke-cluster.yml

$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}-0:
$GCLOUD compute scp --zone=${ZONE} cke-cluster.yml cybozu@${INSTANCE_NAME}-0:
$GCLOUD compute scp --zone=${ZONE} gcp.private-key cybozu@${INSTANCE_NAME}-0:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME}-0 --command='sudo -H /home/cybozu/run.sh'
RET=$?
if [ "$RET" -eq 0 ]; then
  $GCLOUD compute scp --zone=${ZONE} cybozu@${INSTANCE_NAME}-0:/tmp/sonobuoy.tar.gz /tmp/sonobuoy.tar.gz
fi

exit $RET
