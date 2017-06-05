package main_test

import (
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
	cruciblePath, err := gexec.Build("crucible/cmd/crucible")
	Expect(err).NotTo(HaveOccurred())

	return []byte(cruciblePath)
}, func(data []byte) {
	cruciblePath = string(data)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
