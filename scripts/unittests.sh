#!/bin/sh

go clean -testcache
go test gitlab.com/teserakt/c2backend/cmd/c2backend
