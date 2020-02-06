package cke

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type expectedMachineStatus struct {
	status string
	labels map[string]string
}

type updateOperationRunningTestCase struct {
	name     string
	input    ServerStatus
	expected float64
}

func TestMetrics(t *testing.T) {
	t.Run("UpdateOperationRunning", testUpdateOperationRunning)
}

func testUpdateOperationRunning(t *testing.T) {
	testCases := []updateOperationRunningTestCase{
		{
			name:     "completed",
			input:    ServerStatus{Phase: PhaseCompleted},
			expected: 0,
		},
		{
			name:     "running",
			input:    ServerStatus{Phase: PhaseK8sStart},
			expected: 1,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			defer ctx.Done()

			client := newEtcdClient(t)
			defer client.Close()

			resp, err := client.Grant(ctx, 10)
			if err != nil {
				t.Fatal(err)
			}

			storage := Storage{client}
			storage.SetStatus(ctx, resp.ID, &tt.input)

			updater := NewUpdater(10*time.Millisecond, client)
			updater.updateOperationRunning(ctx)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/metrics", nil)
			GetHandler().ServeHTTP(w, req)

			metricsFamily, err := parseMetrics(w.Result())
			if err != nil {
				t.Fatal(err)
			}

			found := false
			for _, mf := range metricsFamily {
				if *mf.Name != "cke_operation_running" {
					continue
				}
				found = true
				for _, m := range mf.Metric {
					if *m.Gauge.Value != tt.expected {
						t.Errorf("value for cke_operation_running is wrong.  expected: %f, actual: %f", tt.expected, *m.Gauge.Value)
					}
				}
			}
			if !found {
				t.Errorf("metrics cke_operation_running was not found")
			}
		})
	}
}

func parseMetrics(resp *http.Response) ([]*dto.MetricFamily, error) {
	var parser expfmt.TextParser
	parsed, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, err
	}
	var result []*dto.MetricFamily
	for _, mf := range parsed {
		result = append(result, mf)
	}
	return result, nil
}
