#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

echo -n "Starting swagger UI on localhost:8080..."

docker run -p 8080:8080 -e URL=swagger.json -v ${DIR}/../doc/api.swagger.json:/usr/share/nginx/html/swagger.json swaggerapi/swagger-ui
