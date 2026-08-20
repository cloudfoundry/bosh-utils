//go:build !windows

package system_test

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cloudfoundry/bosh-utils/logger/loggerfakes"
	. "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

const ErrExitCode = 14

func unixCommand(cmdName string) Command {
	return map[string]Command{
		"pwd": {
			Name:       "bash",
			Args:       []string{"-c", "echo $PWD"},
			WorkingDir: `/tmp`,
		},
		"stderr": {
			Name: "bash",
			Args: []string{"-c", "echo error-output >&2"},
		},
		"exit": {
			Name: "bash",
			Args: []string{"-c", fmt.Sprintf("exit %d", ErrExitCode)},
		},
		"ls": {
			Name:       "ls",
			Args:       []string{"-l"},
			WorkingDir: ".",
		},
		"env": {
			Name: "env",
			Env: map[string]string{
				"FOO": "BAR",
			},
		},
		"echo": {
			Name: "echo",
			Args: []string{"Hello World!"},
		},
	}[cmdName]
}

func normalizeNiceLevel(kernelNice int) int {
	if runtime.GOOS == "linux" {
		return (kernelNice - 20) * -1
	}
	return kernelNice
}

var _ = Describe("execCmdRunner", func() {
	var (
		runner CmdRunner
	)

	BeforeEach(func() {
		runner = NewExecCmdRunner(boshlog.NewLogger(boshlog.LevelNone))
	})

	Describe("RunComplexCommand", func() {
		It("run complex command with working directory", func() {
			cmd := unixCommand("ls")
			stdout, stderr, status, err := runner.RunComplexCommand(cmd)
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(ContainSubstring("exec_cmd_runner_fixtures"))
			Expect(stderr).To(BeEmpty())
			Expect(status).To(Equal(0))
		})

		It("run complex command with env", func() {
			cmd := unixCommand("env")
			stdout, stderr, status, err := runner.RunComplexCommand(cmd)
			Expect(err).ToNot(HaveOccurred())

			envVars := parseEnvFields(stdout, true)
			Expect(envVars).To(HaveKeyWithValue("FOO", "BAR"))
			Expect(envVars).To(HaveKey("PATH"))
			Expect(stderr).To(BeEmpty())
			Expect(status).To(Equal(0))
		})

		It("uses the env vars specified in the Command", func() {
			GinkgoT().Setenv("_FOO", "BAR")

			cmd := unixCommand("env")
			cmd.Env = map[string]string{
				"_FOO": "BAZZZ",
			}
			stdout, _, _, err := runner.RunComplexCommand(cmd)
			Expect(err).ToNot(HaveOccurred())

			envVars := parseEnvFields(stdout, false)
			Expect(envVars).To(HaveKeyWithValue("_FOO", "BAZZZ"))
		})

		Context("unix specific behavior", func() {
			It("performs a case-sensitive comparison of env vars when on *Nix", func() {
				GinkgoT().Setenv("_FOO", "BAR")

				cmd := unixCommand("env")
				cmd.Env = map[string]string{
					"_foo": "BAZZZ",
					"ABC":  "XYZ",
					"abc":  "xyz",
				}
				stdout, _, _, err := runner.RunComplexCommand(cmd)
				Expect(err).ToNot(HaveOccurred())

				envVars := parseEnvFields(stdout, false)
				Expect(envVars).To(HaveKeyWithValue("_FOO", "BAR"))
				Expect(envVars).To(HaveKeyWithValue("_foo", "BAZZZ"))
				Expect(err).ToNot(HaveOccurred())
				Expect(envVars).To(HaveKeyWithValue("ABC", "XYZ"))
				Expect(envVars).To(HaveKeyWithValue("abc", "xyz"))
			})

			It("runs a command nicer than itself", func() {
				// Calculate what we expect the priority to be
				parentPid := os.Getpid()
				parentKnice, err := syscall.Getpriority(syscall.PRIO_PROCESS, parentPid)
				Expect(err).ToNot(HaveOccurred())

				stdout, _, _, err := runner.RunComplexCommand(Command{Name: priorityPath, SpawnWithLowerPriority: true})
				Expect(err).ToNot(HaveOccurred())

				parentNice := normalizeNiceLevel(parentKnice)
				expectedOutput := fmt.Sprintf("%d\n", min(parentNice+5, 19))
				Expect(stdout).To(Equal(expectedOutput))
			})

			It("runs an async command nicer than itself", func() {
				parentPid := os.Getpid()
				parentKnice, err := syscall.Getpriority(syscall.PRIO_PROCESS, parentPid)
				Expect(err).ToNot(HaveOccurred())

				process, err := runner.RunComplexCommandAsync(Command{Name: priorityPath, SpawnWithLowerPriority: true})
				Expect(err).ToNot(HaveOccurred())
				result := <-process.Wait()
				Expect(result.Error).ToNot(HaveOccurred())

				parentNice := normalizeNiceLevel(parentKnice)
				expectedOutput := fmt.Sprintf("%d\n", min(parentNice+5, 19))
				Expect(result.Stdout).To(Equal(expectedOutput))
			})
		})

		It("run complex command with stdin", func() {
			input := "This is STDIN\nWith another line."
			cmd := Command{
				Name:  catPath,
				Stdin: strings.NewReader(input),
			}
			stdout, stderr, status, err := runner.RunComplexCommand(cmd)
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(Equal(input))
			Expect(stderr).To(BeEmpty())
			Expect(status).To(Equal(0))
		})

		It("prints stdout/stderr to provided I/O object", func() {
			fs := fakesys.NewFakeFileSystem()
			stdoutFile, err := fs.OpenFile("/fake-stdout-path", os.O_RDWR, os.FileMode(0644))
			Expect(err).ToNot(HaveOccurred())

			stderrFile, err := fs.OpenFile("/fake-stderr-path", os.O_RDWR, os.FileMode(0644))
			Expect(err).ToNot(HaveOccurred())

			cmd := Command{
				Name:   catPath,
				Args:   []string{"-stdout", "fake-out", "-stderr", "fake-err"},
				Stdout: stdoutFile,
				Stderr: stderrFile,
			}

			stdout, stderr, status, err := runner.RunComplexCommand(cmd)
			Expect(err).ToNot(HaveOccurred())

			Expect(stdout).To(BeEmpty())
			Expect(stderr).To(BeEmpty())
			Expect(status).To(Equal(0))

			stdoutContents := make([]byte, 1024)
			_, err = stdoutFile.Read(stdoutContents)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(stdoutContents)).To(ContainSubstring("fake-out"))

			stderrContents := make([]byte, 1024)
			_, err = stderrFile.Read(stderrContents)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(stderrContents)).To(ContainSubstring("fake-err"))
		})
	})

	Describe("RunComplexCommandAsync", func() {
		It("populates stdout and stderr", func() {
			cmd := unixCommand("ls")
			process, err := runner.RunComplexCommandAsync(cmd)
			Expect(err).ToNot(HaveOccurred())

			result := <-process.Wait()
			Expect(result.Error).ToNot(HaveOccurred())
			Expect(result.ExitStatus).To(Equal(0))
		})

		It("populates stdout and stderr", func() {
			cmd := Command{
				Name: catPath,
				Args: []string{"-stdout", "STDOUT", "-stderr", "STDERR"},
			}
			process, err := runner.RunComplexCommandAsync(cmd)
			Expect(err).ToNot(HaveOccurred())

			result := <-process.Wait()
			Expect(result.Error).ToNot(HaveOccurred())
			Expect(result.Stdout).To(Equal("STDOUT\n"))
			Expect(result.Stderr).To(Equal("STDERR\n"))
		})

		It("returns error and sets status to exit status of command if it exits with non-0 status", func() {
			cmd := unixCommand("exit")
			process, err := runner.RunComplexCommandAsync(cmd)
			Expect(err).ToNot(HaveOccurred())

			result := <-process.Wait()
			Expect(result.Error).To(HaveOccurred())
			Expect(result.ExitStatus).To(Equal(ErrExitCode))
		})

		It("allows setting custom env variable in addition to inheriting process env variables", func() {
			cmd := unixCommand("env")

			process, err := runner.RunComplexCommandAsync(cmd)
			Expect(err).ToNot(HaveOccurred())

			result := <-process.Wait()
			Expect(result.Error).ToNot(HaveOccurred())
			Expect(result.Stdout).To(ContainSubstring("FOO=BAR"))
			Expect(result.Stdout).To(ContainSubstring("PATH="))
		})

		It("changes working dir", func() {
			cmd := unixCommand("pwd")
			process, err := runner.RunComplexCommandAsync(cmd)
			Expect(err).ToNot(HaveOccurred())

			result := <-process.Wait()
			Expect(result.Error).ToNot(HaveOccurred())
			Expect(result.Stdout).To(ContainSubstring(cmd.WorkingDir))
		})
	})

	Describe("RunCommand", func() {
		It("run command", func() {
			cmd := unixCommand("echo")
			stdout, stderr, status, err := runner.RunCommand(cmd.Name, cmd.Args...)
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(Equal("Hello World!\n"))
			Expect(stderr).To(BeEmpty())
			Expect(status).To(Equal(0))
		})

		It("run command with error output", func() {
			cmd := unixCommand("stderr")
			stdout, stderr, status, err := runner.RunCommand(cmd.Name, cmd.Args...)
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(BeEmpty())
			Expect(stderr).To(ContainSubstring("error-output"))
			Expect(status).To(Equal(0))
		})

		It("run command with non-0 exit status", func() {
			cmd := unixCommand("exit")
			stdout, stderr, status, err := runner.RunCommand(cmd.Name, cmd.Args...)
			Expect(err).To(HaveOccurred())
			Expect(stdout).To(BeEmpty())
			Expect(stderr).To(BeEmpty())
			Expect(status).To(Equal(ErrExitCode))
		})

		It("run command with error", func() {
			stdout, stderr, status, err := runner.RunCommand(falsePath)
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(BeEmpty())
			Expect(status).To(Equal(1))
		})

		It("run command with error with args", func() {
			stdout, stderr, status, err := runner.RunCommand(falsePath, "second arg")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("Running command: '%s second arg', stdout: '', stderr: '': exit status 1", falsePath)))
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(BeEmpty())
			Expect(status).To(Equal(1))
		})

		It("run command with cmd not found", func() {
			stdout, stderr, status, err := runner.RunCommand("something that does not exist")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Or(ContainSubstring("not found"), ContainSubstring("ObjectNotFound")))
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(BeEmpty())
			Expect(status).ToNot(Equal(0))
		})
	})

	Describe("RunCommandWithInput", func() {
		It("run command with input", func() {
			stdout, stderr, status, err := runner.RunCommandWithInput("foo\nbar\nbaz", catPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(Equal("foo\nbar\nbaz"))
			Expect(stderr).To(BeEmpty())
			Expect(status).To(Equal(0))
		})
	})

	Describe("RunCommandQuietly", func() {
		It("run command with input", func() {
			logger := &loggerfakes.FakeLogger{}
			runner = NewExecCmdRunner(logger)

			cmd := unixCommand("echo")
			stdout, stderr, status, err := runner.RunCommandQuietly(cmd.Name, cmd.Args...)
			Expect(err).ToNot(HaveOccurred())
			Expect(logger.DebugCallCount()).To(Equal(2))
			Expect(stdout).To(Equal("Hello World!\n"))
			Expect(stderr).To(BeEmpty())
			Expect(status).To(Equal(0))
		})
	})

	Describe("CommandExists", func() {
		It("command exists", func() {
			Expect(runner.CommandExists("env")).To(BeTrue())
			Expect(runner.CommandExists("absolutely-does-not-exist-ever-please-unicorns")).To(BeFalse())
		})
	})
})
