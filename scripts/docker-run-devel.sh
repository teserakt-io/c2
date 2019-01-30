#!/bin/sh

touch `pwd`/c2.log
docker run -it -v `pwd`/configs:/opt/e4/configs -v `pwd`/c2.log:/var/log/e4_c2backend.log -p 5555:5555 -p 8888:8888 e4/backend:devel
