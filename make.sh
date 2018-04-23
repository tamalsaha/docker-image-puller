#!/bin/bash
set -xeou pipefail

go build -v -o docker-image-puller main.go
chmod +x docker-image-puller

docker build -t appscode/docker-image-puller .
docker push appscode/docker-image-puller

rm -rf docker-image-puller
