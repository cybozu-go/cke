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

type labeledValue struct {
	labels map[string]string
	value  float64
}

type updateOperationPhaseTestCase struct {
	name     string
	input    OperationPhase
	expected []labeledValue
}

type updateLeaderTestCase struct {
	name     string
	input    bool
	expected float64
}

type sabakanInput struct {
	enabled        bool
	successful     bool
	workersByRole  map[string]int
	unusedMachines int
}

type sabakanExpected struct {
	returned       bool
	successful     float64
	workersByRole  []labeledValue
	unusedMachines float64
}

type updateSabakanIntegrationTestCase struct {
	name     string
	input    sabakanInput
	expected sabakanExpected
}

func TestMetrics(t *testing.T) {
	t.Run("UpdateOperationPhase", testUpdateOperationPhase)
	t.Run("UpdateLeader", testUpdateLeader)
	t.Run("UpdateSabakanIntegration", testUpdateSabakanIntegration)
}

func testUpdateOperationPhase(t *testing.T) {
	testCases := []updateOperationPhaseTestCase{
		{
			name:  "completed",
			input: PhaseCompleted,
			expected: []labeledValue{
				{
					labels: map[string]string{"phase": string(PhaseCompleted)},
					value:  1,
				},
				{
					labels: map[string]string{"phase": string(PhaseUpgrade)},
					value:  0,
				},
				{
					labels: map[string]string{"phase": string(PhaseRivers)},
					value:  0,
				},
			},
		},
		{
			name:  "upgrading",
			input: PhaseUpgrade,
			expected: []labeledValue{
				{
					labels: map[string]string{"phase": string(PhaseCompleted)},
					value:  0,
				},
				{
					labels: map[string]string{"phase": string(PhaseUpgrade)},
					value:  1,
				},
				{
					labels: map[string]string{"phase": string(PhaseRivers)},
					value:  0,
				},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			defer ctx.Done()

			client := newEtcdClient(t)
			defer client.Close()

			collector := NewCollector(client)
			handler := GetHandler(collector)

			UpdateOperationPhaseMetrics(tt.input, time.Now().UTC())

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/metrics", nil)
			handler.ServeHTTP(w, req)

			metricsFamily, err := parseMetrics(w.Result())
			if err != nil {
				t.Fatal(err)
			}

			metricsFamilyFound := false
			for _, mf := range metricsFamily {
				if *mf.Name != "cke_operation_phase" {
					continue
				}
				metricsFamilyFound = true
				for _, exp := range tt.expected {
					metricsFound := false
					for _, m := range mf.Metric {
						labels := labelToMap(m.Label)
						if !hasLabels(labels, exp.labels) {
							continue
						}
						metricsFound = true
						if *m.Gauge.Value != exp.value {
							t.Errorf("value for cke_operation_phase with labels of %v is wrong.  expected: %f, actual: %f", exp.labels, exp.value, *m.Gauge.Value)
						}
					}
					if !metricsFound {
						t.Errorf("metrics cke_operation_phase with labels of %v was not found", exp.labels)
					}
				}
			}
			if !metricsFamilyFound {
				t.Errorf("metrics cke_operation_phase was not found")
			}
		})
	}
}

func testUpdateLeader(t *testing.T) {
	testCases := []updateLeaderTestCase{
		{
			name:     "I am the leader",
			input:    true,
			expected: 1,
		},
		{
			name:     "I am not the leader",
			input:    false,
			expected: 0,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			defer ctx.Done()

			client := newEtcdClient(t)
			defer client.Close()

			UpdateLeaderMetrics(tt.input)

			collector := NewCollector(client)
			handler := GetHandler(collector)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/metrics", nil)
			handler.ServeHTTP(w, req)

			metricsFamily, err := parseMetrics(w.Result())
			if err != nil {
				t.Fatal(err)
			}

			found := false
			for _, mf := range metricsFamily {
				if *mf.Name != "cke_leader" {
					continue
				}
				found = true
				for _, m := range mf.Metric {
					if *m.Gauge.Value != tt.expected {
						t.Errorf("value for cke_leader is wrong.  expected: %f, actual: %f", tt.expected, *m.Gauge.Value)
					}
				}
			}
			if !found {
				t.Errorf("metrics cke_leader was not found")
			}
		})
	}
}

func testUpdateSabakanIntegration(t *testing.T) {
	testCases := []updateSabakanIntegrationTestCase{
		{
			name: "disabled",
			input: sabakanInput{
				enabled: false,
			},
			expected: sabakanExpected{
				returned: false,
			},
		},
		{
			name: "failed",
			input: sabakanInput{
				enabled:    true,
				successful: false,
			},
			expected: sabakanExpected{
				returned:   true,
				successful: 0,
			},
		},
		{
			name: "succeeded",
			input: sabakanInput{
				enabled:    true,
				successful: true,
				workersByRole: map[string]int{
					"cs": 17,
					"ss": 29,
				},
				unusedMachines: 42,
			},
			expected: sabakanExpected{
				returned:   true,
				successful: 1,
				workersByRole: []labeledValue{
					{
						labels: map[string]string{"role": "cs"},
						value:  17,
					},
					{
						labels: map[string]string{"role": "ss"},
						value:  29,
					},
				},
				unusedMachines: 42,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			defer ctx.Done()

			client := newEtcdClient(t)
			defer client.Close()

			collector := NewCollector(client)
			handler := GetHandler(collector)

			storage := Storage{client}
			err := storage.EnableSabakan(ctx, tt.input.enabled)
			if err != nil {
				t.Fatal(err)
			}
			UpdateSabakanIntegrationMetrics(tt.input.successful, tt.input.workersByRole, tt.input.unusedMachines, time.Now().UTC())

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/metrics", nil)
			handler.ServeHTTP(w, req)

			metricsFamily, err := parseMetrics(w.Result())
			if err != nil {
				t.Fatal(err)
			}

			successfulFound := false
			workersFound := false
			unusedMachinesFound := false
			for _, mf := range metricsFamily {
				if !tt.expected.returned {
					switch *mf.Name {
					case "cke_sabakan_integration_successful",
						"cke_sabakan_integration_timestamp_seconds",
						"cke_sabakan_workers",
						"cke_sabakan_unused_machines":
						t.Errorf("metrics %q should not be returned", *mf.Name)
					}
					continue
				}
				switch *mf.Name {
				case "cke_sabakan_integration_successful":
					successfulFound = true
					for _, m := range mf.Metric {
						if *m.Gauge.Value != tt.expected.successful {
							t.Errorf("value for cke_sabakan_integration_successful is wrong.  expected: %f, actual: %f", tt.expected.successful, *m.Gauge.Value)
						}
					}
				case "cke_sabakan_workers":
					workersFound = true
					for _, exp := range tt.expected.workersByRole {
						metricsFound := false
						for _, m := range mf.Metric {
							labels := labelToMap(m.Label)
							if !hasLabels(labels, exp.labels) {
								continue
							}
							metricsFound = true
							if *m.Gauge.Value != exp.value {
								t.Errorf("value for cke_sabakan_workers with labels of %v is wrong.  expected: %f, actual: %f", exp.labels, exp.value, *m.Gauge.Value)
							}
						}
						if !metricsFound {
							t.Errorf("metrics cke_sabakan_workers with labels of %v was not found", exp.labels)
						}
					}
				case "cke_sabakan_unused_machines":
					unusedMachinesFound = true
					for _, m := range mf.Metric {
						if *m.Gauge.Value != tt.expected.unusedMachines {
							t.Errorf("value for cke_sabakan_unused_machines is wrong.  expected: %f, actual: %f", tt.expected.unusedMachines, *m.Gauge.Value)
						}
					}
				}
			}
			if tt.expected.returned {
				if !successfulFound {
					t.Errorf("metrics cke_sabakan_integration_successful was not found")
				}
				if tt.expected.successful == 1 {
					if !workersFound {
						t.Errorf("metrics cke_sabakan_workers was not found")
					}
					if !unusedMachinesFound {
						t.Errorf("metrics cke_sabakan_unused_machines was not found")
					}
				}
			}
		})
	}
}

func labelToMap(labelPair []*dto.LabelPair) map[string]string {
	res := make(map[string]string)
	for _, l := range labelPair {
		res[*l.Name] = *l.Value
	}
	return res
}

func hasLabels(lm map[string]string, expectedLabels map[string]string) bool {
	for ek, ev := range expectedLabels {
		found := false
		for k, v := range lm {
			if k == ek && v == ev {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
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
