
# cleanup db
rm -rf /tmp/e4/db
mkdir -p /tmp/e4/db

E4PATH=$GOPATH/src/teserakt/e4go

$E4PATH/scripts/build.sh

printf "\nstarting c2backend..\n"
$E4PATH/bin/c2backend &
BEPID=$!
sleep 3

printf "\nstarting client..\n"
$E4PATH/bin/mqe4client -action sub -broker tcp://localhost:1883 -num 50 -topic testtopic &
CLID=$!

trap terminate INT

function terminate() {
    printf "\nshutting down c2backend.."
    kill -9 $BEPID
    printf "\nshutting down client.."
    kill -9 $CLID
    exit
}

printf "\nTESTING gRPC INTERFACE\n"

C2CLI=$E4PATH/bin/c2cli

printf "\n# adding client to C2 db\n"
$C2CLI -c nc -i "testid" -p "testpwd"

printf "\n# adding a topic to C2\n"
$C2CLI -c nt -t "atopic"

printf "\n# adding this topic to client\n"
$C2CLI -c ntc -t "atopic" -i "testid"

sleep 1

printf "\n# resetting client\n"
$C2CLI -c rsc -i "testid"

sleep 1

printf "\n# changing client key\n"
$C2CLI -c nck -i "testid"

sleep 1

printf "\n# adding topic to client\n"
$C2CLI -c ntc -t "atopic" -i "testid"

sleep 1

printf "\nTESTING HTTP INTERFACE\n"

C2HTTP="localhost:8888"

printf "\n# resetting client key\n"
curl -X PATCH $C2HTTP/e4/client/$CLIENTID

printf "\n# adding a topic to C2\n"
curl -X POST $C2HTTP/e4/topic/newtopic

printf "\n# adding this topic to client\n"
curl -X PUT $C2HTTP/e4/client/$CLIENTID/topic/newtopic

printf "\n# then removing it from client\n"
curl -X DELETE $C2HTTP/e4/client/$CLIENTID/topic/newtopic

printf "\n# remove topic from c2\n"
curl -X DELETE $C2HTTP/e4/topic/newtopic

printf "\n# removing it again should fail with 404\n"
curl -X DELETE $C2HTTP/topic/newtopic

terminate
