package main_test

import (
	"bpm/config"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	yaml "gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	uuid "github.com/satori/go.uuid"
)

var _ = Describe("bpm", func() {
	var (
		command *exec.Cmd

		boshConfigPath,
		jobName,
		procName,
		containerID,
		jobConfigPath,
		stdoutFileLocation,
		stderrFileLocation,
		runcRoot,
		bpmLogFileLocation string

		jobConfig *config.BpmConfig
	)

	var writeConfig = func(cfg *config.BpmConfig) {
		jobConfigDir := filepath.Join(boshConfigPath, "jobs", jobName, "config")
		err := os.MkdirAll(jobConfigDir, 0755)
		Expect(err).NotTo(HaveOccurred())

		jobConfigPath = filepath.Join(jobConfigDir, "bpm.yml")
		Expect(os.RemoveAll(jobConfigPath)).To(Succeed())
		f, err := os.OpenFile(
			jobConfigPath,
			os.O_RDWR|os.O_CREATE,
			0644,
		)
		Expect(err).NotTo(HaveOccurred())

		data, err := yaml.Marshal(cfg)
		Expect(err).NotTo(HaveOccurred())

		n, err := f.Write(data)
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(len(data)))
	}

	var runcCommand = func(args ...string) *exec.Cmd {
		args = append([]string{runcRoot}, args...)
		return exec.Command("runc", args...)
	}

	BeforeEach(func() {
		var err error
		boshConfigPath, err = ioutil.TempDir("", "bpm-main-test")
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(filepath.Join(boshConfigPath, "packages"), 0755)
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(filepath.Join(boshConfigPath, "data", "packages"), 0755)
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(filepath.Join(boshConfigPath, "packages", "runc", "bin"), 0755)
		Expect(err).NotTo(HaveOccurred())

		runcPath, err := exec.LookPath("runc")
		Expect(err).NotTo(HaveOccurred())

		err = os.Link(runcPath, filepath.Join(boshConfigPath, "packages", "runc", "bin", "runc"))
		Expect(err).NotTo(HaveOccurred())

		jobName = fmt.Sprintf("bpm-test-%s", uuid.NewV4().String())
		procName = "sleeper-agent"
		containerID = fmt.Sprintf("%s-%s", jobName, procName)

		jobConfig = &config.BpmConfig{
			Name:       procName,
			Executable: "/bin/bash",
			Args: []string{
				"-c",
				//This script traps the SIGTERM signal and kills the subsequent
				//commands referenced by the PID in the $child variable
				`trap "echo Signalled && kill -9 $child" SIGTERM;
					 echo Foo is $FOO &&
					  (>&2 echo "$FOO is Foo") &&
					  sleep 5 &
					 child=$!;
					 wait $child`,
			},
			Env: []string{"FOO=BAR"},
		}

		stdoutFileLocation = filepath.Join(boshConfigPath, "sys", "log", jobName, fmt.Sprintf("%s.out.log", procName))
		stderrFileLocation = filepath.Join(boshConfigPath, "sys", "log", jobName, fmt.Sprintf("%s.err.log", procName))
		bpmLogFileLocation = filepath.Join(boshConfigPath, "sys", "log", jobName, "bpm.log")

		writeConfig(jobConfig)

		runcRoot = fmt.Sprintf("--root=%s", filepath.Join(boshConfigPath, "data", "bpm", "runc"))
	})

	AfterEach(func() {
		// using force, as we cannot delete a running container.
		runcCommand("delete", "--force", containerID).Run() // TODO: Assert on error when runc is updated to 1.0.0-rc4+

		if CurrentGinkgoTestDescription().Failed {
			fmt.Fprintf(GinkgoWriter, "STDOUT: %s\n", fileContents(stdoutFileLocation)())
			fmt.Fprintf(GinkgoWriter, "STDERR: %s\n", fileContents(stderrFileLocation)())
		}

		err := os.RemoveAll(boshConfigPath)
		Expect(err).NotTo(HaveOccurred())
	})

	runcState := func(cid string) specs.State {
		cmd := runcCommand("state", cid)
		cmd.Stderr = GinkgoWriter

		data, err := cmd.Output()
		Expect(err).NotTo(HaveOccurred())

		stateResponse := specs.State{}
		err = json.Unmarshal(data, &stateResponse)
		Expect(err).NotTo(HaveOccurred())

		return stateResponse
	}

	Context("start", func() {
		JustBeforeEach(func() {
			command = exec.Command(bpmPath, "start", "-j", jobName, "-c", jobConfigPath)
			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))
		})

		It("runs the process in a container with a pidfile", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			state := runcState(containerID)
			Expect(state.Status).To(Equal("running"))
			pidText, err := ioutil.ReadFile(filepath.Join(boshConfigPath, "sys", "run", "bpm", jobName, fmt.Sprintf("%s.pid", procName)))
			Expect(err).NotTo(HaveOccurred())

			pid, err := strconv.Atoi(string(pidText))
			Expect(err).NotTo(HaveOccurred())
			Expect(pid).To(Equal(state.Pid))
		})

		It("redirects the processes stdout and stderr to a standard location", func() {
			Expect(stdoutFileLocation).NotTo(BeAnExistingFile())
			Expect(stderrFileLocation).NotTo(BeAnExistingFile())

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Eventually(fileContents(stdoutFileLocation)).Should(Equal("Foo is BAR\n"))
			Eventually(fileContents(stderrFileLocation)).Should(Equal("BAR is Foo\n"))
		})

		It("logs bpm internal logs to a consistent location", func() {
			Expect(bpmLogFileLocation).NotTo(BeAnExistingFile())

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Eventually(fileContents(bpmLogFileLocation)).Should(ContainSubstring("bpm.start.starting"))
			Eventually(fileContents(bpmLogFileLocation)).Should(ContainSubstring("bpm.start.complete"))
		})

		Context("capabilities", func() {
			BeforeEach(func() {
				jobConfig.Executable = "/bin/bash"
				jobConfig.Args = []string{
					"-c",
					// See https://codegolf.stackexchange.com/questions/24485/create-a-memory-leak-without-any-fork-bombs
					`cat /proc/1/status | grep CapEff`,
				}

				writeConfig(jobConfig)
			})

			It("has no effective capabilities", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Eventually(fileContents(stdoutFileLocation)).Should(MatchRegexp("^\\s?CapEff:\\s?0000000000000000\\s?$"))
			})
		})

		Context("resource limits", func() {
			Context("memory", func() {
				BeforeEach(func() {
					jobConfig.Executable = "/bin/bash"
					jobConfig.Args = []string{
						"-c",
						// See https://codegolf.stackexchange.com/questions/24485/create-a-memory-leak-without-any-fork-bombs
						`start_memory_leak() { :(){ : $@$@;};: : ;};
							trap "kill $child" SIGTERM;
							sleep 100 &
							child=$!;
							wait $child;
							start_memory_leak`,
					}

					limit := "4M"
					jobConfig.Limits = &config.Limits{
						Memory: &limit,
					}

					writeConfig(jobConfig)
				})

				streamOOMEvents := func(stdout io.Reader) chan event {
					oomEvents := make(chan event)

					decoder := json.NewDecoder(stdout)

					go func() {
						defer GinkgoRecover()
						defer close(oomEvents)

						for {
							var actualEvent event
							err := decoder.Decode(&actualEvent)
							if err != nil {
								return
							}

							if actualEvent.Type == "oom" {
								oomEvents <- actualEvent
							}
							time.Sleep(100 * time.Millisecond)
						}
					}()

					return oomEvents
				}

				It("gets OOMed when it exceeds its memory limit", func() {
					session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))

					Eventually(func() string {
						return runcState(containerID).Status
					}).Should(Equal("running"))

					eventsCmd := runcCommand("events", containerID)
					stdout, err := eventsCmd.StdoutPipe()
					Expect(err).NotTo(HaveOccurred())

					oomEventsChan := streamOOMEvents(stdout)
					Expect(eventsCmd.Start()).To(Succeed())

					Expect(runcCommand("kill", containerID).Run()).To(Succeed())

					Eventually(oomEventsChan).Should(Receive())

					Expect(eventsCmd.Process.Kill()).To(Succeed())
					Eventually(oomEventsChan).Should(BeClosed())
				})
			})

			Context("open files", func() {
				BeforeEach(func() {
					jobConfig.Executable = "/bin/bash"
					jobConfig.Args = []string{
						"-c",
						// See https://codegolf.stackexchange.com/questions/24485/create-a-memory-leak-without-any-fork-bombs
						fmt.Sprintf(`file_dir=%s;
						  start_file_leak() { for i in $(seq 1 20); do touch $file_dir/file-$i; done; tail -f $file_dir/* ;};
							trap "kill $child" SIGTERM;
							sleep 100 &
							child=$!;
							wait $child;
							start_file_leak`, filepath.Join(boshConfigPath, "data", jobName, procName)),
					}

					limit := uint64(10)
					jobConfig.Limits = &config.Limits{
						OpenFiles: &limit,
					}

					writeConfig(jobConfig)
				})

				It("cannot open more files than permitted", func() {
					session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))

					Eventually(func() string {
						return runcState(containerID).Status
					}).Should(Equal("running"))

					Expect(runcCommand("kill", containerID).Run()).To(Succeed())

					Eventually(fileContents(stderrFileLocation)).Should(ContainSubstring("Too many open files"))
				})
			})

			Context("processes", func() {
				BeforeEach(func() {
					jobConfig.Executable = "/bin/bash"
					jobConfig.Args = []string{
						"-c",
						// See https://codegolf.stackexchange.com/questions/24485/create-a-memory-leak-without-any-fork-bombs
						` trap "kill $child" SIGTERM;
							sleep 100 &
							child=$!;
							wait $child;
							for i in $(seq 1 999); do sleep 100 & done;
							wait`,
					}

					limit := uint64(1000)
					jobConfig.Limits = &config.Limits{
						Processes: &limit,
					}

					writeConfig(jobConfig)
				})

				It("cannot create more processes than permitted", func() {
					session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))

					Eventually(func() string {
						return runcState(containerID).Status
					}).Should(Equal("running"))

					Expect(runcCommand("kill", containerID).Run()).To(Succeed())

					Eventually(fileContents(stderrFileLocation)).Should(ContainSubstring("fork: retry: Resource temporarily unavailable"))
				})
			})
		})

		Context("when the stdout and stderr files already exist", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Dir(stdoutFileLocation), 0700)).To(Succeed())
				Expect(ioutil.WriteFile(stdoutFileLocation, []byte("STDOUT PREFIX: "), 0700)).To(Succeed())
				Expect(ioutil.WriteFile(stderrFileLocation, []byte("STDERR PREFIX: "), 0700)).To(Succeed())
			})

			It("does not truncate the file", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				Eventually(fileContents(stdoutFileLocation)).Should(Equal("STDOUT PREFIX: Foo is BAR\n"))
				Eventually(fileContents(stderrFileLocation)).Should(Equal("STDERR PREFIX: BAR is Foo\n"))
			})
		})

		Context("when the bpm configuration file does not exist", func() {
			It("exit with a non-zero exit code and prints an error", func() {
				command = exec.Command(bpmPath, "stop", "-j", jobName, "-c", "i am a bogus config path")
				command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).Should(gbytes.Say("i am a bogus config path"))
			})
		})

		Context("when no job name is specified", func() {
			It("exits with a non-zero exit code and prints the usage", func() {
				command = exec.Command(bpmPath, "start")
				command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(session.Err).Should(gbytes.Say("must specify a job"))
			})
		})

		Context("when no config is specified", func() {
			It("exits with a non-zero exit code and prints the usage", func() {
				command = exec.Command(bpmPath, "start", "-j", jobName)
				command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(session.Err).Should(gbytes.Say("must specify a configuration file"))
			})
		})

		Context("when starting the job fails", func() {
			BeforeEach(func() {
				start := exec.Command(bpmPath, "start", "-j", jobName, "-c", jobConfigPath)
				start.Env = append(start.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

				session, err := gexec.Start(start, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
			})

			It("cleans up the associated container and artifacts", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				_, err = os.Open(filepath.Join(boshConfigPath, "data", "bpm", "bundles", jobName, procName))
				Expect(err).To(HaveOccurred())
				Expect(os.IsNotExist(err)).To(BeTrue())

				Expect(runcCommand("state", containerID).Run()).To(HaveOccurred())
			})
		})
	})

	Context("stop", func() {
		BeforeEach(func() {
			startCmd := exec.Command(bpmPath, "start", "-j", jobName, "-c", jobConfigPath)
			startCmd.Env = append(startCmd.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

			session, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})

		JustBeforeEach(func() {
			command = exec.Command(bpmPath, "stop", "-j", jobName, "-c", jobConfigPath)
			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))
		})

		It("signals the container with a SIGTERM", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Eventually(fileContents(stdoutFileLocation)).Should(ContainSubstring("Signalled"))
		})

		It("removes the container and it's corresponding process", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Expect(runcCommand("state", containerID).Run()).To(HaveOccurred())
		})

		It("removes the bundle directory", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			_, err = os.Open(filepath.Join(boshConfigPath, "data", "bpm", "bundles", jobName, procName))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("logs bpm internal logs to a consistent location", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Eventually(fileContents(bpmLogFileLocation)).Should(ContainSubstring("bpm.stop.starting"))
			Eventually(fileContents(bpmLogFileLocation)).Should(ContainSubstring("bpm.stop.complete"))
		})

		Context("when the job name is not specified", func() {
			It("exits with a non-zero exit code and prints the usage", func() {
				command = exec.Command(bpmPath, "stop")
				command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(session.Err).Should(gbytes.Say("must specify a job"))
			})
		})

		Context("when no config is specified", func() {
			It("exits with a non-zero exit code and prints the usage", func() {
				command = exec.Command(bpmPath, "stop", "-j", jobName)
				command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				Expect(session.Err).Should(gbytes.Say("must specify a configuration file"))
			})
		})

		Context("when the bpm configuration file does not exist", func() {
			It("exit with a non-zero exit code and prints an error", func() {
				command = exec.Command(bpmPath, "stop", "-j", jobName, "-c", "i am a bogus config path")
				command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).Should(gbytes.Say("i am a bogus config path"))
			})
		})
	})

	Context("list", func() {
		Context("with running and stopped containers", func() {
			var otherJobName string

			BeforeEach(func() {
				startCmd := exec.Command(bpmPath, "start", "-j", jobName, "-c", jobConfigPath)
				startCmd.Env = append(startCmd.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

				session, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				otherJobName = "example-2"
				Expect(os.MkdirAll(filepath.Join(boshConfigPath, "jobs", otherJobName, "config"), 0755)).NotTo(HaveOccurred())

				startCmd = exec.Command(bpmPath, "start", "-j", otherJobName, "-c", jobConfigPath)
				startCmd.Env = append(startCmd.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

				session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
			})

			It("lists the running jobs and their state", func() {
				listCmd := exec.Command(bpmPath, "list")
				listCmd.Env = append(listCmd.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

				session, err := gexec.Start(listCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())

				state := runcState(containerID)
				otherState := runcState(fmt.Sprintf("%s-%s", otherJobName, procName))

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).Should(gbytes.Say("Name\\s+Pid\\s+Status"))
				Expect(session.Out).Should(gbytes.Say(fmt.Sprintf("%s\\s+%d\\s+%s", state.ID, state.Pid, state.Status)))
				Expect(session.Out).Should(gbytes.Say(fmt.Sprintf("%s\\s+%d\\s+%s", otherState.ID, otherState.Pid, otherState.Status)))
			})
		})

		Context("when no containers are running", func() {
			It("prints no output", func() {
				listCmd := exec.Command(bpmPath, "list")
				listCmd.Env = append(listCmd.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshConfigPath))

				session, err := gexec.Start(listCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).Should(gbytes.Say(""))
			})
		})
	})

	Context("when no flags are provided", func() {
		It("exits with a non-zero exit code and prints the usage", func() {
			command := exec.Command(bpmPath)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).Should(gbytes.Say("Usage:"))
		})
	})
})

func fileContents(path string) func() string {
	return func() string {
		data, err := ioutil.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		return string(data)
	}
}

type event struct {
	Data containerStats `json:"data"`
	Type string         `json:"type"`
	ID   string         `json:"id"`
}

type containerStats struct {
	Memory memory `json:"memory"`
}

type memory struct {
	Cache     uint64            `json:"cache,omitempty"`
	Usage     memoryEntry       `json:"usage,omitempty"`
	Swap      memoryEntry       `json:"swap,omitempty"`
	Kernel    memoryEntry       `json:"kernel,omitempty"`
	KernelTCP memoryEntry       `json:"kernelTCP,omitempty"`
	Raw       map[string]uint64 `json:"raw,omitempty"`
}

type memoryEntry struct {
	Limit   uint64 `json:"limit"`
	Usage   uint64 `json:"usage,omitempty"`
	Max     uint64 `json:"max,omitempty"`
	Failcnt uint64 `json:"failcnt"`
}
