#!/bin/bash

set -euo pipefail

ROOT_PATH="$(pwd)"
export ROOT_PATH

local_bpm_release="${ROOT_PATH}/bpm-dev-release.tgz"
bosh create-release --dir "${ROOT_PATH}/bpm-release/" --tarball "${local_bpm_release}"

# Explicitly deploy the Director using the dev release to validate it.
. start-bosh \
  -o "${ROOT_PATH}/bpm-release/ci/tasks/use-dev-release-ops-file.yml" \
  -v local_bpm_release=${local_bpm_release}
