#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

echo "c2 Docker build script (c) Teserakt AG 2018-2019. All Right Reserved"
echo ""

E4_VERSION="${CI_COMMIT_REF_NAME//\//_}"
E4_GIT_COMMIT="${CI_COMMIT_SHORT_SHA}"

if [ -z "$E4_VERSION" ]; then
    E4_VERSION="devel"
fi

if [ -z "$E4_GIT_COMMIT" ]; then
    E4_GIT_COMMIT=$(git rev-list -1 HEAD)
fi

LINKS=$(ldd ${DIR}/../bin/c2 | grep "not a dynamic executable")
if [ -z "${LINKS}" ]; then
    echo "c2 is not a static binary, please rebuild it with CGO_ENABLED=0"
    exit 1
fi

echo "Building version $E4_VERSION, commit $E4_GIT_COMMIT\n"

printf "=> c2"
docker build \
    --target c2 \
    --build-arg binary_path=./bin/c2 \
    --tag "c2:$E4_VERSION" \
    --tag "c2:$E4_GIT_COMMIT" \
    -f "${DIR}/../docker/c2/Dockerfile" \
    "${DIR}/../"

LINKS=$(ldd ${DIR}/../bin/c2cli | grep "not a dynamic executable")
if [ -z "${LINKS}" ]; then
    echo "c2cli is not a static binary, please rebuild it with CGO_ENABLED=0"
    exit 1
fi

printf "=> c2cli"
docker build \
    --target c2cli \
    --build-arg binary_path=./bin/c2cli \
    --tag "c2cli:$E4_VERSION" \
    --tag "c2cli:$E4_GIT_COMMIT" \
    -f "${DIR}/../docker/c2/Dockerfile" \
    "${DIR}/../"
