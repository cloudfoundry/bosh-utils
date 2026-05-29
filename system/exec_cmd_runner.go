package system

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/hekmon/processpriority"
)

type execCmdRunner struct {
	logger boshlog.Logger
}

func NewExecCmdRunner(logger boshlog.Logger) CmdRunner {
	return execCmdRunner{logger}
}

func (r execCmdRunner) lowerProcessPriority(logTag string, processPid int) error {
	parentPid := os.Getpid()

	parentPrio, rawParentPrio, err := processpriority.Get(parentPid)
	if err != nil {
		r.logger.Error(logTag, "Error getting priority of the current process (pid %d): %s", parentPid, err)
		return err
	}
	r.logger.Debug(logTag, "Current process priority is %s (%d)", parentPrio, rawParentPrio)

	if runtime.GOOS == "windows" {
		r.logger.Debug(logTag, "Setting child process (pid %d) priority to BelowNormal", processPid)
		err = processpriority.Set(processPid, processpriority.BelowNormal)
	} else {
		processPrio := rawParentPrio + 5
		if processPrio > 19 {
			processPrio = 19
		}
		r.logger.Debug(logTag, "Setting child process (pid %d) priority to %d", processPid, processPrio)
		err = processpriority.SetRAW(processPid, processPrio)
	}

	if err != nil {
		r.logger.Error(logTag, "Error setting priority on child process (pid %d): %s", processPid, err)
	}
	return err
}

func (r execCmdRunner) RunComplexCommand(cmd Command) (string, string, int, error) {
	process := NewExecProcess(r.buildComplexCommand(cmd), cmd.KeepAttached, cmd.Quiet, r.logger)

	err := process.Start()
	if err != nil {
		return "", "", -1, err
	}

	if cmd.SpawnWithLowerPriority {
		r.lowerProcessPriority(cmd.Name, process.cmd.Process.Pid) //nolint:errcheck
	}

	result := <-process.Wait()

	return result.Stdout, result.Stderr, result.ExitStatus, result.Error
}

func (r execCmdRunner) RunComplexCommandAsync(cmd Command) (Process, error) {
	process := NewExecProcess(r.buildComplexCommand(cmd), cmd.KeepAttached, cmd.Quiet, r.logger)

	err := process.Start()
	if err != nil {
		return nil, err
	}

	if cmd.SpawnWithLowerPriority {
		r.lowerProcessPriority(cmd.Name, process.cmd.Process.Pid) //nolint:errcheck
	}

	return process, nil
}

func (r execCmdRunner) RunCommand(cmdName string, args ...string) (string, string, int, error) {
	return r.RunComplexCommand(Command{Name: cmdName, Args: args})
}

func (r execCmdRunner) RunCommandQuietly(cmdName string, args ...string) (string, string, int, error) {
	return r.RunComplexCommand(Command{Name: cmdName, Args: args, Quiet: true})
}

func (r execCmdRunner) RunCommandWithInput(input, cmdName string, args ...string) (string, string, int, error) {
	cmd := Command{
		Name:  cmdName,
		Args:  args,
		Stdin: strings.NewReader(input),
	}
	return r.RunComplexCommand(cmd)
}

func (r execCmdRunner) CommandExists(cmdName string) bool {
	_, err := exec.LookPath(cmdName)
	return err == nil
}

func (r execCmdRunner) buildComplexCommand(cmd Command) *exec.Cmd {
	execCmd := newExecCmd(cmd.Name, cmd.Args...)

	if cmd.Stdin != nil {
		execCmd.Stdin = cmd.Stdin
	}

	if cmd.Stdout != nil {
		execCmd.Stdout = cmd.Stdout
	}

	if cmd.Stderr != nil {
		execCmd.Stderr = cmd.Stderr
	}

	execCmd.Dir = cmd.WorkingDir

	execCmd.Env = mergeEnv(os.Environ(), cmd.Env)

	return execCmd
}

func newExecCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
