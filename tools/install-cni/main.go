package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	hostBinDir = "/host/bin"
	cniDir     = "/cni_plugins/bin"
)

func main() {
	err := subMain()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func subMain() error {
	files, err := ioutil.ReadDir(cniDir)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", cniDir, err)
	}

	for _, f := range files {
		if err := copyFile(f.Name()); err != nil {
			return err
		}
	}

	return nil
}

func copyFile(name string) error {
	src := filepath.Join(cniDir, name)
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", src, err)
	}
	defer f.Close()

	dest := filepath.Join(hostBinDir, name)
	err = os.Remove(dest)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove %s: %w", dest, err)
	}

	o, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", dest, err)
	}
	defer o.Close()

	_, err = io.Copy(o, f)
	if err != nil {
		return fmt.Errorf("failed to copy %s: %w", name, err)
	}

	if err := o.Sync(); err != nil {
		return fmt.Errorf("failed to fsync %s: %w", dest, err)
	}

	return o.Chmod(0755)
}
