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

			expectedMemoryLimit := "100G"
			expectedOpenFilesLimit := uint64(100)
			Expect(cfg.Name).To(Equal("server"))
			Expect(cfg.Executable).To(Equal("/var/vcap/packages/program/bin/program-server"))
			Expect(cfg.Args).To(ConsistOf("--port=2424", "--host=\"localhost\""))
			Expect(cfg.Env).To(ConsistOf("FOO=BAR", "BAZ=BUZZ"))
			Expect(cfg.Limits.Memory).To(Equal(&expectedMemoryLimit))
			Expect(cfg.Limits.OpenFiles).To(Equal(&expectedOpenFilesLimit))
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
				configPath = "fixtures/example-invalid-yaml.yml"
			})

			It("returns an error", func() {
				_, err := config.ParseConfig(configPath)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the configuration is not valid", func() {
			BeforeEach(func() {
				configPath = "fixtures/example-invalid.yml"
			})

			It("returns an error", func() {
				_, err := config.ParseConfig(configPath)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Validate", func() {
		var cfg *config.CrucibleConfig

		BeforeEach(func() {
			cfg = &config.CrucibleConfig{
				Name:       "example",
				Executable: "executable",
			}
		})

		It("does not error on a valid config", func() {
			Expect(cfg.Validate()).To(Succeed())
		})

		Context("when the config does not have a Name", func() {
			BeforeEach(func() {
				cfg.Name = ""
			})

			It("returns an error", func() {
				Expect(cfg.Validate()).To(HaveOccurred())
			})
		})

		Context("when the config does not have an Executable", func() {
			BeforeEach(func() {
				cfg.Executable = ""
			})

			It("returns an error", func() {
				Expect(cfg.Validate()).To(HaveOccurred())
			})
		})
	})
})
