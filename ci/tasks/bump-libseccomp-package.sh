#!/bin/bash

set -e

task_dir=$PWD

cd bpm-release

libseccomp_version=$(curl -sL \
  "https://api.github.com/repos/seccomp/libseccomp/releases/latest" \
  | grep '"tag_name"' \
  | sed 's/.*"v\(.*\)".*/\1/')

if grep -q -F "libseccomp-${libseccomp_version}" config/blobs.yml; then
  echo "libseccomp ${libseccomp_version} is already up to date"
  exit 0
fi

echo "Bumping libseccomp to ${libseccomp_version}"

curl -Lo "$task_dir/libseccomp-${libseccomp_version}.tar.gz" \
  "https://github.com/seccomp/libseccomp/releases/download/v${libseccomp_version}/libseccomp-${libseccomp_version}.tar.gz"
curl -Lo "$task_dir/libseccomp-${libseccomp_version}.tar.gz.SHA256SUM" \
  "https://github.com/seccomp/libseccomp/releases/download/v${libseccomp_version}/libseccomp-${libseccomp_version}.tar.gz.SHA256SUM"

(cd "$task_dir" && sha256sum -c "libseccomp-${libseccomp_version}.tar.gz.SHA256SUM")

libseccomp_old_blob=$(grep 'libseccomp/' config/blobs.yml | sed 's/:$//')
bosh remove-blob "$libseccomp_old_blob"

bosh add-blob "$task_dir/libseccomp-${libseccomp_version}.tar.gz" "libseccomp/libseccomp-${libseccomp_version}.tar.gz"
echo "${BOSH_PRIVATE_CONFIG}" > config/private.yml

bosh upload-blobs

rm "$task_dir/libseccomp-${libseccomp_version}.tar.gz" "$task_dir/libseccomp-${libseccomp_version}.tar.gz.SHA256SUM"

if [ "$(git status --porcelain)" != "" ]; then
  git config --global user.email "$GIT_USER_EMAIL"
  git config --global user.name "$GIT_USER_NAME"
  git add .
  git status
  git commit -m "Update libseccomp to ${libseccomp_version}"
fi
