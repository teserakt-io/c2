#!/bin/bash

docker run -d --name vernemq \
    -p 1883:1883 \
     -e "DOCKER_VERNEMQ_ALLOW_ANONYMOUS=on" \
    erlio/docker-vernemq
