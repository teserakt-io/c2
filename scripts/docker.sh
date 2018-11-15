#!/bin/sh

echo "E4GO Docker Build Script (c) Teserakt AG 2018. All Right Reserved"
echo ""

E4_VERSION=""
GIT_COMMIT=$(git rev-list -1 HEAD)

if [[ -z "$BUILDVERSION" ]]; then
    E4_VERSION=$GIT_COMMIT
fi

echo "Building version $E4_VERSION\n"


printf "=> backend"
sudo docker build --target c2 -t "e4/backend:$E4_VERSION" -t "e4/backend:$GIT_COMMIT" -f docker/Dockerfile.c2 .