#!/bin/sh -ex

. $(dirname $0)/env

$GCLOUD compute instances add-metadata ${INSTANCE_NAME:?} --zone ${ZONE} \
  --metadata shutdown-at=$(date -Iseconds -d+30minutes)
