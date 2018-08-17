package mtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/kubernetes/pkg/apis/core"
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
			current, err := currentLeader(service)
			Expect(err).ToNot(HaveOccurred())
			fmt.Printf("current active %s is %s\n", service, current)
			leader[service] = strings.SplitN(current, "_", 2)[0]
			_, _, err = execAt(os.Getenv(strings.ToUpper(leader[service])), "docker", "kill", service)
			Expect(err).ToNot(HaveOccurred())
		}

		By("Switching another one")
		for _, service := range []string{"kube-controller-manager", "kube-scheduler"} {
			Eventually(func() bool {
				current, err := currentLeader(service)
				if err != nil {
					return false
				}
				if current == leader[service] {
					fmt.Printf("active %s has not switched yet\n", service)
					return false
				}
				return true
			}).Should(BeTrue())
		}

		By("Checking component statuses are healthy")
		Expect(checkComponentStatuses(node1)).To(BeTrue())

		By("Checking all nodes status are ready")
		stdout := kubectl("get", "nodes", "-o", "json")
		var nodeList core.NodeList
		err := json.NewDecoder(bytes.NewReader(stdout)).Decode(&nodeList)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			for _, item := range nodeList.Items {
				for _, st := range item.Status.Conditions {
					if st.Type == core.NodeReady && st.Status != core.ConditionTrue {
						return false
					}
				}
			}
			return true
		}).Should(BeTrue())
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

func currentLeader(service string) (string, error) {
	stdout := kubectl("get", "endpoints", "--namespace=kube-system", "-o", "json", service)

	var endpoint core.Endpoints
	err := json.NewDecoder(bytes.NewReader(stdout)).Decode(&endpoint)
	if err != nil {
		return "", err
	}

	var record struct {
		HolderIdentity string `json:"holderIdentity"`
	}
	recordString := endpoint.ObjectMeta.Annotations["control-plane.alpha.kubernetes.io/leader"]
	err = json.NewDecoder(strings.NewReader(recordString)).Decode(&record)
	if err != nil {
		return "", err
	}
	return record.HolderIdentity, nil
}
