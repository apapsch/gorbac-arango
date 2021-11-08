#!/bin/sh

# Absolute path to this script, e.g. /home/user/bin/foo.sh
SCRIPT=$(readlink -f "$0")
# Absolute path this script is in, thus /home/user/bin
SCRIPTPATH=$(dirname "$SCRIPT")

sh "${SCRIPTPATH}/test/arango-up.sh"
sleep 5
go test
sh "${SCRIPTPATH}/test/arango-down.sh"
