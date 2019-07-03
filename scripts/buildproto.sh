#!/bin/bash

go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
go get github.com/golang/protobuf/protoc-gen-go

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

GOOGLEAPI=$(find $GOPATH/pkg/mod/github.com/grpc-ecosystem/ -path */grpc-gateway*/third_party/googleapis -type d | sort -r | head -1)

protoc -I ${DIR}/../ -I $GOOGLEAPI \
    --go_out=plugins=grpc:${DIR}/../pkg/pb \
    --grpc-gateway_out=logtostderr=true:${DIR}/../pkg/pb \
    --swagger_out=logtostderr=true:${DIR}/../doc/ \
    ${DIR}/../api.proto
