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

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cmd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"golang.org/x/crypto/ssh"
	yaml "gopkg.in/yaml.v2"
)

const sshTimeout = 3 * time.Minute

var (
	sshClients = make(map[string]*ssh.Client)
	httpClient = &cmd.HTTPClient{Client: &http.Client{}}
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

func runManagementEtcd(client *ssh.Client) error {
	command := "sudo systemd-run --unit=my-etcd.service /data/etcd --listen-client-urls=http://0.0.0.0:2379 --advertise-client-urls=http://localhost:2379 --data-dir /home/cybozu/default.etcd"
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.Run(command)
}

func stopCke() error {
	for _, host := range []string{host1, host2} {
		c := sshClients[host]
		sess, err := c.NewSession()
		if err != nil {
			return err
		}

		sess.Run("sudo systemctl reset-failed cke.service; sudo systemctl stop cke.service")
		sess.Close()
	}
	return nil
}

func runCke() error {
	for _, host := range []string{host1, host2} {
		c := sshClients[host]
		sess, err := c.NewSession()
		if err != nil {
			return err
		}

		err = sess.Run("sudo systemd-run --unit=cke.service --setenv=GOFAIL_HTTP=0.0.0.0:1234 /data/cke -config /etc/cke.yml -interval 10s -session-ttl 5s")
		sess.Close()
		if err != nil {
			return err
		}
	}
	return nil
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

func localTempFile(body string) *os.File {
	f, err := ioutil.TempFile("", "cke-mtest")
	Expect(err).NotTo(HaveOccurred())
	f.WriteString(body)
	f.Close()
	return f
}

func ckecli(args ...string) []byte {
	args = append([]string{"-config", ckeConfigPath}, args...)
	command := exec.Command(ckecliPath, args...)
	stdout := new(bytes.Buffer)
	session, err := gexec.Start(command, stdout, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
	return stdout.Bytes()
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

func getClusterStatus() (*cke.ClusterStatus, error) {
	controller := cke.NewController(nil, 0)
	cluster := getCluster()
	for _, n := range cluster.Nodes {
		n.ControlPlane = true
	}
	return controller.GetClusterStatus(context.Background(), cluster)
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

func checkEtcdClusterStatus(status *cke.ClusterStatus, controlPlanes, workers []string) bool {
	for _, host := range controlPlanes {
		if !status.NodeStatuses[host].Etcd.Running {
			fmt.Printf("%s is not running\n", host)
			return false
		}
		if !status.NodeStatuses[host].Etcd.HasData {
			fmt.Printf("%s does not have data\n", host)
			return false
		}
	}
	for _, host := range workers {
		if status.NodeStatuses[host].Etcd.Running {
			fmt.Printf("%s is running\n", host)
			return false
		}
		// if status.NodeStatuses[host].Etcd.HasData {
		// 	fmt.Printf("%s has data\n", host)
		// 	return false
		// }
	}
	if len(controlPlanes) != len(status.Etcd.Members) {
		fmt.Printf("len(controlPlanes) != len(status.Etcd.Members), %d != %d\n", len(controlPlanes), len(status.Etcd.Members))
		return false
	}
	for _, host := range controlPlanes {
		member, ok := status.Etcd.Members[host]
		if !ok {
			fmt.Printf("%s is not member\n", host)
			return false
		}
		if member.Name == "" {
			fmt.Printf("%s is unstarted\n", host)
			return false
		}

		health, ok := status.Etcd.MemberHealth[host]
		if !ok {
			fmt.Printf("%s's health is unknown\n", host)
			return false
		}
		if health != cke.EtcdNodeHealthy {
			fmt.Printf("%s is not healthy\n", host)
			return false
		}
	}
	return true
}

func setupCKE() {
	err := stopCke()
	Expect(err).NotTo(HaveOccurred())
	err = runCke()
	Expect(err).NotTo(HaveOccurred())

	// wait cke
	Eventually(func() error {
		_, _, err := execAt(host1, "/data/ckecli", "history")
		return err
	}).Should(Succeed())
}

func initializeControlPlane() {
	ckecli("constraints", "set", "control-plane-count", "3")
	cluster := getCluster()
	for i := 0; i < 3; i++ {
		cluster.Nodes[i].ControlPlane = true
	}
	ckecliClusterSet(cluster)
	Eventually(func() bool {
		controlPlanes := []string{node1, node2, node3}
		workers := []string{node4, node5, node6}
		status, err := getClusterStatus()
		if err != nil {
			return false
		}
		defer status.Destroy()
		return checkEtcdClusterStatus(status, controlPlanes, workers)
	}).Should(BeTrue())
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
