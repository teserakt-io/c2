#!/bin/bash

PROJECT=c2

echo "$PROJECT build script (c) Teserakt AG 2018-2019. All rights reserved."
echo ""

goimports -w cmd/$PROJECT

GIT_COMMIT=$(git rev-list -1 HEAD)
GIT_TAG=$(git describe --exact-match HEAD 2>/dev/null || true)
NOW=$(date "+%Y%m%d")

if [ -z "$GOOS" ]; then 
    GOOS=`uname -s | tr '[:upper:]' '[:lower:]'` 
fi
if [ -z "$GOARCH" ]; then
    GOARCH=amd64
fi

printf "building $PROJECT:\n\tversion $NOW-$GIT_COMMIT\n\tOS $GOOS\n\tarch: $GOARCH\n"

printf "=> $PROJECT...\n"
GOOS=$GOOS GOARCH=$GOARCH go build -o bin/$PROJECT.$GOOS.$GOARCH -ldflags "-X main.gitTag=$GIT_TAG -X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" ${PWD}/cmd/$PROJECT
