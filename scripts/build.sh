#!/bin/bash

echo "C2 backend build script (c) Teserakt AG 2018. All Right Reserved"
echo ""

if ! [ -x "$(command -v goimports)" ]; then 
    echo "Error: goimports not found (or not on the path)"
    echo "To get run go get golang.org/x/tools/cmd/goimports and add \$GOPATH/bin to your path."
    exit 1
fi

APIREPO=
for GOSRC in ${GOPATH//:/ }; do
    if [ -d $GOSRC/src/teserakt/e4/backend/cmd ]; then
        goimports -w $GOSRC/src/teserakt/e4/backend/cmd
    fi

    # locate the backend api directory
    if [ -d $GOSRC/src/teserakt/e4/backend-api ]; then
        APIREPO=$GOSRC/src/teserakt/e4/backend-api
    fi
done

if [[ -z "$APIREPO" ]]; then 
    echo "API repository not found on the GOPATH. Have you cloned the"
    echo "repositories into a gopath location (or symlinked them there?)"
    exit 1
fi

# README: If you are wondering why we are doing this, there is a simple reason
# - golang does not like a package using grpc and a vendored build also using 
# it. Import paths get confused and builds fail. Consequently we have to 
# compile the grpc generated code internally in this project, but we keep it 
# separate as it is shared across multiple projects.
# This code locates the generated protobuf file and copies it locally.
PROTOBUFSRC=$APIREPO/pkg/c2proto/c2.pb.go
echo $PROTOBUFSRC
PROTOBUFDST=pkg/c2proto
if [ ! -f $PROTOBUFSRC ]; then
    echo "c2 protobuf go code not generated. Please check the repository";
    exit 1
fi

mkdir -p $PROTOBUFDST
cp $PROTOBUFSRC $PROTOBUFDST

CMDPATH=teserakt/e4/backend/cmd

if [[ -z "$E4_GIT_COMMIT" ]]; then 
    if [[ ! -x "$(command -v git)" ]]; then 
        echo "Git command not found; can't determine git commit info."
        exit 1
    fi
    if [[ ! -d `pwd`/.git ]];then 
        echo "We are not in a git repository. Cannot deduce build info."
        exit 1
    fi
    GIT_COMMIT=$(git rev-list -1 HEAD)
else
    GIT_COMMIT=$E4_GIT_COMMIT
fi
NOW=$(date "+%Y%m%d")



# see valid values at https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63
#GOOS=linux 

GOOS=`uname -s | tr '[:upper:]' '[:lower:]'` 
GOARCH=amd64

printf "building C2:\n\tversion $NOW-$GIT_COMMIT\n\tOS $GOOS\n\tarch: $GOARCH\n"

printf "=> c2backend...\n"
GOOS=$GOOS GOARCH=$GOARCH go build -o bin/c2backend -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" $CMDPATH/c2backend 
