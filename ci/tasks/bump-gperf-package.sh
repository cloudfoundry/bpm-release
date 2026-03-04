#!/bin/bash

set -e

task_dir=$PWD

cd bpm-release

gperf_version=$(curl -sL https://ftp.gnu.org/pub/gnu/gperf/ \
  | grep -oP 'gperf-\K[0-9]+\.[0-9.]+(?=\.tar\.gz")' \
  | sort -V \
  | tail -1)

if grep -q -F "gperf-${gperf_version}" config/blobs.yml; then
  echo "gperf ${gperf_version} is already up to date"
  exit 0
fi

echo "Bumping gperf to ${gperf_version}"

curl -Lo "$task_dir/gperf-${gperf_version}.tar.gz" \
  "https://ftp.gnu.org/pub/gnu/gperf/gperf-${gperf_version}.tar.gz"
curl -Lo "$task_dir/gperf-${gperf_version}.tar.gz.sig" \
  "https://ftp.gnu.org/pub/gnu/gperf/gperf-${gperf_version}.tar.gz.sig"

# Bruno Haible's key (gperf maintainer)
gpg --batch --keyserver hkps://keys.openpgp.org \
  --recv-keys E0FFBD975397F77A32AB76ECB6301D9E1BBEAC08
gpg --batch --tofu-policy good E0FFBD975397F77A32AB76ECB6301D9E1BBEAC08
gpg --trust-model tofu --verify "$task_dir/gperf-${gperf_version}.tar.gz.sig" \
  "$task_dir/gperf-${gperf_version}.tar.gz"

gperf_old_blob=$(grep 'gperf/' config/blobs.yml | sed 's/:$//')
bosh remove-blob "$gperf_old_blob"

bosh add-blob "$task_dir/gperf-${gperf_version}.tar.gz" "gperf/gperf-${gperf_version}.tar.gz"
echo "${BOSH_PRIVATE_CONFIG}" > config/private.yml

bosh upload-blobs

rm "$task_dir/gperf-${gperf_version}.tar.gz" "$task_dir/gperf-${gperf_version}.tar.gz.sig"

if [ "$(git status --porcelain)" != "" ]; then
  git config --global user.email "$GIT_USER_EMAIL"
  git config --global user.name "$GIT_USER_NAME"
  git add .
  git status
  git commit -m "Update gperf to ${gperf_version}"
fi
