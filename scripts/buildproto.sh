#!/bin/bash

go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
go get github.com/golang/protobuf/protoc-gen-go

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Retrieve path to grpc-gateway modules folder and grep its latest version path only
GRPC_GATEWAY_SRC_PATH=$(find $GOPATH/pkg/mod/github.com/grpc-ecosystem/ -maxdepth 1 -type d -path *grpc-gateway* | sort -r | head -1)

protoc -I ${DIR}/../ -I $GRPC_GATEWAY_SRC_PATH/third_party/googleapis/ -I $GRPC_GATEWAY_SRC_PATH/ \
    --go_out=plugins=grpc:${DIR}/../pkg/pb \
    --grpc-gateway_out=logtostderr=true,allow_delete_body=true,allow_patch_feature=false:${DIR}/../pkg/pb \
    --swagger_out=logtostderr=true,allow_delete_body=true:${DIR}/../doc/ \
    ${DIR}/../api.proto
