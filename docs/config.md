# Configuration Format

**Note:** This is not the final configuration format and is subject to change at
any time.

## Example `monit` Configuration

bpm still sits on top of `monit` as part of the current BOSH job API.
However, the contents of the `monit` file now become simpler and less variable.
The amount of features used is minimized. BOSH would like to remove support
for `monit` eventually and so reducing the exposed feature area will make this
easier.

```
check process the_job
  with pidfile /var/vcap/sys/run/bpm/the_job/the_job.pid
  start program "/var/vcap/jobs/bpm/bin/bpm start the_job"
  stop program "/var/vcap/jobs/bpm/bin/bpm stop the_job"
  group vcap

check process the_job-worker
  with pidfile /var/vcap/sys/run/bpm/the_job/worker.pid
  start program "/var/vcap/jobs/bpm/bin/bpm start the_job -p worker"
  stop program "/var/vcap/jobs/bpm/bin/bpm stop the_job -p worker"
  group vcap
```

## Job Configuration

```yaml
# /var/vcap/jobs/the_job/config/bpm.yml
processes:
  - name: the_job # same as BOSH-job name means `-p <process>` is not needed
    executable: /var/vcap/data/packages/server/serve.sh
    args:
    - --port
    - 2424
    env:
      FOO: BAR
    limits:
      memory: 3G
      processes: 10
      open_files: 100
    ephemeral_disk: true # mount /var/vcap/data/the_job ; default `false`
                         # NOTE: /var/vcap/data/the_job/tmp is always mounted `rw`
    additional_volumes:
    - /var/vcap/data/certificates
    hooks:
      pre_start: /var/vcap/jobs/the_job/bin/server-setup
    capabilities:
    - NET_BIND_SERVICE
  - name: worker
    executable: /var/vcap/data/packages/worker/work.sh
    args:
    - --queues
    - 4
    persistent_disk: true # mount /var/vcap/store/the_job ; default `false`
    additional_volumes:
    - name: /var/vcap/data/sockets
      writable: true # default `false`
    hooks:
      pre_start: /var/vcap/jobs/the_job/bin/worker-setup
```

**Note:** The value of the `args:` are passed literally to the `executable:`.
Consider the following snippet:

```yaml
executable: /path/to/command

args:
- --some-flag="flag-value"
```

The value of `--some-flag` will be the string `"flag-value"` including quotes.

## Setting Sysctl Kernel Parameters

We recommend setting these parameters in your BOSH `pre-start` with the
following command:

```bash
sysctl -e -w net.ipv4.tcp_fin_timeout 10
sysctl -e -w net.ipv4.tcp_tw_reuse 1
```

You could set these in your bpm `pre_start` but since these affect the entire
host and not just the contained job we like to keep them separate.

## Hooks

Your startup hook must finish with time to spare before the `monit start`
timeout (30s by default). We're looking into ways to make this less vague.
