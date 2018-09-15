package cmd

import (
	"context"
	"errors"
	"flag"
	"path/filepath"

	"github.com/cybozu-go/log"
)

var (
	// empty default values indicates unspecified condition.
	logFilename = flag.String("logfile", "", "Log filename")
	logLevel    = flag.String("loglevel", "", "Log level [critical,error,warning,info,debug]")
	logFormat   = flag.String("logformat", "", "Log format [plain,logfmt,json]")

	ignoreLogFilename bool
)

func init() {
	// This is for child processes of graceful restarting server.
	// See graceful.go
	ignoreLogFilename = !isMaster()
}

// LogConfig configures cybozu-go/log's default logger.
//
// Filename, if not an empty string, specifies the output filename.
//
// Level is the log threshold level name.
// Valid levels are "critical", "error", "warning", "info", and "debug".
// Empty string is treated as "info".
//
// Format specifies log formatter to be used.
// Available formatters are "plain", "logfmt", and "json".
// Empty string is treated as "plain".
//
// For details, see https://godoc.org/github.com/cybozu-go/log .
type LogConfig struct {
	Filename string `toml:"filename" json:"filename"`
	Level    string `toml:"level"    json:"level"`
	Format   string `toml:"format"   json:"format"`
}

// Apply applies configurations to the default logger.
//
// Command-line flags take precedence over the struct member values.
func (c LogConfig) Apply() error {
	logger := log.DefaultLogger()

	filename := c.Filename
	if len(*logFilename) > 0 {
		filename = *logFilename
	}
	if len(filename) > 0 && !ignoreLogFilename {
		abspath, err := filepath.Abs(filename)
		if err != nil {
			return err
		}
		w, err := openLogFile(abspath)
		if err != nil {
			return err
		}
		logger.SetOutput(w)
	}

	level := c.Level
	if len(*logLevel) > 0 {
		level = *logLevel
	}
	if len(level) == 0 {
		level = "info"
	}
	err := logger.SetThresholdByName(level)
	if err != nil {
		return err
	}

	format := c.Format
	if len(*logFormat) > 0 {
		format = *logFormat
	}
	switch format {
	case "", "plain":
		logger.SetFormatter(log.PlainFormat{})
	case "logfmt":
		logger.SetFormatter(log.Logfmt{})
	case "json":
		logger.SetFormatter(log.JSONFormat{})
	default:
		return errors.New("invalid format: " + format)
	}

	return nil
}

// FieldsFromContext returns a map of fields containing
// context information.  Currently, request ID field is
// included, if any.
func FieldsFromContext(ctx context.Context) map[string]interface{} {
	m := make(map[string]interface{})
	v := ctx.Value(RequestIDContextKey)
	if v != nil {
		m[log.FnRequestID] = v.(string)
	}
	return m
}
