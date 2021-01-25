package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/fileutil"
	"github.com/cybozu-go/etcdutil"
)

const (
	backupSucceed       = "backup successfully"
	snapshotFilePattern = "snapshot-*.db.gz"
)

// Server is etcdbackup server
type Server struct {
	cfg *Config
}

// NewServer returns etcd backup server
func NewServer(cfg *Config) *Server {
	return &Server{cfg}
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !strings.HasPrefix(r.URL.Path, "/api/v1/backup") {
		renderError(ctx, w, APIErrNotFound)
		return
	}
	p := r.URL.Path[len("/api/v1/backup"):]
	switch r.Method {
	case http.MethodGet:
		if len(p) == 0 {
			s.handleBackupList(w, r)
			return
		} else if strings.HasPrefix(p, "/") && len(p) > 1 {
			s.handleBackupDownload(w, r, p[1:])
			return
		}
	case http.MethodPost:
		if len(p) == 0 {
			s.handleBackupSave(w, r)
			return
		}
	}
	renderError(ctx, w, APIErrNotFound)
}

func (s Server) handleBackupList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	matches, err := filepath.Glob(filepath.Join(s.cfg.BackupDir, snapshotFilePattern))
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}

	var files []string
	for _, f := range matches {
		fi, err := os.Stat(f)
		if err != nil {
			renderError(ctx, w, InternalServerError(err))
			return
		}
		if fi.IsDir() {
			continue
		}
		files = append(files, path.Base(f))
	}

	renderJSON(w, files, http.StatusOK)
}

func (s Server) handleBackupDownload(w http.ResponseWriter, r *http.Request, filename string) {
	ctx := r.Context()
	matched, err := filepath.Match(snapshotFilePattern, filename)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	if !matched {
		renderError(ctx, w, APIErrBadRequest)
		return
	}

	target := filepath.Join(s.cfg.BackupDir, filename)
	fi, err := os.Stat(target)
	if os.IsNotExist(err) || fi.IsDir() {
		renderError(ctx, w, APIErrNotFound)
		return
	}

	f, err := os.Open(target)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	defer f.Close()
	header := w.Header()
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	header.Set("content-type", contentType)
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}

func (s Server) handleBackupSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()
	filename := filepath.Join(s.cfg.BackupDir, snapshotName(now))
	cli, err := etcdutil.NewClient(s.cfg.Etcd)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	defer cli.Close()

	bkName, bkSize, err := saveBackup(ctx, filename, cli)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}

	err = removeOldBackups(s.cfg.BackupDir, s.cfg.Rotate)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}

	renderJSON(w, map[string]interface{}{
		"message":  backupSucceed,
		"filename": bkName,
		"filesize": bkSize,
	}, http.StatusOK)
}

func snapshotName(date time.Time) string {
	return fmt.Sprintf("snapshot-%s.db", date.Format("20060102_150405"))
}

func saveBackup(ctx context.Context, filename string, cli *clientv3.Client) (string, int64, error) {
	// Take snapshot to temp file
	partpath := filename + ".part"
	defer os.RemoveAll(partpath)

	fp, err := os.OpenFile(partpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileutil.PrivateFileMode)
	if err != nil {
		return "", 0, err
	}
	var rd io.ReadCloser
	rd, err = cli.Snapshot(ctx)
	if err != nil {
		return "", 0, err
	}
	_, err = io.Copy(fp, rd)
	if err != nil {
		return "", 0, err
	}
	err = fileutil.Fsync(fp)
	if err != nil {
		return "", 0, err
	}
	err = fp.Close()
	if err != nil {
		return "", 0, err
	}

	// Rename temp file to expected file name
	err = os.Rename(partpath, filename)
	if err != nil {
		return "", 0, err
	}

	// Compress snapshot file
	f, err := os.Open(filename)
	if err != nil {
		return "", 0, err
	}

	gzipName := filename + ".gz"
	zf, err := os.Create(gzipName)
	if err != nil {
		return "", 0, err
	}
	zw := gzip.NewWriter(zf)

	_, err = io.Copy(zw, f)
	if err != nil {
		return "", 0, err
	}
	defer os.Remove(filename)

	err = zw.Close()
	if err != nil {
		return "", 0, err
	}

	fi, err := os.Stat(gzipName)
	if err != nil {
		return "", 0, err
	}
	return fi.Name(), fi.Size(), nil
}

func removeOldBackups(dir string, rotate int) error {
	matches, err := filepath.Glob(filepath.Join(dir, snapshotFilePattern))
	if err != nil {
		return err
	}

	var snapshotFiles []os.FileInfo
	for _, f := range matches {
		fi, err := os.Stat(f)
		if err != nil {
			return err
		}
		snapshotFiles = append(snapshotFiles, fi)
	}

	if len(snapshotFiles) < rotate {
		return nil
	}

	sort.Slice(snapshotFiles, func(i, j int) bool {
		return snapshotFiles[i].ModTime().Unix() > snapshotFiles[j].ModTime().Unix()
	})

	removeFiles := snapshotFiles[rotate:]
	for _, f := range removeFiles {
		if f.IsDir() {
			continue
		}
		err := os.Remove(filepath.Join(dir, f.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}
