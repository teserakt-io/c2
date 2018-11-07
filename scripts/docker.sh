#!/bin/sh

docker build --target buildenv -t teserakt/e4c2backend:0.1 -f docker/Dockerfile.c2 .