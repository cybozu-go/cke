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
  --image-project ubuntu-os-cloud \
  --image-family ubuntu-2004-lts \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE} \
  --metadata shutdown-at=$(date -Iseconds -d+4hours)

cd $(dirname $0)/../sonobuoy
make worker.ign
rm -f gcp_rsa gcp_rsa.pub
ssh-keygen -t rsa -f gcp_rsa -C cybozu -N ''
sed -e "s#PUBLIC_KEY#$(cat gcp_rsa.pub)#g" ./worker.ign > /tmp/worker.ign

#FLATCAR_IMAGE_SPEC='--image-project kinvolk-public --image-family flatcar-stable'
#FLATCAR_IMAGE_SPEC='--image flatcar-stable-v3227-2-0'
FLATCAR_IMAGE_SPEC='--image flatcar-stable-v3139-2-3'
for i in $(seq 3); do
  $GCLOUD compute instances create ${INSTANCE_NAME}-${i} \
    --zone ${ZONE} \
    --machine-type ${MACHINE_TYPE_WORKER} \
    ${FLATCAR_IMAGE_SPEC} \
    --boot-disk-type ${DISK_TYPE} \
    --boot-disk-size ${BOOT_DISK_SIZE} \
    --metadata-from-file user-data=/tmp/worker.ign \
    --metadata shutdown-at=$(date -Iseconds -d+4hours)
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

# Install essential tools
curl -fsSL -o docker.gpg https://download.docker.com/linux/ubuntu/gpg
apt-key add docker.gpg
add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"

apt-get update
apt-get install -y --no-install-recommends \
    git \
    make \
    jq \
    docker-ce \
    docker-ce-cli

curl -fsSL -O https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz

# Run sonobuoy
GOPATH=\$HOME/go
export GOPATH
PATH=/usr/local/go/bin:\$GOPATH/bin:\$PATH
export PATH

git clone https://github.com/${GITHUB_REPOSITORY} \
    \$HOME/go/src/github.com/${GITHUB_REPOSITORY}
cd \$HOME/go/src/github.com/${GITHUB_REPOSITORY}
git checkout -qf ${GITHUB_SHA}

cd sonobuoy
make setup
make run
make sonobuoy
mv sonobuoy.tar.gz e2e-check.log /tmp
EOF
chmod +x run.sh

WORKER1_ADDRESS=$($GCLOUD compute instances describe ${INSTANCE_NAME}-1 --zone ${ZONE} --format='get(networkInterfaces[0].networkIP)')
WORKER2_ADDRESS=$($GCLOUD compute instances describe ${INSTANCE_NAME}-2 --zone ${ZONE} --format='get(networkInterfaces[0].networkIP)')
WORKER3_ADDRESS=$($GCLOUD compute instances describe ${INSTANCE_NAME}-3 --zone ${ZONE} --format='get(networkInterfaces[0].networkIP)')

sed -e "s|@WORKER1_ADDRESS@|${WORKER1_ADDRESS}|" \
  -e "s|@WORKER2_ADDRESS@|${WORKER2_ADDRESS}|" \
  -e "s|@WORKER3_ADDRESS@|${WORKER3_ADDRESS}|" ./cke-cluster.yml.template > cke-cluster.yml

$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}-0:
$GCLOUD compute scp --zone=${ZONE} cke-cluster.yml cybozu@${INSTANCE_NAME}-0:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME}-0 --command='sudo -H /home/cybozu/run.sh'
RET=$?
if [ "$RET" -eq 0 ]; then
  $GCLOUD compute scp --zone=${ZONE} cybozu@${INSTANCE_NAME}-0:/tmp/sonobuoy.tar.gz /tmp/sonobuoy.tar.gz
  $GCLOUD compute scp --zone=${ZONE} cybozu@${INSTANCE_NAME}-0:/tmp/e2e-check.log /tmp/e2e-check.log
fi

exit $RET
