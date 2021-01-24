package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s DIR\n", os.Args[0])
		os.Exit(2)
	}
	if err := subMain(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func subMain(dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", dir, err)
	}

	for _, f := range files {
		target := filepath.Join(dir, f.Name())
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("failed to delete %s: %w", target, err)
		}
	}
	return nil
}
