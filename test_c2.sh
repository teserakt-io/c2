#!/bin/bash

rm -rf /tmp/e4/db
mkdir -p /tmp/e4/db

# sync workspace for dev outside GOPATH
cp -p -r c2proto  $GOPATH/src/teserakt
cp -p -r e4common $GOPATH/src/teserakt
cp -p -r e4client $GOPATH/src/teserakt

printf "\nbuilding c2backend.."
cd c2backend && go build && cd ..

printf "\nbuilding c2cli.."
cd c2cli && go build && cd ..

printf "\nstarting c2backend..\n"
cd c2backend
./c2backend &
BEPID=$!
sleep 3

trap terminate INT

function terminate() {
    printf "\nshutting down c2backend.."
    kill -9 $BEPID
}

cd ../c2cli

printf "\n# newClient\n"
./c2cli -c nc -id "testid" -pwd "testpwd"

printf "\n# removeClient: for unexisting client, should fail\n"
./c2cli -c rc -id "tstid"

printf "\n# removeClient: this time for an existing client\n"
./c2cli -c rc -id "testid"

printf "\n# newTopic\n"
./c2cli -c nt -topic "atopic"

printf "\n# newClient: add a client 'anotherclient'\n"
./c2cli -c nc -id "anotherclient" -pwd "anotherpwd"

printf "\n# newTopicClient: add a topic to 'anotherclient'\n"
./c2cli -c ntc -id "anotherclient" -topic "atopic"

printf "\n# newClientKey: change the key of 'anotherclient'\n"
./c2cli -c nck -id "anotherclient" 

printf "\n# newTopic: add another topic locally\n"
./c2cli -c nt -topic "anothertopic"

printf "\n# newTopicClient: add 'anothertopic' to 'anotherclient'\n"
./c2cli -c ntc -id "anotherclient" -topic "anothertopic"

printf "\n# removeTopicClient: remove 'anothertopic' from 'anotherclient'\n"
./c2cli -c rtc -id "anotherclient" -topic "anothertopic"

printf "\nterminating.."
terminate



