package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	namespace = "cke"
)

// OperationRunning returns True (== 1) if any operations are running.
var OperationRunning = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "operation_running",
		Help:      "1 if any operations are running.",
	},
)

// BootLeader returns True (== 1) if the boot server is the leader of CKE.
var BootLeader = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "boot_leader",
		Help:      "1 if the boot server is the leader of CKE.",
	},
)

// NodeInfo returns the Control Plane and Worker info.
var NodeInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "node_info",
		Help:      "The Control Plane and Worker info.",
	},
	[]string{"rack", "role", "control_plane"},
)
