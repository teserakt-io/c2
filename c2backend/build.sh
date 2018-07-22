#!/bin/bash

gofmt -w .
export GIT_COMMIT=$(git rev-list -1 HEAD)
export NOW=$(date "+%Y%m%d")
go build -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW"
