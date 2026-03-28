# Process Priority

[![Go Reference](https://pkg.go.dev/badge/github.com/hekmon/processpriority.svg)](https://pkg.go.dev/github.com/hekmon/processpriority)

Easily get or set a process priority from a Golang program with transparent Linux/Windows support.

Under the hood, it will uses pre-defined nice values for Linux and the priority classes for Windows.

## Notes

### Linux

On Linux, a user can only decrease its priority (increase its nice value), not increase it. This means that even after lowering the priority, you won't be able to raise it to its original value.

To be able to increase it, the user must be root or have the `CAP_SYS_NICE` capability.

### Windows

On Windows, the real time priority can only be set as an administrator. If you try to set it as a normal user, you will won't get an error but your process will be set at `High` instead.

## Example

```golang
package main

import (
	"fmt"
	"os/exec"

	"github.com/hekmon/processpriority"
)

func main() {
	// Run command
	// cmd := exec.Command("sleep", "1") // linux binary
	cmd := exec.Command("./sleep.exe") // windows binary
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	// Get its current priority
	prio, rawPrio, err := processpriority.Get(cmd.Process.Pid)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Current process priority is %s (%d)\n", prio, rawPrio)
	// Change its priority
	newPriority := processpriority.BelowNormal
	fmt.Printf("Changing process priority to %s\n", newPriority)
	if err = processpriority.Set(cmd.Process.Pid, newPriority); err != nil {
		panic(err)
	}
	// Verifying
	if prio, rawPrio, err = processpriority.Get(cmd.Process.Pid); err != nil {
		panic(err)
	}
	fmt.Printf("Current process priority is %s (%d)\n", prio, rawPrio)
	// Wait for the cmd to end
	if err = cmd.Wait(); err != nil {
		panic(err)
	}
}
```

### Linux output

```
Current process priority is Normal (0)
Changing process priority to Below Normal
Current process priority is Below Normal (5)
```

### Windows output

```
Current process priority is Normal (32)
Changing process priority to Below Normal
Current process priority is Below Normal (16384)
```

## Install

```bash
go get -u github.com/hekmon/processpriority
```
