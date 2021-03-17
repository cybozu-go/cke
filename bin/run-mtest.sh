#!/bin/sh -ex

SUITE=$1
CLUSTER=$2

. $(dirname $0)/env

# Create GCE instance
$GCLOUD compute instances delete ${INSTANCE_NAME} --zone ${ZONE} || true
$GCLOUD compute instances create ${INSTANCE_NAME} \
  --zone ${ZONE} \
  --machine-type ${MACHINE_TYPE} \
  --image vmx-enabled \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE} \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme

# Run multi-host test
for i in $(seq 300); do
  if $GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command=date 2>/dev/null; then
    break
  fi
  sleep 1
done

cat >run.sh <<EOF
#!/bin/sh -e

# mkfs and mount local SSD on /var/scratch
mkfs -t ext4 -F /dev/nvme0n1
mkdir -p /var/scratch
mount -t ext4 /dev/nvme0n1 /var/scratch
chmod 1777 /var/scratch

# Run mtest
GOPATH=\$HOME/go
export GOPATH
PATH=/usr/local/go/bin:\$GOPATH/bin:\$PATH
export PATH

git clone https://github.com/${GITHUB_REPOSITORY} \
    \$HOME/go/src/github.com/${GITHUB_REPOSITORY}
cd \$HOME/go/src/github.com/${GITHUB_REPOSITORY}
git checkout -qf ${GITHUB_SHA}

cd mtest
cp /assets/etcd-*.tar.gz .
cp /assets/ubuntu-*.img .
cp /assets/flatcar_production_qemu_image.img .
make setup
make placemat SUITE=${SUITE} CLUSTER="${CLUSTER}"
sleep 3
exec make test CONTAINER_RUNTIME=remote SUITE=${SUITE} CLUSTER="${CLUSTER}"
EOF
chmod +x run.sh

$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo -H /home/cybozu/run.sh'
RET=$?

exit $RET
