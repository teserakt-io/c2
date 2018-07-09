
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

printf "\nbuilding e4demoapp.."
cd e4demoapp && go build && cd ..

printf "\nstarting c2backend..\n"
cd c2backend
./c2backend &
BEPID=$!
sleep 3

cd ../e4demoapp

CLIENTID=2dd31f9cbe1ccf9f3f67520a8bc9594b7fe095ea69945408b83c861021372169 

printf "\nstarting client..\n"
./e4demoapp -action sub -broker test.mosquitto.org:1883 -num 50 -topic e4/$CLIENTID &
CLID=$!

trap terminate INT

    function terminate() {
    printf "\nshutting down c2backend.."
    kill -9 $BEPID
    printf "\nshutting down client.."
    kill -9 $CLID
}

cd ../c2cli

printf "\nTESTING gRPC INTERFACE\n"

printf "\n# adding client to C2 db\n"
./c2cli -c nc -id "testid" -pwd "testpwd"

printf "\n# adding a topic to C2\n"
./c2cli -c nt -topic "atopic"

printf "\n# adding this topic to client\n"
./c2cli -c ntc -topic "atopic" -id "testid"

sleep 1

printf "\n# resetting client\n"
./c2cli -c rsc -id "testid"

sleep 1

printf "\n# changing client key\n"
./c2cli -c nck -id "testid"

sleep 1

printf "\n# adding topic to client\n"
./c2cli -c ntc -topic "atopic" -id "testid"

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