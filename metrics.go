package cke

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	v3 "github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

type logger struct{}

func (l logger) Println(v ...interface{}) {
	log.Error(fmt.Sprint(v...), nil)
}

const (
	namespace = "cke"
)

// Metric represents collector and updater of metric.
type Metric struct {
	collector prometheus.Collector
	updater   func(context.Context, *Storage) error
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
			"operation_running": {
				collector: OperationRunning,
				updater:   updateOperationRunning,
			},
			"leader": {
				collector: Leader,
				updater:   updateLeader,
			},
			"node_info": {
				collector: NodeInfo,
				updater:   updateNodeInfo,
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
		metric.collector.Describe(ch)
	}
}

// Collect implements Collector.Collect().
func (c Collector) Collect(ch chan<- prometheus.Metric) {
	env := well.NewEnvironment(context.Background())
	env.Go(c.updateAllMetrics)
	env.Stop()
	err := env.Wait()
	if err != nil {
		log.Warn("some metrics failed to be updated", map[string]interface{}{
			log.FnError: err.Error(),
		})
	}

	for _, metric := range c.metrics {
		metric.collector.Collect(ch)
	}
}

// UpdateAllMetrics is the func to update all metrics once
func (c Collector) updateAllMetrics(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for key, metric := range c.metrics {
		key, metric := key, metric
		g.Go(func() error {
			err := metric.updater(ctx, c.storage)
			if err != nil {
				log.Warn("unable to update metrics", map[string]interface{}{
					"funcname":  key,
					log.FnError: err,
				})
			}
			return err
		})
	}
	return g.Wait()
}

// OperationRunning returns True (== 1) if any operations are running.
var OperationRunning = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "operation_running",
		Help:      "1 if any operations are running.",
	},
)

// Leader returns True (== 1) if the boot server is the leader of CKE.
var Leader = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "leader",
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
	[]string{"address", "rack", "role", "control_plane"},
)

func updateOperationRunning(ctx context.Context, storage *Storage) error {
	st, err := storage.GetStatus(ctx)
	if err != nil {
		return err
	}
	if st.Phase == PhaseCompleted {
		OperationRunning.Set(0)
	} else {
		OperationRunning.Set(1)
	}
	return nil
}

func updateLeader(ctx context.Context, storage *Storage) error {
	leader, err := storage.GetLeaderHostname(ctx)
	if err != nil {
		if err == ErrLeaderNotExist {
			Leader.Set(0)
		}
		return err
	}

	myName, err := os.Hostname()
	if err != nil {
		return err
	}

	if leader == myName {
		Leader.Set(1)
	} else {
		Leader.Set(0)
	}
	return nil
}

func updateNodeInfo(ctx context.Context, storage *Storage) error {
	cluster, err := storage.GetCluster(ctx)
	if err != nil {
		return err
	}

	for _, node := range cluster.Nodes {
		rack := node.Labels["cke.cybozu.com/rack"]
		role := node.Labels["cke.cybozu.com/role"]
		NodeInfo.WithLabelValues(node.Address, rack, role, strconv.FormatBool(node.ControlPlane)).Set(1)
		NodeInfo.WithLabelValues(node.Address, rack, role, strconv.FormatBool(!node.ControlPlane)).Set(0)
	}
	return nil
}
