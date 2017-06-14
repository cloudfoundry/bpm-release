package crucible_acceptance_test

import (
	"flag"
	"log"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCrucibleAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CrucibleAcceptance Suite")
}

var (
	agentURI string
	client   *http.Client
)

func init() {
	flag.StringVar(&agentURI, "agent-uri", "", "http address of the crucible-test-agent")
	flag.Parse()

	if agentURI == "" {
		log.Fatal("Agent URI must be provided")
	}
}

var _ = BeforeSuite(func() {
	client = &http.Client{
		Timeout: 10 * time.Second,
	}
})
