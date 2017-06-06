package config_test

import (
	"crucible/config"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("ParseConfig", func() {
		var configPath string

		BeforeEach(func() {
			configPath = "fixtures/example.yml"
		})

		It("parses a yaml file into a crucible config", func() {
			cfg, err := config.ParseConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(cfg.Process.Name).To(Equal("server"))
			Expect(cfg.Process.Executable).To(Equal("/var/vcap/packages/program/bin/program-server"))
			Expect(cfg.Process.Args).To(ConsistOf("--port=2424", "--host=\"localhost\""))
			Expect(cfg.Process.Env).To(ConsistOf("FOO=BAR", "BAZ=BUZZ"))
		})

		Context("when reading the file fails", func() {
			BeforeEach(func() {
				configPath = "does-not-exist"
			})

			It("returns an error", func() {
				_, err := config.ParseConfig(configPath)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the yaml is invalid", func() {
			BeforeEach(func() {
				configPath = "fixtures/example-invalid.yml"
			})

			It("returns an error", func() {
				_, err := config.ParseConfig(configPath)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ConfigPath", func() {
		It("returns the default BOSH config path", func() {
			path := config.ConfigPath("jim")
			Expect(path).To(Equal("/var/vcap/jobs/jim/config/crucible.yml"))
		})

		Context("when the CRUCIBLE_BOSH_ROOT env var is set", func() {
			It("returns the BOSH config path based from that root", func() {
				originalVal := os.Getenv("CRUCIBLE_BOSH_ROOT")
				Expect(os.Setenv("CRUCIBLE_BOSH_ROOT", "/home/cf")).To(Succeed())

				path := config.ConfigPath("jim")
				Expect(path).To(Equal("/home/cf/jobs/jim/config/crucible.yml"))

				Expect(os.Setenv("CRUCIBLE_BOSH_ROOT", originalVal)).To(Succeed())
			})
		})
	})
})
