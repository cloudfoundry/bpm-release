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

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"code.cloudfoundry.org/clock/fakeclock"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"bpm/bosh"
	"bpm/config"
	"bpm/jobid"
	"bpm/models"
	"bpm/runc/client"
	"bpm/runc/lifecycle"
	"bpm/runc/lifecycle/mock_lifecycle"
)

var _ = Describe("RuncJobLifecycle", func() {
	var (
		mockCtrl *gomock.Controller

		fakeRuncAdapter   *mock_lifecycle.MockRuncAdapter
		fakeRuncClient    *mock_lifecycle.MockRuncClient
		fakeUserFinder    *mock_lifecycle.MockUserFinder
		fakeCommandRunner *mock_lifecycle.MockCommandRunner
		fakeFileRemover   *fileRemover

		logger *lagertest.TestLogger

		bpmCfg  *config.BPMConfig
		procCfg *config.ProcessConfig
		jobSpec specs.Spec

		expectedJobName,
		expectedProcName,
		expectedContainerID,
		expectedSystemRoot string

		boshEnv *bosh.Env

		expectedStdout, expectedStderr *os.File

		expectedUser specs.User

		fakeClock *fakeclock.FakeClock

		runcLifecycle *lifecycle.RuncLifecycle
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())

		fakeClock = fakeclock.NewFakeClock(time.Now())
		fakeRuncAdapter = mock_lifecycle.NewMockRuncAdapter(mockCtrl)
		fakeRuncClient = mock_lifecycle.NewMockRuncClient(mockCtrl)
		fakeUserFinder = mock_lifecycle.NewMockUserFinder(mockCtrl)
		fakeCommandRunner = mock_lifecycle.NewMockCommandRunner(mockCtrl)
		fakeFileRemover = &fileRemover{}

		logger = lagertest.NewTestLogger("lifecycle")

		expectedUser = specs.User{Username: "vcap", UID: 300, GID: 400}

		var err error
		expectedStdout, err = ioutil.TempFile("", "runc-lifecycle-stdout")
		Expect(err).NotTo(HaveOccurred())
		expectedStderr, err = ioutil.TempFile("", "runc-lifecycle-stderr")
		Expect(err).NotTo(HaveOccurred())

		expectedJobName = "example"
		expectedProcName = "server"
		expectedContainerID = jobid.Encode(fmt.Sprintf("%s.%s", expectedJobName, expectedProcName))

		procCfg = &config.ProcessConfig{
			Executable: "/bin/sleep",
		}
		jobSpec = specs.Spec{
			Process: &specs.Process{
				Env: []string{"foo=bar"},
			},
			Version: "example-version",
		}
		expectedSystemRoot = "system-root"
		boshEnv = bosh.NewEnv(expectedSystemRoot)

		runcLifecycle = lifecycle.NewRuncLifecycle(
			fakeRuncClient,
			fakeRuncAdapter,
			fakeUserFinder,
			fakeCommandRunner,
			fakeClock,
			fakeFileRemover.Remove,
		)
		bpmCfg = config.NewBPMConfig(boshEnv, expectedJobName, expectedProcName)
	})

	AfterEach(func() {
		mockCtrl.Finish()

		Expect(os.RemoveAll(expectedStdout.Name())).To(Succeed())
		Expect(os.RemoveAll(expectedStderr.Name())).To(Succeed())
	})

	// gomock evaluates call matches FIFO which means we can't setup more
	// specific matches as the test goes on. To get around this we setup
	// the test-specific matches first and then call this to setup the
	// defaults.
	//
	// You are not expected to be happy about this.
	setupMockDefaults := func() {
		fakeUserFinder.
			EXPECT().
			Lookup("vcap").
			Return(expectedUser, nil).
			AnyTimes()

		fakeRuncAdapter.
			EXPECT().
			CreateJobPrerequisites(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(expectedStdout, expectedStderr, nil).
			AnyTimes()

		fakeRuncAdapter.
			EXPECT().
			BuildSpec(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(jobSpec, nil).
			AnyTimes()

		fakeRuncClient.
			EXPECT().
			CreateBundle(gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes()

		fakeRuncClient.
			EXPECT().
			RunContainer(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes()

		fakeRuncClient.
			EXPECT().
			DeleteContainer(gomock.Any()).
			AnyTimes()

		fakeRuncClient.
			EXPECT().
			DestroyBundle(gomock.Any()).
			AnyTimes()

		fakeCommandRunner.
			EXPECT().
			Run(gomock.Any()).
			AnyTimes()
	}

	var ItSetsUpAndRunsAProcess = func(run func(logger lager.Logger, bpmCfg *config.BPMConfig, procCfg *config.ProcessConfig) error) {
		Context("when a PreStart Hook is provided", func() {
			BeforeEach(func() {
				procCfg.Hooks = &config.Hooks{
					PreStart: "/please/execute/me",
				}
			})

			It("executes the pre start hook", func() {
				expectedCommand := exec.Command(procCfg.Hooks.PreStart)
				expectedCommand.Stdout = expectedStdout
				expectedCommand.Stderr = expectedStderr
				expectedCommand.Env = []string{"foo=bar"}

				fakeCommandRunner.
					EXPECT().
					Run(expectedCommand).
					Times(1)

				err := run(logger, bpmCfg, procCfg)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when the PreStart Hook fails", func() {
				BeforeEach(func() {
					fakeCommandRunner.
						EXPECT().
						Run(gomock.Any()).
						Return(errors.New("fake test error")).
						Times(1)
				})

				It("returns an error", func() {
					err := run(logger, bpmCfg, procCfg)
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
				err := run(logger, bpmCfg, procCfg)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the process name is the same as the job name", func() {
			BeforeEach(func() {
				bpmCfg = config.NewBPMConfig(boshEnv, expectedJobName, expectedJobName)

				fakeRuncClient.
					EXPECT().
					RunContainer(gomock.Any(), gomock.Any(), jobid.Encode(expectedJobName), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)
			})

			It("simplifies the container ID", func() {
				err := run(logger, bpmCfg, procCfg)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when looking up the vcap user fails", func() {
			BeforeEach(func() {
				fakeUserFinder.
					EXPECT().
					Lookup(gomock.Any()).
					Return(specs.User{}, errors.New("fake test error")).
					AnyTimes()
			})

			It("returns an error", func() {
				err := run(logger, bpmCfg, procCfg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when creating the system files fails", func() {
			BeforeEach(func() {
				fakeRuncAdapter.
					EXPECT().
					CreateJobPrerequisites(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil, errors.New("fake test error"))
			})

			It("returns an error", func() {
				err := run(logger, bpmCfg, procCfg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when building the runc spec fails", func() {
			BeforeEach(func() {
				fakeRuncAdapter.
					EXPECT().
					BuildSpec(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(specs.Spec{}, errors.New("fake test error"))
			})

			It("returns an error", func() {
				err := run(logger, bpmCfg, procCfg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when building the bundle fails", func() {
			BeforeEach(func() {
				fakeRuncClient.
					EXPECT().
					CreateBundle(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("fake test error"))
			})

			It("returns an error", func() {
				err := run(logger, bpmCfg, procCfg)
				Expect(err).To(HaveOccurred())
			})
		})

	}

	Describe("StartProcess", func() {
		It("builds the runc spec, bundle, and runs the container", func() {
			fakeRuncAdapter.
				EXPECT().
				CreateJobPrerequisites(bpmCfg, procCfg, expectedUser).
				Return(expectedStdout, expectedStderr, nil).
				Times(1)

			fakeRuncAdapter.
				EXPECT().
				BuildSpec(gomock.Any(), bpmCfg, procCfg, expectedUser).
				Return(jobSpec, nil).
				Times(1)

			rootPath := filepath.Join(expectedSystemRoot, "data", "bpm", "bundles", expectedJobName, expectedProcName)
			fakeRuncClient.
				EXPECT().
				CreateBundle(rootPath, jobSpec, expectedUser).
				Times(1)

			fakeRuncClient.
				EXPECT().
				RunContainer(
					bpmCfg.PidFile().External(),
					rootPath,
					expectedContainerID,
					true,
					expectedStdout,
					expectedStderr,
				).
				Times(1)

			setupMockDefaults()

			err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when running the container fails", func() {
			BeforeEach(func() {
				fakeRuncClient.
					EXPECT().
					RunContainer(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(1, errors.New("fake test error"))
			})

			It("returns an error", func() {
				setupMockDefaults()

				err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
				Expect(err).To(HaveOccurred())
			})
		})

		ItSetsUpAndRunsAProcess(func(logger lager.Logger, bpmCfg *config.BPMConfig, procCfg *config.ProcessConfig) error {
			setupMockDefaults()

			return runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
		})
	})

	Describe("RunProcess", func() {
		It("builds the runc spec, bundle, and runs the container", func() {
			fakeRuncAdapter.
				EXPECT().
				CreateJobPrerequisites(bpmCfg, procCfg, expectedUser).
				Return(expectedStdout, expectedStderr, nil).
				Times(1)

			fakeRuncAdapter.
				EXPECT().
				BuildSpec(gomock.Any(), bpmCfg, procCfg, expectedUser).
				Return(jobSpec, nil).
				Times(1)

			rootPath := filepath.Join(expectedSystemRoot, "data", "bpm", "bundles", expectedJobName, expectedProcName)
			fakeRuncClient.
				EXPECT().
				CreateBundle(rootPath, jobSpec, expectedUser).
				Times(1)

			fakeRuncClient.
				EXPECT().
				RunContainer(
					bpmCfg.PidFile().External(),
					rootPath,
					expectedContainerID,
					false,
					gomock.Any(), // We can't assert on these because the function wraps them in io.MultiWriters.
					gomock.Any(),
				).
				Return(0, nil).
				Times(1)

			setupMockDefaults()

			status, err := runcLifecycle.RunProcess(logger, bpmCfg, procCfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(0))
		})

		Context("when running the container fails", func() {
			BeforeEach(func() {
				fakeRuncClient.
					EXPECT().
					RunContainer(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					Return(1, errors.New("fake test error"))
			})

			It("returns an error", func() {
				setupMockDefaults()

				status, err := runcLifecycle.RunProcess(logger, bpmCfg, procCfg)
				Expect(err).To(HaveOccurred())
				Expect(status).To(Equal(1))
			})
		})

		ItSetsUpAndRunsAProcess(func(logger lager.Logger, bpmCfg *config.BPMConfig, procCfg *config.ProcessConfig) error {
			setupMockDefaults()
			// status is tested separately
			_, err := runcLifecycle.RunProcess(logger, bpmCfg, procCfg)
			return err
		})
	})

	Describe("StopProcess", func() {
		var exitTimeout time.Duration

		BeforeEach(func() {
			exitTimeout = 5 * time.Second
		})

		It("stops the container", func() {
			fakeRuncClient.
				EXPECT().
				ContainerState(expectedContainerID).
				Return(&specs.State{
					Status: "stopped",
				}, nil)

			fakeRuncClient.
				EXPECT().
				SignalContainer(expectedContainerID, client.Term).
				Times(1)

			setupMockDefaults()
			err := runcLifecycle.StopProcess(logger, bpmCfg, procCfg, exitTimeout)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when the shutdown signal is SIGINT", func() {
			BeforeEach(func() {
				procCfg.ShutdownSignal = "INT"
			})

			It("stops the container with an interrupt signal", func() {
				fakeRuncClient.
					EXPECT().
					ContainerState(expectedContainerID).
					Return(&specs.State{
						Status: "stopped",
					}, nil)

				fakeRuncClient.
					EXPECT().
					SignalContainer(expectedContainerID, client.Int).
					Times(1)

				setupMockDefaults()
				err := runcLifecycle.StopProcess(logger, bpmCfg, procCfg, exitTimeout)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when the container does not stop immediately", func() {
			It("polls the container state every second until it stops", func() {
				gomock.InOrder(
					fakeRuncClient.
						EXPECT().
						SignalContainer(expectedContainerID, client.Term).
						Times(1),
					fakeRuncClient.
						EXPECT().
						ContainerState(expectedContainerID).
						DoAndReturn(func(id string) (*specs.State, error) {
							go fakeClock.WaitForNWatchersAndIncrement(lifecycle.ContainerStatePollInterval, 2)
							return &specs.State{Status: "running"}, nil
						}).
						Times(1),
					fakeRuncClient.
						EXPECT().
						ContainerState(expectedContainerID).
						DoAndReturn(func(id string) (*specs.State, error) {
							go fakeClock.WaitForNWatchersAndIncrement(lifecycle.ContainerStatePollInterval, 2)
							return &specs.State{Status: "running"}, nil
						}).
						Times(1),
					fakeRuncClient.
						EXPECT().
						ContainerState(expectedContainerID).
						DoAndReturn(func(id string) (*specs.State, error) {
							return &specs.State{Status: "stopped"}, nil
						}).
						Times(1),
				)

				setupMockDefaults()
				err := runcLifecycle.StopProcess(logger, bpmCfg, procCfg, exitTimeout)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("and the exit timeout has passed", func() {
				It("sends a SIGQUIT and returns a timeout error", func() {
					gomock.InOrder(
						fakeRuncClient.
							EXPECT().
							SignalContainer(expectedContainerID, client.Term).
							Times(1),
						fakeRuncClient.
							EXPECT().
							ContainerState(expectedContainerID).
							DoAndReturn(func(id string) (*specs.State, error) {
								go fakeClock.WaitForNWatchersAndIncrement(lifecycle.ContainerStatePollInterval, 2)
								return &specs.State{Status: "running"}, nil
							}).
							Times(1),
						fakeRuncClient.
							EXPECT().
							ContainerState(expectedContainerID).
							DoAndReturn(func(id string) (*specs.State, error) {
								go fakeClock.WaitForNWatchersAndIncrement(exitTimeout, 2)
								return &specs.State{Status: "running"}, nil
							}).
							AnyTimes(),
						fakeRuncClient.
							EXPECT().
							SignalContainer(expectedContainerID, client.Quit).
							Do(func(id string, signal client.Signal) {
								go fakeClock.WaitForNWatchersAndIncrement(lifecycle.ContainerSigQuitGracePeriod, 2)
							}).
							Times(1),
					)

					setupMockDefaults()
					err := runcLifecycle.StopProcess(logger, bpmCfg, procCfg, exitTimeout)
					Expect(err).To(MatchError("failed to stop job within timeout"))
				})
			})
		})

		Context("when fetching the container state fails", func() {
			It("keeps attempting to fetch the state", func() {
				gomock.InOrder(
					fakeRuncClient.
						EXPECT().
						SignalContainer(expectedContainerID, client.Term).
						Times(1),
					fakeRuncClient.
						EXPECT().
						ContainerState(expectedContainerID).
						DoAndReturn(func(id string) (*specs.State, error) {
							go fakeClock.WaitForNWatchersAndIncrement(lifecycle.ContainerStatePollInterval, 2)
							return nil, errors.New("fake test error")
						}).
						Times(1),
					fakeRuncClient.
						EXPECT().
						ContainerState(expectedContainerID).
						DoAndReturn(func(id string) (*specs.State, error) {
							go fakeClock.WaitForNWatchersAndIncrement(exitTimeout, 2)
							return nil, errors.New("fake test error")
						}).
						AnyTimes(),
					fakeRuncClient.
						EXPECT().
						SignalContainer(expectedContainerID, client.Quit).
						Do(func(id string, signal client.Signal) {
							go fakeClock.WaitForNWatchersAndIncrement(lifecycle.ContainerSigQuitGracePeriod, 2)
						}).
						Times(1),
				)

				setupMockDefaults()
				err := runcLifecycle.StopProcess(logger, bpmCfg, procCfg, exitTimeout)
				Expect(err).To(MatchError("failed to stop job within timeout"))
			})
		})

		Context("when stopping a container fails", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("an error")
				fakeRuncClient.
					EXPECT().
					SignalContainer(gomock.Any(), gomock.Any()).
					Return(expectedErr)
			})

			It("returns an error", func() {
				setupMockDefaults()
				err := runcLifecycle.StopProcess(logger, bpmCfg, procCfg, exitTimeout)
				Expect(err).To(Equal(expectedErr))
			})
		})
	})

	Describe("RemoveProcess", func() {
		It("deletes the container", func() {
			fakeRuncClient.
				EXPECT().
				DeleteContainer(expectedContainerID).
				Times(1)

			setupMockDefaults()
			err := runcLifecycle.RemoveProcess(logger, bpmCfg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("destroys the bundle", func() {
			bundlePath := filepath.Join(expectedSystemRoot, "data", "bpm", "bundles", expectedJobName, expectedProcName)
			fakeRuncClient.
				EXPECT().
				DestroyBundle(bundlePath).
				Times(1)

			setupMockDefaults()
			err := runcLifecycle.RemoveProcess(logger, bpmCfg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes the pidfile", func() {
			setupMockDefaults()
			err := runcLifecycle.RemoveProcess(logger, bpmCfg)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeFileRemover.deletedFiles).To(ConsistOf(bpmCfg.PidFile().External()))
		})

		Context("when the process name is the same as the job name", func() {
			BeforeEach(func() {
				bpmCfg = config.NewBPMConfig(boshEnv, expectedJobName, expectedJobName)
			})

			It("simplifies the container id", func() {
				fakeRuncClient.
					EXPECT().
					DeleteContainer(jobid.Encode(expectedJobName)).
					Times(1)

				setupMockDefaults()
				err := runcLifecycle.RemoveProcess(logger, bpmCfg)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when deleting a container fails", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("an error")
				fakeRuncClient.
					EXPECT().
					DeleteContainer(gomock.Any()).
					Return(expectedErr)
			})

			It("returns an error", func() {
				setupMockDefaults()
				err := runcLifecycle.RemoveProcess(logger, bpmCfg)
				Expect(err).To(Equal(expectedErr))
			})
		})

		Context("when destroying a bundle fails", func() {
			It("returns an error", func() {
				expectedErr := errors.New("an error2")
				fakeRuncClient.
					EXPECT().
					DestroyBundle(gomock.Any()).
					Return(expectedErr)

				setupMockDefaults()
				err := runcLifecycle.RemoveProcess(logger, bpmCfg)
				Expect(err).To(Equal(expectedErr))
			})
		})
	})

	Describe("ListProcesses", func() {
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
			fakeRuncClient.
				EXPECT().
				ListContainers().
				Return(containerStates, nil)

			setupMockDefaults()
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
				fakeRuncClient.
					EXPECT().
					ListContainers().
					Return([]client.ContainerState{}, expectedErr)

				setupMockDefaults()
				_, err := runcLifecycle.ListProcesses()
				Expect(err).To(Equal(expectedErr))
			})
		})
	})

	Describe("StatProcess", func() {
		It("fetches the container state and translates it into a job", func() {
			fakeRuncClient.
				EXPECT().
				ContainerState(expectedContainerID).
				Return(&specs.State{ID: expectedContainerID, Pid: 1234, Status: "running"}, nil).
				Times(1)

			setupMockDefaults()
			process, err := runcLifecycle.StatProcess(bpmCfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(process).To(Equal(&models.Process{
				Name:   expectedContainerID,
				Pid:    1234,
				Status: "running",
			}))
		})

		Context("when the container state is stopped", func() {
			BeforeEach(func() {
				fakeRuncClient.
					EXPECT().
					ContainerState(expectedContainerID).
					Return(&specs.State{ID: expectedContainerID, Pid: 0, Status: "stopped"}, nil).
					Times(1)
			})

			It("fetches the container state and translates it into a job", func() {
				setupMockDefaults()
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
				bpmCfg = config.NewBPMConfig(boshEnv, expectedJobName, expectedJobName)
			})

			It("simplifies the container id", func() {
				containerID := jobid.Encode(expectedJobName)
				fakeRuncClient.
					EXPECT().
					ContainerState(containerID).
					Return(&specs.State{ID: containerID, Pid: 1234, Status: "running"}, nil).
					Times(1)

				setupMockDefaults()
				_, err := runcLifecycle.StatProcess(bpmCfg)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when fetching the container state returns nil state", func() {
			BeforeEach(func() {
				fakeRuncClient.
					EXPECT().
					ContainerState(gomock.Any()).
					Return(nil, nil)
			})

			It("returns nil and an 'IsNotExist' error", func() {
				setupMockDefaults()
				process, err := runcLifecycle.StatProcess(bpmCfg)
				Expect(err).To(HaveOccurred())
				Expect(process).To(BeNil())

				Expect(lifecycle.IsNotExist(err)).To(BeTrue())
			})
		})

		Context("when fetching the container state fails", func() {
			err := errors.New("fake test error")

			BeforeEach(func() {
				fakeRuncClient.
					EXPECT().
					ContainerState(gomock.Any()).
					Return(nil, err)
			})

			It("returns the underlying error", func() {
				setupMockDefaults()
				_, err := runcLifecycle.StatProcess(bpmCfg)
				Expect(err).To(MatchError(err))
				Expect(lifecycle.IsNotExist(err)).To(BeFalse())
			})
		})
	})

	Describe("OpenShell", func() {
		var expectedStdin *gbytes.Buffer

		BeforeEach(func() {
			expectedStdin = gbytes.BufferWithBytes([]byte("stdin"))
		})

		It("execs /bin/bash inside the container", func() {
			fakeRuncClient.
				EXPECT().
				Exec(expectedContainerID, "/bin/bash", expectedStdin, expectedStdout, expectedStderr).
				Times(1)

			setupMockDefaults()
			err := runcLifecycle.OpenShell(bpmCfg, expectedStdin, expectedStdout, expectedStderr)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the process name is the same as the job name", func() {
			BeforeEach(func() {
				bpmCfg = config.NewBPMConfig(boshEnv, expectedJobName, expectedJobName)
			})

			It("simplifies the container id", func() {
				fakeRuncClient.
					EXPECT().
					Exec(jobid.Encode(expectedJobName), "/bin/bash", expectedStdin, expectedStdout, expectedStderr).
					Times(1)
				setupMockDefaults()
				err := runcLifecycle.OpenShell(bpmCfg, expectedStdin, expectedStdout, expectedStderr)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the exec command fails", func() {
			BeforeEach(func() {
				fakeRuncClient.
					EXPECT().
					Exec(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("fake test error"))
			})

			It("returns an error", func() {
				setupMockDefaults()
				err := runcLifecycle.OpenShell(bpmCfg, expectedStdin, expectedStdout, expectedStderr)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

type fileRemover struct {
	deletedFiles []string
}

func (f *fileRemover) Remove(path string) error {
	f.deletedFiles = append(f.deletedFiles, path)
	return nil
}
