package config_test

import (
	"crucible/config"

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

			Expect(cfg.Name).To(Equal("server"))
			Expect(cfg.Executable).To(Equal("/var/vcap/packages/program/bin/program-server"))
			Expect(cfg.Args).To(ConsistOf("--port=2424", "--host=\"localhost\""))
			Expect(cfg.Env).To(ConsistOf("FOO=BAR", "BAZ=BUZZ"))
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
})
