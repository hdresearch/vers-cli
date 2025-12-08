#!/bin/bash

docker rm vers_cli

set -e

docker build -t vers_cli .

docker run --init -d --name vers_cli vers_cli sleep infinity

docker cp vers_cli:/src/bin/vers .

docker stop vers_cli
