package system_test

import (
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cloudfoundry/bosh-utils/logger/loggerfakes"
	. "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

const ErrExitCode = 14

func windowsCommand(cmdName string) Command {
	return map[string]Command{
		"pwd": {
			Name:       "powershell",
			Args:       []string{"echo $PWD"},
			WorkingDir: `C:\windows\temp`,
		},
		"stderr": {
			Name: "powershell",
			Args: []string{"[Console]::Error.WriteLine('error-output')"},
		},
		"exit": {
			Name: "powershell",
			Args: []string{fmt.Sprintf("exit %d", ErrExitCode)},
		},
		"ls": {
			Name:       "powershell",
			Args:       []string{"dir"},
			WorkingDir: ".",
		},
		"env": {
			Name: "cmd.exe",
			Args: []string{"/C", "SET"},
			Env: map[string]string{
				"FOO": "BAR",
			},
		},
		"echo": {
			Name: "powershell",
			Args: []string{"Write-Host", "Hello World!"},
		},
	}[cmdName]
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
			cmd := windowsCommand("ls")
			stdout, stderr, status, err := runner.RunComplexCommand(cmd)
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(ContainSubstring("exec_cmd_runner_fixtures"))
			Expect(stderr).To(BeEmpty())
			Expect(status).To(Equal(0))
		})

		It("run complex command with env", func() {
			cmd := windowsCommand("env")
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

			cmd := windowsCommand("env")
			cmd.Env = map[string]string{
				"_FOO": "BAZZZ",
			}
			stdout, _, _, err := runner.RunComplexCommand(cmd)
			Expect(err).ToNot(HaveOccurred())

			envVars := parseEnvFields(stdout, false)
			Expect(envVars).To(HaveKeyWithValue("_FOO", "BAZZZ"))
		})

		Context("windows specific behavior", func() {
			setupWindowsEnvTest := func(cmdVars map[string]string) (map[string]string, error) {
				os.Setenv("_FOO", "BAR") //nolint:errcheck
				defer os.Unsetenv("_FOO")

				cmd := windowsCommand("env")
				cmd.Env = cmdVars
				stdout, _, _, err := runner.RunComplexCommand(cmd)
				if err != nil {
					return nil, err
				}

				// don't upper case key names we want to assert that the lower case
				// duplicates provided in Command.Env are used.  also, Windows does
				// not care about key case.
				envVars := parseEnvFields(stdout, false)
				return envVars, nil
			}

			It("uses the env vars specified in the Command", func() {
				envVars, err := setupWindowsEnvTest(map[string]string{
					"_FOO": "BAZZZ",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(envVars).To(HaveKeyWithValue("_FOO", "BAZZZ"))
			})

			It("env var comparison is case-insensitive on Windows", func() {
				envVars, err := setupWindowsEnvTest(map[string]string{
					"_foo": "BAZZZ",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(envVars).ToNot(HaveKey("_FOO"))
				Expect(envVars).To(HaveKeyWithValue("_foo", "BAZZZ"))
			})

			It("deterministically handles duplicate env vars on Windows", func() {
				envVars, err := setupWindowsEnvTest(map[string]string{
					"_foo": "BAZZZ",
					"_bar": "alpha=second",
					"_BAR": "alpha=first",
				})
				Expect(err).ToNot(HaveOccurred())

				// vars in Command.Env replace System vars with the same name,
				// compared case-insensitively. Therefore, the lower case '_foo'
				// replaces the upper case '_FOO'.
				//
				Expect(envVars).ToNot(HaveKey("_FOO"))
				Expect(envVars).To(HaveKeyWithValue("_foo", "BAZZZ"))

				// Duplicate env vars in Command.Env are de-duped before being
				// merged with the System env vars.  Since the Command.Env is
				// a map we sort the keys alphabetically before de-duping so
				// that the result is deterministic.
				//
				Expect(envVars).ToNot(HaveKey("_bar"))
				Expect(envVars).To(HaveKeyWithValue("_BAR", "alpha=first"))
			})

			It("runs a command nicer than itself", func() {
				cmd := Command{Name: priorityPath, SpawnWithLowerPriority: true}
				stdout, _, _, err := runner.RunComplexCommand(cmd)

				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal("BELOW_NORMAL_PRIORITY_CLASS\r\n"))
			})

			It("runs an async command nicer than itself", func() {
				cmd := Command{Name: priorityPath, SpawnWithLowerPriority: true}
				process, err := runner.RunComplexCommandAsync(cmd)
				Expect(err).ToNot(HaveOccurred())

				result := <-process.Wait()
				Expect(result.Error).ToNot(HaveOccurred())
				Expect(result.Stdout).To(Equal("BELOW_NORMAL_PRIORITY_CLASS\r\n"))
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
			cmd := windowsCommand("ls")
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
			cmd := windowsCommand("exit")
			process, err := runner.RunComplexCommandAsync(cmd)
			Expect(err).ToNot(HaveOccurred())

			result := <-process.Wait()
			Expect(result.Error).To(HaveOccurred())
			Expect(result.ExitStatus).To(Equal(ErrExitCode))
		})

		It("allows setting custom env variable in addition to inheriting process env variables", func() {
			cmd := windowsCommand("env")

			process, err := runner.RunComplexCommandAsync(cmd)
			Expect(err).ToNot(HaveOccurred())

			result := <-process.Wait()
			Expect(result.Error).ToNot(HaveOccurred())
			Expect(result.Stdout).To(ContainSubstring("FOO=BAR"))
			Expect(result.Stdout).To(ContainSubstring("PATH="))
		})

		It("changes working dir", func() {
			cmd := windowsCommand("pwd")
			process, err := runner.RunComplexCommandAsync(cmd)
			Expect(err).ToNot(HaveOccurred())

			result := <-process.Wait()
			Expect(result.Error).ToNot(HaveOccurred())
			Expect(result.Stdout).To(ContainSubstring(cmd.WorkingDir))
		})
	})

	Describe("RunCommand", func() {
		It("run command", func() {
			cmd := windowsCommand("echo")
			stdout, stderr, status, err := runner.RunCommand(cmd.Name, cmd.Args...)
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(Equal("Hello World!\n"))
			Expect(stderr).To(BeEmpty())
			Expect(status).To(Equal(0))
		})

		It("run command with error output", func() {
			cmd := windowsCommand("stderr")
			stdout, stderr, status, err := runner.RunCommand(cmd.Name, cmd.Args...)
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(BeEmpty())
			Expect(stderr).To(ContainSubstring("error-output"))
			Expect(status).To(Equal(0))
		})

		It("run command with non-0 exit status", func() {
			cmd := windowsCommand("exit")
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

			cmd := windowsCommand("echo")
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
			Expect(runner.CommandExists("cmd.exe")).To(BeTrue())
			Expect(runner.CommandExists("absolutely-does-not-exist-ever-please-unicorns")).To(BeFalse())
		})
	})
})
