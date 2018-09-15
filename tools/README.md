# tools for CKE test

[CKE](https://github.com/cybozu-go/cke/) is tested by integrated style test(=mtest).

This directory contains tools to be used in CKE mtest.

The purpose of the following tools is to shorten the docker pull time
which accounts for the major part of mtest initialization processing.

- `create-disk-if-not-exists`

    If the checksum made from [cke.AllImages()](https://github.com/cybozu-go/cke/blob/a34bf2c4525c9231fab390e910b09d1f42c30994/images.go#L31)'s results is changed,
    `create-disk-if-not-exists` creates GCE instance then execute `create-qcow` on that VM.
    
- `create-qcow`

    1. create QCOW2 image and mounts it on `/var/lib/docker`.
    2. execute `docker pull` each images used CKE.
    3. detach and upload it on google cloud storage.
    
    
