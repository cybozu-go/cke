package main

import (
	"compress/bzip2"
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
)

const (
	dockerImage = "docker.img"
)

var (
	flgBucket = flag.String("bucket", "cke-test", "GCP Cloud Storage's bucket name")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.ErrorExit(err)
	}
}

func run() error {
	images := cke.AllImages()
	h := sha256.New()
	h.Write([]byte(strings.Join(images, "")))
	checkSum := h.Sum(nil)

	targetFile := fmt.Sprintf("var-lib-docker.%x.img", checkSum)

	_, err = os.Stat(targetFile)
	if err == nil {
		log.Info(targetFile+" already exists", nil)

		// Use targetFile as dockerImage.
		return link(targetFile)
	}

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	storageClient, err := storage.NewClient(ctxWithTimeout)
	if err != nil {
		return err
	}

	rc, err := storageClient.Bucket(*flgBucket).Object(targetFile + ".bz2").NewReader(ctxWithTimeout)
	if err == storage.ErrObjectNotExist {
		log.Info(targetFile+".bz2 not uploaded", nil)
		return nil
	} else if err != nil {
		return err
	}
	defer rc.Close()

	// Download to targetFile first, and then make symlink as dockerImage.
	// The intermediate targetFile is necessary for checking existence in later executions.

	err = download(targetFile, rc)
	if err != nil {
		return err
	}

	return link(targetFile)
}

func download(filename string, r io.Reader) error {
	expanded := bzip2.NewReader(r)

	w, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, expanded)
	if err != nil {
		return err
	}

	return w.Sync()
}

func link(filename string) error {
	err := os.Remove(dockerImage)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return os.Symlink(filename, dockerImage)
}
