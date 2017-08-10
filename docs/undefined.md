# Undefined Things

This is a list of behavior you may not rely on. Some of it may be defined
eventually. This is not the most user friendly format for this data (it should
be next to the documentation that it is related with) but it helps while
developing it to see what we may need to make decisions on.

* What happens if I `bpm start` a job which is already running?

* What happens to a process if it reaches its memory limit?

* The content, format, or location of log files apart from their inclusion
  inside `/var/vcap/sys/log/[JOB]`.
  * Bind-mounted filesystems operations may not be atomic from within/without
    the container
