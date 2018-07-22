#!/bin/bash

gofmt -w .
export GIT_COMMIT=$(git rev-list -1 HEAD)
export NOW=$(date "+%Y%m%d")
go build -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" -o ./bin/c2backend ./cmd/c2backend 
go build -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" -o ./bin/c2cli ./cmd/c2cli
go build -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" -o ./bin/mqe4client ./cmd/mqe4client
