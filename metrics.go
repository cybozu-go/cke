package cke

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	v3 "github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/log"
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
	[]string{"address", "rack", "role", "control_plane"},
)

// GetHandler return http.Handler for prometheus metrics
func GetHandler() http.Handler {
	registry := prometheus.NewRegistry()
	registerMetrics(registry)

	handler := promhttp.HandlerFor(registry,
		promhttp.HandlerOpts{
			ErrorLog:      logger{},
			ErrorHandling: promhttp.ContinueOnError,
		})

	return handler
}

func registerMetrics(registry *prometheus.Registry) {
	registry.MustRegister(OperationRunning)
	registry.MustRegister(BootLeader)
	registry.MustRegister(NodeInfo)
}

// Updater updates Prometheus metrics periodically
type Updater struct {
	interval time.Duration
	storage  *Storage
}

// NewUpdater is the constructor for Updater
func NewUpdater(interval time.Duration, client *v3.Client) *Updater {
	storage := Storage{
		Client: client,
	}
	return &Updater{interval, &storage}
}

// UpdateLoop is the func to update all metrics continuously
func (u *Updater) UpdateLoop(ctx context.Context) error {
	ticker := time.NewTicker(u.interval)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := u.UpdateAllMetrics(ctx)
			if err != nil {
				log.Warn("failed to update metrics", map[string]interface{}{
					log.FnError: err.Error(),
				})
			}
		}
	}
}

// UpdateAllMetrics is the func to update all metrics once
func (u *Updater) UpdateAllMetrics(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	tasks := map[string]func(ctx context.Context) error{
		"updateOperationRunning": u.updateOperationRunning,
		"updateBootLeader":       u.updateBootLeader,
		"updateNodeInfo":         u.updateNodeInfo,
	}
	for key, task := range tasks {
		key, task := key, task
		g.Go(func() error {
			err := task(ctx)
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

func (u *Updater) updateOperationRunning(ctx context.Context) error {
	st, err := u.storage.GetStatus(ctx)
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

func (u *Updater) updateBootLeader(ctx context.Context) error {
	leader, err := u.storage.GetLeaderHostname(ctx)
	if err != nil {
		if err == ErrLeaderNotExist {
			BootLeader.Set(0)
		}
		return err
	}

	myName, err := os.Hostname()
	if err != nil {
		return err
	}

	if leader == myName {
		BootLeader.Set(1)
	} else {
		BootLeader.Set(0)
	}
	return nil
}

func (u *Updater) updateNodeInfo(ctx context.Context) error {
	cluster, err := u.storage.GetCluster(ctx)
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
