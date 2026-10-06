#!/bin/sh -ex

. $(dirname $0)/env

$GCLOUD compute instances delete ${INSTANCE_NAME:?} --zone ${ZONE}
