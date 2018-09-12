package main

import (
	"compress/bzip2"
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	err := deleteDockerImage()
	if err != nil {
		return err
	}

	images := cke.AllImages()
	h := sha256.New()
	h.Write([]byte(strings.Join(images, "")))
	checkSum := h.Sum(nil)

	targetFile := fmt.Sprintf("var-lib-docker.%x.img", checkSum)

	_, err = os.Stat(targetFile)
	if err == nil {
		log.Info(targetFile+" already exists", nil)

		// Use targetFile as dockerImage.
		return os.Symlink(targetFile, dockerImage)
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
		return createDummyImage()
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

	return os.Symlink(targetFile, dockerImage)
}

func deleteDockerImage() error {
	err := os.Remove(dockerImage)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func download(filename string, r io.Reader) error {
	log.Info("downloading "+filename, nil)
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

func executeCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	defer func() {
		cmd.Process.Kill()
	}()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Info(name+" "+strings.Join(args, " "), nil)
	return cmd.Run()
}

func createDummyImage() error {
	log.Info("creating dummy image "+dockerImage, nil)
	prepareDeviceCommands := [][]string{
		{"qemu-img", "create", "-f", "qcow2", dockerImage, "2G"},
		{"sudo", "modprobe", "nbd"},
		{"sudo", "qemu-nbd", "-c", "/dev/nbd0", dockerImage},
		{"sudo", "mkfs", "-t", "btrfs", "/dev/nbd0"},
		{"sudo", "qemu-nbd", "-d", "/dev/nbd0"},
	}
	for _, command := range prepareDeviceCommands {
		err := executeCommand(command[0], command[1:]...)
		if err != nil {
			return err
		}
	}
	return nil
}
