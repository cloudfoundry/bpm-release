# Configuration Format

**Note:** This is not the final configuration format and is subject to
change at any time.

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
    - --queus
    - 4
  env:
    - FOO=BAR
```
