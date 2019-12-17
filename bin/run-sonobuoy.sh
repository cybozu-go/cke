#!/bin/sh -ex

. $(dirname $0)/env-sonobuoy

delete_instance() {
  if [ $RET -ne 0 ]; then
    # do not delete GCP instance upon test failure to help debugging.
    return
  fi
  $GCLOUD compute instances delete ${INSTANCE_NAME} --zone ${ZONE} || true
}

# Create GCE instance
$GCLOUD compute instances delete ${INSTANCE_NAME} --zone ${ZONE} || true
$GCLOUD compute instances create ${INSTANCE_NAME} \
  --zone ${ZONE} \
  --machine-type ${MACHINE_TYPE} \
  --image vmx-enabled \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE}

RET=0
trap delete_instance INT QUIT TERM 0

for i in $(seq 300); do
  if $GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command=date 2>/dev/null; then
    break
  fi
  sleep 1
done

# Extend instance life to complete sonobuoy test
$GCLOUD compute instances add-metadata ${INSTANCE_NAME} --zone ${ZONE} \
  --metadata extended=$(date -Iseconds -d+3hours)

cat >run.sh <<EOF
#!/bin/sh -e

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

$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo -H /home/cybozu/run.sh'
RET=$?
if [ "$RET" -eq 0 ]; then
  $GCLOUD compute scp --zone=${ZONE} cybozu@${INSTANCE_NAME}:/tmp/sonobuoy.tar.gz /tmp/sonobuoy.tar.gz
fi

exit $RET
