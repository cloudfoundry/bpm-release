# Transitioning to BPM

## Update the `monit` File

You need to convert the job's `monit` file to use `bpm` to start and stop the
process. In order to allow `monit` to accurately track the status of the
process, you will also need to update the location of the `pidfile` to the
standard `bpm` location, like so:

```
# jobs/<job>/monit
check process <job>
  with pidfile /var/vcap/sys/run/bpm/<job>/<job>.pid
  start program "/var/vcap/jobs/bpm/bin/bpm start <job>"
  stop program "/var/vcap/jobs/bpm/bin/bpm stop <job>"
  group vcap
```

*Note*: You should use `bpm` from the canonical location of
`/var/vcap/jobs/bpm/bin/bpm`.

## Create `bpm.yml` Configuration File

In order for `bpm` to successfully start your process, you will need to create
a `bpm` configuration file in the canonical location:
`/var/vcap/jobs/<job>/config/bpm.yml`. This can be done by adding a template in
your job definition with the following structure:

```yaml
# jobs/<job>/templates/bpm.yml.erb
processes:
  - name: <job>
    executable: /path/to/executable
    args:
    - --port
    - 2424
    env:
      FOO: BAR
```

Also you will need to add the following to the list of templates in the job's
`spec` file:

```
# jobs/<job>/spec
templates:
  bpm.yml.erb: config/bpm.yml
```

### Converting `_ctl` Scripts to `bpm.yml`

A common pattern in BOSH releases is to use a `_ctl` bash script to control the
execution of a process. Because bpm handles process execution, you will need to
move functionality from the `_ctl` script to `bpm.yml`. We have found the
following patterns useful:

#### Executable

Most `_ctl` scripts `exec` a binary as their last step with de-escalated
privileges. Because `bpm` will always execute your process with de-escalated
privileges, it is no longer necessary to manually de-escalate. Thus the
executable and its corresponding arguments can be moved into `bpm.yml` as
follows:

```yaml
executable: /path/to/executable
args:
- arg1
- -flag
- arg2
```

#### Environment Variables

Any necessary environment variables that are set in the `_ctl` script can be
moved into the `env:` block of `bpm.yml`. The format of this configuration is
as follows:

```yaml
env:
  KEY: VALUE
```

*Note*: that it is not currently possible for one value to interpolate values
from another. If this is required, perform the interpolation earlier in the erb
rendering, or later in a shell script that is called by BPM.

#### Runtime Configuration

Often times `_ctl` scripts need to modify runtime configuration, such as the
`$PATH` environment variable. This functionality is not currently supported in
the static `bpm.yml` configuration file, so we have found it useful to extract
runtime configuration into smaller, more auditable, bash scripts. An example
`bpm.yml` and bash script would work as follows:

```yaml
# jobs/<job>/templates/bpm.yml.erb
executable: /path/to/helper/bash/script
```

```yaml
# jobs/<job>/templates/script.erb
#!/bin/bash

export PATH=$PATH:/path/to/runtime/config

/path/to/executable arg1 -flag arg2
```

#### Stop / Usage Blocks

Many `_ctl` scripts provide implementations for `monit stop` and `usage`. Due
to the fact that `bpm` manages stopping processes, these blocks are no longer
necessary.

#### Complex BOSH Property Templating

Oftentimes releases will consume configuration from BOSH properties. A good
pattern for templating these properties into the `bpm.yml.erb` is to use a Ruby
hash and then perform a `YAML.dump()` to format and escape it correctly. For
example:

```ruby
# jobs/<job>/templates/bpm.yml.erb
<%=

config= {}
config["executable"] = /path/to/executable
config["args"] = []
config["args"] << p("example.property")
config["env"] = { "KEY" => "#{p("another.example.property")}" }

YAML.dump(config)
%>
```

### Errands

The `bpm run` command can be used to run short-lived processes in the same
environment as other long-lived processes. Each errand's conversion is going to
be slightly different but most will follow these approximate steps.

1. Rename the `run` script to describe what it does. e.g. `drop_database.sh`
2. Add a `bpm.yml` file by following the other advice in this file. The
   `executable:` should be the script from step one.
3. Re-add the `run` script but change the contents to include the snippet from
   below.
4. Update the templates in the `spec` for all your new files.

```bash
#!/bin/bash

set -e

/var/vcap/jobs/bpm/bin/bpm run <job> [-p <process>]"
```

The `bpm run` command will exit with the same exit status as the internal
executable. It will tee logs to both standard out and standard error as well as
the typical log files.

## Feature Flagging

We have found that in order to integrate `bpm` into certain releases, it needs
to be introduced under a feature flag. If that is the case with your release,
we recommend the following pattern.

```yaml
# jobs/<job>/spec
properties:
  bpm.enabled:
    description: "Enable Bosh Process Manager"
    default: false
```

```
# jobs/<job>/monit
<% if p("bpm.enabled") %>
check process <job>
  with pidfile /var/vcap/sys/run/bpm/<job>/<job>.pid
  start program "/var/vcap/jobs/bpm/bin/bpm start <job>"
  stop program "/var/vcap/jobs/bpm/bin/bpm stop <job>"
  group vcap
<% else %>
check process <job>
  with pidfile /var/vcap/sys/run/<job>/<job>.pid
  start program "/var/vcap/jobs/<job>/bin/<job>_ctl start"
  stop program "/var/vcap/jobs/<job>/bin/<job>_ctl stop"
  group vcap
<% end %>
```

In most cases, when feature flagging, the `_ctl` script will not need to be
modified, as the `bpm.yml` can replace it entirely if the flag is active.

If a helper script is necessary to provide functionality missing from `bpm`,
the same pattern as described in the "Runtime Configuration" section above
still applies. In addition, we have found it useful to modify the `_ctl` script
to call this helper function as it unifies the execution path between `bpm` and
the `_ctl` script.

## Updating Deployment Manifest

In order to successfully deploy your release using BPM, you will need to add
the BPM release to the list of releases in your deployment manifest. As of
right now, there is not a canonical location to consume final releases of BPM
from, thus you will need to create and upload the release to your BOSH
director. This should look as follows:

```yaml
releases:
- name: bpm
  version: latest
```

You can also add bpm using the following operation file:

```yaml
- type: replace
  path: /releases/-
  value:
    name: bpm
    version: create
    url: file:///path/to/bpm-release
```

*Note*: This operation file takes advantage of the `version: create` syntax
which will create and upload the release for you.

Once you have added `bpm` to the list of releases, you will also need to
colocate the `bpm` job on any `instance_groups` that utilize it. This can be
done by adding the following to your `templates` configuration for the
`instance_group` in the deployment manifest:

```yaml
templates:
- name: bpm
  release: bpm
```

This also can be achieved using an operation file similar to the following:

```yaml
- type: replace
  path: /instance_groups/name=<job>/jobs/-
  value:
    name: bpm
    release: bpm

- type: replace
  path: /instance_groups/name=<job>/jobs/name=<job>/properties/bpm?/enabled?
  value: true
```

## Log Forwarding

Some `_ctl` scripts forward the output and error streams from their server to
the `logger` program which sends those logs to syslog. 

For example, your job's `_ctl` script may look like this:

```bash
log_dir=/var/vcap/sys/log/<job>

/var/vcap/packages/<package>/bin/<job> \
  -config=$conf_dir/<job>.json \
  2> >(tee -a $log_dir/<job>.stderr.log | logger -p user.error -t vcap.<job>) \
  1> >(tee -a $log_dir/<job>.stdout.log | logger -p user.info -t vcap.<job>) & echo $! > $pidfile
```


BPM does not forward logs and so you should use something like the [syslog release][syslog-release]
to send your logs from the machine to an external log aggregator and store:

```yaml
 jobs:
  - name: syslog_forwarder
    properties:
      syslog:
        address: ((syslog_address))
        custom_rule: |
          if ($programname startswith "vcap.") then stop
        port: ((syslog_port))
        transport: ((syslog_transport))
    release: syslog
  - name: <job>
    ...
```

This decision was made to encourage consistency and to keep features in one
place: BPM puts logs in the BOSH log directory and the syslog release takes
those logs and puts them somewhere else. This means this if we need to fix a
bug in the syslog forwarding then we don't need to bump every release.

If you have some jobs using bpm, but they are co-located with other jobs using
`_ctl` scripts, avoid double logs by adding providing a `custom_rule` argument
to filter out messages originating from logger. `_ctl` scripts write to logger
with the program name `vcap.[JOB_NAME]`, so we can filter out logs where the
program name starts with `vcap`.

[syslog-release]: https://github.com/cloudfoundry/syslog-release

## Post Deployment Checklist

You can use the `bpm list` command to verify that your jobs are all running as
expected. If you'd like to tail the logs for a particular job then you can run
`bpm logs --all -f JOB`.
