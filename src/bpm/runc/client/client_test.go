package client_test

import (
	"bpm/runc/client"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("RuncClient", func() {
	var (
		runcClient client.RuncClient
		jobSpec    specs.Spec
		bundlePath string
		user       specs.User
	)

	BeforeEach(func() {
		user = specs.User{UID: 200, GID: 300, Username: "vcap"}
		runcClient = client.NewRuncClient(
			"/var/vcap/packages/runc/bin/runc",
			"/var/vcap/data/bpm/runc",
		)
	})

	Context("CreateBundle", func() {
		var bundlesRoot string

		BeforeEach(func() {
			jobSpec = specs.Spec{
				Version: "example-version",
			}

			var err error
			bundlesRoot, err = ioutil.TempDir("", "bundle-builder")
			Expect(err).ToNot(HaveOccurred())

			bundlePath = filepath.Join(bundlesRoot, "bundle")
		})

		AfterEach(func() {
			Expect(os.RemoveAll(bundlesRoot)).To(Succeed())
		})

		It("makes the bundle directory", func() {
			err := runcClient.CreateBundle(bundlePath, jobSpec, user)
			Expect(err).ToNot(HaveOccurred())

			f, err := os.Stat(bundlePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.IsDir()).To(BeTrue())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
		})

		It("makes an empty rootfs directory", func() {
			err := runcClient.CreateBundle(bundlePath, jobSpec, user)
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
			err := runcClient.CreateBundle(bundlePath, jobSpec, user)
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
				_, err := os.Create(bundlePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				err := runcClient.CreateBundle(bundlePath, jobSpec, user)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when creating the rootfs directory fails", func() {
			BeforeEach(func() {
				err := os.MkdirAll(bundlePath, 0700)
				Expect(err).ToNot(HaveOccurred())

				_, err = os.Create(filepath.Join(bundlePath, "rootfs"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				err := runcClient.CreateBundle(bundlePath, jobSpec, user)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("DestroyBundle", func() {
		var bundlePath string

		BeforeEach(func() {
			var err error
			bundlePath, err = ioutil.TempDir("", "bundle-builder")
			Expect(err).ToNot(HaveOccurred())

			jobSpec := specs.Spec{
				Version: "test-version",
			}
			user := specs.User{Username: "vcap", UID: 300, GID: 400}

			err = runcClient.CreateBundle(bundlePath, jobSpec, user)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(bundlePath)).To(Succeed())
		})

		It("deletes the bundle", func() {
			err := runcClient.DestroyBundle(bundlePath)
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(bundlePath)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})
