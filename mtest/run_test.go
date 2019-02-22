package mtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/server"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/well"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"
	yaml "gopkg.in/yaml.v2"
)

const sshTimeout = 3 * time.Minute

var (
	sshClients = make(map[string]*ssh.Client)
	httpClient = &well.HTTPClient{Client: &http.Client{}}
)

func sshTo(address string, sshKey ssh.Signer) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: "cybozu",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(sshKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	return ssh.Dial("tcp", address+":22", config)
}

func parsePrivateKey() (ssh.Signer, error) {
	f, err := os.Open(os.Getenv("SSH_PRIVKEY"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(data)
}

func prepareSSHClients(addresses ...string) error {
	sshKey, err := parsePrivateKey()
	if err != nil {
		return err
	}

	ch := time.After(sshTimeout)
	for _, a := range addresses {
	RETRY:
		select {
		case <-ch:
			return errors.New("timed out")
		default:
		}
		client, err := sshTo(a, sshKey)
		if err != nil {
			time.Sleep(5 * time.Second)
			goto RETRY
		}
		sshClients[a] = client
	}

	return nil
}

func reconnectSSH(address string) error {
	if c, ok := sshClients[address]; ok {
		c.Close()
		delete(sshClients, address)
	}

	sshKey, err := parsePrivateKey()
	if err != nil {
		return err
	}
	ch := time.After(sshTimeout)
RETRY:
	select {
	case <-ch:
		return errors.New("timed out")
	default:
	}
	c, err := sshTo(address, sshKey)
	if err != nil {
		time.Sleep(5 * time.Second)
		goto RETRY
	}
	sshClients[address] = c
	return nil
}

func stopCKE() error {
	env := well.NewEnvironment(context.Background())
	for _, host := range []string{host1, host2} {
		host2 := host
		env.Go(func(ctx context.Context) error {
			c := sshClients[host2]
			sess, err := c.NewSession()
			if err != nil {
				return err
			}
			defer sess.Close()

			sess.Run("sudo systemctl reset-failed cke.service; sudo systemctl stop cke.service")

			return nil // Ignore error if cke was not running
		})
	}
	env.Stop()
	return env.Wait()
}

func runCKE() error {
	env := well.NewEnvironment(context.Background())
	for _, host := range []string{host1, host2} {
		host2 := host
		env.Go(func(ctx context.Context) error {
			c := sshClients[host2]
			sess, err := c.NewSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			return sess.Run("sudo systemd-run --unit=cke.service --setenv=GOFAIL_HTTP=0.0.0.0:1234 /opt/bin/cke --interval 3s --session-ttl 5s")
		})
	}
	env.Stop()
	return env.Wait()
}

func execAt(host string, args ...string) (stdout, stderr []byte, e error) {
	client := sshClients[host]
	sess, err := client.NewSession()
	if err != nil {
		return nil, nil, err
	}
	defer sess.Close()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	sess.Stdout = outBuf
	sess.Stderr = errBuf
	err = sess.Run(strings.Join(args, " "))
	return outBuf.Bytes(), errBuf.Bytes(), err
}

func execSafeAt(host string, args ...string) string {
	stdout, _, err := execAt(host, args...)
	ExpectWithOffset(1, err).To(Succeed())
	return string(stdout)
}

func execAtLocal(cmd string, args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	command := exec.Command(cmd, args...)
	command.Stdout = &stdout
	command.Stderr = GinkgoWriter
	err := command.Run()
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func localTempFile(body string) *os.File {
	f, err := ioutil.TempFile("", "cke-mtest")
	Expect(err).NotTo(HaveOccurred())
	f.WriteString(body)
	f.Close()
	return f
}

func ckecli(args ...string) []byte {
	stdout, err := ckecliUnsafe(args...)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return stdout
}

func ckecliUnsafe(args ...string) ([]byte, error) {
	args = append([]string{"--config", ckeConfigPath}, args...)
	var stdout bytes.Buffer
	command := exec.Command(ckecliPath, args...)
	command.Stdout = &stdout
	command.Stderr = GinkgoWriter
	timer := time.AfterFunc(10*time.Second, func() {
		command.Process.Signal(os.Interrupt)
	})
	defer timer.Stop()
	err := command.Run()
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func getCluster() *cke.Cluster {
	f, err := os.Open(ckeClusterPath)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()

	var cluster cke.Cluster
	err = yaml.NewDecoder(f).Decode(&cluster)
	Expect(err).NotTo(HaveOccurred())
	err = cluster.Validate()
	Expect(err).NotTo(HaveOccurred())

	return &cluster
}

func connectEtcd() (*clientv3.Client, error) {
	etcdConfig := cke.NewEtcdConfig()
	etcdConfig.Endpoints = []string{"http://" + host1 + ":2379"}
	return etcdutil.NewClient(etcdConfig)
}

func getClusterStatus(cluster *cke.Cluster) (*cke.ClusterStatus, error) {
	controller := server.NewController(nil, 0, time.Second*2, nil)

	etcd, err := connectEtcd()
	if err != nil {
		return nil, err
	}
	defer etcd.Close()

	inf, err := cke.NewInfrastructure(context.Background(), cluster, cke.Storage{Client: etcd})
	if err != nil {
		return nil, err
	}
	defer inf.Close()

	return controller.GetClusterStatus(context.Background(), cluster, inf)
}

func ckecliClusterSet(cluster *cke.Cluster) error {
	y, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}

	f := localTempFile(string(y))
	ckecli("cluster", "set", f.Name())
	return nil
}

func stopManagementEtcd(client *ssh.Client) error {
	command := "sudo systemctl stop my-etcd.service; sudo rm -rf /home/cybozu/default.etcd"
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	sess.Run(command)
	return nil
}

func stopVault(client *ssh.Client) error {
	command := "sudo systemctl stop my-vault.service"
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	sess.Run(command)
	return nil
}

func setupCKE() {
	err := stopCKE()
	Expect(err).NotTo(HaveOccurred())
	err = runCKE()
	Expect(err).NotTo(HaveOccurred())
}

type checkError struct {
	Ops    []string
	Status *cke.ClusterStatus
}

func (e checkError) Error() string {
	return strings.Join(e.Ops, ",")
}

func checkCluster(c *cke.Cluster) error {
	status, err := getClusterStatus(c)
	if err != nil {
		return err
	}

	nf := server.NewNodeFilter(c, status)
	if !nf.EtcdIsGood() {
		return errors.New("etcd cluster is not good")
	}

	ops := server.DecideOps(c, status)
	if len(ops) == 0 {
		return nil
	}
	opNames := make([]string, len(ops))
	for i, op := range ops {
		opNames[i] = op.Name()
	}
	return checkError{opNames, status}
}

func initializeControlPlane() {
	ckecli("constraints", "set", "control-plane-count", "3")
	cluster := getCluster()
	for i := 0; i < 3; i++ {
		cluster.Nodes[i].ControlPlane = true
	}
	ckecliClusterSet(cluster)
	Eventually(func() error {
		return checkCluster(cluster)
	}).Should(Succeed())
}

func setFailurePoint(failurePoint, code string) {
	leader := strings.TrimSpace(string(ckecli("leader")))
	Expect(leader).To(Or(Equal("host1"), Equal("host2")))
	var leaderAddress string
	if leader == "host1" {
		leaderAddress = host1
	} else {
		leaderAddress = host2
	}

	u := fmt.Sprintf("http://%s:1234/github.com/cybozu-go/cke/%s", leaderAddress, failurePoint)
	req, _ := http.NewRequest(http.MethodPut, u, strings.NewReader(code))
	resp, err := httpClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()
	Expect(resp.StatusCode / 100).To(Equal(2))
}

func injectFailure(failurePoint string) {
	setFailurePoint(failurePoint, "panic(\"cke-mtest\")")
}

func deleteFailure(failurePoint string) {
	setFailurePoint(failurePoint, "")
}

func etcdctl(crt, key, ca string, args ...string) error {
	args = append([]string{"--endpoints=https://" + node1 + ":2379,https://" + node2 + ":2379,https://" + node3 + ":2379",
		"--cert=" + crt, "--key=" + key, "--cacert=" + ca}, args...)
	cmd := exec.Command("output/etcdctl", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "ETCDCTL_API=3")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func kubectl(args ...string) ([]byte, []byte, error) {
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	args = append([]string{"--kubeconfig=/tmp/cke-mtest-kubeconfig"}, args...)
	cmd := exec.Command(kubectlPath, args...)
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err := cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}
