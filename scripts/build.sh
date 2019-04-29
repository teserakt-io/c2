#!/bin/bash

set -ex

PROJECT=c2

echo "$PROJECT build script (c) Teserakt AG 2018-2019. All rights reserved."
echo ""

GIT_COMMIT=$(git rev-parse --short HEAD)
GIT_TAG=$(git describe --exact-match HEAD 2>/dev/null || true)
NOW=$(date "+%Y%m%d")

GOOS=`uname -s | tr '[:upper:]' '[:lower:]'`
GOARCH=amd64

printf "building $PROJECT:\n\tversion:\t$NOW-$GIT_COMMIT\n\tOS:\t\t$GOOS\n\tarch:\t\t$GOARCH\n"

printf "=> $PROJECT...\n"
GOOS=$GOOS GOARCH=$GOARCH go build -v -race -o bin/$PROJECT -ldflags "-X main.gitTag=$GIT_TAG -X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" ${PWD}/cmd/$PROJECT
