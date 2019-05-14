#!/bin/sh

C2TEST_POSTGRES="${C2TEST_POSTGRES:-}" \
    go test -v -race ./...
