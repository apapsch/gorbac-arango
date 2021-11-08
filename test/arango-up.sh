#!/usr/bin/env bash
set -euo pipefail

##
## Sets up Arango test container suitable for local development.
##

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
. "${DIR}/env.sh"

mkdir -p "${DATADIR}" || exit 1

arango_pid=$($DOCKER run \
    -d \
    -e "ARANGO_NO_AUTH=1" \
    -p 8529:8529 \
    arangodb) || exit 1

echo "${arango_pid}" > "${PID_FILE}"

echo "Arango test instance up and running"
