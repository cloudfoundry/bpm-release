# The Docker CLI does not need to be run as root in order to be safe in all
# cases. For example, when running on MacOS it is quarantined to a different
# virtual machine. If you are running daemon on a local machine then practice
# caution with sudo-less docker invocations.
user_can_docker() {
  ! docker info 2>&1 | grep -q "permission denied"
}
