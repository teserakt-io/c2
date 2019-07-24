#!/bin/sh

GIT_TAG=$(git describe --exact-match HEAD 2>/dev/null || true)
GIT_BRANCH=`git branch | sed -n '/\* /s///p'`


if ! [[ "${GIT_BRANCH}" == "master" ]]; then
    echo "Releases are only performed on master!"
    exit 1
fi

if [[ -z "${VERSION}" && -z "${GIT_TAG}" ]]; then
    echo "Release not tagged, refusing to build."
    exit 1
fi

if ! [[ -z "${VERSION}" ]]; then
    V=$VERSION
elif ! [[ -z "${GIT_TAG}" ]]; then
    V=$GIT_TAG
else
    echo "Bug in release script."
    return 1
fi

OUTDIR=build/$V

echo "Producing release $GIT_TAG"

mkdir -p $OUTDIR

OUTDIR=$OUTDIR GOOS=linux GOARCH=amd64 ./scripts/xbuild.sh
OUTDIR=$OUTDIR GOOS=darwin GOARCH=amd64 ./scripts/xbuild.sh
OUTDIR=$OUTDIR GOOS=windows GOARCH=amd64 ./scripts/xbuild.sh

mkdir -p $OUTDIR/configs/

cp -v configs/config.yaml.example $OUTDIR/configs/
cp -v configs/ocagent.yaml $OUTDIR/configs/
cp -v configs/prometheus.yaml $OUTDIR/configs/
cp -v configs/kibana.yml $OUTDIR/configs/
cp -v configs/kibana_objects.json $OUTDIR/configs/

pushd build/$V
tar cjf ../e4-c2-$V.tar.gz *
popd
