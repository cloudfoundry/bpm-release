module bpm

go 1.13

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20190819182555-854d396b647c
	code.cloudfoundry.org/clock v0.0.0-20180518195852-02e53af36e6c
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/golang/mock v1.4.1-0.20200222213444-b48cb6623c04
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/kr/pty v1.1.8
	github.com/moby/sys/mountinfo v0.4.1
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.0
	github.com/opencontainers/runc v1.0.0-rc94
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00 // indirect
	golang.org/x/net v0.0.0-20210510120150-4163338589ed // indirect
	golang.org/x/sys v0.0.0-20210511113859-b0526f3d8744
	gopkg.in/yaml.v2 v2.2.4
)

exclude github.com/willf/bitset v1.2.0
