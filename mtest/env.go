package mtest

import (
	"os"
)

var (
	host1            = os.Getenv("HOST1")
	host2            = os.Getenv("HOST2")
	node1            = os.Getenv("NODE1")
	node2            = os.Getenv("NODE2")
	node3            = os.Getenv("NODE3")
	node4            = os.Getenv("NODE4")
	node5            = os.Getenv("NODE5")
	node6            = os.Getenv("NODE6")
	ckecliPath       = os.Getenv("CKECLI")
	kubectlPath      = os.Getenv("KUBECTL")
	etcdctlPath      = os.Getenv("ETCDCTL")
	ckeClusterPath   = os.Getenv("CKECLUSTER")
	ckeConfigPath    = os.Getenv("CKECONFIG")
	policyYAMLPath   = os.Getenv("POLICYYAML")
	nginxYAMLPath    = os.Getenv("NGINXYAML")
	localPVYAMLPath  = os.Getenv("LOCALPVYAML")
	containerRuntime = os.Getenv("CONTAINER_RUNTIME")
)
