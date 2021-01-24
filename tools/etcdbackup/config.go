package main

import "github.com/cybozu-go/etcdutil"

const (
	defaultBackupDir = "/etcd-backup"
	defaultListen    = "0.0.0.0:8080"
	defaultRotate    = 14
)

// NewConfig returns configuration for etcdbackup
func NewConfig() *Config {
	return &Config{
		BackupDir: defaultBackupDir,
		Listen:    defaultListen,
		Rotate:    defaultRotate,
		Etcd:      etcdutil.NewConfig(""),
	}
}

// Config is configuration parameters
type Config struct {
	BackupDir string           `json:"backup-dir,omitempty"`
	Listen    string           `json:"listen,omitempty"`
	Rotate    int              `json:"rotate,omitempty"`
	Etcd      *etcdutil.Config `json:"etcd"`
}
