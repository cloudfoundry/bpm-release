# bpm [![ci.bosh-ecosystem.cf-app.com](https://ci.bosh-ecosystem.cf-app.com/api/v1/teams/main/pipelines/bpm/jobs/test-acceptance-jammy/badge)](https://ci.bosh-ecosystem.cf-app.com/teams/main/pipelines/bpm)

> Isolated BOSH Jobs

## About

bpm (BOSH process manager) is a layer between `monit` and your BOSH jobs which
adds additional features while removing nearly all boilerplate startup scripts.
It is backwards compatible with any BOSH version released in the past few years.

### Well-defined Lifecycle

The current job lifecycle is very dependent on `monit` semantics. Job and
process start order is not guaranteed and there are hidden timeouts you can hit
which will put your system in an unexpected state.

bpm makes its expectations of your job very clear. It defines how long things
should take, how bpm will communicate with your process, and how your job
should behave under certain scenarios. Most jobs will already be compliant.

### Isolation

Jobs using bpm are isolated from one another. All operating system resources
(with the exception of networking) are namespaced such that a job cannot see or
interact with other processes outside their containing job.

This provides a far smaller and easier to maintain interface between your jobs
and the system but crucially provides a security barrier such that if one of
the jobs on your machine is compromised then the incident is limited to just
that job rather than all jobs on the same machine.

### Resource Limits

bpm is also able to offer resource limiting due to the technologies chosen for
the above features. This stops any one job from starving other collocated jobs
of the operating system resources they need in order to work.

## Documentation

Documentation can be found in the [docs](docs) directory. As we're developing
bpm this documentation may lead the implementation changes briefly, but it will
eventually become the official source of bpm documentation.

## Usage

bpm has now reached 1.0 and has a stable [public API](docs/public_interface.md) 
which should be usable for the majority of BOSH jobs. We do not plan on making
any more backwards incompatible changes to the public API before 2.0.

You can start to read about the [ethos and glossary](docs/bpm.md), [runtime
environment](docs/runtime.md) which bpm provides to your job, the
[configuration format](docs/config.md), and the [undefined
behavior](docs/undefined.md) of the system.

## Development

Development is not currently supported on anything other than Linux, though
running the docker based tests is possible on macOS.

Dependencies required for local testing:

* Docker
* Go

The following steps should allow you to run the tests in a local docker
container:

* Enable swap accounting by running the following commands as root:

    ```sh
    # sed -i 's/GRUB_CMDLINE_LINUX=""/GRUB_CMDLINE_LINUX="swapaccount=1"/' /etc/default/grub
    # update-grub
    # reboot
    ```

* Clone this repository and submodules:

    ```sh
    $ cd ~/workspace
    $ git clone https://github.com/pivotal-cf/bpm-release.git
    $ cd ~/workspace/bpm-release
    ```

* Run tests:

    ```sh
    $ cd ~/workspace/bpm-release
    $ ./scripts/test-with-docker
    ```
