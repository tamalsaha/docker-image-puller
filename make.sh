#!/bin/bash

go build main.go

docker build -t appscode/docker-image-puller .
docker push appscode/docker-image-puller

