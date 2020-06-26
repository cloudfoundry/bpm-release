#!/bin/bash

set -e
set -u
set -o pipefail

# Shared-CF BPM/bpm-pipeline-creds
NOTE_ID="4061975348382289614"

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

fly -t bosh set-pipeline \
  -p bpm \
  -c "${SCRIPT_DIR}/bpm.yml" \
  --load-vars-from <(lpass show "$NOTE_ID" --notes)

