//go:build linux

package swarm

import (
	"os"
	"os/exec"
	"syscall"
)

func applyLocalSandbox(cmd *exec.Cmd, workspace string) {
	if os.Getenv("QUICKPLAN_DISABLE_LOCAL_SANDBOX") == "1" {
		return
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWUSER | syscall.CLONE_NEWPID,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
		GidMappingsEnableSetgroups: false,
	}
}
