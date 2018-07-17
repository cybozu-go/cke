package cke

import (
	"bytes"
	"time"

	"net"

	"path/filepath"

	"github.com/cybozu-go/log"
	"golang.org/x/crypto/ssh"
)

const (
	defaultTimeout     = 10 * time.Minute
	defaultEtcdDataDir = "/var/lib/etcd"
)

type SSHAgent struct {
	node   *Node
	client *ssh.Client
}

func NewSSHAgent(node *Node) (SSHAgent, error) {
	config := &ssh.ClientConfig{
		User: node.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(node.signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         defaultTimeout,
	}

	client, err := ssh.Dial("tcp", node.Address+":22", config)
	if err != nil {
		log.Error("failed to dial: ", map[string]interface{}{
			log.FnError: err,
			"address":   node.Address,
		})
		return SSHAgent{}, err
	}

	return SSHAgent{
		node:   node,
		client: client,
	}, nil
}

func (a SSHAgent) Run(command string) (string, string, error) {
	session, err := a.client.NewSession()
	if err != nil {
		log.Error("failed to create session: ", map[string]interface{}{
			log.FnError: err,
		})
		return "", "", err
	}
	defer session.Close()

	var stdoutBuff bytes.Buffer
	var stderrBuff bytes.Buffer
	session.Stdout = &stdoutBuff
	session.Stderr = &stderrBuff
	if err := session.Run(command); err != nil {
		log.Error("failed to run command: ", map[string]interface{}{
			log.FnError: err,
			"command":   command,
		})
		return "", "", err
	}
	return stdoutBuff.String(), stderrBuff.String(), nil
}

func (a *SSHAgent) GetNodeStatus(cluster *Cluster) (*NodeStatus, error) {
	status := &NodeStatus{
		Address: net.ParseIP(a.node.Address),
	}

	etcd, err := a.getEtcdStatus(cluster.Options.Etcd)
	if err != nil {
		return status, nil
	}
	status.Etcd = etcd

	return status, nil
}

func (a *SSHAgent) getEtcdStatus(params EtcdParams) (ServiceStatus, error) {
	var status ServiceStatus
	dataDir := params.DataDir
	if len(dataDir) == 0 {
		dataDir = defaultEtcdDataDir
	}
	_, _, err := a.Run("test -d " + filepath.Join(dataDir, "default.etcd"))
	if err != nil {
		return status, err
	}
	status.Configured = true

	_, _, err = a.Run("test -d " + filepath.Join(dataDir, "default.etcd"))
	if err != nil {
		return status, err
	}

	return status, nil
}
