package main_test

import (
	"crucible/config"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func chownR(path string, uid, gid int) error {
	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			err = os.Chown(name, uid, gid)
		}
		return err
	})
}

var _ = Describe("Crucible", func() {
	var boshConfigPath string

	BeforeEach(func() {
		var err error
		boshConfigPath, err = ioutil.TempDir("", "crucible-main-test")
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(filepath.Join(boshConfigPath, "packages"), 0755)
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(filepath.Join(boshConfigPath, "data", "packages"), 0755)
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(filepath.Join(boshConfigPath, "packages", "runc", "bin"), 0755)
		Expect(err).NotTo(HaveOccurred())

		runcPath, err := exec.LookPath("runc")
		Expect(err).NotTo(HaveOccurred())

		err = os.Link(runcPath, filepath.Join(boshConfigPath, "packages", "runc", "bin", "runc"))
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(boshConfigPath)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("start", func() {
		var (
			command *exec.Cmd
			jobName string
			jobDir  string
		)

		BeforeEach(func() {
			jobName = fmt.Sprintf("example-%d", GinkgoParallelNode())

			jobDir = filepath.Join(boshConfigPath, "jobs", jobName)
			jobConfigPath := filepath.Join(jobDir, "config")
			err := os.MkdirAll(jobConfigPath, 0755)
			Expect(err).NotTo(HaveOccurred())

			jobConfig := config.CrucibleConfig{
				Process: &config.Process{
					Name:       jobName,
					Executable: "/bin/sleep",
					Args:       []string{"10"},
					Env:        []string{"FOO=BAR"},
				},
			}

			f, err := os.OpenFile(
				filepath.Join(jobConfigPath, "crucible.yml"),
				os.O_RDWR|os.O_CREATE,
				0644,
			)
			Expect(err).NotTo(HaveOccurred())

			data, err := yaml.Marshal(jobConfig)
			Expect(err).NotTo(HaveOccurred())

			n, err := f.Write(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(len(data)))

			err = chownR(boshConfigPath, 2000, 3000)
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			command = exec.Command(cruciblePath, "start", jobName)
			command.Env = append(command.Env, fmt.Sprintf("CRUCIBLE_BOSH_ROOT=%s", boshConfigPath))
		})

		AfterEach(func() {
			// using force, as we cannot delete a running container.
			cmd := exec.Command("runc", "delete", "--force", jobName)
			combinedOutput, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(combinedOutput))
		})

		It("runs", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			cmd := exec.Command("runc", "state", jobName)
			data, err := cmd.Output()
			Expect(err).NotTo(HaveOccurred())

			stateResponse := specs.State{}
			err = json.Unmarshal(data, &stateResponse)
			Expect(err).NotTo(HaveOccurred())

			Expect(stateResponse.ID).To(Equal(jobName))
			Expect(stateResponse.Status).To(Equal("running"))
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
