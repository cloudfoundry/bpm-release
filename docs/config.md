# Configuration Format

## Example `monit` Configuration

bpm still sits on top of `monit` as part of the current BOSH job API. However,
the contents of the `monit` file become simpler and less variable. BOSH would
like to remove support for `monit` eventually and so reducing the exposed
feature area will make this easier.

```
check process server
  with pidfile /var/vcap/sys/run/bpm/server/server.pid
  start program "/var/vcap/jobs/bpm/bin/bpm start server"
  stop program "/var/vcap/jobs/bpm/bin/bpm stop server"
  group vcap

check process worker
  with pidfile /var/vcap/sys/run/bpm/server/worker.pid
  start program "/var/vcap/jobs/bpm/bin/bpm start server -p worker"
  stop program "/var/vcap/jobs/bpm/bin/bpm stop server -p worker"
  group vcap
```

## Job Configuration

Your job configuration must be in a file called `bpm.yml` in the `config`
directory of your job.

### Schema

| **Property** | **Type**  | **Required?** | **Description**                                          |
|--------------|-----------|---------------|----------------------------------------------------------|
| `processes`  | process[] | Yes           | A top-level listing of all of the processes in your job. |

#### `process` Schema

| **Property**         | **Type**         | **Required?** | **Description**                                                                                                                   |
| -------------------- | ---------------- | ------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `name`               | string           | Yes           | The name of this process.                                                                                                         |
| `executable`         | string           | Yes           | The path to the executable file for this process.                                                                                 |
| `args`               | string[]         | No            | The arguments which will be passed to the `executable` of this process.                                                           |
| `env`                | string => string | No            | Any additional environment variables to be included in the environment of this process.                                           |
| `workdir`            | string           | No            | The working directory for this process. If not specified this is the value `/var/vcap/jobs/JOB`.                                  |
| `hooks`              | hooks            | No            | The hook configuration for this process (see below).                                                                              |
| `capabilities`       | string[]         | No            | The list of [capabilities][capabilities] (without CAP_) which should be granted to this process.                                  |
| `limits`             | limits           | No            | The limit configuration for this process (see below).                                                                             |
| `ephemeral_disk`     | boolean          | No            | Whether or not an ephemeral disk should be mounted into the container at `/var/vcap/data/JOB`.                                    |
| `persistent_disk`    | boolean          | No            | Whether or not an persistent disk should be mounted into the container at `/var/vcap/store/JOB`.                                  |
| `additional_volumes` | volume[]         | No            | A list of additional volumes to mount inside this process (see below). They must be inside `/var/vcap/data` or `/var/vcap/store`. |
| `unsafe`             | unsafe           | No            | The unsafe configuration for this process (see below).                                                                            |

[capabilities]: http://man7.org/linux/man-pages/man7/capabilities.7.html

#### `hooks` Schema

| **Property** | **Type** | **Required** | **Description**                                                                       |
|--------------|----------|--------------|---------------------------------------------------------------------------------------|
| `pre_start`  | string   | No           | The path to an executable to run before starting the main executable of this process. |

#### `limits` Schema

| **Property** | **Type** | **Required** | **Description**                                                                                                             |
|--------------|----------|--------------|-----------------------------------------------------------------------------------------------------------------------------|
| `memory`     | string   | No           | The memory limit to apply to this process. It is formatted as a number and then a single character for units e.g. 1G, 256M. |
| `open_files` | int      | No           | The number of files this process is allowed to have open at any one time.                                                   |
| `processes`  | int      | No           | The number of processes which this process is allowed to have running at any one moment (inclusive of the main process).    |

#### `unsafe` Schema

| **Property**           | **Type**  | **Required** | **Description**                                                                           |
|--------------          |---------- |--------------|-------------------------------------------------------------------------------------------|
| `privileged`           | boolean   | No           | Whether or not this process should execute with increased privileges (see details below). |
| `unrestricted_volumes` | volume[]  | No           | An unrestricted list of additional volumes to mount inside this process (see below).      |

#### `volume` Schema

| **Property**       | **Type** | **Required** | **Description**                                                                                                |
|--------------------|----------|--------------|----------------------------------------------------------------------------------------------------------------|
| `path`             | string   | Yes          | The path of the volume inside this process.                                                                    |
| `writable`         | boolean  | No           | Whether or not this volume is writable by the process.                                                         |
| `allow_executions` | boolean  | No           | Whether or not executable files can be executed from this volume.                                              |
| `mount_only`       | boolean  | No           | Whether or not BPM should just mount this directory rather than creating and chowning a backing directory too. |


### Example

```yaml
# /var/vcap/jobs/server/config/bpm.yml
processes:
- name: server
  executable: /var/vcap/data/packages/server/serve.sh
  args:
  - --port
  - 2424

  env:
    FOO: BAR

  limits:
    processes: 10

  ephemeral_disk: true

  additional_volumes:
  - path: /var/vcap/data/sockets
    writable: true

  capabilities:
  - NET_BIND_SERVICE

- name: worker
  executable: /var/vcap/data/packages/worker/work.sh
  args:
  - --queues
  - 4

  additional_volumes:
  - path: /var/vcap/data/sockets
    writable: true

  hooks:
    pre_start: /var/vcap/jobs/server/bin/worker-setup
```

## Setting Sysctl Kernel Parameters

We recommend setting these parameters in your BOSH `pre-start` with the
following command:

```bash
sysctl -e -w net.ipv4.tcp_fin_timeout 10
sysctl -e -w net.ipv4.tcp_tw_reuse 1
```

You could set these in your bpm `pre_start` but since these affect the entire
host and not just the contained job we like to keep them separate.

## Passing Configuration at Runtime

You are also able to pass volumes to mount and environment variables into the
process when using `bpm run`. This is useful to mount volumes and pass
configuration which you don't know about until runtime.  The syntax for this
is:

```
bpm run -v /var/vcap/data/database:writable,allow_executions -v ... [...]
```

```
bpm run -e KEY=value -e ... [...]
```

**Note:** The environment variable flag should not be used for secret values as
these strings will appear in the process table.

The both flags can be specified multiple times. The volume flag can use the
`writable`, `mount_only`, or `allow_executions` options.

The same validations and limitations which apply to the file-based
configuration also apply here.

## Hooks

Your startup hook must finish with time to spare before the `monit start`
timeout (30s by default). We're looking into ways to make this less vague.

## Privileged Jobs

Processes can be marked as privileged by setting the `unsafe: {privileged:
true}` attribute in their configuration. Jobs should almost never use this
configuration option as it was only added for jobs which truly need to run as a
superuser such as Garden.

Running a privilieged job removes some of the safeguards which surround a bpm
process. The full list of effects is:

* run the job as user `root` and group `root`
* grant a larger list of privileges (taken from docker's privileged list)
* allow new privileges to be gained
* remove seccomp limitations
* remove masked and readonly paths (still applies to volumes and
  `/var/vcap/{data,store}`)
* all mounts have their nosuid option removed
