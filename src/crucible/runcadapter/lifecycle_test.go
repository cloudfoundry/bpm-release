package runcadapter_test

import (
	"crucible/config"
	"crucible/runcadapter"
	"crucible/runcadapter/runcadapterfakes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"

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
		expectedProcName,
		expectedContainerID,
		expectedBundlePath,
		expectedPidDir string

		expectedStdout, expectedStderr *os.File

		fakeClock *fakeclock.FakeClock

		runcLifecycle *runcadapter.RuncJobLifecycle
	)

	BeforeEach(func() {
		fakeClock = fakeclock.NewFakeClock(time.Now())

		expectedJobName = "example"
		expectedProcName = "server"
		expectedContainerID = fmt.Sprintf("%s-%s", expectedJobName, expectedProcName)

		jobConfig = &config.CrucibleConfig{
			Executable: "/bin/sleep",
			Name:       "server",
		}
		jobSpec = specs.Spec{
			Version: "example-version",
		}

		fakeRuncAdapter = &runcadapterfakes.FakeRuncAdapter{}
		fakeRuncAdapter.BuildSpecReturns(jobSpec, nil)

		expectedBundlePath = "a-bundle-path"
		fakeRuncAdapter.CreateBundleReturns(expectedBundlePath, nil)

		var err error
		expectedPidDir = "a-pid-dir"
		expectedStdout, err = ioutil.TempFile("", "runc-lifecycle-stdout")
		Expect(err).NotTo(HaveOccurred())
		expectedStderr, err = ioutil.TempFile("", "runc-lifecycle-stderr")
		Expect(err).NotTo(HaveOccurred())
		fakeRuncAdapter.CreateJobPrerequisitesReturns(expectedPidDir, expectedStdout, expectedStderr, nil)

		runcLifecycle = runcadapter.NewRuncJobLifecycle(
			fakeRuncAdapter,
			fakeClock,
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
			bundleRoot, jobName, procName, spec := fakeRuncAdapter.CreateBundleArgsForCall(0)
			Expect(bundleRoot).To(Equal(config.BundlesRoot()))
			Expect(jobName).To(Equal(expectedJobName))
			Expect(procName).To(Equal(expectedProcName))
			Expect(spec).To(Equal(jobSpec))

			Expect(fakeRuncAdapter.CreateJobPrerequisitesCallCount()).To(Equal(1))
			systemRoot, jobName, procName := fakeRuncAdapter.CreateJobPrerequisitesArgsForCall(0)
			Expect(systemRoot).To(Equal(config.BoshRoot()))
			Expect(jobName).To(Equal(expectedJobName))
			Expect(procName).To(Equal(expectedProcName))

			Expect(fakeRuncAdapter.RunContainerCallCount()).To(Equal(1))
			pidFilePath, bundlePath, containerID, stdout, stderr := fakeRuncAdapter.RunContainerArgsForCall(0)
			Expect(pidFilePath).To(Equal(filepath.Join(expectedPidDir, fmt.Sprintf("%s.pid", expectedProcName))))
			Expect(bundlePath).To(Equal(expectedBundlePath))
			Expect(containerID).To(Equal(expectedContainerID))
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
		var exitTimeout time.Duration
		BeforeEach(func() {
			exitTimeout = 5 * time.Second

			fakeRuncAdapter.ContainerStateReturns(specs.State{
				Status: "stopped",
			}, nil)
		})

		It("stops the container", func() {
			err := runcLifecycle.StopJob(exitTimeout)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRuncAdapter.StopContainerCallCount()).To(Equal(1))
			containerID := fakeRuncAdapter.StopContainerArgsForCall(0)
			Expect(containerID).To(Equal(expectedContainerID))

			Expect(fakeRuncAdapter.ContainerStateCallCount()).To(Equal(1))
			containerID = fakeRuncAdapter.ContainerStateArgsForCall(0)
			Expect(containerID).To(Equal(expectedContainerID))
		})

		Context("when the container does not stop immediately", func() {
			var stopped chan struct{}
			BeforeEach(func() {
				stopped = make(chan struct{})

				fakeRuncAdapter.ContainerStateStub = func(containerID string) (specs.State, error) {
					select {
					case <-stopped:
						return specs.State{Status: "stopped"}, nil
					default:
						return specs.State{Status: "running"}, nil
					}
				}
			})

			It("polls the container state every second until it stops", func() {
				errChan := make(chan error)
				go func() {
					defer GinkgoRecover()
					errChan <- runcLifecycle.StopJob(exitTimeout)
				}()

				Eventually(fakeRuncAdapter.StopContainerCallCount).Should(Equal(1))
				Expect(fakeRuncAdapter.StopContainerArgsForCall(0)).To(Equal(expectedContainerID))

				Eventually(fakeRuncAdapter.ContainerStateCallCount).Should(Equal(1))
				Expect(fakeRuncAdapter.ContainerStateArgsForCall(0)).To(Equal(expectedContainerID))

				fakeClock.WaitForWatcherAndIncrement(1 * time.Second)

				Eventually(fakeRuncAdapter.ContainerStateCallCount).Should(Equal(2))
				Expect(fakeRuncAdapter.ContainerStateArgsForCall(1)).To(Equal(expectedContainerID))

				close(stopped)
				fakeClock.WaitForWatcherAndIncrement(1 * time.Second)

				Eventually(fakeRuncAdapter.ContainerStateCallCount).Should(Equal(3))
				Expect(fakeRuncAdapter.ContainerStateArgsForCall(2)).To(Equal(expectedContainerID))

				Eventually(errChan).Should(Receive(BeNil()), "stop job did not exit in time")
			})

			Context("and the exit timeout has passed", func() {
				It("forcefully removes the container", func() {
					errChan := make(chan error)
					go func() {
						defer GinkgoRecover()
						errChan <- runcLifecycle.StopJob(exitTimeout)
					}()

					Eventually(fakeRuncAdapter.StopContainerCallCount).Should(Equal(1))
					Expect(fakeRuncAdapter.StopContainerArgsForCall(0)).To(Equal(expectedContainerID))

					Eventually(fakeRuncAdapter.ContainerStateCallCount).Should(Equal(1))
					Expect(fakeRuncAdapter.ContainerStateArgsForCall(0)).To(Equal(expectedContainerID))

					fakeClock.WaitForWatcherAndIncrement(1 * time.Second)

					Eventually(fakeRuncAdapter.ContainerStateCallCount).Should(Equal(2))
					Expect(fakeRuncAdapter.ContainerStateArgsForCall(1)).To(Equal(expectedContainerID))

					fakeClock.WaitForWatcherAndIncrement(exitTimeout)

					var actualError error
					Eventually(errChan).Should(Receive(&actualError))
					Expect(actualError).To(Equal(runcadapter.TimeoutError))
				})
			})
		})

		Context("when fetching the container state fails", func() {
			BeforeEach(func() {
				fakeRuncAdapter.ContainerStateReturns(specs.State{}, errors.New("boom"))
			})

			It("keeps attempting to fetch the state", func() {
				errChan := make(chan error)
				go func() {
					defer GinkgoRecover()
					errChan <- runcLifecycle.StopJob(exitTimeout)
				}()

				Eventually(fakeRuncAdapter.ContainerStateCallCount).Should(Equal(1))
				Expect(fakeRuncAdapter.ContainerStateArgsForCall(0)).To(Equal(expectedContainerID))

				fakeClock.WaitForWatcherAndIncrement(1 * time.Second)

				Eventually(fakeRuncAdapter.ContainerStateCallCount).Should(Equal(2))
				Expect(fakeRuncAdapter.ContainerStateArgsForCall(1)).To(Equal(expectedContainerID))

				fakeClock.WaitForWatcherAndIncrement(1 * time.Second)

				Eventually(fakeRuncAdapter.ContainerStateCallCount).Should(Equal(3))
				Expect(fakeRuncAdapter.ContainerStateArgsForCall(2)).To(Equal(expectedContainerID))

				fakeClock.WaitForWatcherAndIncrement(exitTimeout)

				var actualError error
				Eventually(errChan).Should(Receive(&actualError))
				Expect(actualError).To(Equal(runcadapter.TimeoutError))
			})
		})

		Context("when stopping a container fails", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("an error")
				fakeRuncAdapter.StopContainerReturns(expectedErr)
			})

			It("returns an error", func() {
				err := runcLifecycle.StopJob(exitTimeout)
				Expect(err).To(Equal(expectedErr))
			})
		})
	})

	Describe("RemoveJob", func() {
		It("deletes the container", func() {
			err := runcLifecycle.RemoveJob()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRuncAdapter.DeleteContainerCallCount()).To(Equal(1))
			containerID := fakeRuncAdapter.DeleteContainerArgsForCall(0)
			Expect(containerID).To(Equal(expectedContainerID))
		})

		It("destroys the bundle", func() {
			err := runcLifecycle.RemoveJob()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRuncAdapter.DestroyBundleCallCount()).To(Equal(1))
			bundleRoot, jobName, procName := fakeRuncAdapter.DestroyBundleArgsForCall(0)
			Expect(bundleRoot).To(Equal(config.BundlesRoot()))
			Expect(jobName).To(Equal(expectedJobName))
			Expect(procName).To(Equal(expectedProcName))
		})

		Context("when deleting a container fails", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("an error")
				fakeRuncAdapter.DeleteContainerReturns(expectedErr)
			})

			It("returns an error", func() {
				err := runcLifecycle.RemoveJob()
				Expect(err).To(Equal(expectedErr))
			})
		})

		Context("when destroying a bundle fails", func() {
			It("returns an error", func() {
				expectedErr := errors.New("an error2")
				fakeRuncAdapter.DestroyBundleReturns(expectedErr)
				err := runcLifecycle.RemoveJob()
				Expect(err).To(Equal(expectedErr))
			})
		})
	})
})
