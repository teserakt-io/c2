#!/bin/bash

## TODO: DO NOT DELETE THIS SCRIPT
## TODO: this script should eventually be incorporated into the functional 
## TODO: testing code. 
## TODO: For now we will leave it here.

set -ex

C2HOST=https://localhost:8888

curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test01/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test02/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test03/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test04/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test05/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test06/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test07/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test08/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test09/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test10/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test11/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test12/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test13/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test14/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test15/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test16/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test17/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test18/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test19/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/test20/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/clients/count
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/clients/0/10
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/clients/10/10
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/topic/anewtopic
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/topic/anotherexampletopic
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X PUT ${C2HOST}/e4/client/name/test01/topic/anewtopic
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients/count
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients/0/10
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X DELETE ${C2HOST}/e4/client/name/test01/topic/anewtopic
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients/0/10
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X PUT ${C2HOST}/e4/client/name/test01/topic/anewtopic
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/clients/0/10
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients/0/10
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X DELETE ${C2HOST}/e4/client/name/test20/topic/anewtopic
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X DELETE ${C2HOST}/e4/client/name/test18/topic/anewtopic
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients/0/10
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X DELETE ${C2HOST}/e4/topic/anewtopic
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/topic/anewtopic/clients/0/10
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/clients/count

RANDOMTOPICS=`shuf -n20 /usr/share/dict/words`

for rtopic in $RANDOMTOPICS; do
    curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/topic/$rtopic
    curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X PUT ${C2HOST}/e4/client/name/test02/topic/$rtopic
done

curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X GET ${C2HOST}/e4/client/name/test02/topics/0/50
