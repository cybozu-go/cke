package mtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("kubernetes strategy", func() {
	AfterEach(initializeControlPlane)

	It("should deploy HA control plane", func() {
		By("Checking cluster status")
		Eventually(func() error {
			controlPlanes := []string{node1, node2, node3}
			workers := []string{node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return err
			}
			return checkKubernetesClusterStatus(status, controlPlanes, workers)
		}).Should(Succeed())

		By("Killing the active service")
		leader := make(map[string]string)
		for _, service := range []string{"kube-controller-manager", "kube-scheduler"} {
			current, err := currentLeader(service)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("current active %s is %s\n", service, current)
			leader[service] = strings.SplitN(current, "_", 2)[0]
			_, _, err = execAt(os.Getenv(strings.ToUpper(leader[service])), "docker", "kill", service)
			Expect(err).NotTo(HaveOccurred())
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
		Expect(checkComponentStatuses(node1)).To(Succeed())

		By("Checking all nodes status are ready")
		stdout := kubectl("get", "nodes", "-o", "json")
		var nodeList struct {
			Items []struct {
				Status struct {
					Conditions []struct {
						LastHeartbeatTime  time.Time `json:"lastHeartbeatTime"`
						LastTransitionTime time.Time `json:"lastTransitionTime"`
						Message            string    `json:"message"`
						Reason             string    `json:"reason"`
						Status             string    `json:"status"`
						Type               string    `json:"type"`
					} `json:"conditions"`
				} `json:"status"`
			} `json:"items"`
		}
		err := json.NewDecoder(bytes.NewReader(stdout)).Decode(&nodeList)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			for _, item := range nodeList.Items {
				for _, st := range item.Status.Conditions {
					if st.Type == "Ready" && st.Status != "True" {
						return false
					}
				}
			}
			return true
		}).Should(BeTrue())
	})
})

func currentLeader(service string) (string, error) {
	stdout := kubectl("get", "endpoints", "--namespace=kube-system", "-o", "json", service)

	var endpoint struct {
		Metadata struct {
			Annotations struct {
				Leader string `json:"control-plane.alpha.kubernetes.io/leader"`
			} `json:"annotations"`
		} `json:"metadata"`
	}
	err := json.NewDecoder(bytes.NewReader(stdout)).Decode(&endpoint)
	if err != nil {
		return "", err
	}

	var record struct {
		HolderIdentity string `json:"holderIdentity"`
	}
	err = json.NewDecoder(strings.NewReader(endpoint.Metadata.Annotations.Leader)).Decode(&record)
	if err != nil {
		return "", err
	}

	return record.HolderIdentity, nil
}
