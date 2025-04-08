package system

import (
	"os"
	"os/exec"
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

func (r execCmdRunner) RunComplexCommand(cmd Command) (string, string, int, error) {
	process := NewExecProcess(r.buildComplexCommand(cmd), cmd.KeepAttached, cmd.Quiet, r.logger)

	err := process.Start()
	if err != nil {
		return "", "", -1, err
	}

	if cmd.RunNicer == true {
		parentName := os.Args[0]
		parentPid := os.Getpid()
		parentPrio, rawParentPrio, err := processpriority.Get(parentPid)

		if err == nil {
			r.logger.Debug(parentName, "Current process priority is %s (%d)", parentPrio, rawParentPrio)

			processPrio := rawParentPrio - 5
			// Clamp priority to be max 19 (idle)
			if processPrio < -19 {
				processPrio = -19
			}

			processPid := process.cmd.Process.Pid

			r.logger.Debug(parentName, "Setting new process priority to %d", processPrio)
			err := processpriority.SetRAW(processPid, processPrio)
			if err != nil {
				r.logger.Debug(cmd.Name, "Error setting priority %d on the command %s (%d): %s", processPrio, cmd.Name, processPid, err)
			}
		} else {
			r.logger.Debug(parentName, "Error getting priority of the current process (%d): %s", parentPid, err)
		}
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
