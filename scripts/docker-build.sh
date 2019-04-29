#!/bin/sh

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

echo "c2 Docker build script (c) Teserakt AG 2018-2019. All Right Reserved"
echo ""

# docker buikd --ssh require at least docker 18.09
if [ -z "$(docker --version | grep 18.09)" ]; then
    echo "docker >= 18.09 is required"
    exit 1
fi

# ssh-agent is required to forward the connection to the docker build process.
# This allow to reuse host ssh / git configuration to clone private repositories from gitlab.
if ps -p $SSH_AGENT_PID > /dev/null
then
   echo "ssh-agent is already running"
else
    eval `ssh-agent -s`
fi

E4_VERSION=""
E4_GIT_COMMIT=$(git rev-list -1 HEAD)

if [[ -z "$BUILDVERSION" ]]; then
    E4_VERSION="devel"
fi

echo "Building version $E4_VERSION, commit $E4_GIT_COMMIT\n"

printf "=> c2"
DOCKER_BUILDKIT=1 docker build \
    --ssh default \
    --target c2 \
    --tag "e4/c2:$E4_VERSION" \
    --tag "e4/c2:$E4_GIT_COMMIT" \
    -f "${DIR}/../docker/c2/Dockerfile" \
    "${DIR}/../"
