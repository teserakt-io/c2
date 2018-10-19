#!/bin/sh

go clean -testcache

go build teserakt/e4go/pkg/e4test

go test teserakt/e4go/pkg/e4common
go test teserakt/e4go/pkg/e4client
go test teserakt/e4go/cmd/c2backend