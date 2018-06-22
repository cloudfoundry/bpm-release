# v0.8.0

* add the `bpm run` command for executing processes as short-lived commands
* add support for unrestricted volumes in unsafe configuration

# v0.7.1

* sort mounts by ascending length of elements in destination path

# v0.7.0

* do not limit swap space on hosts which do not support it
* mounting reserved directories provides a more useful validation error
* remove the restriction on allowed capabilities

# v0.6.0

* change ownership of `/etc/profile.d/bpm.sh` to `vcap` group
* improved consistency of error messages
* add `/sbin` to the default system mounts
* add support for privileged containers

# v0.5.0

* mount cgroup subsystems when executing `bpm` command
* add `bpm version` command and global `bpm --version` flag
* include stopped processes in `bpm list` output
