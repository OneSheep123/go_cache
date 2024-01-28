// create by chencanhua in 2024/1/28
package service

import (
	"os"
	"syscall"
)

var signalsWindow = []os.Signal{
	os.Interrupt, os.Kill, syscall.SIGKILL,
	syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGILL, syscall.SIGTRAP,
	syscall.SIGABRT, syscall.SIGTERM,
}
