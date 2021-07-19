#!/bin/bash

set -e #stops the execution if error

task_dir=$PWD

cd bpm-release

pushd src/bpm
  go get -u ./...
  go mod tidy
  go mod vendor
popd

runc_version_go_mod=$(grep 'runc' src/bpm/go.mod | sed 's/.* v//')
if ! $(grep -q "$runc_version_go_mod" config/blobs.yml); then
  curl -o $task_dir/runc_filename.tar.xz -L https://github.com/opencontainers/runc/releases/download/v${runc_version_go_mod}/runc.tar.xz
  runc_old_version=$(grep 'runc' config/blobs.yml |  sed 's/.$//')
  bosh remove-blob $runc_old_version

  bosh add-blob $task_dir/runc_filename.tar.xz runc/runc-${runc_version_go_mod}.tar.xz
  echo "${BOSH_PRIVATE_CONFIG}" > config/private.yml

  bosh upload-blobs

  rm $task_dir/runc_filename.tar.xz
fi

if [ "$(git status --porcelain)" != "" ]; then
  git status
  git config --global user.email "cf-bpm@pivotal.io"
  git config --global user.name "CF BPM"
  git add .
  git commit -m "Updated dependencies"
fi