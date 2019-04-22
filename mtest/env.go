package mtest

import (
	"os"
)

var (
	host1 = os.Getenv("HOST1")
	host2 = os.Getenv("HOST2")
	node1 = os.Getenv("NODE1")
	node2 = os.Getenv("NODE2")
	node3 = os.Getenv("NODE3")
	node4 = os.Getenv("NODE4")
	node5 = os.Getenv("NODE5")
	node6 = os.Getenv("NODE6")

	ckeClusterPath  = os.Getenv("CKECLUSTER")
	ckeConfigPath   = os.Getenv("CKECONFIG")
	ckecliPath      = os.Getenv("CKECLI")
	ckeImagePath    = os.Getenv("CKE_IMAGE")
	ckeImageURL     = os.Getenv("CKE_IMAGE_URL")
	etcdctlPath     = os.Getenv("ETCDCTL")
	kubectlPath     = os.Getenv("KUBECTL")
	localPVYAMLPath = os.Getenv("LOCALPVYAML")
	nginxYAMLPath   = os.Getenv("NGINXYAML")
	policyYAMLPath  = os.Getenv("POLICYYAML")

	containerRuntime = os.Getenv("CONTAINER_RUNTIME")
	sshKeyFile       = os.Getenv("SSH_PRIVKEY")
)
