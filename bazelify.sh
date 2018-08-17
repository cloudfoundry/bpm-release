#!/bin/bash

set -e

alias run="chronic"

echo "getting deps..."
go get github.com/bazelbuild/buildtools/buildozer
go get github.com/maxbrunsfeld/counterfeiter

# clear out existing source
echo "clearing out existing source..."
rm -rf bpm

# get the new sources from master
echo "fetching new code..."
git checkout master -- src
mv src/bpm bpm
rm -r src
git add src

# setup the monorepo imports
echo "setting up monorepo imports..."
echo "# gazelle:prefix bpm" > bpm/BUILD.bazel

# manually generate fakes
echo "generating fakes..."
pushd bpm/runc/lifecycle >/dev/null
  chronic counterfeiter . UserFinder
  chronic counterfeiter . CommandRunner
  chronic counterfeiter . RuncAdapter
  chronic counterfeiter . RuncClient

  pushd lifecyclefakes >/dev/null
    sed -i -e 's!code.cloudfoundry.org/bpm/!!g' *.go
  popd >/dev/null
popd >/dev/null

# generate build files
echo "generating build files..."
chronic bazel run :gazelle

# pend tests which haven't been converted yet
echo "pending failing tests..."
chronic buildozer 'add tags manual' \
  //bpm/acceptance:go_default_test \
  //bpm/config:go_default_test \
  //bpm/integration:go_default_test \
  //bpm/integration2:go_default_test \
  //bpm/mount:go_default_test \
  //bpm/runc/adapter:go_default_test \
  //bpm/runc/client:go_default_test \
  //bpm/usertools:go_default_test \

# check it all worked
echo "checking it all worked..."
chronic bazel build //...
chronic bazel test //...
