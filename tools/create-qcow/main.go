package main

import (
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
	"github.com/cybozu-go/cke/tools/cli"
	"github.com/cybozu-go/log"
)

var (
	flgZone        = flag.String("zone", "asia-northeast1-c", "GCP compute zone name")
	flgBucket      = flag.String("bucket", "cke-test", "GCP Cloud Storage's bucket name")
	flgAccountJSON = flag.String("account-json", "", "file path of service account's account.json ")
)

type client struct {
	*storage.Client
	Project      string
	Account      string
	Zone         string
	InstanceName string
}

func main() {
	flag.Parse()
	images := cke.AllImages()
	h := sha256.New()
	h.Write([]byte(strings.Join(images, "")))
	checkSum := h.Sum(nil)
	fileName := fmt.Sprintf("var-lib-docker.%x.img", checkSum)

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	storageClient, err := storage.NewClient(ctxWithTimeout)
	if err != nil {
		log.ErrorExit(err)
	}

	ss, err := cli.LoadAccountJSON(*flgAccountJSON)
	if err != nil {
		log.ErrorExit(err)
	}

	c := client{
		Client:       storageClient,
		Account:      ss.ClientEmail,
		Project:      ss.ProjectID,
		Zone:         *flgZone,
		InstanceName: "docker-" + fileName,
	}

	prepareDeviceCommands := [][]string{
		{"gcloud", "auth", "activate-service-account", "--key-file=" + *flgAccountJSON},
		{"apt", "update", "-qq"},
		{"apt", "install", "-y", "-qq", "docker.io", "qemu-utils"},
		{"systemctl", "stop", "docker.service"},
		{"qemu-img", "create", "-f", "qcow2", fileName, "8G"},
		{"modprobe", "nbd"},
		{"qemu-nbd", "-c", "/dev/nbd0", fileName},
		{"mkfs", "-t", "ext4", "/dev/nbd0"},
		{"mount", "/dev/nbd0", "/var/lib/docker"},
		{"systemctl", "start", "docker.service"},
	}
	for _, command := range prepareDeviceCommands {
		err = run(command[0], command[1:]...)
		if err != nil {
			log.ErrorExit(err)
		}
	}

	for _, command := range dockerPullImagesCommands(images) {
		err = run(command[0], command[1:]...)
		if err != nil {
			log.ErrorExit(err)
		}
	}

	ejectDeviceCommands := [][]string{
		{"systemctl", "stop", "docker.service"},
		{"sync"},
		{"umount", "/var/lib/docker"},
		{"qemu-nbd", "-d", "/dev/nbd0"},
		{"bzip2", fileName},
	}
	for _, command := range ejectDeviceCommands {
		err = run(command[0], command[1:]...)
		if err != nil {
			log.ErrorExit(err)
		}
	}

	err = c.upload(ctxWithTimeout, fileName+".bz2")
	if err != nil {
		log.ErrorExit(err)
	}
	log.Info("succeeded upload: "+fileName+".bz2", nil)
}

func dockerPullImagesCommands(images []string) [][]string {
	var res [][]string
	for _, image := range images {
		res = append(res, []string{"docker", "pull", image})
	}
	return res
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	defer func() {
		cmd.Process.Kill()
	}()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Info(name+" "+strings.Join(args, " "), nil)
	return cmd.Run()
}

func (c *client) upload(ctx context.Context, fileName string) error {
	log.Info("uploading file...", nil)

	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	wc := c.Client.Bucket(*flgBucket).Object(fileName).NewWriter(ctx)
	_, err = io.Copy(wc, f)
	if err != nil {
		return err
	}

	err = wc.Close()
	if err != nil {
		return err
	}

	acl := c.Client.Bucket(*flgBucket).Object(fileName).ACL()
	return acl.Set(ctx, storage.AllUsers, storage.RoleReader)
}
