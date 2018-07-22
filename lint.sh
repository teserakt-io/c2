#!/bin/bash

if [ ! $(command -v gometalinter) ]
then
	go get github.com/alecthomas/gometalinter
	gometalinter --install
fi

gofmt -w .

time gometalinter \
   	--exclude='error return value not checked.*(Close|Log|Print).*\(errcheck\)$' \
	--concurrency=2 \
	--deadline=300s \
    ./...