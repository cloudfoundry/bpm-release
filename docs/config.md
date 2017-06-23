# Configuration Format

**Note:** This is not the final configuration format and is subject to change at
any time.

## Job Configuration

``` yaml
# /var/vcap/jobs/job/config/server.yml
name: server

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
```

``` yaml
# /var/vcap/jobs/job/config/worker.yml
name: worker

executable: /var/vcap/packages/program/bin/program-worker

args:
  - --queues
  - 4
```

## Example `monit` Configuration

Crucible still sits on top of `monit` as part of the current BOSH job API.
However, the contents of the `monit` file now become simpler and less variable.
The amount of features used is minimized. BOSH would like to remove support
for `monit` eventually and so reducing the exposed feature area will make this
easier.

```
check process job-server
  with pidfile /var/vcap/sys/run/crucible/job/server.pid
  start program "/var/vcap/packages/crucible/bin/crucible start -j job -c /var/vcap/jobs/job/config/server.yml"
  stop program "/var/vcap/packages/crucible/bin/crucible stop -j job -c /var/vcap/jobs/job/config/server.yml"
  group vcap

check process job-worker
  with pidfile /var/vcap/sys/run/crucible/job/worker.pid
  start program "/var/vcap/packages/crucible/bin/crucible start -j job -c /var/vcap/jobs/job/config/worker.yml"
  stop program "/var/vcap/packages/crucible/bin/crucible stop -j job -c /var/vcap/jobs/job/config/worker.yml"
  group vcap
```

## Setting Sysctl Kernel Parameters

We recommend setting these parameters in your `pre-start` with the following
command:

```bash
sysctl -e -w net.ipv4.tcp_fin_timeout 10
sysctl -e -w net.ipv4.tcp_tw_reuse 1
```
