#!/bin/sh

echo "E4GO Docker Build Script (c) Teserakt AG 2018. All Right Reserved"
echo ""

E4_VERSION=""
E4_GIT_COMMIT=$(git rev-list -1 HEAD)

if [[ -z "$BUILDVERSION" ]]; then
    E4_VERSION="devel"
fi

echo "Building version $E4_VERSION, commit $E4_GIT_COMMIT\n"


printf "=> backend"
sudo -E docker build --build-arg E4_GIT_COMMIT="$E4_GIT_COMMIT" --target c2 -t "e4/backend:$E4_VERSION" -t "e4/backend:$E4_GIT_COMMIT" -f docker/Dockerfile.c2 .