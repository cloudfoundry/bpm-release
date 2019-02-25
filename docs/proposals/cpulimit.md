# CPU Resource Limits

Chris Tarazi
Christopher Brown  

*February 2019*

## Background

Some services have strict latency goals which need to be met in order for the
broader system to be acceptable to users. Some services don't.

If these different classes of job are collocated on the same machine then an
over-excited best-effort (BE) service may negatively impact the
latency-critical (LC) service and cause it to miss its responsiveness goals. 

The Loggregator team is toying with a new agent architecture which would see an
increased number of agents be present on each machine to handle the various log
sinks. They would like to ensure that even if an agent is misbehaving that it
isn't able to cause the system to miss its performance goals.

There is a Google/Stanford project called [*Heracles*][heracles] which has
similar goals but instead is able to dynamically change the scheduling
parameters based on information from service metrics. It's an interesting paper
and I'd recommend you read it before continuing through this proposal.
Unfortunately, while impressive, it requires a level of integration which I
don't think we have available to us today. Having BPM be able to query the
metrics of the services it was running to ensure they're not starved of
resource and being able to dynamically change constraints to better accommodate
them is a non-goal for now. Many of the resource constraints which Heracles is
able to tweak also do not necessarily apply in the virtualised environment
which we run Cloud Foundry on (such as Intel specific CPU cache controls).

[heracles]: https://ai.google/research/pubs/pub45351

## Objective

We should be able to deploy Cloud Foundry with (rigged) Loggregator agents
which generate excessive load and see that the system is negatively affected.
After deploying a modified BPM program we should be able to run the same test
and see improvement in the latency of the system.

The misbehaving agents may need to employ back-pressure or load-shedding to
cope with their new resource constraints.

## Changes

We're currently toying around with the idea of assigning processes to different
performance classes. The configuration could look something like this:

    processes:
    - name: network-server
      type: latency-critical

    - name: logging-agent
      type: best-effort

The exact ways that this affects the scheduling performance is pretty nebulous
and will probably require some experimentation to get right. We're currently
planning on trying an 80/20% split of CPU time between all of the latency
sensitive jobs and the best effort jobs. Jobs will be allowed to burst out of
this constraint if the other job types are not currently using it.

The [`cpu.shares` § 7][shares] parameter of the `cpu` cgroup can be used to
influence the scheduler to achieve the constraints we want.

This style of configuration would also allow us to add network traffic
prioritization behind the same class system at a later date.

[shares]: https://www.kernel.org/doc/Documentation/scheduler/sched-design-CFS.txt

## Backwards Incompatibilities

There are no backwards incompatibilities associated with this change. If a
release author doesn't specify a type then we will assume that the process is
latency critical. This attribute may become a requirement to set in later major
versions of BPM.

## Caveats & Risks

### Are these tasks actually all "best-effort"?

The classes of "latency-critical" and "best-effort" are taken from the Heracles
paper above. However, I'm not sure if they apply directly to our use case.
Google is able to dynamically fill servers with workloads which are batch
workloads which have extremely loose performance requirements (MapReduce jobs,
Google Brain number-crunching, etc.). Our agents still have performance
requirements and goals which they need to achieve for our customers but they're
not the most important thing on the system.

I think this is going to bring up difficult-to-answer questions like: "How much
latency are we willing to give up per percentage of log lines lost?".

### Difficulties in Testing

Testing scheduling reliably is challenging. There are so many factors which
influence the performance of a system which are out of our control that
creating a small test which generates reliable feedback is going to be
challenging.

### Interactions with Garden

The layout of how cgroups are organized on a system has important semantic
meaning to how the `cpu` cgroup resource constraints are applied. Up until now,
Garden and BPM have (largely) pretended that each other didn't exist at
runtime. We may need to do some more coordination to ensure that we both get
the results we're looking for.

### Interactions with Non-BPM Jobs

We have no control over the scheduling of processes which do not use BPM. There
is a risk they they do not behave properly and we then exacerbate that by
trying to fiddle with BPM'd processes.

The Linux CFS Scheduler seems to do a pretty good job giving reasonable fair
CPU time to processes with different session IDs. We already give each job a
different session ID so things may be "good enough" for now.

### How do we let operators debug that a process is CPU starved?

The overall machine has an attribute called "Steal %" which is a global
percentage of the times that the kernel asked the hypervisor for some CPU time
and was denied. It would be good if we could expose something like this for
each job/process but I don't know if it's availiable.
