#!/bin/bash

set -e

task_dir=$PWD

cd bpm-release

# tini only publishes checksums for prebuilt binaries, not source archives.
# Tags are lightweight/unsigned. HTTPS transport security is the only protection.
tini_version=$(curl -sL \
  "https://api.github.com/repos/krallin/tini/releases/latest" \
  | grep '"tag_name"' \
  | sed 's/.*"v\(.*\)".*/\1/')

if grep -q -F "tini-${tini_version}" config/blobs.yml; then
  echo "tini ${tini_version} is already up to date"
  exit 0
fi

echo "Bumping tini to ${tini_version}"

curl -Lo "$task_dir/tini-${tini_version}.tar.gz" \
  "https://github.com/krallin/tini/archive/refs/tags/v${tini_version}.tar.gz"

tini_old_blob=$(grep 'tini/' config/blobs.yml | sed 's/:$//')
bosh remove-blob "$tini_old_blob"

bosh add-blob "$task_dir/tini-${tini_version}.tar.gz" "tini/tini-${tini_version}.tar.gz"
echo "${BOSH_PRIVATE_CONFIG}" > config/private.yml

bosh upload-blobs

rm "$task_dir/tini-${tini_version}.tar.gz"

if [ "$(git status --porcelain)" != "" ]; then
  git config --global user.email "$GIT_USER_EMAIL"
  git config --global user.name "$GIT_USER_NAME"
  git add .
  git status
  git commit -m "Update tini to ${tini_version}"
fi
