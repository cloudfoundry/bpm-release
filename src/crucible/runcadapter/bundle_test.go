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
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("Bundlebuilder", func() {
	var (
		adapter          runcadapter.RuncAdapter
		jobName          string
		bundlesRoot      string
		fakeUserIDFinder *specbuilderfakes.FakeUserIDFinder
		jobSpec          specs.Spec
	)

	BeforeEach(func() {
		fakeUserIDFinder = &specbuilderfakes.FakeUserIDFinder{}
		fakeUserIDFinder.LookupReturns(specs.User{UID: 200, GID: 300, Username: "jim"}, nil)

		adapter = runcadapter.NewRuncAdapater("/var/vcap/packages/runc/bin/runc", fakeUserIDFinder)
		jobName = "example"

		jobConfig := &config.CrucibleConfig{
			Process: &config.Process{
				Executable: "/bin/sleep",
				Args:       []string{"100"},
				Env:        []string{"FOO=BAR"},
			},
		}

		var err error
		jobSpec, err = specbuilder.Build(jobName, jobConfig, fakeUserIDFinder)

		bundlesRoot, err = ioutil.TempDir("", "Bundlebuilder")
		Expect(err).ToNot(HaveOccurred())
	})

	Context("BuildBundle", func() {
		It("makes the bundle directory", func() {
			bundlePath, err := adapter.BuildBundle(bundlesRoot, jobName, jobSpec)
			Expect(err).ToNot(HaveOccurred())

			f, err := os.Stat(bundlePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.IsDir()).To(BeTrue())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
		})

		It("makes an empty rootfs directory", func() {
			bundlePath, err := adapter.BuildBundle(bundlesRoot, jobName, jobSpec)
			Expect(err).ToNot(HaveOccurred())

			bundlefs := filepath.Join(bundlePath, "rootfs")
			f, err := os.Stat(bundlefs)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.IsDir()).To(BeTrue())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(f.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(f.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			infos, err := ioutil.ReadDir(bundlefs)
			Expect(err).ToNot(HaveOccurred())
			Expect(infos).To(HaveLen(0))
		})

		It("writes a config.json in the root bundle directory", func() {
			bundlePath, err := adapter.BuildBundle(bundlesRoot, jobName, jobSpec)
			Expect(err).ToNot(HaveOccurred())

			configPath := filepath.Join(bundlePath, "config.json")
			f, err := os.Stat(configPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0600)))

			expectedConfigData, err := json.MarshalIndent(&jobSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			configData, err := ioutil.ReadFile(configPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(configData).To(MatchJSON(expectedConfigData))
		})

		Context("when creating the bundle directory fails", func() {
			BeforeEach(func() {
				_, err := os.Create(filepath.Join(bundlesRoot, jobName))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				_, err := adapter.BuildBundle(bundlesRoot, jobName, jobSpec)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when creating the rootfs directory fails", func() {
			BeforeEach(func() {
				bundlePath := filepath.Join(bundlesRoot, jobName)
				err := os.MkdirAll(bundlePath, 0700)
				Expect(err).ToNot(HaveOccurred())

				_, err = os.Create(filepath.Join(bundlePath, "rootfs"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				_, err := adapter.BuildBundle(bundlesRoot, jobName, jobSpec)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("DestroyBundle", func() {
		var bundlePath string
		BeforeEach(func() {
			var err error
			bundlePath, err = adapter.BuildBundle(bundlesRoot, jobName, jobSpec)
			Expect(err).ToNot(HaveOccurred())
		})

		It("deletes the bundle", func() {
			err := adapter.DestroyBundle(bundlesRoot, jobName)
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(bundlePath)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})
