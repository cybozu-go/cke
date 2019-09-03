#!/bin/bash -e

for i in {1..10}; do
  if docker-compose ps setup | grep -q "Exit 0"; then
    exit 0
  fi
  sleep 1
done
exit 1
