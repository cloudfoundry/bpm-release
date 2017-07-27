# Configuration Format

**Note:** This is not the final configuration format and is subject to change at
any time.

## Job Configuration

``` yaml
# /var/vcap/jobs/job/config/bpm/job.yml
executable: /var/vcap/packages/program/bin/program-server

args:
  - --port
  - 2424

env:
  - FOO=BAR

limits:
  memory: 3G
  processes: 10
  open_files: 100

volumes:
- name: certificates
- name: sockets

hooks:
  pre_start: /var/vcap/job/program/bin/bpm-pre-start
```

``` yaml
# /var/vcap/jobs/job/config/bpm/worker.yml
executable: /var/vcap/packages/program/bin/program-worker

args:
  - --queues
  - 4

volumes:
- name: sockets
```

## Example `monit` Configuration

bpm still sits on top of `monit` as part of the current BOSH job API.
However, the contents of the `monit` file now become simpler and less variable.
The amount of features used is minimized. BOSH would like to remove support
for `monit` eventually and so reducing the exposed feature area will make this
easier.

```
check process job
  with pidfile /var/vcap/sys/run/bpm/job/job.pid
  start program "/var/vcap/packages/bpm/bin/bpm start job"
  stop program "/var/vcap/packages/bpm/bin/bpm stop job"
  group vcap

check process job-worker
  with pidfile /var/vcap/sys/run/bpm/job/worker.pid
  start program "/var/vcap/packages/bpm/bin/bpm start job -p worker"
  stop program "/var/vcap/packages/bpm/bin/bpm stop job -p worker"
  group vcap
```

## Setting Sysctl Kernel Parameters

We recommend setting these parameters in your `pre-start` with the following
command:

```bash
sysctl -e -w net.ipv4.tcp_fin_timeout 10
sysctl -e -w net.ipv4.tcp_tw_reuse 1
```
