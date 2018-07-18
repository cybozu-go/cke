package cke

import (
	"bytes"
	"time"

	"github.com/cybozu-go/log"
	"golang.org/x/crypto/ssh"
)

const (
	defaultTimeout = 30 * time.Second
)

// Agent is the interface to run commands on a node.
type Agent interface {
	// Close closes the underlying connection.
	Close() error

	// Run command on the node.
	Run(command string) (stdout, stderr []byte, err error)
}

type sshAgent struct {
	node   *Node
	client *ssh.Client
}

// SSHAgent creates an Agent that communicates over SSH.
// It returns non-nil error when connection could not be established.
func SSHAgent(node *Node) (Agent, error) {
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
		return nil, err
	}

	a := sshAgent{
		node:   node,
		client: client,
	}

	_, _, err = a.Run("docker version")
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (a sshAgent) Close() error {
	err := a.client.Close()
	a.client = nil
	return err
}

func (a sshAgent) Run(command string) (stdout, stderr []byte, e error) {
	session, err := a.client.NewSession()
	if err != nil {
		log.Error("failed to create session: ", map[string]interface{}{
			log.FnError: err,
		})
		return nil, nil, err
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
		return nil, nil, err
	}
	return stdoutBuff.Bytes(), stderrBuff.Bytes(), nil
}
