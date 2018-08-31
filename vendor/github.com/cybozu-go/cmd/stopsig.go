// +build !windows

package cmd

import (
	"os"
	"syscall"
)

var stopSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
