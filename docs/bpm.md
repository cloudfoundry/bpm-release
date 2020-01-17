# bpm

bpm, as the README outlines, is a stepping stone to isolate collocated BOSH
jobs from one another. Ideally this project should not exist and instead we
would use an existing container scheduler but we have a huge number of existing
BOSH releases which would benefit from this isolation. bpm provides a
straightforward but opinionated runtime for existing BOSH jobs.

This document, along with the others in the same directory, is the de-facto
source of documentation for bpm. This file in particular introduces some of the
nouns and semantics used by bpm.

## Entities

### Jobs

Jobs in bpm are semantically identical to those in BOSH. A job is an
independently schedule-able server or collection of servers which provide some
kind of service. Jobs can be collocated with other jobs, but should not require
this. They should be able to speak over the network to wherever the other job
is located. However, a minority jobs are designed to be deployed as a sidecar
to other jobs, so this definition isn't perfect.

bpm isolates collocated jobs from one another. Namespaces are applied to the
host filesystem and process tables such that jobs are not aware of each other.
Performance isolation can be enabled on jobs to prevent them starving their
neighbors of shared physical resources.

Notably, the network is not namespaced due to complexities it would introduce
around service discovery. For example, if a process ends up listening on a
different port externally than in its namespace then how does it know what to
advertise to its service discovery registry? This pattern of allowing traffic
over `localhost` also allows for the use of the popular sidecar proxy pattern
i.e. Envoy, linkerd, Istio.

### Processes

Jobs can be made up of multiple processes, though most jobs will only have a single
process. Processes are isolated from one another and can have independent
performance restrictions imposed upon them but processes do share portions of
the filesystem. Processes are *always* collocated on the same machine as each
other.

Due to how common the case of a single process per job is you can generally
omit the process argument from many of the `bpm` commands. In this case the job
name is reused as the process name.
