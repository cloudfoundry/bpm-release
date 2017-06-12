#! /bin/bash

mount_subsystem() {
  subsystem=$1
  mountpoint=/cgroup/crucible/$subsystem

  # Idempotently mount cgroup subsytem
  if mount | grep cgroup | grep -q  $subsystem; then
    return
  fi

  mkdir -p $mountpoint
  mountpoint -q $mountpoint || mount -n -t cgroup -o $subsystem cgroup $mountpoint
}

mount_subsystem blkio
mount_subsystem cpu
mount_subsystem cpuacct
mount_subsystem cpuset
mount_subsystem devices
mount_subsystem freezer
mount_subsystem hugetlb
mount_subsystem memory
mount_subsystem perf_event
mount_subsystem pids
