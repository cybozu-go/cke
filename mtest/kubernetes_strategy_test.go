package mtest

import (
	"fmt"
	"os"
	"path"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("kubernetes strategy", func() {
	AfterEach(initializeControlPlane)

	It("should deploy HA control plane", func() {
		By("Checking cluster status")
		Eventually(func() bool {
			controlPlanes := []string{node1, node2, node3}
			workers := []string{node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return false
			}
			defer status.Destroy()

			return checkKubernetesClusterStatus(status, controlPlanes, workers)
		}).Should(BeTrue())

		By("Killing the active service")
		leader := make(map[string]string)
		for _, service := range []string{"kube-controller-manager", "kube-scheduler"} {
			stdout, _, err := execAt(node1, getLeaderCommands(service)...)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(stdout)).ToNot(BeZero())

			holderIdentity := string(stdout)
			fmt.Printf("current active %s is %s\n", service, holderIdentity)

			leader[service] = strings.SplitN(string(stdout), "_", 2)[0]

			stdout, _, err = execAt(os.Getenv(strings.ToUpper(leader[service])), "docker", "kill", service)
			Expect(err).ToNot(HaveOccurred())
		}

		By("Switching another one")
		for _, service := range []string{"kube-controller-manager", "kube-scheduler"} {
			Eventually(func() bool {
				stdout, _, err := execAt(node1, getLeaderCommands(service)...)
				if err != nil {
					return false
				}
				if string(stdout) == leader[service] {
					fmt.Printf("active %s has not switched yet\n", service)
					return false
				}
				return true
			}).Should(BeTrue())
		}

		By("Checking component statuses are healthy")
		Expect(checkComponentStatuses(node1)).To(BeTrue())

		By("Checking all nodes status are ready")
		countReadyNodes := []string{"get", "nodes", "-o", "json", "|",
			"jq", `'[.items[].status.conditions[] | select( .type | contains("Ready") ) | select( .status | contains("True") )] | length'`}
		stdout := kubectl(countReadyNodes...)
		Expect(string(stdout)).To(Equal("6"))
	})

	It("should update node4 as control plane", func() {
		By("Changing definition of node4")
		ckecli("constraints", "set", "control-plane-count", "4")
		cluster := getCluster()
		for i := 0; i < 4; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		ckecliClusterSet(cluster)

		By("Checking cluster status")
		Eventually(func() bool {
			controlPlanes := []string{node1, node2, node3, node4}
			workers := []string{node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return false
			}
			defer status.Destroy()
			return checkKubernetesClusterStatus(status, controlPlanes, workers)
		}).Should(BeTrue())
	})

	It("should remove not-in-cluster node1 from cluster", func() {
		By("Removing definition of node1")
		ckecli("constraints", "set", "control-plane-count", "2")
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.Nodes = cluster.Nodes[1:]
		ckecliClusterSet(cluster)

		By("Checking cluster status")
		Eventually(func() bool {
			controlPlanes := []string{node2, node3}
			workers := []string{node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return false
			}
			defer status.Destroy()
			return checkKubernetesClusterStatus(status, controlPlanes, workers)
		}).Should(BeTrue())
	})
})

func getLeaderCommands(service string) []string {
	return []string{
		"curl", "-s", path.Join("localhost:18080/api/v1/namespaces/kube-system/endpoints/", service), "|",
		"jq", "-r", `.metadata.annotations'."control-plane.alpha.kubernetes.io/leader"'`, "|",
		"jq", "-r", ".holderIdentity"}
}
