package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/snapshot"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var config struct {
	maxBackups int
	dir        string
}

const backupTimeFormat = "20060102-150405"

func backupFilename(t time.Time) string {
	return fmt.Sprintf("etcd-%s.backup", t.UTC().Format(backupTimeFormat))
}

var etcdLocalBackupCmd = &cobra.Command{
	Use:   "local-backup",
	Short: "take a snapshot of CKE-managed etcd data and save it",
	Long: `This command takes a snapshot of CKE-managed etcd that stores Kubernetes data.

The snapshots are saved in a directory specified with --dir flag
with this format: etcd-YYYYMMDD-hhmmss.backup
The date and time is UTC.

Old backups are automatically removed when the number of backup files
exceed the maximum specified with --max-backups flag.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		well.Go(func(ctx context.Context) error {
			etcd, err := inf.NewEtcdClient(ctx, nil)
			if err != nil {
				return err
			}
			err = backup(ctx, etcd)
			if err != nil {
				return fmt.Errorf("failed to take a backup: %w", err)
			}
			return removeOldBackups()
		})
		well.Stop()
		return well.Wait()
	},
}

func backup(ctx context.Context, etcd *clientv3.Client) error {
	r, err := etcd.Snapshot(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	switch fi, err := os.Stat(config.dir); {
	case err == nil:
		if !fi.IsDir() {
			return fmt.Errorf("%s is not a directory", config.dir)
		}
	case os.IsNotExist(err):
		if err := os.MkdirAll(config.dir, 0755); err != nil {
			return err
		}
	default:
		return err
	}

	fname := backupFilename(time.Now())
	fullName := filepath.Join(config.dir, fname)
	w, err := os.Create(fullName)
	if err != nil {
		return err
	}
	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		os.Remove(fullName)
		return err
	}

	if err := w.Sync(); err != nil {
		os.Remove(fullName)
		return err
	}

	ss := snapshot.NewV3(nil)
	if _, err := ss.Status(fullName); err != nil {
		os.Remove(fullName)
		return fmt.Errorf("failed to check status of the new backup: %w", err)
	}

	fmt.Printf("created backup %s\n", fname)
	return nil
}

func removeOldBackups() error {
	fis, err := os.ReadDir(config.dir)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(fis))
	for _, fi := range fis {
		name := fi.Name()
		if strings.HasPrefix(name, "etcd-") && strings.HasSuffix(name, ".backup") {
			names = append(names, name)
		}
	}

	sort.Strings(names)
	toRemove := len(names) - config.maxBackups
	if toRemove > 0 {
		for _, name := range names[0:toRemove] {
			err := os.Remove(filepath.Join(config.dir, name))
			if err != nil {
				return fmt.Errorf("failed to remove %s: %w", name, err)
			}
			fmt.Printf("removed %s\n", name)
		}
	}

	d, err := os.Open(config.dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func init() {
	etcdCmd.AddCommand(etcdLocalBackupCmd)

	f := etcdLocalBackupCmd.Flags()
	f.IntVar(&config.maxBackups, "max-backups", 10, "the maximum number of backups to keep")
	f.StringVar(&config.dir, "dir", "/var/cke/etcd-backups", "the directory to keep the backup files")
}
