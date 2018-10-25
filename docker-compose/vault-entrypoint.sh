#!/bin/bash -e

/usr/local/vault/install-tools
/usr/local/vault/bin/vault server -config=/etc/vault/config.hcl
