# Configuration Format

**Note:** This is not the final configuration format and is subject to
change at any time.

## Job Configuration

``` yaml
# job.yml

processes:
- name: server
  executable: /var/vcap/packages/program/bin/program-server
  args:
    - --port
    - 2424
  env:
    - FOO=BAR

- name: worker
  executable: /var/vcap/packages/program/bin/program-worker
  args:
    - --queues
    - 4
  env:
    - FOO=BAR
```

## Example `monit` Configuration

Crucible still sits on top of `monit` as part of the current BOSH job API.
However, the contents of the `monit` file now become simpler and less variable.
The amount of features used is minimized. BOSH would like to remove support for
`monit` eventually and so reducing the exposed feature area will make this
easier.

```
check process job-server
  with pidfile /var/vcap/sys/run/crucible/job-server.pid
  start program "/var/vcap/packages/crucible/bin/crucible start job/server"
  stop program "/var/vcap/packages/crucible/bin/crucible stop job/server"
  group vcap

check process job-worker
  with pidfile /var/vcap/sys/run/crucible/job-worker.pid
  start program "/var/vcap/packages/crucible/bin/crucible start job/worker"
  stop program "/var/vcap/packages/crucible/bin/crucible stop job/worker"
  group vcap
```
