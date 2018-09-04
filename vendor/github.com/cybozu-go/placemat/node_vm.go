package placemat

import (
	"io"
	"net"

	"github.com/cybozu-go/cmd"
)

// NodeVM holds resources to manage and monitor a QEMU process.
type NodeVM struct {
	cmd     *cmd.LogCmd
	monitor net.Conn
	running bool
	cleanup func()
}

// IsRunning returns true if the VM is running.
func (n *NodeVM) IsRunning() bool {
	return n.running
}

// PowerOn turns on the power of the VM.
func (n *NodeVM) PowerOn() {
	if n.running {
		return
	}

	io.WriteString(n.monitor, "system_reset\ncont\n")
	n.running = true
}

// PowerOff turns off the power of the VM.
func (n *NodeVM) PowerOff() {
	if !n.running {
		return
	}

	io.WriteString(n.monitor, "stop\n")
	n.running = false
}
