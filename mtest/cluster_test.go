package mtest

import (
	"bytes"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("cluster", func() {
	AfterEach(initializeControlPlane)

	It("should remove not-in-cluster node2 from cluster", func() {
		By("Removing definition of node2")
		ckecli("constraints", "set", "control-plane-count", "2")
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.Nodes = append(cluster.Nodes[:1], cluster.Nodes[2:]...)
		ckecliClusterSet(cluster)

		By("Checking cluster status")
		Eventually(func() error {
			controlPlanes := []string{node1, node3}
			workers := []string{node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return err
			}

			if err := checkEtcdClusterStatus(status, controlPlanes, workers); err != nil {
				return err
			}
			return checkKubernetesClusterStatus(status, controlPlanes, workers)
		}).Should(Succeed())

		By("Checking that CKE did not remove non-cluster node's etcd data")
		status, err := getClusterStatus()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.NodeStatuses[node2].Etcd.HasData).To(BeTrue())
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
		Eventually(func() error {
			controlPlanes := []string{node1, node2, node3, node4}
			workers := []string{node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return err
			}

			if err := checkEtcdClusterStatus(status, controlPlanes, workers); err != nil {
				return err
			}
			return checkKubernetesClusterStatus(status, controlPlanes, workers)
		}).Should(Succeed())
	})

	It("should adjust command arguments", func() {
		etcdArg := "--experimental-enable-v2v3=/v2/"
		controllerManagerArg := "--concurrent-deployment-syncs=5"

		By("Updating container options")
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.Options.Etcd.ExtraArguments = []string{etcdArg}
		cluster.Options.ControllerManager.ExtraArguments = []string{controllerManagerArg}
		ckecliClusterSet(cluster)

		By("Checking that etcd members and controller managers restarted with new arguments")
		Eventually(func() error {
			controlPlanes := []string{node1, node2, node3}
			workers := []string{node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return err
			}

			for _, node := range controlPlanes {
				cmds, err := inspect(node, "etcd")
				if err != nil {
					return errors.Wrap(err, "failed to exec docker inspect etcd")
				}

				ok := false
				for _, val := range cmds {
					if val == etcdArg {
						ok = true
					}
				}
				if !ok {
					return errors.New("etcd argument is not updated yet")
				}
			}

			for _, node := range controlPlanes {
				cmds, err := inspect(node, "kube-controller-manager")
				if err != nil {
					return errors.Wrap(err, "failed to exec docker inspect kube-controller-manager")
				}

				ok := false
				for _, val := range cmds {
					if val == controllerManagerArg {
						ok = true
					}
				}
				if !ok {
					return errors.New("kube-controller-manager argument is not updated yet")
				}
			}

			if err := checkEtcdClusterStatus(status, controlPlanes, workers); err != nil {
				return err
			}
			return checkKubernetesClusterStatus(status, controlPlanes, workers)
		}).Should(Succeed())

		// Revert and check here.
		// Though they will be performed in AfterEach, arguments are not checked there.
		// Checking arguments is too specific to this test, so do it here.
		By("Reverting container options")
		initializeControlPlane()
		Eventually(func() bool {
			controlPlanes := []string{node1, node2, node3}
			for _, node := range controlPlanes {
				cmds, err := inspect(node, "etcd")
				if err != nil {
					fmt.Println("failed to exec docker inspect etcd", err)
					return false
				}

				for _, val := range cmds {
					if val == etcdArg {
						fmt.Println("etcd argument is not reverted yet")
						return false
					}
				}
			}

			for _, node := range controlPlanes {
				cmds, err := inspect(node, "kube-controller-manager")
				if err != nil {
					fmt.Println("failed to exec docker inspect kube-controller-manager", err)
					return false
				}

				for _, val := range cmds {
					if val == controllerManagerArg {
						fmt.Println("kube-controller-manager argument is not reverted yet")
						return false
					}
				}
			}

			return true
		}).Should(BeTrue())
	})
})

func inspect(node, name string) ([]string, error) {
	stdout, _, err := execAt(node, "docker", "inspect", name, "--format='{{json .Config.Cmd}}'")
	if err != nil {
		return nil, err
	}

	var cmds = []string{}
	err = json.NewDecoder(bytes.NewReader(stdout)).Decode(&cmds)
	if err != nil {
		return nil, err
	}

	return cmds, nil
}
