# Crucible

> Isolated BOSH Jobs

## About

Crucible is a layer between `monit` and your BOSH jobs which adds additional
features while removing nearly all boilerplate startup scripts. It is backwards
compatible with any BOSH version released in the past few years.

### Well-defined Lifecycle

The current job lifecycle is very dependent on `monit` semantics. Job start
order is not guaranteed and there are many hidden timeouts you can hit which
will put your system in an unexpected state.

Crucible makes its expectations of your job very clear. It defines how long
things should take, how Crucible will communicate with your process and how
your job should behave under certain scenarios.  These will be unchanged from
your expectations for the vast majority of jobs. Due to the use of cgroups to
contain processes we're able to reliably stop the job whatever the state.

### Isolation

TODO: This section.

### Resource Limits

TODO: This section.

## Usage

Not yet, please.

