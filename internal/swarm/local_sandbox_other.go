//go:build !linux

package swarm

import "os/exec"

func applyLocalSandbox(cmd *exec.Cmd, workspace string) {
}
