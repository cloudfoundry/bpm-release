package runcadapter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRuncadapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Runcadapter Suite")
}
