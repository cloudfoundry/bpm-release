# Transitioning `bbr` Scripts to `bpm`

`bbr` scripts can be run as `bpm` errands. This document outlines the processes
for transitioning a `bbr` script to use `bpm`.

## Create `bpm.yml` Configuration File

In order for `bpm` to successfully call `bbr` scripts, a `bpm` configuration
file will need to be created in the canonical location:
`/var/vcap/jobs/<bbr-job>/config/bpm.yml`. This can be done by adding a template
in the `bbr` job definition with the following structure for each of the `bbr`
scripts:

```yaml
# jobs/<bbr-job>/templates/bpm.yml.erb
processes:
- name: <script>
  executable: /var/vcap/jobs/<bbr-job>/bin/bbr/<script>
  args:
  - run
...
```

Also, the following will need to be added to the list of templates in the job's
`spec` file:

```yaml
# jobs/<bbr-job>/spec
templates:
  bpm.yml.erb: config/bpm.yml
```

### Using the `backup-and-restore-sdk-release`

In order to make use of the `database-backup-restorer`, the following
`unsafe unrestricted_volumes` mount must be added to the `bpm.yml.erb` for each
script that needs to use the SDK:

```yaml
processes:
- name: <script>
  executable: /var/vcap/jobs/<bbr-job>/bin/bbr/<script>
  args:
  - run
  unsafe:
    unrestricted_volumes:
      - path: /var/vcap/jobs/database-backup-restorer/bin
        allow_executions: true
...
```

This will allow the scripts to access the SDK even when running in BPM
containers.

## Update `bbr` scripts to use `bpm`

Move the contents of the `bbr` script into a bash function which will be called
if the script is executed with the argument `run` or invoked without `bpm`. Then
change the script to call `bpm run <bbr-job> -p <script>` if called and `bpm` is
enabled.

As an explicit example, here is a backup script which uses `bpm`
```bash
#!/usr/bin/env bash

set -eu

<% if p('enabled') %>
backup() {
  <code to be executed when backup script called>
}

case ${1:-} in
  run)
    backup
    ;;

  *)

    <% if p("bpm.enabled") %>
      /var/vcap/jobs/bpm/bin/bpm run <bbr-job> \
        -p backup \
        -v "${BBR_ARTIFACT_DIRECTORY%/}:writable" \
        -e BBR_ARTIFACT_DIRECTORY="$BBR_ARTIFACT_DIRECTORY"
    <% else %>
      backup
    <% end %>
    ;;

esac

<% end %>
```
