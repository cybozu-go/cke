package mtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	"sigs.k8s.io/yaml"
)

const (
	sshTimeout         = 3 * time.Minute
	defaultDialTimeout = 30 * time.Second
	defaultKeepAlive   = 5 * time.Second

	// DefaultRunTimeout is the timeout value for Agent.Run().
	DefaultRunTimeout = 10 * time.Minute
)

var (
	sshClients = make(map[string]*sshAgent)
	httpClient = &well.HTTPClient{Client: &http.Client{}}

	agentDialer = &net.Dialer{
		Timeout:   defaultDialTimeout,
		KeepAlive: defaultKeepAlive,
	}
)

type sshAgent struct {
	client *ssh.Client
	conn   net.Conn
}

func sshTo(address string, sshKey ssh.Signer, userName string) (*sshAgent, error) {
	conn, err := agentDialer.Dial("tcp", address+":22")
	if err != nil {
		fmt.Printf("failed to dial: %s\n", address)
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(sshKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	err = conn.SetDeadline(time.Now().Add(defaultDialTimeout))
	if err != nil {
		conn.Close()
		return nil, err
	}
	clientConn, channelCh, reqCh, err := ssh.NewClientConn(conn, "tcp", config)
	if err != nil {
		// conn was already closed in ssh.NewClientConn
		return nil, err
	}
	err = conn.SetDeadline(time.Time{})
	if err != nil {
		clientConn.Close()
		return nil, err
	}
	a := sshAgent{
		client: ssh.NewClient(clientConn, channelCh, reqCh),
		conn:   conn,
	}
	return &a, nil
}

func prepareSSHClients(addresses ...string) error {
	sshKey, err := parsePrivateKey(sshKeyFile)
	if err != nil {
		return err
	}

	ch := time.After(sshTimeout)
	for _, a := range addresses {
	RETRY:
		select {
		case <-ch:
			return errors.New("prepareSSHClients timed out")
		default:
		}
		agent, err := sshTo(a, sshKey, "cybozu")
		if err != nil {
			time.Sleep(time.Second)
			goto RETRY
		}
		sshClients[a] = agent
	}

	return nil
}

func parsePrivateKey(keyPath string) (ssh.Signer, error) {
	f, err := os.Open(keyPath)
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

func reconnectSSH(address string) error {
	sshClients[address].client.Close()
	delete(sshClients, address)

	sshKey, err := parsePrivateKey(sshKeyFile)
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
	c, err := sshTo(address, sshKey, "cybozu")
	if err != nil {
		time.Sleep(5 * time.Second)
		goto RETRY
	}
	sshClients[address] = c
	return nil
}

func loadImage(path string) error {
	env := well.NewEnvironment(context.Background())
	for _, host := range []string{host1, host2} {
		host := host
		f, err := os.Open(path)
		Expect(err).NotTo(HaveOccurred())
		env.Go(func(ctx context.Context) error {
			_, _, err := execAtWithStream(host, f, "docker", "load")
			return err
		})
	}
	env.Stop()
	return env.Wait()
}

func installTools(image string) error {
	env := well.NewEnvironment(context.Background())
	for _, host := range []string{host1, host2} {
		host := host
		env.Go(func(ctx context.Context) error {
			sess, err := sshClients[host].client.NewSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			return sess.Run("docker run --rm -u root:root " +
				"--entrypoint /usr/local/cke/install-tools " +
				"--mount type=bind,src=/opt/bin,target=/host/ " + image)
		})
	}
	env.Stop()
	return env.Wait()
}

func stopCKE() error {
	env := well.NewEnvironment(context.Background())
	for _, host := range []string{host1, host2} {
		host2 := host
		env.Go(func(ctx context.Context) error {
			sess, err := sshClients[host2].client.NewSession()
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

func runCKE(image string) error {
	env := well.NewEnvironment(context.Background())
	for _, host := range []string{host1, host2} {
		host2 := host
		env.Go(func(ctx context.Context) error {
			sess, err := sshClients[host2].client.NewSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			return sess.Run("sudo mkdir -p /var/lib/cke && sudo systemd-run --unit=cke.service " +
				"docker run --rm --network=host --name cke " +
				"-e GOFAIL_HTTP=0.0.0.0:1234 " +
				"--mount type=bind,source=/var/lib/cke,target=/var/lib/cke " +
				"--mount type=bind,source=/etc/cke/,target=/etc/cke/ " +
				image + " --config /etc/cke/cke.yml --interval 3s --certs-gc-interval 5m --session-ttl 5s --loglevel debug")
		})
	}
	env.Stop()
	return env.Wait()
}

func execAt(host string, args ...string) (stdout, stderr []byte, e error) {
	return execAtWithStream(host, nil, args...)
}

// WARNING: `input` can contain secret data.  Never output `input` to console.
func execAtWithInput(host string, input []byte, args ...string) (stdout, stderr []byte, e error) {
	var r io.Reader
	if input != nil {
		r = bytes.NewReader(input)
	}
	return execAtWithStream(host, r, args...)
}

// WARNING: `input` can contain secret data.  Never output `input` to console.
func execAtWithStream(host string, input io.Reader, args ...string) (stdout, stderr []byte, e error) {
	agent := sshClients[host]
	return doExec(agent, input, args...)
}

// WARNING: `input` can contain secret data.  Never output `input` to console.
func doExec(agent *sshAgent, input io.Reader, args ...string) ([]byte, []byte, error) {
	err := agent.conn.SetDeadline(time.Now().Add(DefaultRunTimeout))
	if err != nil {
		return nil, nil, err
	}
	defer agent.conn.SetDeadline(time.Time{})

	sess, err := agent.client.NewSession()
	if err != nil {
		return nil, nil, err
	}
	defer sess.Close()

	if input != nil {
		sess.Stdin = input
	}
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	sess.Stdout = outBuf
	sess.Stderr = errBuf
	err = sess.Run(strings.Join(args, " "))
	return outBuf.Bytes(), errBuf.Bytes(), err
}

func execSafeAt(host string, args ...string) []byte {
	stdout, stderr, err := execAt(host, args...)
	ExpectWithOffset(1, err).To(Succeed(), "[%s] %v: %s", host, args, stderr)
	return stdout
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

func ckecli(args ...string) ([]byte, []byte, error) {
	args = append([]string{"/opt/bin/ckecli"}, args...)
	return execAt(host1, args...)
}

func ckecliSafe(args ...string) []byte {
	args = append([]string{"/opt/bin/ckecli"}, args...)
	return execSafeAt(host1, args...)
}

func ckecliWithInput(input []byte, args ...string) ([]byte, []byte, error) {
	args = append([]string{"/opt/bin/ckecli"}, args...)
	return execAtWithInput(host1, input, args...)
}

func localTempFile(body string) *os.File {
	f, err := ioutil.TempFile("", "cke-mtest")
	Expect(err).NotTo(HaveOccurred())
	_, err = f.WriteString(body)
	Expect(err).NotTo(HaveOccurred())
	err = f.Close()
	Expect(err).NotTo(HaveOccurred())
	return f
}

func remoteTempFile(body string) string {
	f, err := ioutil.TempFile("", "cke-mtest")
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	_, err = f.WriteString(body)
	Expect(err).NotTo(HaveOccurred())
	_, err = f.Seek(0, os.SEEK_SET)
	Expect(err).NotTo(HaveOccurred())
	remoteFile := filepath.Join("/tmp", filepath.Base(f.Name()))
	_, _, err = execAtWithStream(host1, f, "dd", "of="+f.Name())
	Expect(err).NotTo(HaveOccurred())
	return remoteFile
}

func getCluster() *cke.Cluster {
	b, err := ioutil.ReadFile(ckeClusterPath)
	Expect(err).NotTo(HaveOccurred())

	var cluster cke.Cluster

	err = yaml.Unmarshal(b, &cluster)
	Expect(err).NotTo(HaveOccurred())
	err = cluster.Validate(false)
	Expect(err).NotTo(HaveOccurred())

	return &cluster
}

func connectEtcd() (*clientv3.Client, error) {
	etcdConfig := cke.NewEtcdConfig()
	etcdConfig.Endpoints = []string{"http://" + host1 + ":2379"}
	return etcdutil.NewClient(etcdConfig)
}

func getClusterStatus(cluster *cke.Cluster) (*cke.ClusterStatus, []cke.ResourceDefinition, error) {
	controller := server.NewController(nil, 0, time.Hour, time.Second*2, nil)

	etcd, err := connectEtcd()
	if err != nil {
		return nil, nil, err
	}
	defer etcd.Close()

	st := cke.Storage{Client: etcd}
	ctx := context.Background()
	resources, err := st.GetAllResources(ctx)
	if err != nil {
		return nil, nil, err
	}

	inf, err := cke.NewInfrastructure(ctx, cluster, st)
	if err != nil {
		return nil, nil, err
	}
	defer inf.Close()

	cs, err := controller.GetClusterStatus(ctx, cluster, inf)
	if err != nil {
		return nil, nil, err
	}

	return cs, resources, err
}

func getServerStatus() (*cke.ServerStatus, error) {
	etcd, err := connectEtcd()
	if err != nil {
		return nil, err
	}
	defer etcd.Close()

	st := cke.Storage{Client: etcd}
	ctx := context.Background()
	return st.GetStatus(ctx)
}

func ckecliClusterSet(cluster *cke.Cluster) (time.Time, error) {
	y, err := yaml.Marshal(cluster)
	if err != nil {
		return time.Time{}, err
	}

	// TODO: remove this workaround added in #334
	data := string(y) + "\npod_subnet: 10.1.0.0/16"

	rf := remoteTempFile(data)
	_, _, err = ckecli("cluster", "set", rf)
	return time.Now(), err
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

func setupCKE(img string) {
	err := stopCKE()
	Expect(err).NotTo(HaveOccurred())
	err = runCKE(img)
	Expect(err).NotTo(HaveOccurred())
}

type checkError struct {
	Ops    []string
	Status *cke.ClusterStatus
}

func (e checkError) Error() string {
	return strings.Join(e.Ops, ",")
}

func checkCluster(c *cke.Cluster, ts time.Time) error {
	st, err := getServerStatus()
	if err != nil {
		if err == cke.ErrNotFound {
			return errors.New("server status is not found")
		}
		return err
	}

	if st.Phase != cke.PhaseCompleted {
		return fmt.Errorf("status:%+v", st)
	}
	if st.Timestamp.Before(ts) {
		return errors.New("server status is not yet updated")
	}
	return nil
}

func clusterSetAndWait(cluster *cke.Cluster) {
	ts, err := ckecliClusterSet(cluster)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	EventuallyWithOffset(1, func() error {
		return checkCluster(cluster, ts)
	}).Should(Succeed())
}

func initializeControlPlane() {
	ckecliSafe("constraints", "set", "control-plane-count", "3")
	cluster := getCluster()
	for i := 0; i < 3; i++ {
		cluster.Nodes[i].ControlPlane = true
	}
	clusterSetAndWait(cluster)
}

func setFailurePoint(failurePoint, code string) {
	leader := strings.TrimSpace(string(ckecliSafe("leader")))
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

func etcdctl(crt, key, ca string, args ...string) ([]byte, []byte, error) {
	args = append([]string{"env", "ETCDCTL_API=3", "/opt/bin/etcdctl", "--endpoints=https://" + node1 + ":2379,https://" + node2 + ":2379,https://" + node3 + ":2379",
		"--cert=" + crt, "--key=" + key, "--cacert=" + ca}, args...)
	return execAt(host1, args...)
}

func kubectl(args ...string) ([]byte, []byte, error) {
	args = append([]string{"/opt/bin/kubectl"}, args...)
	return execAt(host1, args...)
}

func kubectlWithInput(input []byte, args ...string) ([]byte, []byte, error) {
	args = append([]string{"/opt/bin/kubectl"}, args...)
	return execAtWithInput(host1, input, args...)
}

func getRandomNumber() *rand.Rand {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	return r1
}
