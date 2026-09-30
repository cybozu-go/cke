package op

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/cybozu-go/log"
)

// These errors are returned by runCommand instead of the raw error from cmd.Run().
// The command, its exit status, stdout, and stderr are already recorded by the log call in runCommand.
// So returning that same detail again would just duplicate it wherever the caller logs err.
var (
	errCommandTimedOut = errors.New("command timed out")
	errCommandFailed   = errors.New("command failed")
	errContextDone     = errors.New("context done before command finished")
)

// runCommand runs a command (command[0] is the executable, the rest are its arguments). It blocks until the command exits.
// If timeoutSeconds is nonzero, the command is killed after that many seconds; a zero value means no timeout is applied.
// This function logs the result (elapsed time, stdout, and stderr of the command) and returns the captured stdout.
func runCommand(ctx context.Context, timeoutSeconds int, fnType string, command []string) (stdout string, err error) {
	execCtx := ctx
	if timeoutSeconds != 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(timeoutSeconds))
		defer cancel()
	}

	cmd := exec.CommandContext(execCtx, command[0], command[1:]...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	st := time.Now()
	err = cmd.Run()
	et := time.Now()
	stdout = stdoutBuf.String()
	stderr := stderrBuf.String()

	fields := map[string]any{
		log.FnType:         fnType,
		log.FnResponseTime: et.Sub(st).Seconds(),
		"command":          strings.Join(command, " "),
	}
	if stdout != "" {
		fields["stdout"] = stdout
	}
	if stderr != "" {
		fields["stderr"] = stderr
	}

	if err != nil {
		fields[log.FnError] = err

		// Logs the failure causes in detail here for diagnosability.
		// The raw error is not returned to the caller since that detail is already recorded in the log.
		switch {
		case ctx.Err() != nil:
			log.Error("exec: context done", fields)
			return stdout, errContextDone
		case timeoutSeconds != 0 && execCtx.Err() == context.DeadlineExceeded:
			log.Error("exec: timed out", fields)
			return stdout, errCommandTimedOut
		default:
			log.Error("exec: failed", fields)
			return stdout, errCommandFailed
		}
	}

	log.Info("exec: succeeded", fields)
	return stdout, nil
}
