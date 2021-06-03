#!/usr/bin/env bash

set -eu

task_dir=$PWD

git config --global user.email "ci@localhost"
git config --global user.name "CI Bot"

cd bpm-release

echo "${BOSH_PRIVATE_CONFIG}" > bumped-bosh-release/config/private.yml

bosh vendor-package golang-1-linux "$task_dir/golang-release"

if [ -z "$(git status --porcelain)" ]; then
  exit
fi

git add -A

git commit -m "Update golang packages from golang-release"
