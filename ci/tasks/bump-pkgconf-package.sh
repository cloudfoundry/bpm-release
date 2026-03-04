#!/bin/bash

set -e

task_dir=$PWD

cd bpm-release

# pkgconf uses lightweight unsigned tags and publishes no checksums.
# HTTPS transport security is the only protection available.
pkgconf_version=$(curl -sL \
  "https://api.github.com/repos/pkgconf/pkgconf/git/matching-refs/tags/pkgconf-" \
  | grep '"ref"' \
  | sed 's|.*refs/tags/pkgconf-\(.*\)".*|\1|' \
  | sort -V \
  | tail -1)

if grep -q -F "pkgconf-${pkgconf_version}" config/blobs.yml; then
  echo "pkgconf ${pkgconf_version} is already up to date"
  exit 0
fi

echo "Bumping pkgconf to ${pkgconf_version}"

curl -Lo "$task_dir/pkgconf-${pkgconf_version}.tar.gz" \
  "https://github.com/pkgconf/pkgconf/archive/refs/tags/pkgconf-${pkgconf_version}.tar.gz"

pkgconf_old_blob=$(grep -E 'pkg-config/|pkgconf/' config/blobs.yml | sed 's/:$//')
if [ -n "$pkgconf_old_blob" ]; then
  bosh remove-blob "$pkgconf_old_blob"
fi

bosh add-blob "$task_dir/pkgconf-${pkgconf_version}.tar.gz" "pkgconf/pkgconf-${pkgconf_version}.tar.gz"
echo "${BOSH_PRIVATE_CONFIG}" > config/private.yml

bosh upload-blobs

rm "$task_dir/pkgconf-${pkgconf_version}.tar.gz"

if [ "$(git status --porcelain)" != "" ]; then
  git config --global user.email "$GIT_USER_EMAIL"
  git config --global user.name "$GIT_USER_NAME"
  git add .
  git status
  git commit -m "Update pkgconf to ${pkgconf_version} (replaces pkg-config)"
fi
