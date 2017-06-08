package runcadapter_test

import (
	"crucible/config"
	"crucible/runcadapter"
	"crucible/specbuilder"
	"crucible/specbuilder/specbuilderfakes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("Bundlebuilder", func() {
	var (
		adapter          runcadapter.RuncAdapater
		jobName          string
		bundleRoot       string
		fakeUserIDFinder *specbuilderfakes.FakeUserIDFinder
		jobSpec          specs.Spec
	)

	BeforeEach(func() {
		adapter = runcadapter.NewRuncAdapater("/var/vcap/packages/runc/bin/runc")
		jobName = "example"

		jobConfig := &config.CrucibleConfig{
			Process: &config.Process{
				Name:       jobName,
				Executable: "/bin/sleep",
				Args:       []string{"100"},
				Env:        []string{"FOO=BAR"},
			},
		}

		var err error
		fakeUserIDFinder = &specbuilderfakes.FakeUserIDFinder{}
		fakeUserIDFinder.LookupReturns(specs.User{UID: 200, GID: 300, Username: "jim"}, nil)
		jobSpec, err = specbuilder.Build(jobName, jobConfig, fakeUserIDFinder)

		bundleRoot, err = ioutil.TempDir("", "Bundlebuilder")
		Expect(err).ToNot(HaveOccurred())
	})

	Context("BuildBundle", func() {
		It("makes the bundle directory", func() {
			bundlePath, err := adapter.BuildBundle(bundleRoot, jobName, jobSpec)
			Expect(err).ToNot(HaveOccurred())

			f, err := os.Stat(bundlePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.IsDir()).To(BeTrue())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
		})

		It("makes an empty rootfs directory", func() {
			bundlePath, err := adapter.BuildBundle(bundleRoot, jobName, jobSpec)
			Expect(err).ToNot(HaveOccurred())

			bundlefs := filepath.Join(bundlePath, "rootfs")
			f, err := os.Stat(bundlefs)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.IsDir()).To(BeTrue())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))

			infos, err := ioutil.ReadDir(bundlefs)
			Expect(err).ToNot(HaveOccurred())
			Expect(infos).To(HaveLen(0))
		})

		It("writes a config.json in the root bundle directory", func() {
			bundlePath, err := adapter.BuildBundle(bundleRoot, jobName, jobSpec)
			Expect(err).ToNot(HaveOccurred())

			configPath := filepath.Join(bundlePath, "config.json")
			f, err := os.Stat(configPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))

			expectedConfigData, err := json.MarshalIndent(&jobSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			configData, err := ioutil.ReadFile(configPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(configData).To(MatchJSON(expectedConfigData))
		})

		Context("when creating the bundle directory fails", func() {
			BeforeEach(func() {
				_, err := os.Create(filepath.Join(bundleRoot, jobName))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				_, err := adapter.BuildBundle(bundleRoot, jobName, jobSpec)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when creating the rootfs directory fails", func() {
			BeforeEach(func() {
				bundlePath := filepath.Join(bundleRoot, jobName)
				err := os.MkdirAll(bundlePath, 0700)
				Expect(err).ToNot(HaveOccurred())

				_, err = os.Create(filepath.Join(bundlePath, "rootfs"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				_, err := adapter.BuildBundle(bundleRoot, jobName, jobSpec)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
