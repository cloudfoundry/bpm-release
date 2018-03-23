// Copyright (C) 2017-Present CloudFoundry.org Foundation, Inc. All rights reserved.
//
// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License”);
// you may not use this file except in compliance with the License.
//
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
// License for the specific language governing permissions and limitations
// under the License.

package lifecycle_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"bpm/config"
	"bpm/models"
	"bpm/runc/client"
	"bpm/runc/lifecycle"
	"bpm/runc/lifecycle/lifecyclefakes"
	"bpm/usertools"
)

var _ = Describe("RuncJobLifecycle", func() {
	var (
		fakeRuncAdapter   *lifecyclefakes.FakeRuncAdapter
		fakeRuncClient    *lifecyclefakes.FakeRuncClient
		fakeUserFinder    *lifecyclefakes.FakeUserFinder
		fakeCommandRunner *lifecyclefakes.FakeCommandRunner

		logger *lagertest.TestLogger

		bpmCfg  *config.BPMConfig
		procCfg *config.ProcessConfig
		jobSpec specs.Spec

		expectedJobName,
		expectedProcName,
		expectedContainerID,
		expectedSystemRoot string

		expectedStdout, expectedStderr *os.File

		expectedUser specs.User

		fakeClock *fakeclock.FakeClock

		runcLifecycle *lifecycle.RuncLifecycle
	)

	BeforeEach(func() {
		fakeClock = fakeclock.NewFakeClock(time.Now())
		fakeRuncAdapter = &lifecyclefakes.FakeRuncAdapter{}
		fakeRuncClient = &lifecyclefakes.FakeRuncClient{}
		fakeUserFinder = &lifecyclefakes.FakeUserFinder{}
		fakeCommandRunner = &lifecyclefakes.FakeCommandRunner{}

		logger = lagertest.NewTestLogger("lifecycle")

		expectedUser = specs.User{Username: "vcap", UID: 300, GID: 400}
		fakeUserFinder.LookupReturns(expectedUser, nil)

		var err error
		expectedStdout, err = ioutil.TempFile("", "runc-lifecycle-stdout")
		Expect(err).NotTo(HaveOccurred())
		expectedStderr, err = ioutil.TempFile("", "runc-lifecycle-stderr")
		Expect(err).NotTo(HaveOccurred())
		fakeRuncAdapter.CreateJobPrerequisitesReturns(expectedStdout, expectedStderr, nil)

		expectedJobName = "example"
		expectedProcName = "server"
		expectedContainerID = config.Encode(fmt.Sprintf("%s.%s", expectedJobName, expectedProcName))

		procCfg = &config.ProcessConfig{
			Executable: "/bin/sleep",
		}
		jobSpec = specs.Spec{
			Process: &specs.Process{
				Env: []string{"foo=bar"},
			},
			Version: "example-version",
		}
		fakeRuncAdapter.BuildSpecReturns(jobSpec, nil)

		expectedSystemRoot = "system-root"

		runcLifecycle = lifecycle.NewRuncLifecycle(
			fakeRuncClient,
			fakeRuncAdapter,
			fakeUserFinder,
			fakeCommandRunner,
			fakeClock,
		)
		bpmCfg = config.NewBPMConfig(expectedSystemRoot, expectedJobName, expectedProcName)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(expectedStdout.Name())).To(Succeed())
		Expect(os.RemoveAll(expectedStderr.Name())).To(Succeed())
	})

	Describe("StartProcess", func() {
		It("builds the runc spec, bundle, and runs the container", func() {
			err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeUserFinder.LookupCallCount()).To(Equal(1))
			Expect(fakeUserFinder.LookupArgsForCall(0)).To(Equal(usertools.VcapUser))

			Expect(fakeRuncAdapter.CreateJobPrerequisitesCallCount()).To(Equal(1))
			actualBPMCfg, actualProcCfg, user := fakeRuncAdapter.CreateJobPrerequisitesArgsForCall(0)
			Expect(actualBPMCfg).To(Equal(bpmCfg))
			Expect(actualProcCfg).To(Equal(procCfg))
			Expect(user).To(Equal(expectedUser))

			Expect(fakeRuncAdapter.BuildSpecCallCount()).To(Equal(1))
			_, actualBPMCfg, actualProcCfg, user = fakeRuncAdapter.BuildSpecArgsForCall(0)
			Expect(actualBPMCfg).To(Equal(bpmCfg))
			Expect(actualProcCfg).To(Equal(procCfg))
			Expect(user).To(Equal(expectedUser))

			Expect(fakeRuncClient.CreateBundleCallCount()).To(Equal(1))
			bundlePath, spec, user := fakeRuncClient.CreateBundleArgsForCall(0)
			Expect(bundlePath).To(Equal(filepath.Join(expectedSystemRoot, "data", "bpm", "bundles", expectedJobName, expectedProcName)))
			Expect(spec).To(Equal(jobSpec))
			Expect(user).To(Equal(expectedUser))

			Expect(fakeRuncClient.RunContainerCallCount()).To(Equal(1))
			pidFilePath, bundlePath, cid, stdout, stderr := fakeRuncClient.RunContainerArgsForCall(0)
			Expect(pidFilePath).To(Equal(bpmCfg.PidFile()))
			Expect(bundlePath).To(Equal(filepath.Join(expectedSystemRoot, "data", "bpm", "bundles", expectedJobName, expectedProcName)))
			Expect(cid).To(Equal(expectedContainerID))
			Expect(stdout).To(Equal(expectedStdout))
			Expect(stderr).To(Equal(expectedStderr))
		})

		Context("when a PreStart Hook is provided", func() {
			BeforeEach(func() {
				procCfg.Hooks = &config.Hooks{
					PreStart: "/please/execute/me",
				}
			})

			It("executes the pre start hook", func() {
				err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
				Expect(err).NotTo(HaveOccurred())

				expectedCommand := exec.Command(procCfg.Hooks.PreStart)
				expectedCommand.Stdout = expectedStdout
				expectedCommand.Stderr = expectedStderr
				expectedCommand.Env = []string{"foo=bar"}

				Expect(fakeCommandRunner.RunCallCount()).To(Equal(1))
				Expect(fakeCommandRunner.RunArgsForCall(0)).To(Equal(expectedCommand))
			})

			Context("when the PreStart Hook fails", func() {
				BeforeEach(func() {
					fakeCommandRunner.RunReturns(errors.New("fake test error"))
				})

				It("returns an error", func() {
					err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when PreStart Hook is empty", func() {
			BeforeEach(func() {
				procCfg.Hooks = &config.Hooks{
					PreStart: "",
				}
			})

			It("ignores the pre start hook", func() {
				err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the process name is the same as the job name", func() {
			BeforeEach(func() {
				bpmCfg = config.NewBPMConfig(expectedSystemRoot, expectedJobName, expectedJobName)
			})

			It("simplifies the container ID", func() {
				err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeRuncClient.RunContainerCallCount()).To(Equal(1))
				_, _, cid, _, _ := fakeRuncClient.RunContainerArgsForCall(0)
				Expect(cid).To(Equal(config.Encode(expectedJobName)))
			})
		})

		Context("when looking up the vcap user fails", func() {
			BeforeEach(func() {
				fakeUserFinder.LookupReturns(specs.User{}, errors.New("fake test error"))
			})

			It("returns an error", func() {
				err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when creating the system files fails", func() {
			BeforeEach(func() {
				fakeRuncAdapter.CreateJobPrerequisitesReturns(nil, nil, errors.New("fake test error"))
			})

			It("returns an error", func() {
				err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when building the runc spec fails", func() {
			BeforeEach(func() {
				fakeRuncAdapter.BuildSpecReturns(specs.Spec{}, errors.New("fake test error"))
			})

			It("returns an error", func() {
				err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when building the bundle fails", func() {
			BeforeEach(func() {
				fakeRuncClient.CreateBundleReturns(errors.New("fake test error"))
			})

			It("returns an error", func() {
				err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when running the container fails", func() {
			BeforeEach(func() {
				fakeRuncClient.RunContainerReturns(errors.New("fake test error"))
			})

			It("returns an error", func() {
				err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("StopProcess", func() {
		var exitTimeout time.Duration

		BeforeEach(func() {
			exitTimeout = 5 * time.Second

			fakeRuncClient.ContainerStateReturns(&specs.State{
				Status: "stopped",
			}, nil)
		})

		It("stops the container", func() {
			errChan := make(chan error)
			go func() {
				defer GinkgoRecover()
				errChan <- runcLifecycle.StopProcess(logger, bpmCfg, exitTimeout)
			}()

			Eventually(fakeRuncClient.SignalContainerCallCount).Should(Equal(1))
			cid, signal := fakeRuncClient.SignalContainerArgsForCall(0)
			Expect(cid).To(Equal(expectedContainerID))
			Expect(signal).To(Equal(client.Term))
		})

		Context("when the container does not stop immediately", func() {
			var stopped chan struct{}

			BeforeEach(func() {
				stopped = make(chan struct{})

				fakeRuncClient.ContainerStateStub = func(containerID string) (*specs.State, error) {
					select {
					case <-stopped:
						return &specs.State{Status: "stopped"}, nil
					default:
						return &specs.State{Status: "running"}, nil
					}
				}
			})

			It("polls the container state every second until it stops", func() {
				errChan := make(chan error)
				go func() {
					defer GinkgoRecover()
					errChan <- runcLifecycle.StopProcess(logger, bpmCfg, exitTimeout)
				}()

				Eventually(fakeRuncClient.SignalContainerCallCount).Should(Equal(1))
				cid, signal := fakeRuncClient.SignalContainerArgsForCall(0)
				Expect(cid).To(Equal(expectedContainerID))
				Expect(signal).To(Equal(client.Term))

				Eventually(fakeRuncClient.ContainerStateCallCount).Should(Equal(1))
				Expect(fakeRuncClient.ContainerStateArgsForCall(0)).To(Equal(expectedContainerID))

				fakeClock.WaitForWatcherAndIncrement(lifecycle.ContainerStatePollInterval)

				Eventually(fakeRuncClient.ContainerStateCallCount).Should(Equal(2))
				Expect(fakeRuncClient.ContainerStateArgsForCall(1)).To(Equal(expectedContainerID))

				close(stopped)
				fakeClock.WaitForWatcherAndIncrement(lifecycle.ContainerStatePollInterval)

				Eventually(fakeRuncClient.ContainerStateCallCount).Should(Equal(3))
				Expect(fakeRuncClient.ContainerStateArgsForCall(2)).To(Equal(expectedContainerID))

				Eventually(errChan).Should(Receive(BeNil()), "stop job did not exit in time")
			})

			Context("and the exit timeout has passed", func() {
				It("sends a SIGQUIT and returns a timeout error", func() {
					errChan := make(chan error)
					go func() {
						defer GinkgoRecover()
						errChan <- runcLifecycle.StopProcess(logger, bpmCfg, exitTimeout)
					}()

					Eventually(fakeRuncClient.SignalContainerCallCount).Should(Equal(1))
					cid, signal := fakeRuncClient.SignalContainerArgsForCall(0)
					Expect(cid).To(Equal(expectedContainerID))
					Expect(signal).To(Equal(client.Term))

					Eventually(fakeRuncClient.ContainerStateCallCount).Should(Equal(1))
					Expect(fakeRuncClient.ContainerStateArgsForCall(0)).To(Equal(expectedContainerID))

					fakeClock.WaitForWatcherAndIncrement(lifecycle.ContainerStatePollInterval)

					Eventually(fakeRuncClient.ContainerStateCallCount).Should(Equal(2))
					Expect(fakeRuncClient.ContainerStateArgsForCall(1)).To(Equal(expectedContainerID))

					fakeClock.WaitForWatcherAndIncrement(exitTimeout)

					Eventually(fakeRuncClient.SignalContainerCallCount).Should(Equal(2))
					cid, signal = fakeRuncClient.SignalContainerArgsForCall(1)
					Expect(cid).To(Equal(expectedContainerID))
					Expect(signal).To(Equal(client.Quit))

					fakeClock.WaitForWatcherAndIncrement(lifecycle.ContainerSigQuitGracePeriod)

					var actualError error
					Eventually(errChan).Should(Receive(&actualError))
					Expect(actualError).To(MatchError("failed to stop job within timeout"))
				})
			})
		})

		Context("when fetching the container state fails", func() {
			BeforeEach(func() {
				fakeRuncClient.ContainerStateReturns(nil, errors.New("fake test error"))
			})

			It("keeps attempting to fetch the state", func() {
				errChan := make(chan error)
				go func() {
					defer GinkgoRecover()
					errChan <- runcLifecycle.StopProcess(logger, bpmCfg, exitTimeout)
				}()

				Eventually(fakeRuncClient.ContainerStateCallCount).Should(Equal(1))
				Expect(fakeRuncClient.ContainerStateArgsForCall(0)).To(Equal(expectedContainerID))

				fakeClock.WaitForWatcherAndIncrement(lifecycle.ContainerStatePollInterval)

				Eventually(fakeRuncClient.ContainerStateCallCount).Should(Equal(2))
				Expect(fakeRuncClient.ContainerStateArgsForCall(1)).To(Equal(expectedContainerID))

				fakeClock.WaitForWatcherAndIncrement(lifecycle.ContainerStatePollInterval)

				Eventually(fakeRuncClient.ContainerStateCallCount).Should(Equal(3))
				Expect(fakeRuncClient.ContainerStateArgsForCall(2)).To(Equal(expectedContainerID))

				fakeClock.WaitForWatcherAndIncrement(exitTimeout)

				Eventually(fakeRuncClient.SignalContainerCallCount).Should(Equal(2))
				cid, signal := fakeRuncClient.SignalContainerArgsForCall(1)
				Expect(cid).To(Equal(expectedContainerID))
				Expect(signal).To(Equal(client.Quit))

				fakeClock.WaitForWatcherAndIncrement(lifecycle.ContainerSigQuitGracePeriod)

				var actualError error
				Eventually(errChan).Should(Receive(&actualError))
				Expect(actualError).To(MatchError("failed to stop job within timeout"))
			})
		})

		Context("when stopping a container fails", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("an error")
				fakeRuncClient.SignalContainerReturns(expectedErr)
			})

			It("returns an error", func() {
				err := runcLifecycle.StopProcess(logger, bpmCfg, exitTimeout)
				Expect(err).To(Equal(expectedErr))
			})
		})
	})

	Describe("RemoveProcess", func() {
		It("deletes the container", func() {
			err := runcLifecycle.RemoveProcess(bpmCfg)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRuncClient.DeleteContainerCallCount()).To(Equal(1))
			containerID := fakeRuncClient.DeleteContainerArgsForCall(0)
			Expect(containerID).To(Equal(expectedContainerID))
		})

		It("destroys the bundle", func() {
			err := runcLifecycle.RemoveProcess(bpmCfg)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRuncClient.DestroyBundleCallCount()).To(Equal(1))
			bundlePath := fakeRuncClient.DestroyBundleArgsForCall(0)
			Expect(bundlePath).To(Equal(filepath.Join(expectedSystemRoot, "data", "bpm", "bundles", expectedJobName, expectedProcName)))
		})

		Context("when the process name is the same as the job name", func() {
			BeforeEach(func() {
				bpmCfg = config.NewBPMConfig(expectedSystemRoot, expectedJobName, expectedJobName)
			})

			It("simplifies the container id", func() {
				err := runcLifecycle.RemoveProcess(bpmCfg)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeRuncClient.DeleteContainerCallCount()).To(Equal(1))
				containerID := fakeRuncClient.DeleteContainerArgsForCall(0)
				Expect(containerID).To(Equal(config.Encode(expectedJobName)))
			})
		})

		Context("when deleting a container fails", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("an error")
				fakeRuncClient.DeleteContainerReturns(expectedErr)
			})

			It("returns an error", func() {
				err := runcLifecycle.RemoveProcess(bpmCfg)
				Expect(err).To(Equal(expectedErr))
			})
		})

		Context("when destroying a bundle fails", func() {
			It("returns an error", func() {
				expectedErr := errors.New("an error2")
				fakeRuncClient.DestroyBundleReturns(expectedErr)
				err := runcLifecycle.RemoveProcess(bpmCfg)
				Expect(err).To(Equal(expectedErr))
			})
		})
	})

	Describe("ListProcesses", func() {
		It("calls the runc client", func() {
			_, err := runcLifecycle.ListProcesses()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRuncClient.ListContainersCallCount()).To(Equal(1))
		})

		It("returns a list of bpm jobs", func() {
			containerStates := []client.ContainerState{
				{
					ID:             "job-process-2",
					InitProcessPid: 23456,
					Status:         "created",
				},
				{
					ID:             "job-process-1",
					InitProcessPid: 34567,
					Status:         "running",
				},
				{
					ID:             "job-process-3",
					InitProcessPid: 0,
					Status:         "stopped",
				},
			}
			fakeRuncClient.ListContainersReturns(containerStates, nil)

			bpmJobs, err := runcLifecycle.ListProcesses()
			Expect(err).NotTo(HaveOccurred())

			Expect(bpmJobs).To(ConsistOf([]*models.Process{
				{Name: "job-process-2", Pid: 23456, Status: "created"},
				{Name: "job-process-1", Pid: 34567, Status: "running"},
				{Name: "job-process-3", Pid: 0, Status: "failed"},
			}))
		})

		Context("when listing jobs fails", func() {
			It("returns an error", func() {
				expectedErr := errors.New("list jobs error")
				fakeRuncClient.ListContainersReturns([]client.ContainerState{}, expectedErr)

				_, err := runcLifecycle.ListProcesses()
				Expect(err).To(Equal(expectedErr))
			})
		})
	})

	Describe("StatProcess", func() {
		BeforeEach(func() {
			fakeRuncClient.ContainerStateReturns(&specs.State{ID: expectedContainerID, Pid: 1234, Status: "running"}, nil)
		})

		It("fetches the container state and translates it into a job", func() {
			process, err := runcLifecycle.StatProcess(bpmCfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeRuncClient.ContainerStateCallCount()).To(Equal(1))
			Expect(fakeRuncClient.ContainerStateArgsForCall(0)).To(Equal(expectedContainerID))
			Expect(process).To(Equal(&models.Process{
				Name:   expectedContainerID,
				Pid:    1234,
				Status: "running",
			}))
		})

		Context("when the container state is stopped", func() {
			BeforeEach(func() {
				fakeRuncClient.ContainerStateReturns(&specs.State{ID: expectedContainerID, Pid: 0, Status: "stopped"}, nil)
			})

			It("fetches the container state and translates it into a job", func() {
				process, err := runcLifecycle.StatProcess(bpmCfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(process).To(Equal(&models.Process{
					Name:   expectedContainerID,
					Pid:    0,
					Status: "failed",
				}))
			})
		})

		Context("when the process name is the same as the job name", func() {
			BeforeEach(func() {
				bpmCfg = config.NewBPMConfig(expectedSystemRoot, expectedJobName, expectedJobName)
			})

			It("simplifies the container id", func() {
				_, err := runcLifecycle.StatProcess(bpmCfg)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeRuncClient.ContainerStateCallCount()).To(Equal(1))
				Expect(fakeRuncClient.ContainerStateArgsForCall(0)).To(Equal(config.Encode(expectedJobName)))
			})
		})

		Context("when fetching the container state returns nil state", func() {
			BeforeEach(func() {
				fakeRuncClient.ContainerStateReturns(nil, nil)
			})

			It("returns nil and an 'IsNotExist' error", func() {
				process, err := runcLifecycle.StatProcess(bpmCfg)
				Expect(err).To(HaveOccurred())
				Expect(process).To(BeNil())

				Expect(lifecycle.IsNotExist(err)).To(BeTrue())
			})
		})

		Context("when fetching the container state fails", func() {
			err := errors.New("fake test error")

			BeforeEach(func() {
				fakeRuncClient.ContainerStateReturns(nil, err)
			})

			It("returns the underlying error", func() {
				_, err := runcLifecycle.StatProcess(bpmCfg)
				Expect(err).To(MatchError(err))
				Expect(lifecycle.IsNotExist(err)).To(BeFalse())
			})
		})
	})

	Describe("OpenShell", func() {
		var expectedStdin *gbytes.Buffer

		BeforeEach(func() {
			fakeRuncClient.ExecReturns(nil)
			expectedStdin = gbytes.BufferWithBytes([]byte("stdin"))
		})

		It("execs /bin/bash inside the container", func() {
			err := runcLifecycle.OpenShell(bpmCfg, expectedStdin, expectedStdout, expectedStderr)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRuncClient.ExecCallCount()).To(Equal(1))
			cid, command, stdin, stdout, stderr := fakeRuncClient.ExecArgsForCall(0)
			Expect(cid).To(Equal(expectedContainerID))
			Expect(command).To(Equal("/bin/bash"))
			Expect(stdin).To(Equal(expectedStdin))
			Expect(stdout).To(Equal(expectedStdout))
			Expect(stderr).To(Equal(expectedStderr))
		})

		Context("when the process name is the same as the job name", func() {
			BeforeEach(func() {
				bpmCfg = config.NewBPMConfig(expectedSystemRoot, expectedJobName, expectedJobName)
			})

			It("simplifies the container id", func() {
				err := runcLifecycle.OpenShell(bpmCfg, expectedStdin, expectedStdout, expectedStderr)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeRuncClient.ExecCallCount()).To(Equal(1))
				cid, _, _, _, _ := fakeRuncClient.ExecArgsForCall(0)
				Expect(cid).To(Equal(config.Encode(expectedJobName)))
			})
		})

		Context("when the exec command fails", func() {
			BeforeEach(func() {
				fakeRuncClient.ExecReturns(errors.New("fake test error"))
			})

			It("returns an error", func() {
				err := runcLifecycle.OpenShell(bpmCfg, expectedStdin, expectedStdout, expectedStderr)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
