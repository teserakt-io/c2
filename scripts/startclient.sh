#!/bin/bash

./bin/mqe4client -action sub -broker tcp://mqtt.fail:1883 -num 50 -topic testtopic 
