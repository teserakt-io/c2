#!/bin/bash

BROKER=tcp://localhost:1883

./bin/mqe4client -action sub -broker $BROKER -num 50 -topic testtopic 
