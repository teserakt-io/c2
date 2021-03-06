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

if [ -z "$OUTDIR" ]; then
    OUTDIR=bin
fi

printf "building $PROJECT:\n\tversion $NOW-$GIT_COMMIT\n\tOS $GOOS\n\tarch: $GOARCH\n"

mkdir -p $OUTDIR/${GOOS}_${GOARCH}/

printf "=> $PROJECT...\n"
CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -o $OUTDIR/${GOOS}_${GOARCH}/$PROJECT -ldflags "-X main.gitTag=$GIT_TAG -X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" ${PWD}/cmd/$PROJECT

PROJECT=c2cli
printf "=> $PROJECT...\n"
CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -o $OUTDIR/${GOOS}_${GOARCH}/$PROJECT -ldflags "-X main.gitTag=$GIT_TAG -X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" ${PWD}/cmd/$PROJECT
