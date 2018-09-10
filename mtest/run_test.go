package mtest

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/etcdutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
	"k8s.io/kubernetes/pkg/apis/core"
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

func stopCKE() error {
	env := cmd.NewEnvironment(context.Background())
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
	env := cmd.NewEnvironment(context.Background())
	for _, host := range []string{host1, host2} {
		host2 := host
		env.Go(func(ctx context.Context) error {
			c := sshClients[host2]
			sess, err := c.NewSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			return sess.Run("sudo systemd-run --unit=cke.service --setenv=GOFAIL_HTTP=0.0.0.0:1234 /data/cke -interval 3s -session-ttl 5s")
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

func localTempFile(body string) *os.File {
	f, err := ioutil.TempFile("", "cke-mtest")
	Expect(err).NotTo(HaveOccurred())
	f.WriteString(body)
	f.Close()
	return f
}

func ckecli(args ...string) []byte {
	args = append([]string{"-config", ckeConfigPath}, args...)
	var stdout bytes.Buffer
	command := exec.Command(ckecliPath, args...)
	command.Stdout = &stdout
	command.Stderr = GinkgoWriter
	err := command.Run()
	Expect(err).NotTo(HaveOccurred())
	return stdout.Bytes()
}

func kubectl(args ...string) []byte {
	args = append([]string{"--kubeconfig", kubeconfigPath}, args...)
	var stdout bytes.Buffer
	command := exec.Command(kubectlPath, args...)
	command.Stdout = &stdout
	command.Stderr = GinkgoWriter
	err := command.Run()
	Expect(err).NotTo(HaveOccurred())
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

func connectEtcd() (*clientv3.Client, error) {
	etcdConfig := cke.NewEtcdConfig()
	etcdConfig.Endpoints = []string{"http://" + host1 + ":2379"}
	return etcdutil.NewClient(etcdConfig)
}

func getClusterStatus() (*cke.ClusterStatus, error) {
	controller := cke.NewController(nil, 0, time.Second*2)
	cluster := getCluster()

	etcd, err := connectEtcd()
	if err != nil {
		return nil, err
	}
	defer etcd.Close()

	inf, err := cke.NewInfrastructure(context.Background(), cluster, cke.Storage{etcd})
	if err != nil {
		return nil, err
	}
	defer inf.Close()

	for _, n := range cluster.Nodes {
		n.ControlPlane = true
	}
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

type clusterStatusError struct {
	message       string
	host          string
	port          uint16
	status        *cke.ClusterStatus
	controlPlanes []string
	workers       []string
	stdout        string
	stderr        string
}

func (e clusterStatusError) Error() string {
	return fmt.Sprintf(`%s
host          : %s
port          : %d
control planes: %v
workers       : %v
stdout        : %s
stderr        : %s
status:
%+v
`, e.message, e.host, e.port, e.controlPlanes, e.workers, e.stdout, e.stderr, e.status)
}

func checkEtcdClusterStatus(status *cke.ClusterStatus, controlPlanes, workers []string) error {
	newError := func(msg, host string) error {
		return clusterStatusError{
			msg, host, 0, status, controlPlanes, workers, "", "",
		}
	}

	for _, host := range controlPlanes {
		if !status.NodeStatuses[host].Etcd.Running {
			return newError("etcd is not running", host)
		}
		if !status.NodeStatuses[host].Etcd.HasData {
			return newError("no etcd data", host)
		}
	}
	for _, host := range workers {
		if status.NodeStatuses[host].Etcd.Running {
			return newError("etcd is still running", host)
		}
	}
	if len(controlPlanes) != len(status.Etcd.Members) {
		return newError("wrong number of etcd members", "")
	}
	for _, host := range controlPlanes {
		member, ok := status.Etcd.Members[host]
		if !ok {
			return newError("host is not a member of etcd cluster", host)
		}
		if member.Name == "" {
			return newError("host has not joined to etcd cluster", host)
		}
		if !status.Etcd.InSyncMembers[host] {
			return newError("local etcd is out of sync", host)
		}
	}
	return nil
}

func isRunningAllControlPlaneComponents(status *cke.ClusterStatus, host string) error {
	newError := func(msg string) error {
		return clusterStatusError{
			msg, host, 0, status, nil, nil, "", "",
		}
	}

	if !status.NodeStatuses[host].APIServer.Running {
		return newError("kube-apiserver is not running")
	}
	if !status.NodeStatuses[host].ControllerManager.Running {
		return newError("kube-controller-manager is not running")
	}
	if !status.NodeStatuses[host].Scheduler.Running {
		return newError("kube-scheduler is not running")
	}
	return nil
}

func isRunningAnyControlPlaneComponents(status *cke.ClusterStatus, host string) error {
	newError := func(msg string) error {
		return clusterStatusError{
			msg, host, 0, status, nil, nil, "", "",
		}
	}

	if status.NodeStatuses[host].APIServer.Running {
		return newError("kube-apiserver is running")
	}
	if status.NodeStatuses[host].ControllerManager.Running {
		return newError("kube-controller-manager is running")
	}
	if status.NodeStatuses[host].Scheduler.Running {
		return newError("kube-scheduler is running")
	}
	return nil
}

func isRunningAllCommonComponents(status *cke.ClusterStatus, host string) error {
	newError := func(msg string) error {
		return clusterStatusError{
			msg, host, 0, status, nil, nil, "", "",
		}
	}

	if !status.NodeStatuses[host].Rivers.Running {
		return newError("rivers is not running")
	}
	if !status.NodeStatuses[host].Proxy.Running {
		return newError("kube-proxy is not running")
	}
	if !status.NodeStatuses[host].Kubelet.Running {
		return newError("kubelet is not running")
	}
	return nil
}

func checkKubernetesClusterStatus(status *cke.ClusterStatus, controlPlanes, workers []string) (err error) {
	defer func() {
		e, ok := err.(clusterStatusError)
		if !ok {
			return
		}
		e.status = status
		e.controlPlanes = controlPlanes
		e.workers = workers
	}()

	nodes := append(controlPlanes, workers...)

	for _, host := range controlPlanes {
		if err = isRunningAllControlPlaneComponents(status, host); err != nil {
			return
		}
	}
	for _, host := range workers {
		if err = isRunningAnyControlPlaneComponents(status, host); err != nil {
			return
		}
	}
	for _, host := range nodes {
		if err = isRunningAllCommonComponents(status, host); err != nil {
			return
		}
	}

	newError := func(msg, host string, port uint16, stdout, stderr string) error {
		return clusterStatusError{
			msg, host, port, status, controlPlanes, workers, stdout, stderr,
		}
	}

	for _, host := range controlPlanes {
		// 8080: apiserver, 10252: controller-manager, 10251: scheduler
		for _, port := range []uint16{8080, 10252, 10251} {
			stdout, stderr, err := execAt(host, "curl", "-sf", fmt.Sprintf("localhost:%d/healthz", port))
			if err != nil {
				return newError(err.Error(), host, port, string(stdout), string(stderr))
			}
			if string(stdout) != "ok" {
				return newError("stdout is not ok", host, port, string(stdout), string(stderr))
			}
		}
		if err = checkComponentStatuses(host); err != nil {
			return
		}
	}

	for _, host := range nodes {
		// 10248: kubelet, 10256: kube-proxy, 16443: rivers (to apiserver)
		for _, port := range []uint16{10248, 10256, 16443} {
			stdout, stderr, err := execAt(host, "curl", "-sf", fmt.Sprintf("localhost:%d/healthz", port))
			if err != nil {
				return newError(err.Error(), host, port, string(stdout), string(stderr))
			}
			if string(stdout) != "ok" && port != 10256 { // kube-proxy does not return "ok"
				return newError("stdout is not ok", host, port, string(stdout), string(stderr))
			}
		}
	}

	return nil
}

func checkComponentStatuses(host string) error {
	newError := func(msg, stdout string) error {
		return clusterStatusError{
			msg, host, 0, nil, nil, nil, stdout, "",
		}
	}

	stdout, _, err := execAt(host, "curl", "https://localhost:16443/api/v1/componentstatuses")
	if err != nil {
		return newError(err.Error(), string(stdout))
	}
	var csl core.ComponentStatusList
	err = json.NewDecoder(bytes.NewReader(stdout)).Decode(&csl)
	if err != nil {
		return newError(err.Error(), string(stdout))
	}
	for _, item := range csl.Items {
		for _, condition := range item.Conditions {
			if condition.Type != core.ComponentHealthy {
				return newError(item.Name+" is unhealthy", "")
			}
		}
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

func initializeControlPlane() {
	ckecli("constraints", "set", "control-plane-count", "3")
	cluster := getCluster()
	for i := 0; i < 3; i++ {
		cluster.Nodes[i].ControlPlane = true
	}
	ckecliClusterSet(cluster)
	Eventually(func() error {
		controlPlanes := []string{node1, node2, node3}
		workers := []string{node4, node5, node6}
		status, err := getClusterStatus()
		if err != nil {
			return err
		}
		if err := checkEtcdClusterStatus(status, controlPlanes, workers); err != nil {
			return err
		}
		return checkKubernetesClusterStatus(status, controlPlanes, workers)
	}, 5*time.Minute).Should(Succeed())
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
