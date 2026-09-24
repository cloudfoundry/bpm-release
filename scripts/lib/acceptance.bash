# shellcheck shell=bash
# Helpers for the run-acceptance-specs-* scripts. Expects RELEASE_PATH, and
# BOSH_DEPLOYMENT naming a deployment of manifests/bosh-lite-ci.yml.

BUNDLE_CONFIG=/var/vcap/data/bpm/bundles/test-server/test-server/config.json

vm() {
  bosh ssh bpm/0 -c "set -euo pipefail; $1"
}

vm_output() {
  bosh --column=stdout ssh bpm/0 -r -c "set -euo pipefail; $1" | tr -d '\r'
}

# The stemcell has python3 but not jq.
bundle_arg0() {
  vm_output "sudo python3 -c 'import json; print(json.load(open(\"${BUNDLE_CONFIG}\"))[\"process\"][\"args\"][0])'" | tr -d '[:space:]'
}

# Fails unless the test-server container runs under the given tini.
expect_arg0() {
  local actual
  actual="$(bundle_arg0)"
  if [ "${actual}" != "$1" ]; then
    echo "expected process.args[0] to be $1, got ${actual}" >&2
    exit 1
  fi
}

# Runs the acceptance specs against the test-server on bpm/0.
run_specs() {
  local agent_host
  agent_host="$(bosh instances | grep running | awk '{ print $4 }')"

  "${RELEASE_PATH}/scripts/go-generate"

  pushd "${RELEASE_PATH}/src/bpm/acceptance" > /dev/null || return
    # GINKGO_EXTRA_ARGS is for local runs, e.g. --skip-package=fixtures on macOS.
    # shellcheck disable=SC2086
    go run github.com/onsi/ginkgo/v2/ginkgo -r -p --race --randomize-all ${GINKGO_EXTRA_ARGS:-} -- \
      --agent-uri="http://${agent_host}:1337" \
      --observer-uri="http://${agent_host}:1339"
  popd > /dev/null || return
}
