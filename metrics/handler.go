package metrics

import (
	"fmt"
	"net/http"

	"github.com/cybozu-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type logger struct{}

func (l logger) Println(v ...interface{}) {
	log.Error(fmt.Sprint(v...), nil)
}

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
