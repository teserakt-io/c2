#!/bin/sh

/opt/e4/bin/c2cli --endpoint ${C2_API_ENDPOINT} --cert ${C2_API_CERT} "$@"
