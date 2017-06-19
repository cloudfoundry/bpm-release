package runcadapter_test

import (
	"crucible/config"
	"crucible/runcadapter"
	"crucible/runcadapter/runcadapterfakes"
	"errors"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("RuncJobLifecycle", func() {
	var (
		fakeRuncAdapter *runcadapterfakes.FakeRuncAdapter

		jobConfig *config.CrucibleConfig
		jobSpec   specs.Spec

		expectedJobName,
		expectedBundlePath,
		expectedPidDir string

		expectedStdout, expectedStderr *os.File

		runcLifecycle *runcadapter.RuncJobLifecycle
	)

	BeforeEach(func() {
		expectedJobName = "example"
		jobConfig = &config.CrucibleConfig{
			Process: &config.Process{
				Executable: "/bin/sleep",
			},
		}
		jobSpec = specs.Spec{
			Version: "example-version",
		}

		fakeRuncAdapter = &runcadapterfakes.FakeRuncAdapter{}
		fakeRuncAdapter.BuildSpecReturns(jobSpec, nil)

		expectedBundlePath = "a-bundle-path"
		fakeRuncAdapter.CreateBundleReturns(expectedBundlePath, nil)

		var err error
		expectedPidDir = "a pid dir"
		expectedStdout, err = ioutil.TempFile("", "runc-lifecycle-stdout")
		Expect(err).NotTo(HaveOccurred())
		expectedStderr, err = ioutil.TempFile("", "runc-lifecycle-stderr")
		Expect(err).NotTo(HaveOccurred())
		fakeRuncAdapter.CreateJobPrerequisitesReturns(expectedPidDir, expectedStdout, expectedStderr, nil)

		runcLifecycle = runcadapter.NewRuncJobLifecycle(
			fakeRuncAdapter,
			expectedJobName,
			jobConfig,
		)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(expectedStdout.Name())).To(Succeed())
		Expect(os.RemoveAll(expectedStderr.Name())).To(Succeed())
	})

	Describe("StartJob", func() {
		It("builds the runc spec, bundle, and runs the container", func() {
			err := runcLifecycle.StartJob()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRuncAdapter.BuildSpecCallCount()).To(Equal(1))
			jobName, cfg := fakeRuncAdapter.BuildSpecArgsForCall(0)
			Expect(jobName).To(Equal(expectedJobName))
			Expect(cfg).To(Equal(jobConfig))

			Expect(fakeRuncAdapter.CreateBundleCallCount()).To(Equal(1))
			bundleRoot, jobName, spec := fakeRuncAdapter.CreateBundleArgsForCall(0)
			Expect(bundleRoot).To(Equal(config.BundlesRoot()))
			Expect(jobName).To(Equal(expectedJobName))
			Expect(spec).To(Equal(jobSpec))

			Expect(fakeRuncAdapter.CreateJobPrerequisitesCallCount()).To(Equal(1))
			systemRoot, jobName := fakeRuncAdapter.CreateJobPrerequisitesArgsForCall(0)
			Expect(systemRoot).To(Equal(config.BoshRoot()))
			Expect(jobName).To(Equal(expectedJobName))

			Expect(fakeRuncAdapter.RunContainerCallCount()).To(Equal(1))
			pidDir, bundlePath, jobName, stdout, stderr := fakeRuncAdapter.RunContainerArgsForCall(0)
			Expect(pidDir).To(Equal(expectedPidDir))
			Expect(bundlePath).To(Equal(expectedBundlePath))
			Expect(jobName).To(Equal(expectedJobName))
			Expect(stdout).To(Equal(expectedStdout))
			Expect(stderr).To(Equal(expectedStderr))
		})

		Context("when building the spec fails", func() {
			BeforeEach(func() {
				fakeRuncAdapter.BuildSpecReturns(specs.Spec{}, errors.New("boom!"))
			})

			It("returns an error", func() {
				err := runcLifecycle.StartJob()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when building the bundle fails", func() {
			BeforeEach(func() {
				fakeRuncAdapter.CreateBundleReturns("", errors.New("boom!"))
			})

			It("returns an error", func() {
				err := runcLifecycle.StartJob()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when creating the system files fails", func() {
			BeforeEach(func() {
				fakeRuncAdapter.CreateJobPrerequisitesReturns("", nil, nil, errors.New("boom"))
			})

			It("returns an error", func() {
				err := runcLifecycle.StartJob()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when running the container fails", func() {
			BeforeEach(func() {
				fakeRuncAdapter.RunContainerReturns(errors.New("boom!"))
			})

			It("returns an error", func() {
				err := runcLifecycle.StartJob()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("StopJob", func() {
		It("stops the container", func() {
			err := runcLifecycle.StopJob()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRuncAdapter.StopContainerCallCount()).To(Equal(1))
			jobName := fakeRuncAdapter.StopContainerArgsForCall(0)
			Expect(jobName).To(Equal(expectedJobName))
		})

		It("destroys the bundle", func() {
			err := runcLifecycle.StopJob()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRuncAdapter.DestroyBundleCallCount()).To(Equal(1))
			bundleRoot, jobName := fakeRuncAdapter.DestroyBundleArgsForCall(0)
			Expect(bundleRoot).To(Equal(config.BundlesRoot()))
			Expect(jobName).To(Equal(expectedJobName))
		})

		Context("when stopping a container fails", func() {
			It("returns an error", func() {
				expectedErr := errors.New("an error")
				fakeRuncAdapter.StopContainerReturns(expectedErr)
				err := runcLifecycle.StopJob()
				Expect(err).To(Equal(expectedErr))
			})
		})

		Context("when destroying a bundle fails", func() {
			It("returns an error", func() {
				expectedErr := errors.New("an error2")
				fakeRuncAdapter.DestroyBundleReturns(expectedErr)
				err := runcLifecycle.StopJob()
				Expect(err).To(Equal(expectedErr))
			})
		})
	})
})
