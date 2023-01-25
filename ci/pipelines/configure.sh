#!/bin/bash

set -e
set -u
set -o pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

fly -t bosh-ecosystem set-pipeline \
  -p bpm \
  -c "${SCRIPT_DIR}/bpm.yml"
