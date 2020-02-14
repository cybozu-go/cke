package cke

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	v3 "github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type logger struct{}

func (l logger) Println(v ...interface{}) {
	log.Error(fmt.Sprint(v...), nil)
}

const (
	namespace     = "cke"
	scrapeTimeout = time.Second * 10
)

// Metric represents collector and availability of metric.
type Metric struct {
	collectors  []prometheus.Collector
	isAvailable func(context.Context, *Storage) (bool, error)
}

// Collector is a metrics collector for CKE.
type Collector struct {
	metrics map[string]Metric
	storage *Storage
}

// NewCollector returns a new Collector.
func NewCollector(client *v3.Client) *Collector {
	return &Collector{
		metrics: map[string]Metric{
			"operation_phase": {
				collectors:  []prometheus.Collector{operationPhase, operationPhaseTimestampSeconds},
				isAvailable: alwaysAvailable,
			},
			"leader": {
				collectors:  []prometheus.Collector{leader},
				isAvailable: alwaysAvailable,
			},
			"sabakan_integration": {
				collectors:  []prometheus.Collector{sabakanIntegrationSuccessful, sabakanIntegrationTimestampSeconds, sabakanWorkers, sabakanUnusedMachines},
				isAvailable: isSabakanIntegrationMetricsAvailable,
			},
		},
		storage: &Storage{client},
	}
}

// GetHandler returns http.Handler for prometheus metrics.
func GetHandler(collector *Collector) http.Handler {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	handler := promhttp.HandlerFor(registry,
		promhttp.HandlerOpts{
			ErrorLog:      logger{},
			ErrorHandling: promhttp.ContinueOnError,
		})

	return handler
}

// Describe implements Collector.Describe().
func (c Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.metrics {
		for _, col := range metric.collectors {
			col.Describe(ch)
		}
	}
}

// Collect implements Collector.Collect().
func (c Collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), scrapeTimeout)
	defer cancel()

	var wg sync.WaitGroup
	for key, metric := range c.metrics {
		wg.Add(1)
		go func(key string, metric Metric) {
			defer wg.Done()
			isAvailable, err := metric.isAvailable(ctx, c.storage)
			if err != nil {
				log.Warn("unable to decide whether metrics are available", map[string]interface{}{
					"name":      key,
					log.FnError: err,
				})
				return
			}
			if !isAvailable {
				return
			}

			for _, col := range metric.collectors {
				col.Collect(ch)
			}
		}(key, metric)
	}
	wg.Wait()
}

func alwaysAvailable(_ context.Context, _ *Storage) (bool, error) {
	return true, nil
}

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

// UpdateOperationPhaseMetrics updates "operation_phase" and its timestamp.
func UpdateOperationPhaseMetrics(phase OperationPhase, ts time.Time) {
	for _, labelPhase := range AllOperationPhases {
		if labelPhase == phase {
			operationPhase.WithLabelValues(string(labelPhase)).Set(1)
		} else {
			operationPhase.WithLabelValues(string(labelPhase)).Set(0)
		}
	}
	operationPhaseTimestampSeconds.Set(float64(ts.Unix()))
}

var leader = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "leader",
		Help:      "1 if this server is the leader of CKE.",
	},
)

// UpdateLeaderMetrics updates "leader".
func UpdateLeaderMetrics(isLeader bool) {
	if isLeader {
		leader.Set(1)
	} else {
		leader.Set(0)
	}
}

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

// UpdateSabakanIntegrationMetrics updates Sabakan integration metrics.
func UpdateSabakanIntegrationMetrics(isSuccessful bool, workersByRole map[string]int, unusedMachines int, ts time.Time) {
	sabakanIntegrationTimestampSeconds.Set(float64(ts.Unix()))
	if !isSuccessful {
		sabakanIntegrationSuccessful.Set(0)
		return
	}

	sabakanIntegrationSuccessful.Set(1)
	for role, num := range workersByRole {
		sabakanWorkers.WithLabelValues(role).Set(float64(num))
	}
	sabakanUnusedMachines.Set(float64(unusedMachines))
}

func isSabakanIntegrationMetricsAvailable(ctx context.Context, st *Storage) (bool, error) {
	disabled, err := st.IsSabakanDisabled(ctx)
	if err != nil {
		return false, err
	}
	return !disabled, nil
}
