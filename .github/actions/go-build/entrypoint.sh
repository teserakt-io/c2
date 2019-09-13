#!/bin/bash

set -e

PROJECT="${INPUT_PROJECT}"
GOOS="${INPUT_GOOS}"
GOARCH="${INPUT_GOARCH}"
RACE="${INPUT_RACE}"

# Create netrc file with given username and access token so go get could authenticate to
# github and fetch private repositories.
if [ ! -z "${CI_USERNAME}" ] && [ ! -z "${CI_ACCESS_TOKEN}" ]; then
    echo "machine github.com login ${CI_USERNAME} password ${CI_ACCESS_TOKEN}" > ~/.netrc
else
    echo "No CI_USERNAME or CI_ACCESS_TOKEN defined, skipping authentication."
fi

echo "$PROJECT build script"
echo ""

GIT_COMMIT=$(git rev-parse --short HEAD)
GIT_TAG=$(git describe --exact-match HEAD 2>/dev/null || true)
NOW=$(date "+%Y%m%d")

RACEDETECTOR=$(if [[ "$RACE" -ne "" ]]; then echo "-race"; else echo ""; fi)

BINARY_OUT="bin/${PROJECT}-${GOOS}-${GOARCH}"

if [ "${GOOS}" == "windows" ]; then
    BINARY_OUT="${BINARY_OUT}.exe"
fi

printf "building $BINARY_OUT:\n\tversion:\t$NOW-$GIT_COMMIT\n\tOS:\t\t$GOOS\n\tarch:\t\t$GOARCH\n"
go build $RACEDETECTOR -o ${BINARY_OUT} -ldflags "-X main.gitTag=$GIT_TAG -X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" ${PWD}/cmd/$PROJECT

echo ::set-output name=binary-path::$BINARY_OUT
