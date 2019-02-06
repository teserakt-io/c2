#!/bin/bash

export TAG=6.6.0

docker-compose -f configs/elk-docker-compose.yml up -d