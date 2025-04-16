#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

fly -t "${CONCOURSE_TARGET:-bosh}" set-pipeline \
  -p bpm \
  -c "${SCRIPT_DIR}/pipeline.yml"
