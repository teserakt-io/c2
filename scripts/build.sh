#!/bin/bash

goimports -w cmd/ pkg/ 

CMDPATH=teserakt/e4go/cmd

GIT_COMMIT=$(git rev-list -1 HEAD)
NOW=$(date "+%Y%m%d")

printf "building c2backend...\n"
go build -o bin/c2backend -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" $CMDPATH/c2backend 

printf "building c2cli...\n"
go build -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" -o ./bin/c2cli $CMDPATH/c2cli

printf "building mqe4client...\n"
go build -o bin/mqe4client -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" $CMDPATH/mqe4client
