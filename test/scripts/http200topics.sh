#!/bin/bash

## TODO: DO NOT DELETE THIS SCRIPT
## TODO: this script should eventually be incorporated into the functional 
## TODO: testing code. 
## TODO: For now we will leave it here.

set -ex

C2HOST=https://34.90.149.110:8765
#C2HOST=https://127.0.0.1:8888

HOSTS=200

for i in $(seq 1 $HOSTS); do
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/topic/57topicsandtheresnothingon$i
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/topic/БабаМарта$i
done
