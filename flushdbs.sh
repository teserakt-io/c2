#!/bin/bash

DBFILES="MANIFEST LOCK *.vlog"

cd dbs/id/ && rm -f $DBFILES && cd ../..
cd dbs/topic/ && rm -f $DBFILES 
