package cke

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
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

type updateBootLeaderTestCase struct {
	name     string
	input    func(*concurrency.Election) error
	expected float64
}

func TestMetrics(t *testing.T) {
	t.Run("UpdateOperationRunning", testUpdateOperationRunning)
	t.Run("UpdateBootLeader", testUpdateBootLeader)
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

func testUpdateBootLeader(t *testing.T) {
	testCases := []updateBootLeaderTestCase{
		{
			name:     "leader does not exist",
			input:    func(_ *concurrency.Election) error { return nil },
			expected: 0,
		},
		{
			name: "I am the leader",
			input: func(e *concurrency.Election) error {
				hostname, err := os.Hostname()
				if err != nil {
					return err
				}
				return e.Campaign(context.Background(), hostname)
			},
			expected: 1,
		},
		{
			name: "I am no longer the leader",
			input: func(e *concurrency.Election) error {
				hostname, err := os.Hostname()
				if err != nil {
					return err
				}
				err = e.Campaign(context.Background(), hostname)
				if err != nil {
					return err
				}
				return e.Resign(context.Background())
			},
			expected: 0,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			defer ctx.Done()

			client := newEtcdClient(t)
			defer client.Close()

			session, err := concurrency.NewSession(client, concurrency.WithTTL(int(time.Second)))
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()

			err = tt.input(concurrency.NewElection(session, KeyLeader))
			if err != nil {
				t.Fatal(err)
			}

			updater := NewUpdater(10*time.Millisecond, client)
			updater.updateBootLeader(ctx)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/metrics", nil)
			GetHandler().ServeHTTP(w, req)

			metricsFamily, err := parseMetrics(w.Result())
			if err != nil {
				t.Fatal(err)
			}

			found := false
			for _, mf := range metricsFamily {
				if *mf.Name != "cke_boot_leader" {
					continue
				}
				found = true
				for _, m := range mf.Metric {
					if *m.Gauge.Value != tt.expected {
						t.Errorf("value for cke_boot_leader is wrong.  expected: %f, actual: %f", tt.expected, *m.Gauge.Value)
					}
				}
			}
			if !found {
				t.Errorf("metrics cke_boot_leader was not found")
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
