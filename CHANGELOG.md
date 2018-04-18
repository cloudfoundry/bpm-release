# v0.6.0

* change ownership of `/etc/profile.d/bpm.sh` to `vcap` group
* improved consistency of error messages
* add `/sbin` to the default system mounts
* add support for privileged containers

# v0.5.0

* mount cgroup subsystems when executing `bpm` command
* add `bpm version` command and global `bpm --version` flag
* include stopped processes in `bpm list` output
