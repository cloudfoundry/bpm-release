#!/usr/bin/env bash
set -euo pipefail

set -x

RELEASE_PATH="$(cd "$(dirname "$0")/../.." && pwd)"

version="$(cat release_metadata/version)"

"${RELEASE_PATH}/scripts/build-system-install-tarball" \
  --compiled-release "$(ls compiled-release/bpm-"${version}"-*.tgz)" \
  --output-dir tarball-output
"${RELEASE_PATH}/scripts/check-system-install-tarball" --expect-version "${version}" \
  "tarball-output/bpm-${version}-linux-amd64.tar.gz"
