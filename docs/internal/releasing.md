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

2. Make sure the CHANGELOG is up-to-date!

3. Trigger the appropriate concourse build to bump the semantic version

   We manage the bpm version through the
   [semantic version resource](https://github.com/concourse/semver-resource).
   In order to increment the version, you will need to trigger the appropriate
   build to bump either the patch, minor, or major versions. All of these builds
   can be found [here](https://wings.pivotal.io/teams/bpm/pipelines/bpm?groups=version).

4. Trigger the [create-final-release](https://wings.pivotal.io/teams/bpm/pipelines/bpm/jobs/create-final-release/builds/15) build

   This build will perform the necessary steps to create a final BOSH release and
   tag the corresponding commit associated with the final release. Specifically it
   will: remove the `+dev` from the [local version file](../../src/version), create a
   final BOSH release and upload the corresponding artifacts, tag the commit
   associated with the final release, and re-add the `+dev` to the
   [local version file](../../src/version) with the newly released version of bpm.

5. Create a Github release off the appropriate tag

   Run `scripts/release.sh`! This will release the latest tag in the
   repository. When an editor appears you should make the first line of the
   file be the name of the release (the version number with a "v" prefix) and
   then delete the rest of the file so that only the current release's
   changelog is left.
