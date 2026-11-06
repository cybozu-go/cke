package main

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExtract(t *testing.T) {
	files := []struct {
		name string
		mode int64
		data string
		// expected permission bits of the extracted file
		expected os.FileMode
	}{
		{name: "etc/foo.yaml", mode: 0o644, data: "foo", expected: 0o644},
		{name: "etc/secret.yaml", mode: 0o600, data: "secret", expected: 0o600},
		// an entry without a mode falls back to the default
		{name: "etc/bar.yaml", mode: 0, data: "bar", expected: 0o640},
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	for _, f := range files {
		hdr := &tar.Header{
			Name: f.name,
			Mode: f.mode,
			Size: int64(len(f.data)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(f.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := extract(dir, buf, 0o640); err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		dest := filepath.Join(dir, f.name)
		fi, err := os.Stat(dest)
		if err != nil {
			t.Error(err)
			continue
		}
		if fi.Mode().Perm() != f.expected {
			t.Errorf("%s: unexpected mode: expected=%o actual=%o", f.name, f.expected, fi.Mode().Perm())
		}
		data, err := os.ReadFile(dest)
		if err != nil {
			t.Error(err)
			continue
		}
		if string(data) != f.data {
			t.Errorf("%s: unexpected contents: expected=%q actual=%q", f.name, f.data, string(data))
		}
	}
}

func TestExtractOverwrite(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "etc/secret.yaml")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{dest, dest + ".tmp"} {
		if err := os.WriteFile(name, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	data := "secret"
	if err := tw.WriteHeader(&tar.Header{Name: "etc/secret.yaml", Mode: 0o600, Size: int64(len(data))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(data)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	if err := extract(dir, buf, 0o644); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("unexpected mode: expected=600 actual=%o", fi.Mode().Perm())
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != data {
		t.Errorf("unexpected contents: expected=%q actual=%q", data, string(got))
	}
}
