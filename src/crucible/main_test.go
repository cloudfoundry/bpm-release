package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Crucible", func() {
	var boshConfigPath string

	BeforeEach(func() {
		var err error
		boshConfigPath, err = ioutil.TempDir("", "crucible-main-test")
		Expect(err).NotTo(HaveOccurred())
	})

	Context("start", func() {
		var (
			command *exec.Cmd
			jobName string
		)

		BeforeEach(func() {
			jobName = "odd-job"
			jobConfigPath := filepath.Join(boshConfigPath, "jobs", jobName, "config")
			err := os.MkdirAll(jobConfigPath, 0755)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.OpenFile(
				filepath.Join(jobConfigPath, "crucible.yml"),
				os.O_RDONLY|os.O_CREATE,
				0644,
			)
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			command = exec.Command(cruciblePath, "start", jobName)
			command.Env = append(command.Env, fmt.Sprintf("CRUCIBLE_BOSH_ROOT=%s", boshConfigPath))
		})

		It("runs", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})

		Context("when the crucible configuration file does not exist", func() {
			BeforeEach(func() {
				jobName = "even-job"
			})

			It("exit with a non-zero exit code and prints an error", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(session.Err).Should(gbytes.Say(
					"Error: failed to load config at %s: ",
					filepath.Join(boshConfigPath, "jobs", "even-job", "config", "crucible.yml"),
				))
			})
		})

		Context("when no job name is specified", func() {
			It("exits with a non-zero exit code and prints the usage", func() {
				command = exec.Command(cruciblePath, "start")
				command.Env = append(command.Env, fmt.Sprintf("CRUCIBLE_BOSH_ROOT=%s", boshConfigPath))

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(session.Err).Should(gbytes.Say("must specify a job name"))
			})
		})
	})

	Context("when no arguments are provided", func() {
		It("exits with a non-zero exit code and prints the usage", func() {
			command := exec.Command(cruciblePath)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).Should(gbytes.Say("Usage:"))
		})
	})
})
