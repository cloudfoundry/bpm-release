# Contributing to BOSH-Process-Manager (BPM)

The BPM team uses GitHub and accepts contributions via [pull
request](https://help.github.com/articles/using-pull-requests).

The `bpm-release` repository is a [BOSH](https://github.com/cloudfoundry/bosh)
release for BPM.

Changes are via pull request (PR) to the master branch of its repository. 

Please verify your changes before submitting the PR by following the
[Testing](#testing) section below.

---

## Contributor License Agreement

Follow these steps to make a contribution to any of our open source
repositories:

1. Ensure that you have completed our CLA Agreement for
   [individuals](https://www.cloudfoundry.org/wp-content/uploads/2015/07/CFF_Individual_CLA.pdf)
   or
   [corporations](https://www.cloudfoundry.org/wp-content/uploads/2015/07/CFF_Corporate_CLA.pdf).

2. Set your name and email (these should match the information on your
   submitted CLA)

   ```
   git config --global user.name "Firstname Lastname"
   git config --global user.email "your_email@example.com"
   ```

3. All contributions must be sent using GitHub pull requests as they create a
   nice audit trail and structured approach.
   
   The originating github user has to either have a github id on-file with the
   list of approved users that have signed the CLA or they can be a public
   "member" of a GitHub organization for a group that has signed the corporate
   CLA.  This enables the corporations to manage their users themselves instead of
   having to tell us when someone joins/leaves an organization. By removing a user
   from an organization's GitHub account, their new contributions are no longer
   approved because they are no longer covered under a CLA.
   
   If a contribution is deemed to be covered by an existing CLA, then it is
   analyzed for engineering quality and product fit before merging it.
   
   If a contribution is not covered by the CLA, then the automated CLA system
   notifies the submitter politely that we cannot identify their CLA and ask them
   to sign either an individual or corporate CLA. This happens automatically as a
   comment on pull requests.
   
   When the project receives a new CLA, it is recorded in the project records, the
   CLA is added to the database for the automated system uses, then we manually
   make the Pull Request as having a CLA on-file.

----

## Testing

### Prerequisites:
  - [Install Docker](https://docs.docker.com/engine/installation/)

1. To run the unit test suite in a docker image
```bash
$ cd bpm-release
$ ./scripts/test-with-docker
```

2. To run the acceptance tests against bosh-lite
```bash
$ cd bpm-release
$ ./scripts/bosh-lite-acceptance-test
```

## Updating Dependencies

BPM has a few dependencies:

* Go
* runc
  * pkg-config
  * libseccomp
* tini

We want to keep these up to date to avoid security vulnerabilities sneaking
into the tool. However, we eschew blanket automatic dependency updates as they
can often cause more problems than they solve via supply chain attacks.

We update the Go dependency automatically in CI as we have decided to trust
that supply chain and it is the dependency which updates the most frequently.

The other dependencies are updated manually using their distributables (be that
source code or, in the case of libseccomp, the amended source code from their
GitHub release). We have an RSS bot in Slack watching these releases so that we
can respond to them quickly.
