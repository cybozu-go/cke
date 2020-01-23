#!/bin/sh -ex

. $(dirname $0)/env-sonobuoy

delete_instance() {
  if [ $RET -ne 0 ]; then
    # do not delete GCP instance upon test failure to help debugging.
    return
  fi
  $GCLOUD -q compute firewall-rules delete ${FIREWALL_RULE_NAME} || true
  for i in $(seq 0 3); do
    $GCLOUD compute instances delete ${INSTANCE_NAME}-${i} --zone ${ZONE} || true
  done
}

$GCLOUD -q compute firewall-rules delete ${FIREWALL_RULE_NAME} || true
$GCLOUD compute firewall-rules create ${FIREWALL_RULE_NAME} \
  --allow ipip \
  --network default \
  --source-ranges 10.128.0.0/9

$GCLOUD compute instances delete ${INSTANCE_NAME}-0 --zone ${ZONE} || true
$GCLOUD compute instances create ${INSTANCE_NAME}-0 \
  --zone ${ZONE} \
  --machine-type ${MACHINE_TYPE_SONOBUOY} \
  --image vmx-enabled \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE}

for i in $(seq 3); do
  $GCLOUD compute instances delete ${INSTANCE_NAME}-${i} --zone ${ZONE} || true
  $GCLOUD compute instances create ${INSTANCE_NAME}-${i} \
    --zone ${ZONE} \
    --machine-type ${MACHINE_TYPE_WORKER} \
    --image-project coreos-cloud \
    --image-family coreos-stable \
    --boot-disk-type ${DISK_TYPE} \
    --boot-disk-size ${BOOT_DISK_SIZE} \
    --metadata-from-file user-data=$(dirname $0)/../sonobuoy/worker.ign \
    --local-ssd interface=nvme \
    --local-ssd interface=nvme \
    --local-ssd interface=nvme \
    --local-ssd interface=nvme
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

# Register SSH key and extend instance life to complete sonobuoy test
ssh-keygen -t rsa -f gcp_rsa -C cybozu -N ''
cat gcp_rsa.pub | sed -e "s/^/cybozu:/g" > ./gcp_public_key.txt
for i in $(seq 0 3); do
  $GCLOUD compute instances add-metadata ${INSTANCE_NAME}-${i} --zone ${ZONE} \
    --metadata extended=$(date -Iseconds -d+4hours)
  $GCLOUD compute instances add-metadata ${INSTANCE_NAME}-${i} --zone ${ZONE} \
    --metadata-from-file ssh-keys=./gcp_public_key.txt
done
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
