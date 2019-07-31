#!/bin/bash

set -e
set -u
set -o pipefail

check_installed()
{
  if ! command -v "${1}" > /dev/null ; then
    echo "error: ${1} is not installed." >&2
    exit 1
  fi
}

check_installed git
check_installed hub
check_installed bosh

git pull -r --ff-only

WORKDIR="$(git rev-parse --show-toplevel)"
VERSION="$(git tag --sort=committerdate | tail -1)"
VERSION=${VERSION#"v"} # strip leading v

echo "this will build and release bpm v${VERSION}!"
read -r -p "Are you sure? [y/N] " response
case "$response" in
    [yY][eE][sS]|[yY])
      pushd "${WORKDIR}"
        TARBALL="bpm-release-${VERSION}.tgz"

        bosh create-release "releases/bpm/bpm-${VERSION}.yml" \
          --tarball "${TARBALL}"

        hub release create \
          --attach "${TARBALL}" \
          --file CHANGELOG.md \
          --edit \
          "v${VERSION}"

        rm "${TARBALL}"
      popd
        ;;
esac
