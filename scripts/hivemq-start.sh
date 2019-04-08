#!/bin/bash

docker run -d --name hivemq \
    -p 1883:1883 \
    hivemq/hivemq3
