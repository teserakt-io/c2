#!/bin/bash

protoc -I c2proto/ c2proto/c2.proto --go_out=plugins=grpc:c2proto
