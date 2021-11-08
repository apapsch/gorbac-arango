#!/usr/bin/env bash
set -euo pipefail

DOCKER="$(which docker)"
DATADIR="${TMPDIR:-/tmp}/gorbac-arango-test"
PID_FILE="${DATADIR}/arangod-docker.pid"
