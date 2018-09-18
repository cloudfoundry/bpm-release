# Transitioning BBR Scripts to BPM

BBR Scripts are are run as BPM Errands.

## Create `bpm.yml` Configuration File

In order for `bpm` to successfully call your `bbr` scripts, you will need to create
a `bpm` configuration file in the canonical location:
`/var/vcap/jobs/<bbr-job>/config/bpm.yml`. This can be done by adding a template in
your `bbr` job definition with the following structure for each of your `bbr` scripts:

```yaml
# jobs/<bbr-job>/templates/bpm.yml.erb
processes:
  - name: backup
    executable: /var/vcap/jobs/<bbr-job>/bin/bbr/backup
    args:
    - run
  ...
```

Also you will need to add the following to the list of templates in the job's
`spec` file:

```
# jobs/<bbr-job>/spec
templates:
  bpm.yml.erb: config/bpm.yml
```

### Using the `backup-and-restore-sdk-release`

You will have to add the following `unsafe unrestricted_volumes` to your `bpm.yml.erb` for each job that needs to use the SDK
```yaml
processes:
  - name: backup
    executable: /var/vcap/jobs/<bbr-job>/bin/bbr/backup
    args:
    - run
    unsafe:
      unrestricted_volumes:
        - path: /var/vcap/jobs/database-backup-restorer/bin
          allow_executions: true
  ...
```

This will allow your scripts to access the SDK even when running in BPM containers.

## Update `bbr` scripts to use `bpm`

Move contents of `bbr` script into a bash function which will be called if the script is executed with the argument `run` or invoked without bpm. Then change the script to call `bpm run <bbr-job>` if called and `bpm` is enabled.

As an explicit example, here is a backup script which uses bpm
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
