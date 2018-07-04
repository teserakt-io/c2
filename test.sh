#!/bin/bash

rm -rf /tmp/e4/db
mkdir -p /tmp/e4/db

# sync workspace for dev outside GOPATH
cp -p -r c2proto  $GOPATH/src/teserakt
cp -p -r e4common $GOPATH/src/teserakt
cp -p -r e4client $GOPATH/src/teserakt

echo "building c2backend.."
cd c2backend && go build && cd ..

echo "building c2cli.."
cd c2cli && go build && cd ..

echo "starting c2backend.."
cd c2backend
./c2backend &
BEPID=$!
sleep 3

trap terminate INT

function terminate() {
    echo "shutting down c2backend.."
    kill -9 $BEPID
}

echo "running c2cli.."
cd ../c2cli

echo ""
echo "# adding a client"
./c2cli -c nc -id "testid" -pwd "testpwd"
echo ""
echo "# removing a client that doesnt exist (should fail)"
./c2cli -c rc -id "tstid"
echo ""
echo "# removing the good client"
./c2cli -c rc -id "testid"
echo ""
echo "# add a topic to C2"
./c2cli -c nt -topic "atopic"
echo ""
echo "# adding another client"
./c2cli -c nc -id "anotherclient" -pwd "anotherpwd"
echo ""
echo "# add this topic to this client"
./c2cli -c ntc -id "anotherclient" -topic "atopic"
echo ""
echo "# modify the client's key"
./c2cli -c nck -id "anotherclient" 
echo ""
echo "# add another topic locally then to the client"
./c2cli -c nt -topic "anothertopic"
./c2cli -c ntc -id "anotherclient" -topic "anothertopic"

echo "terminating.."
terminate



