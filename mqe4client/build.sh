#!/bin/bash

export GIT_COMMIT=$(git rev-list -1 HEAD)
export NOW=$(date "+%Y%m%d")
go build -ldflags "-X main.GitCommit=$GIT_COMMIT -X main.BuildDate=$NOW"
