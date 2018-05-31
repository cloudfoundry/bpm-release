# Releasing

Releasing a new version of bpm involves creating a [final
release](https://bosh.io/docs/create-release/#final-release), tagging the commit
that associates with the final release, and creating a Github release associated
with the tagged commit.

## Process

Currently, creating a new release of bpm involves manually triggering a couple
of concourse builds and creating a Github release through the web interface. The
process is as follows:

1. Finish accepting any delivered or finished stories.

   We create new bpm releases from the `master` branch. Due to the fact that all
   development also occurs on the `master` branch, we should finish accepting any
   finished or delivered stories in the
   [bpm tracker project](https://www.pivotaltracker.com/n/projects/2070399).

1. Trigger the appropriate concourse build to bump the semantic version

   We manage the bpm version through the
   [semantic version resource](https://github.com/concourse/semver-resource).
   In order to increment the version, you will need to trigger the appropriate
   build to bump either the patch, minor, or major versions. All of these builds
   can be found [here](https://wings.pivotal.io/teams/bpm/pipelines/bpm?groups=version).

1. Trigger the [create-final-release](https://wings.pivotal.io/teams/bpm/pipelines/bpm/jobs/create-final-release/builds/15) build

   This build will perform the necessary steps to create a final BOSH release and
   tag the corresponding commit associated with the final release. Specifically it
   will: remove the `+dev` from the [local version file](../../src/version), create a
   final BOSH release and upload the corresponding artifacts, tag the commit
   associated with the final release, and re-add the `+dev` to the
   [local version file](../../src/version) with the newly released version of bpm.

1. Create a Github release off the appropriate tag

   As a final step to the release process, we need to create a Github release
   associated with the tagged commit. This can be done by
   [drafting a new release](https://github.com/cloudfoundry-incubator/bpm-release/releases/new).
   The name of the release should be the semantic version of the latest final
   release i.e. `vx.y.z`. The contents of the release notes can be copied from the
   [changelog](../../CHANGELOG.md). It is also a good idea to upload the latest final
   release as an artifact associated with the Github release. This can be done by
   running `bosh create-release releases/bpm/bpm-x.y.z.yml --tarball
   bpm-release-x.y.z.tgz` from the root of bpm-release.
