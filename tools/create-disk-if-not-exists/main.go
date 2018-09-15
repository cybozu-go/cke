package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
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
	flgInstance    = flag.String("instance", "", "GCE instance name to create")
	flgAccountJSON = flag.String("account-json", "", "file path of service account's account.json ")
	flgCleanUp     = flag.Bool("cleanup", false, "whether to delete the created instance after execution(do not delete by default)")
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
	if len(*flgInstance) == 0 {
		log.ErrorExit(errors.New("must specify -instance"))
	}
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

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	storageClient, err := storage.NewClient(ctxWithTimeout)
	if err != nil {
		return err
	}

	file, err := os.Open(*flgAccountJSON)
	if err != nil {
		return err
	}
	defer file.Close()
	var ss cli.ServiceAccount
	err = json.NewDecoder(file).Decode(&ss)
	if err != nil {
		return err
	}

	c := client{
		Client:       storageClient,
		Account:      ss.ClientEmail,
		Project:      ss.ProjectID,
		Zone:         *flgZone,
		InstanceName: *flgInstance,
	}

	targetFile := fmt.Sprintf("var-lib-docker.%x.img.bz2", checkSum)
	_, err = c.Bucket(*flgBucket).Object(targetFile).Attrs(ctxWithTimeout)
	if err == nil {
		log.Info(targetFile+" already exists", nil)
		return nil
	} else if err != storage.ErrObjectNotExist {
		return err
	}
	log.Info(targetFile+" does not exist, I will start to create it", nil)

	c.gcloud("auth", "activate-service-account", "--key-file="+*flgAccountJSON)

	err = c.createInstance()
	if err != nil {
		return err
	}
	if *flgCleanUp {
		defer c.deleteInstance()
	} else {
		log.Info("since -cleanup flag is false, create-disk-if-not-exists does not delete the created GCE instance", nil)
	}

	err = c.waitCreatingInstance(60)
	if err != nil {
		return err
	}

	err = c.gcloud("compute", "scp", "--project", c.Project, "--zone", c.Zone, *flgAccountJSON, "./create-qcow", "cybozu@"+*flgInstance+":")
	if err != nil {
		return err
	}

	return c.ssh("sudo /home/cybozu/create-qcow -account-json=/home/cybozu/" + path.Base(*flgAccountJSON))
}

func (c client) ssh(command string) error {
	return c.gcloud("--project", c.Project, "compute", "ssh",
		"--zone", c.Zone, "cybozu@"+c.InstanceName, fmt.Sprintf("--command=%s", command))
}

func (c client) gcloud(args ...string) error {
	log.Info("gcloud "+strings.Join(args, " "), nil)
	cmd := exec.Command("gcloud", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c client) createInstance() error {
	log.Info("Creating GCE instance ("+c.InstanceName+"), please delete the instance later", nil)
	return c.gcloud("compute", "instances",
		"create", c.InstanceName,
		"--project", c.Project,
		"--zone", c.Zone,
		"--boot-disk-size=20GiB",
		"--preemptible",
		"--boot-disk-type=pd-ssd",
		"--machine-type=n1-standard-1",
		"--image-family=ubuntu-1804-lts",
		"--image-project=ubuntu-os-cloud",
		"--scopes=cloud-platform")
}

func (c client) waitCreatingInstance(durationSecond uint) error {
	for i := uint(0); i < durationSecond; i++ {
		err := c.ssh("echo Hi")
		if err != nil {
			log.Info("waiting for creating instance...", nil)
			time.Sleep(time.Second)
			continue
		}
		return nil
	}
	return errors.New("timeout")
}

func (c client) deleteInstance() error {
	return c.gcloud("compute", "instances",
		"delete", c.InstanceName,
		"--quiet",
		"--project", c.Project,
		"--zone", c.Zone)
}
