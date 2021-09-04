module bpm

go 1.13

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20210608160410-67692ebc98de
	code.cloudfoundry.org/clock v1.0.0
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/golang/mock v1.6.0
	github.com/kr/pty v1.1.8
	github.com/moby/sys/mountinfo v0.4.1
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/opencontainers/runc v1.0.2
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/satori/go.uuid v1.2.0
	github.com/spf13/cobra v1.2.1
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00 // indirect
	golang.org/x/net v0.0.0-20210510120150-4163338589ed // indirect
	golang.org/x/sys v0.0.0-20210903071746-97244b99971b
	gopkg.in/yaml.v2 v2.4.0
)

exclude github.com/willf/bitset v1.2.0
