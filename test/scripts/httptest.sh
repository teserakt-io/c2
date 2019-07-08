#!/bin/bash

## TODO: DO NOT DELETE THIS SCRIPT
## TODO: this script should eventually be incorporated into the functional
## TODO: testing code.
## TODO: For now we will leave it here.

set -ex

C2HOST=https://localhost:8888

# List all clients
curl -k -w "\nHTTP Response Code: %{http_code}\n\n" -X GET "${C2HOST}/e4/clients?offset=0&count=100"
# Count clients
curl -k -w "\nHTTP Response Code: %{http_code}\n\n" -X GET "${C2HOST}/e4/clients/count"
# Add some clients
for i in {1..20}
do
    curl -k -w "\nHTTP Response Code: %{http_code}\n\n" -X POST "${C2HOST}/e4/client" -d "{\"client\":{\"name\":\"test${i}\"}, \"key\": \"$(dd if=/dev/urandom bs=1 count=32 2>/dev/null | base64 -w 0)\"}"
done
# Count clients
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/clients/count
# List client pages
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/clients?offset=0\&count=10
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/clients?offset=10\&count=10
# Create some topics
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/topic/anewtopic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/topic/anotherexampletopic
# Add a client to the topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X PUT ${C2HOST}/e4/client/topic -d "{\"client\":{\"name\":\"test1\"},\"topic\":\"anewtopic\"}"
# Get count of clients on a topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients/count
# Get first page of clients on a topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients?offset=0\&count=10
# Remove a client from topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X DELETE ${C2HOST}/e4/client/topic -d "{\"client\":{\"name\":\"test1\"},\"topic\":\"anewtopic\"}"
# Get count of clients on a topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients/count
# Add a client to the topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X PUT ${C2HOST}/e4/client/topic -d "{\"client\":{\"name\":\"test1\"},\"topic\":\"anewtopic\"}"
# Count clients
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/clients?offset=0\&count=10
# Get first 10 clients on a topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients?offset=0\&count=10
# Delete an client which is not on a topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X DELETE ${C2HOST}/e4/client/topic -d "{\"client\":{\"name\":\"test20\"},\"topic\":\"anewtopic\"}"
# Delete another client which is not on a topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X DELETE ${C2HOST}/e4/client/topic -d "{\"client\":{\"name\":\"test18\"},\"topic\":\"anewtopic\"}"
# List first 10 clients on a topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients?offset=0\&count=10
# Delete a topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X DELETE ${C2HOST}/e4/topic/anewtopic
# List first 10 clients on a deleted topic
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients?offset=0\&count=10
# Count the number of clients
curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/clients/count

RANDOMTOPICS=`shuf -n20 /usr/share/dict/words`

for rtopic in $RANDOMTOPICS; do
    curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/topic/$rtopic
    curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X PUT ${C2HOST}/e4/client/topic -d "{\"client\":{\"name\":\"test2\"},\"topic\":\"$rtopic\"}"
done

curl --insecure -w "\nHTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/client/topics?client.name=test2\&offset=0\&count=50
