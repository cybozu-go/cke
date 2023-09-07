package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "cke"
)

var leader = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "leader",
		Help:      "1 if this server is the leader of CKE.",
	},
)

var operationPhase = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "operation_phase",
		Help:      "The phase where CKE is currently operating.",
	},
	[]string{"phase"},
)

var operationPhaseTimestampSeconds = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "operation_phase_timestamp_seconds",
		Help:      "The Unix timestamp when operation_phase was last updated.",
	},
)

var rebootQueueEntries = prometheus.NewDesc(
	namespace+"_reboot_queue_entries",
	"The number of reboot queue entries remaining.",
	nil,
	nil,
)

var rebootQueueItems = prometheus.NewDesc(
	namespace+"_reboot_queue_items",
	"The number of reboot queue entries remaining per status.",
	[]string{"status"},
	nil,
)

var nodeRebootStatus = prometheus.NewDesc(
	namespace+"_node_reboot_status",
	"The reboot status of a node.",
	[]string{"node", "status"},
	nil,
)

var sabakanIntegrationSuccessful = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "sabakan_integration_successful",
		Help:      "1 if sabakan-integration satisfies constraints.",
	},
)

var sabakanIntegrationTimestampSeconds = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "sabakan_integration_timestamp_seconds",
		Help:      "The Unix timestamp when sabakan_integration_successful was last updated.",
	},
)

var sabakanWorkers = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "sabakan_workers",
		Help:      "The number of worker nodes for each role.",
	},
	[]string{"role"},
)

var sabakanUnusedMachines = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "sabakan_unused_machines",
		Help:      "The number of unused machines.",
	},
)
