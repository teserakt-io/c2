#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

echo "c2 Docker build script (c) Teserakt AG 2018-2019. All Right Reserved"
echo ""

# docker buikd --ssh require at least docker 18.09
if [ -z "$(docker --version | grep 18.09)" ]; then
    echo "docker >= 18.09 is required"
    exit 1
fi

# A running ssh-agent is required to forward the connection to the docker build process.
# This allow to reuse host ssh / git configuration to clone private repositories from gitlab.
if [ -z "$SSH_AUTH_SOCK" ]; then
    echo "SSH_AUTH_SOCK is required to run the docker build"
    exit 1
else
    echo "Using SSH_AUTH_SOCK=$SSH_AUTH_SOCK"
fi

E4_VERSION="${BUILDVERSION}"
E4_GIT_COMMIT=$(git rev-list -1 HEAD)

if [[ -z "$E4_VERSION" ]]; then
    E4_VERSION="devel"
fi

echo "Building version $E4_VERSION, commit $E4_GIT_COMMIT\n"

printf "=> c2"
DOCKER_BUILDKIT=1 docker build \
    --ssh default \
    --target c2 \
    --tag "registry.gitlab.com/teserakt/c2:$E4_VERSION" \
    --tag "registry.gitlab.com/teserakt/c2:$E4_GIT_COMMIT" \
    -f "${DIR}/../docker/c2/Dockerfile" \
    "${DIR}/../"
