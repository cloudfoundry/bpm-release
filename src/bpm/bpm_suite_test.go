package main_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestBpm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "bpm Suite")
}

var (
	bpmPath string
)

var _ = SynchronizedBeforeSuite(func() []byte {
	bpmPath, err := gexec.Build("bpm")
	Expect(err).NotTo(HaveOccurred())

	return []byte(bpmPath)
}, func(data []byte) {
	bpmPath = string(data)
	SetDefaultEventuallyTimeout(2 * time.Second)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
