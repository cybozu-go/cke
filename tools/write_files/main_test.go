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
