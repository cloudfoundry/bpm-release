# Runtime Environment

bpm is explicit about the interface that it provides to your job. If there is
anything in this specification which is unclear or non-specific then please let
us know so that we can be explicit about the interface and guarantees provided.

## Lifecycle

Your process is started and has an unlimited amount of time to start up. You
should use a [post-start script][post-start] and a health check if you want
your job to only say it has completed deploying after it has started up. You do
not need to manage any PID files yourself.

On shutdown your job will receive a `SIGTERM`. You then have 20 seconds to
shutdown your application before it will be sent `SIGQUIT` to dump the stack
(this is default behavior in the Go and Java runtimes) before being forcibly
terminated.

If you require longer than this then you should use a [drain script][drain] for
your server. The drain script should put your server in such a state that it
can shutdown within 15 seconds. It is acceptable and supported to terminate
your process while running the drain script.

[post-start]:https://bosh.io/docs/post-start.html 
[drain]:https://bosh.io/docs/drain.html

### Zombie Processes and Forwarding Signals

bpm will run the process you specify in your configuration file as the `init`
process in the container and it will be the process to receive the signals
listed above. If your process starts other processes then it is responsible for
making sure that they are not left as zombies by `wait`ing for them.

A workaround for this if you don't want to manage this is to have the process
in your configuration file be `/bin/bash -c` and have it invoke your real
process. `bash` will make sure that you don't end up with zombies. Unfortunately
`bash` does not forward signals to child processes by default and so you'll need
to make sure that you forward on any signals to your subprocess too.

We're going to remove this rough edge in the next major version of BPM.

## Environment Variables

| *Name* | *Value*                          |
|--------|----------------------------------|
| TMPDIR | `/var/vcap/data/JOB/tmp`         |

## Logging

Your process should write logs to standard output and standard error file
descriptors. This data will be written to
`/var/vcap/sys/log/JOB/PROCESS.stdout.log` and
`/var/vcap/sys/log/JOB/PROCESS.stderr.log` respectively.

Any other files which are written to `/var/vcap/sys/log/JOB` inside the
container will be written to `/var/vcap/sys/log/JOB` in the host system.

## Resource Limits

bpm can enforce various [resource limits][limits] on your processes. There are
currently 3 different types: memory, open files, and processes.

### Memory

If your process tries to allocate more memory that your configuration allows
then it will be immediately killed by the OOM killer. This setting is more
useful for agent jobs which do not use more memory under user load and do not
want to affect the more important user-facing processes.

### Open Files

The open files setting sets a limit on the number of open files (including
network connections) which your job is allowed to have open at once. This is
equivalent to setting `ulimit -n` for your process.

### Processes

This setting places a limit on the number of PIDs which your process is allowed
to create. This is to protect against fork-bombs and other resource exhaustion
mistakes or attacks. Threads also count towards this limit as they are
also given PIDs.

[limits]: config.md#limits-schema

## Storing Data

### Temporary Files

Applications may store temporary files in either `/tmp` or
`/var/vcap/data/JOB/tmp` (as per the BOSH recommendation). Both paths,
and `/var/tmp` are mapped onto the same host volume
(`/var/vcap/data/JOB/tmp`) and changes made to one will instantly appear
in the other.

bpm will set the `TMPDIR` environment variable when executing your job to one
of the paths listed above. This environment variable is respected by the
majority of in-use standard libraries used by Cloud Foundry.

> **Note:** Per the BOSH team's guidance the temporary filesystem is *not*
> mounted as `tmpfs`.

### Ephemeral Data

You should store ephemeral data in `/var/vcap/data/JOB`. There are no
guarantees that this data will be present in a different invocation of your
job.  The lifecycle of this data is tied with the underlying BOSH stemcell.

This storage area is useful for cached data that can be re-created by other
means when necessary.

This storage area is shared between all processes in your job. It is your
responsibility to make sure that the data saved by each of your job's processes
does not collide.

### Persistent Data

When the path `/var/vcap/store` exists bpm will mount the path
`/var/vcap/store/JOB` into the job filesystem. As with
`/var/vcap/data/JOB` bpm will create the leaf directory if it does not
exist.

> **NOTE:** Because persistent data access defaults to `/var/vcap/store/JOB`,
> job name changes will cause persistent data to no longer be accessible.  When
> changing the name of a job the BOSH `pre-start` script should idempotently
> move the persistent data from `/var/vcap/store/OLD-JOB` to
> `/var/vcap/store/NEW-JOB`.

### Shared Data

> **Note:** This feature has been added to enable support for older jobs which
> use the file system to communicate with other jobs. It is preferable in
> nearly all cases to use the network to communicate across job boundaries.
> Using the network reduces scheduling constraints such that a job can be
> un-collocated and allows more modern access control (key rotation, etc.) of
> the information being transferred between your jobs.

If your job configuration lists shared volumes (`additional_volumes:` key) then these
paths will be created if they do not exist before being mounted into the job
filesystem. These volumes will be mounted read-write. All collocated jobs which
list a particular volume path will be given a volume they can all share.

Only volume paths inside the `/var/vcap/data`, and (when present) the
`/var/vcap/store` directories are currently permitted. Specifying paths which
are not inside this directory will cause the job to fail to start.

## `monit` Workarounds

There are various `monit` quirks that bpm attempts to hide or smooth over.

### Instant Restart

If an operator runs `monit stop JOB && monit start JOB` then `monit` will wait
until the monitored process has completely stopped and the `stop program`
listed in the configuration has completely finished before attempting to start
the job again. This is unsurprising behavior.

Unfortunately `monit restart` [does not wait][monit-mail] until the existing
job has stopped before starting the same job again. As soon as the existing job
has removed its PID file `monit` forks and starts the new process. This
behavior is unexpected, subtle, and full of race conditions. A considerable
number of BOSH releases are unknowingly suffering from bugs caused by this.

bpm enforces its own locking around process operations to avoid these race
conditions. It is completely safe (from a correctness perspective, you may
still break your service) to run `monit restart` on a job which uses bpm.

[monit-mail]: https://lists.nongnu.org/archive/html/monit-general/2012-09/msg00103.html
