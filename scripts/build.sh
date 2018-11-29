#!/bin/bash

PROJECT=backend

echo "$PROJECT build script (c) Teserakt AG 2018. All Right Reserved"
echo ""

if ! [ -x "$(command -v goimports)" ]; then 
    echo "Error: goimports not found (or not on the path)"
    echo "To get run go get golang.org/x/tools/cmd/goimports and add \$GOPATH/bin to your path."
    exit 1
fi

goimports -w cmd/c2backend

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

printf "building $PROJECT:\n\tversion $NOW-$GIT_COMMIT\n\tOS $GOOS\n\tarch: $GOARCH\n"

printf "=> $PROJECT...\n"
GOOS=$GOOS GOARCH=$GOARCH go build -o bin/c2backend -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.buildDate=$NOW" ${PWD}/cmd/c2backend 
