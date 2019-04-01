#!/bin/sh

$PWD/scripts/build.sh
$PWD/scripts/unittests.sh

GIT_COMMIT=$(git rev-list -1 HEAD)
NOW=$(date "+%Y%m%d")

FUNCPATH=`pwd`/functests

GOOS=`uname -s | tr '[:upper:]' '[:lower:]'` 
GOARCH=amd64

GOOS=$GOOS GOARCH=$GOARCH go build -o test/c2httpapi -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" $FUNCPATH/c2httpapi
GOOS=$GOOS GOARCH=$GOARCH go build -o test/c2grpcapi -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" $FUNCPATH/c2grpcapi

./test/c2httpapi
./test/c2grpcapi
