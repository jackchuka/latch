//go:build unix

package queuecmd

import "syscall"

func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
