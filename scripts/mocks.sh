#!/bin/sh

echo "Regenerating all mock files"

# mockgen fail to properly import golang.org/x/crypto packages when running on go >= 1.13
# and import native crypto instead, making the build fail on go1.12.
# So we boot a go1.12 container instead to generate the mocks.
if [[ "$(go version | grep 1.12)" -eq "" ]]; then
    docker run -u $(id -u):$(id -g) \
        --rm -v $(pwd):/app \
        -v $(go env GOPATH):/go \
        -v $(go env GOCACHE):/.cache/go-build/ \
        -e GO111MODULE=auto \
        -w /app golang:1.12 \
        go generate ./...
else
    go generate ./...
fi
