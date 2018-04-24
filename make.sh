#!/bin/bash
set -eou pipefail

GOPATH=$(go env GOPATH)
PKG=github.com/tamalsaha/docker-image-puller
REPO_ROOT="$GOPATH/src/$PKG"

DOCKER_REGISTRY=appscode
IMG=docker-image-puller

main() {
    pushd $REPO_ROOT
    echo "building alpine based binary ..."
    docker run                                                              \
        --rm                                                                \
        -u $(id -u):$(id -g)                                                \
        -v /tmp:/.cache                                                     \
        -v "$REPO_ROOT:/go/src/$PKG"                                        \
        -w "/go/src/$PKG"                                                   \
        -e GOOS=linux                                                       \
        -e GOARCH=amd64                                                     \
        -e CGO_ENABLED=0                                                    \
        golang:1.10.0-alpine                                                \
        go build -a -installsuffix cgo -o docker-image-puller main.go
	chmod +x docker-image-puller

    echo "Building docker image..."
    local cmd="docker build -t $DOCKER_REGISTRY/$IMG ."
    echo $cmd; $cmd

    echo "Push docker image..."
    local cmd="docker push $DOCKER_REGISTRY/$IMG"
    echo $cmd; $cmd

    rm -rf docker-image-puller
    popd
}

main

# go build -v -o docker-image-puller main.go
# chmod +x docker-image-puller

# docker build -t appscode/docker-image-puller .
# docker push appscode/docker-image-puller

# rm -rf docker-image-puller
