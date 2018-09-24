#!/bin/bash


goimports -w $GOPATH/src/teserakt/e4go/cmd
goimports -w $GOPATH/src/teserakt/e4go/pkg

CMDPATH=teserakt/e4go/cmd

GIT_COMMIT=$(git rev-list -1 HEAD)
NOW=$(date "+%Y%m%d")

# see valid values at https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63
#GOOS=linux 

GOOS=`uname -s | tr '[:upper:]' '[:lower:]'` 
GOARCH=amd64

printf "building E4:\n\tversion $NOW-$GIT_COMMIT\n\tOS $GOOS\n\tarch: $GOARCH\n"

printf "=> c2backend...\n"
GOOS=$GOOS GOARCH=$GOARCH go build -o bin/c2backend -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" $CMDPATH/c2backend 

printf "=> c2cli...\n"
GOOS=$GOOS GOARCH=$GOARCH go build -o bin/c2cli -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" $CMDPATH/c2cli

printf "=> mqe4client...\n"
GOOS=$GOOS GOARCH=$GOARCH go build -o bin/mqe4client -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" $CMDPATH/mqe4client
