# Crucible

> Isolated BOSH Jobs

## About

Crucible is a layer between `monit` and your BOSH jobs which adds additional
features while removing nearly all boilerplate startup scripts. It is backwards
compatible with any BOSH version released in the past few years.

### Well-defined Lifecycle

The current job lifecycle is very dependent on `monit` semantics. Job and
process start order is not guaranteed and there are hidden timeouts you can hit
which will put your system in an unexpected state.

Crucible makes its expectations of your job very clear. It defines how long
things should take, how Crucible will communicate with your process, and how
your job should behave under certain scenarios. Most jobs will already be
compliant.

### Isolation

Jobs using Crucible are isolated from one another. All operating system
resources (with the exception of networking) are namespaced such that a job
cannot see or interact with other processes outside their containing job.

This provides a far smaller and easier to maintain interface between your jobs
and the system but crucially provides a security barrier such that if one of the
jobs on your machine is compromised then the incident is limited to just that
job rather than all jobs on the same machine.

### Resource Limits

Crucible is also able to offer resource limiting due to the technologies chosen
for the above features. This stops any one job from starving other collocated
jobs of the operating system resources they need in order to work.

## Documentation

Documentation can be found in the [docs](docs) directory. As we're developing
Crucible this documentation my lead the implementation changes briefly but will
eventually become the official source of Crucible documentation.

## Plans

We're currently planning on switching [Diego][diego-release] to use Crucible as
a first step. This change will initially be behind a feature flag. If this is
successful then we'd like to continue migrating both open and closed source
releases over.

This entire project can also be viewed as a step towards the isolation of BOSH
jobs such that they can be run on many different work schedulers without code
changes.

[diego-release]: https://github.com/cloudfoundry/diego-release

## Usage

Not yet, please.

You can start to read about the [runtime environment](docs/runtime.md) which
Crucible provides to your job, the [configuration format](docs/config.md), and
the [undefined behavior](docs/undefined.md) of the system.

## Development

Development is not currently supported on anything other than Linux, though running the docker based tests is possible on OSX.

Dependencies required for local testing:
* docker
* golang

The following steps should allow you to run the tests in a local docker container:

* Enable swap accounting
  ```
  $ sudo sed -i 's/GRUB_CMDLINE_LINUX=""/GRUB_CMDLINE_LINUX="swapaccount=1"/' /etc/default/grub 
  $ sudo update-grub
  $ sudo reboot
  ```
* Clone the repo and submodules
  ```
  $ cd ~/workspace
  $ git clone git@github.com:pivotal-cf/crucible.git
  $ cd ~/workspace/crucible
  $ git submodule update --init --recursive
  ```
* Install `counterfiter` for generating fakes
  ```
  $ cd && go get github.com/maxbrunsfeld/counterfeiter
  ```
* Enable `direnv` and run tests
  ```
  $ cd ~/workspace/crucible
  $ direnv allow .envrc
  $ ./scripts/test-with-docker 
  ```
