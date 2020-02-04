package metrics

import "github.com/prometheus/client_golang/prometheus"

// AssetsBytesTotal returns the total bytes of assets
var AssetsBytesTotal = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "assets_bytes_total",
		Help:      "The total bytes of assets.",
	},
)
