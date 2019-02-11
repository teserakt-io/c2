 #!/bin/bash

 export ELKTAG=6.6.0
 export E4CONFDIR=../configs

 docker-compose -f docker/elk-docker-compose.yml up -d
