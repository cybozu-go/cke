// +build !windows

package cmd

import (
	"io"
	"syscall"

	"github.com/cybozu-go/log"
)

func openLogFile(filename string) (io.Writer, error) {
	return log.NewFileReopener(filename, syscall.SIGUSR1)
}
