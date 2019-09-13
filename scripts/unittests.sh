#!/bin/sh

C2TEST_POSTGRES="${C2TEST_POSTGRES:-}" \
C2TEST_KAFKA="${C2TEST_KAFKA:-}" \
    go test -failfast -race ./...
