package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type logger struct{}

func (l logger) Println(v ...interface{}) {
	log.Error(fmt.Sprint(v...), nil)
}

const (
	scrapeTimeout = time.Second * 8
)

// collector is a metrics collector for CKE.
type collector struct {
	metrics map[string]metricGroup
	storage storage
}

// metricGroup represents collectors and availability of metric.
type metricGroup struct {
	collectors  []prometheus.Collector
	isAvailable func(context.Context, storage) (bool, error)
}

// storage is abstraction of cke.Storage.
// This abstraction is for mock test.
type storage interface {
	IsSabakanDisabled(context.Context) (bool, error)
	GetRebootsEntries(ctx context.Context) ([]*cke.RebootQueueEntry, error)
	GetCluster(ctx context.Context) (*cke.Cluster, error)
}

// NewCollector returns a new prometheus.Collector.
func NewCollector(storage storage) prometheus.Collector {

	return &collector{
		metrics: map[string]metricGroup{
			"leader": {
				collectors:  []prometheus.Collector{leader},
				isAvailable: alwaysAvailable,
			},
			"operation_phase": {
				collectors:  []prometheus.Collector{operationPhase, operationPhaseTimestampSeconds},
				isAvailable: isOperationPhaseAvailable,
			},
			"reboot": {
				collectors:  []prometheus.Collector{nodeMetricsCollector{storage}},
				isAvailable: isRebootAvailable,
			},
			"sabakan_integration": {
				collectors:  []prometheus.Collector{sabakanIntegrationSuccessful, sabakanIntegrationTimestampSeconds, sabakanWorkers, sabakanUnusedMachines},
				isAvailable: isSabakanIntegrationAvailable,
			},
		},
		storage: storage,
	}
}

// GetHandler returns http.Handler for prometheus metrics.
func GetHandler(collector prometheus.Collector) http.Handler {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	gathers := prometheus.Gatherers{registry, prometheus.DefaultGatherer}
	handler := promhttp.HandlerFor(gathers,
		promhttp.HandlerOpts{
			ErrorLog:      logger{},
			ErrorHandling: promhttp.ContinueOnError,
		})

	return handler
}

// Describe implements Collector.Describe().
func (c collector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.metrics {
		for _, col := range metric.collectors {
			col.Describe(ch)
		}
	}
}

// Collect implements Collector.Collect().
func (c collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), scrapeTimeout)
	defer cancel()

	var wg sync.WaitGroup
	for key, metric := range c.metrics {
		wg.Add(1)
		go func(key string, metric metricGroup) {
			defer wg.Done()
			available, err := metric.isAvailable(ctx, c.storage)
			if err != nil {
				log.Warn("unable to decide whether metrics are available", map[string]interface{}{
					"name":      key,
					log.FnError: err,
				})
				return
			}
			if !available {
				return
			}

			for _, col := range metric.collectors {
				col.Collect(ch)
			}
		}(key, metric)
	}
	wg.Wait()
}

// nodeMetricsCollector implements prometheus.Collector interface.
type nodeMetricsCollector struct {
	storage storage
}

var _ prometheus.Collector = &nodeMetricsCollector{}

func (c nodeMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- rebootQueueEntries
	ch <- rebootQueueItems
	ch <- nodeRebootStatus
}

func (c nodeMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rqEntries, err := c.storage.GetRebootsEntries(ctx)
	if err != nil {
		log.Error("failed to get reboots entries", map[string]interface{}{
			log.FnError: err,
		})
		return
	}

	cluster, err := c.storage.GetCluster(ctx)
	if err != nil {
		log.Error("failed to get reboots entries", map[string]interface{}{
			log.FnError: err,
		})
		return
	}
	itemCounts := cke.CountRebootQueueEntries(rqEntries)
	nodeStatus := cke.BuildNodeRebootStatus(cluster.Nodes, rqEntries)

	ch <- prometheus.MustNewConstMetric(
		rebootQueueEntries,
		prometheus.GaugeValue,
		float64(len(rqEntries)),
	)
	for status, count := range itemCounts {
		ch <- prometheus.MustNewConstMetric(
			rebootQueueItems,
			prometheus.GaugeValue,
			float64(count),
			status,
		)
	}
	for node, statuses := range nodeStatus {
		for status, matches := range statuses {
			value := float64(0)
			if matches {
				value = 1
			}
			ch <- prometheus.MustNewConstMetric(
				nodeRebootStatus,
				prometheus.GaugeValue,
				value,
				node,
				status,
			)
		}
	}
}
