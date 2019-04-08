
FULL_NAME="teserakt/e4go"
RESOURCES="cmd pkg"
BRANCH=`git branch | sed -n '/\* /s///p'`
VERSION=`cat VERSION`
GIT_COMMIT=$(git rev-list -1 HEAD)
NOW=$(date "+%Y%m%d")

if ! [[ "${BRANCH}" == "master" ]]
then
    echo "Releases are only performed on master!"
    exit -1
fi

# Start with a fresh Golang environment everytime
rm -r -f tmp/go
# Remove any existing of the same version
rm -r -f builds/${VERSION}

echo "Vetting..."
VET=`go tool vet cmd pkg 2>&1 >/dev/null`

cur=`pwd`

if ! [ -n "$VET" ]
then
  echo "All good"
  mkdir -p builds/

  # Allocate support for the builds
  mkdir -p builds/${VERSION}/darwin_amd64 
  mkdir -p builds/${VERSION}/linux_amd64
  mkdir -p builds/${VERSION}/windows_amd64

  CMDPATH=teserakt/e4go/cmd
  # Cross compile
  BINS="c2 c2cli"
  OSS="darwin linux windows"
  for BIN in $BINS; do
    for OS in $OSS; do
    echo "Building ${BIN} for ${OS}..."
    GOOS=$OS GOARCH=amd64 go build -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" -o builds/${VERSION}/darwin_amd64/${BIN} $CMDPATH/$BIN
    done
  done
else
  echo "$VET"
  exit -1
fi
