# Releasing

Releasing a new version of bpm involves running a pipeline to create a [final
release](https://bosh.io/docs/create-release/#final-release) and tag and creating a Github release associated
with the tagged commit.

## Process

Currently, creating a new release of bpm involves manually triggering a couple
of concourse builds and creating a Github release through the web interface. The
process is as follows:

1. Finish accepting any delivered or finished stories.

1. Trigger the appropriate concourse build to bump the semantic version

   We manage the bpm version through the
   [semantic version resource](https://github.com/concourse/semver-resource).
   In order to increment the version, you will need to trigger the appropriate
   build to bump either the patch, minor, or major versions. All of these builds
   can be found [here](https://ci.bosh-ecosystem.cf-app.com/teams/main/pipelines/bpm).

1. Trigger the [create-final-release](https://ci.bosh-ecosystem.cf-app.com/teams/main/pipelines/bpm/jobs/create-final-release) build

   This build will perform the necessary steps to create a final BOSH release and
   tag the corresponding commit associated with the final release. Specifically it
   will: remove the `+dev` from the [local version file](../../src/version), create a
   final BOSH release and upload the corresponding artifacts, tag the commit
   associated with the final release, and re-add the `+dev` to the
   [local version file](../../src/version) with the newly released version of bpm.

1. Create a Github release off the appropriate tag
