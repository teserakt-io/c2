#!/bin/sh

echo "Regenerating all mock files"

go generate ./...
