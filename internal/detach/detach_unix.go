//go:build unix

package detach

import "syscall"

func DetachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
