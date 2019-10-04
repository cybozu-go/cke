package mtest

import (
	. "github.com/onsi/ginkgo"
)

// TestStopCP stops 1 control plane for succeeding tests
func TestStopCP() {
	It("should stop CP", func() {
		execAt(node2, "sudo", "systemd-run", "poweroff", "-f", "-f")
		execAt(node3, "sudo", "systemctl", "stop", "sshd.socket")
	})
}
