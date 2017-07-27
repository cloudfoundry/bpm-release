# Runtime Environment

bpm is explicit about the interface that it provides to your job. If there is
anything in this specification which is unclear or non-specific then please let
us know so that we can be explicit about the interface and guarantees provided.

## Lifecycle

Your process is started and has an unlimited amount of time to start up. You
should use a [post-start script][post-start] and a health check if you want your
job to only say it has completed deploying after it has started up. You do not
need to manage any PID files yourself.

On shutdown your job will receive a `SIGTERM`. You then have 20 seconds to
shutdown your application before it will be sent `SIGQUIT` to dump the stack
(this is default behavior in the Go and Java runtimes) before being forcibly
terminated.

If you require longer than this then you should use a [drain script][drain] for
your server. The drain script should put your server in such a state that it can
shutdown within 20 seconds. It is acceptable and supported to terminate your
process while running the drain script.

[post-start]: https://bosh.io/docs/post-start.html
[drain]: https://bosh.io/docs/drain.html

## Environment Variables

None of interest yet.

## Logging

Your process should write logs to standard output and
standard error file descriptors. This data will be written
to `/var/vcap/sys/log/[JOB]/[PROCESS].out.log` and
`/var/vcap/sys/log/[JOB]/[PROCESS].err.log` respectively.

Any other files which are written to `/var/vcap/sys/log/[JOB]` inside the
container will be written to `/var/vcap/sys/log/[JOB]` in the host system.

## Storing Data

### Temporary Files

Applications may store temporary files in either `/tmp` or
`/var/vcap/data/[JOB]/tmp` (as per the BOSH recommendation). Both paths are
mapped onto the same host volume (`/var/vcap/data/[JOB]/tmp`) and changes made
to one will instantly appear in the other.

bpm will set the `TMPDIR` environment variable when execuring your job to one of
the paths listed above. This environment variable is respected by the majority
of in-use standard libraries used by Cloud Foundry.

> **Note:** Per the BOSH team's guidance the temporary filesystem is *not*
> mounted as `tmpfs`.

### Ephemeral Data

You should store ephemeral data in `/var/vcap/data/[JOB NAME]`. There are no
guarantees that this data will be present in a different invocation of your job.
The lifecycle of this data is tied with the underlying BOSH stemcell.

This storage area is useful for cached data that can be re-created by other
means when necessary.

This storage area is shared between all processes in your job. It is your
responsibility to make sure that the data saved by each of your job's processes
does not collide.

### Shared Data

> **Note:** This feature has been added to enable support for older jobs which
> use the file system to communicate with other jobs. It is preferable in nearly
> all cases to use the network to communicate across job boundaries. Using the
> network reduces scheduling constraints such that a job can be un-collocated
> and allows more modern access control (key rotation, etc.) of the information
> being transferred between your jobs.

If your job configuration lists shared volumes (`volumes:` key) then these
paths will be created if they do not exist before being mounted into the job
filesystem. These volumes will be mounted read-write. All collocated jobs which
list a particular volume path will be given a volume they can all share.

Only volume paths inside the `/var/vcap/data` directory are currently permitted.
Paths which are not inside this directory will cause the job to fail to start.

### Persistent Data

TODO: Work out if there is anything complex about persistent data across
restarts (probably not).
