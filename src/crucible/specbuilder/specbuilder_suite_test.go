package specbuilder_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSpecbuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Specbuilder Suite")
}
