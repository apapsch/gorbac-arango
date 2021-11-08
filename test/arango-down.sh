#!/usr/bin/env bash
set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
. "${DIR}/env.sh"

if [ ! -f "${PID_FILE}" ]; then
    echo "File not found: ${PID_FILE}"
    echo "arangod does not seem to be running"
    exit 1
fi

arango_pid=$(tr -d '\n' < "${PID_FILE}") || exit 1

$DOCKER stop "${arango_pid}" || exit 1
$DOCKER rm "${arango_pid}" || exit 1

rm -rf "${DATADIR}" || exit 1

echo "Arango test instance and data directory deleted"
