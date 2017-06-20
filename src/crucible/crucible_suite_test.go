package main_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestCrucible(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Crucible Suite")
}

var (
	cruciblePath string
)

var _ = SynchronizedBeforeSuite(func() []byte {
	cruciblePath, err := gexec.Build("crucible")
	Expect(err).NotTo(HaveOccurred())

	return []byte(cruciblePath)
}, func(data []byte) {
	cruciblePath = string(data)
	SetDefaultEventuallyTimeout(2 * time.Second)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
