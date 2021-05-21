package cke

import (
	"os"
	"os/exec"
	"testing"

	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/log"
	"go.etcd.io/etcd/clientv3"
)

const (
	etcdClientURL = "http://localhost:12379"
	etcdPeerURL   = "http://localhost:12380"
)

func testMain(m *testing.M) int {
	ci := os.Getenv("CI") == "true"
	if ci {
		code := m.Run()
		os.Exit(code)
	}

	etcdPath, err := os.MkdirTemp("", "cke-test")
	if err != nil {
		log.ErrorExit(err)
	}

	cmd := exec.Command("etcd",
		"--data-dir", etcdPath,
		"--initial-cluster", "default="+etcdPeerURL,
		"--listen-peer-urls", etcdPeerURL,
		"--initial-advertise-peer-urls", etcdPeerURL,
		"--listen-client-urls", etcdClientURL,
		"--advertise-client-urls", etcdClientURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.ErrorExit(err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
		os.RemoveAll(etcdPath)
	}()

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func newEtcdClient(t *testing.T) *clientv3.Client {
	var clientURL string
	ci := os.Getenv("CI") == "true"
	if ci {
		clientURL = "http://localhost:2379"
	} else {
		clientURL = etcdClientURL
	}

	cfg := etcdutil.NewConfig(t.Name() + "/")
	cfg.Endpoints = []string{clientURL}

	etcd, err := etcdutil.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return etcd
}
