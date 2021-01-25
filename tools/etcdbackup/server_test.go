package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testHandleBackup(t *testing.T) {
	t.Parallel()

	backupDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(backupDir)

	cfg := NewConfig()
	cfg.BackupDir = backupDir
	cfg.Rotate = 1
	s := NewServer(cfg)

	// backupDir is empty
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/backup", nil)
	s.ServeHTTP(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Error("wrong status code:", resp.StatusCode)
	}
	resp.Body.Close()

	// Save
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/api/v1/backup", nil)
	s.ServeHTTP(w, r)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Error("wrong status code:", resp.StatusCode)
	}
	resp.Body.Close()

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/api/v1/backup", nil)
	s.ServeHTTP(w, r)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Error("wrong status code:", resp.StatusCode)
	}
	resp.Body.Close()

	var list []string
	err = json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Error("len(list) != 1:", len(list))
	}
	if !strings.Contains(list[0], "snapshot-") {
		t.Error("file does not contain \"snapshot-\"", list[0])
	}
	backup1 := list[0]

	// Rotate backups
	time.Sleep(time.Second)

	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/api/v1/backup", nil)
	s.ServeHTTP(w, r)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Error("wrong status code:", resp.StatusCode)
	}
	resp.Body.Close()

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/api/v1/backup", nil)
	s.ServeHTTP(w, r)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Error("wrong status code:", resp.StatusCode)
	}
	resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Error("len(list) != 1:", len(list))
	}
	if list[0] == backup1 {
		t.Error("backup is not rotated", list[0])
	}

	// Download
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", path.Join("/api/v1/backup", list[0]), nil)
	s.ServeHTTP(w, r)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Error("wrong status code:", resp.StatusCode)
	}
	resp.Body.Close()

	files, err := ioutil.ReadDir(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Error("len(files) != 1:", len(files))
	}
	if files[0].Name() != list[0] {
		t.Error("files[0].Name() != list[0]:", files[0].Name())
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/api/v1/backup/foobar", nil)
	s.ServeHTTP(w, r)

	resp = w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Error("wrong status code:", resp.StatusCode)
	}
	resp.Body.Close()

	// File does not exist
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/api/v1/backup/snapshot-foobar.db.gz", nil)
	s.ServeHTTP(w, r)

	resp = w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Error("wrong status code:", resp.StatusCode)
	}
	resp.Body.Close()

	// File exists but directory
	err = os.MkdirAll(filepath.Join(backupDir, "snapshot-foobar.db.gz"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/api/v1/backup/snapshot-foobar.db.gz", nil)
	s.ServeHTTP(w, r)

	resp = w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Error("wrong status code:", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestServer(t *testing.T) {
	t.Run("Backup", testHandleBackup)
}
