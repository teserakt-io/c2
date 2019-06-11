#!/bin/bash

## TODO: DO NOT DELETE THIS SCRIPT
## TODO: this script should eventually be incorporated into the functional 
## TODO: testing code. 
## TODO: For now we will leave it here.

set -ex

C2HOST=https://34.90.149.110:8765

HOSTS=200

for i in $(seq 1 $HOSTS); do
curl --insecure -w "HTTP Response Code: %{http_code}\n\n" -X POST ${C2HOST}/e4/client/name/somedevice$i/key/`hexdump -n 32 -e '4/4 "%08x"' /dev/urandom`
done
