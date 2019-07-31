#!/bin/bash

set -e
set -u
set -o pipefail

VERSION=$1

git pull -r --ff-only

bosh create-release "releases/bpm/bpm-${VERSION}.yml" --tarball "bpm-release-${VERSION}.tgz"

hub release create \
  --attach "bpm-release-${VERSION}.tgz" \
  --file CHANGELOG.md \
  --edit \
  "v${VERSION}"
