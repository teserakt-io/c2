#!/bin/sh

set -e

$PWD/scripts/build.sh

GIT_COMMIT=$(git rev-list -1 HEAD)
NOW=$(date "+%Y%m%d")

FUNCPATH=`pwd`/functests

GOOS=`uname -s | tr '[:upper:]' '[:lower:]'`
GOARCH=amd64

GOOS=$GOOS GOARCH=$GOARCH go build -o test/bin/c2test -race -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" $FUNCPATH/c2/main.go

# Allow to define the MQTT used by the test server via env.
C2TEST_MQTT="${C2TEST_MQTT:-}"

./test/bin/c2test
