#!/bin/bash

PROJECT=c2backend

echo "$PROJECT build script (c) Teserakt AG 2018-2019. All rights reserved."
echo ""

goimports -w cmd/$PROJECT

GIT_COMMIT=$(git rev-list -1 HEAD)
NOW=$(date "+%Y%m%d")

GOOS=`uname -s | tr '[:upper:]' '[:lower:]'` 
GOARCH=amd64

printf "building $PROJECT:\n\tversion $NOW-$GIT_COMMIT\n\tOS $GOOS\n\tarch: $GOARCH\n"

printf "=> $PROJECT...\n"
GOOS=$GOOS GOARCH=$GOARCH go build -o bin/$PROJECT -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" ${PWD}/cmd/$PROJECT
