# This script is deprecated. However we're slowly replacing 
# it with integration testing code elsewhere

E4PATH=""

for GOSRC in ${GOPATH//:/ }; do
    if [ -d $GOSRC/src/teserakt/e4go ]; then
        E4PATH=$GOSRC/src/teserakt/e4go
        break
    fi
done

$E4PATH/scripts/build.sh
$E4PATH/scripts/unittests.sh

GIT_COMMIT=$(git rev-list -1 HEAD)
NOW=$(date "+%Y%m%d")

INTPATH=teserakt/e4go/inttest

GOOS=`uname -s | tr '[:upper:]' '[:lower:]'` 
GOARCH=amd64

# build.sh should build e4tclient, so we don't need to do that.
# we do however need to build the unit tests.

GOOS=$GOOS GOARCH=$GOARCH go build -o test/c2httpapi -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" $INTPATH/c2httpapi
GOOS=$GOOS GOARCH=$GOARCH go build -o test/c2testcli -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" $INTPATH/c2testcli

