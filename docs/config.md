# Configuration Format

**Note:** This is not the final configuration format and is subject to change at
any time.

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
    memory: 3G
    processes: 10
    open_files: 100
  ephemeral_disk: true # mount /var/vcap/data/server ; default `false`
                       # NOTE: /var/vcap/data/server/tmp is always mounted `rw`
  additional_volumes:
  - path: /var/vcap/data/certificates
    writable: true # default `false`
  hooks:
    pre_start: /var/vcap/jobs/server/bin/server-setup
  capabilities:
  - NET_BIND_SERVICE

- name: worker
  executable: /var/vcap/data/packages/worker/work.sh
  args:
  - --queues
  - 4
  persistent_disk: true # mount /var/vcap/store/server ; default `false`
  additional_volumes:
  - path: /var/vcap/data/sockets
    writable: true # default `false`
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

## Hooks

Your startup hook must finish with time to spare before the `monit start`
timeout (30s by default). We're looking into ways to make this less vague.
