# Privileged Jobs

Aram Price  
Christopher Brown  
James Myers  
*February 2018*

## Background
Although we’d like to avoid running BOSH jobs as the root user it is sometimes
an inevitable fact that some pieces of software require this privilege. This
software either has esoteric requirements or interacts with the shared
resources of the operating system. BPM improved the isolation of the jobs which
use it but has shown it also provides usability benefits for BOSH release
authors. We’d like for all jobs, even those which need to run as root, to be
able to take advantage of the usability improvements provided by BPM.

# Objective
There are two releases (which we know of) which would like to run jobs as root:
Garden and KuBo. This proposal must meet their needs and the resulting
implementation should be considered incomplete or failed if it does not.

# Changes

Existing jobs should continue to run as the unprivileged *vcap* user without
any configuration changes. This means that we need new configuration to allow
for this case.

    unsafe:
      privileged: true

This configuration would be defined on a per-process basis.

Setting privileged to true would set the user to run as the root user and group
rather than the vcap user and group. The directories created by BPM would
retain their current ownership by the vcap user and group. This is to provide
an easy (easier) migration path forward if a user is able to remove their
dependency on the root user. The only place this would normally matter is the
persistent store directory.

This setting would also disable all current execution restrictions (seccomp,
capabilities). If we add any additional execution restrictions in the future
(AppArmor, etc.) then these would initially be behind feature flags before
being enabled. It is likely that these would also be disabled by this flag too.

This option does not affect the mount namespace or improvements we have made to
making the job lifecycle explicit.

## Backwards Incompatibilities

There are no backwards incompatibilities associated with this change.

## Caveats & Risks

### Unnecessary Usage

Users may see this new feature as a way out of doing the work required to make
sure that their job doesn’t need to run as root. We can try to prevent this in
two ways: prevention and auditing. Although not a particularly potent
deterrence we have included this new configuration under a new unsafe:
configuration section to warn users that it is not normal or safe to use this
section. This also serves as a rare and therefore easy-to-grep-for word which
can be used to find a list of all jobs which are using this feature.

### What is unsafe?

There are other parts of the configuration which could be called unsafe e.g.
the `allow_executions` setting on a volume. These can’t fit into this new
unsafe block due to their logical place in the configuration file. Does this
reduce the value of the above mitigation? How do users know which options are
safe when there are an ever increasing number of dials?

## Errata

There were a number of additional changes required by Garden in order for them
to adopt BPM.

* All volumes had their nosuid mount option removed if the process was run as
  privileged.
* The privileged capabilities were taken from the default list of capabilities
  which Docker uses.

