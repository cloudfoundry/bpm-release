package crucible_acceptance_test

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CrucibleAcceptance", func() {
	It("returns a 200 response with a body", func() {
		resp, err := client.Get(agentURI)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(Equal("Crucible is SWEET!\n"))
	})

	It("runs as the vcap user", func() {
		resp, err := client.Get(fmt.Sprintf("%s/whoami", agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(Equal("vcap\n"))
	})

	It("has the correct hostname", func() {
		resp, err := client.Get(fmt.Sprintf("%s/hostname", agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(Equal("crucible-test-agent\n"))
	})

	It("has the correct mounts", func() {
		resp, err := client.Get(fmt.Sprintf("%s/mounts", agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		mounts := parseMounts(string(body))

		expectedMountPaths := []string{"/var/vcap/packages", "/var/vcap/data/packages", "/var/vcap/jobs/crucible-test-agent"}
		var found []string
		for _, mount := range mounts {
			if strings.Contains(mount.path, "/var/vcap") {
				found = append(found, mount.path)
				Expect(mount.options).To(ContainElement("ro"), fmt.Sprintf("no read only permissions for %s", mount.path))
			}
		}

		Expect(found).To(ConsistOf(expectedMountPaths))
	})

	It("only has access to data, jobs, and packages in /var/vcap", func() {
		resp, err := client.Get(fmt.Sprintf("%s/var-vcap", agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		directories := strings.Split(strings.Trim(string(body), "\n"), "\n")
		Expect(directories).To(ConsistOf("data", "jobs", "packages"))
	})

	It("is contained in a pid namespace", func() {
		resp, err := client.Get(fmt.Sprintf("%s/processes", agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		processes := strings.Split(strings.Trim(string(body), "\n"), "\n")

		// We expect the test agent to be the only process with the root PID
		Expect(processes).To(HaveLen(1))
		Expect(processes).To(ConsistOf(MatchRegexp("1 /var/vcap/packages/crucible-test-agent/bin/crucible-test-agent.*")))
	})
})

func containsString(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

type mount struct {
	path    string
	options []string
}

func parseMounts(mountData string) []mount {
	results := []mount{}
	scanner := bufio.NewScanner(strings.NewReader(mountData))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		columns := strings.Split(line, " ")
		options := strings.Split(columns[3], ",")
		sort.Strings(options)

		results = append(results, mount{
			path:    columns[1],
			options: options,
		})
	}

	return results
}
