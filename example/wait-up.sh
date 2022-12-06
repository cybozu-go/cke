#!/bin/bash -e

for i in {1..10}; do
  if $(docker inspect setup | jq 'any(.Name == "/setup" and .State.Status == "exited" and .State.ExitCode == 0)'); then
    exit 0
  fi
  sleep 1
done
exit 1
