package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cybozu-go/cke"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type labeledValue struct {
	labels map[string]string
	value  float64
}

type updateLeaderTestCase struct {
	name     string
	input    bool
	expected float64
}

type operationPhaseInput struct {
	isLeader bool
	phase    cke.OperationPhase
}

type operationPhaseExpected struct {
	returned bool
	values   []labeledValue
}

type updateOperationPhaseTestCase struct {
	name     string
	input    operationPhaseInput
	expected operationPhaseExpected
}

type sabakanInput struct {
	isLeader       bool
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

func TestMetricsUpdater(t *testing.T) {
	t.Run("UpdateLeader", testUpdateLeader)
	t.Run("UpdateOperationPhase", testUpdateOperationPhase)
	t.Run("UpdateSabakanIntegration", testUpdateSabakanIntegration)
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

			UpdateLeader(tt.input)

			collector, _ := newTestCollector()
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

func testUpdateOperationPhase(t *testing.T) {
	testCases := []updateOperationPhaseTestCase{
		{
			name: "not leader",
			input: operationPhaseInput{
				isLeader: false,
			},
			expected: operationPhaseExpected{
				returned: false,
			},
		},
		{
			name: "completed",
			input: operationPhaseInput{
				isLeader: true,
				phase:    cke.PhaseCompleted,
			},
			expected: operationPhaseExpected{
				returned: true,
				values: []labeledValue{
					{
						labels: map[string]string{"phase": string(cke.PhaseCompleted)},
						value:  1,
					},
					{
						labels: map[string]string{"phase": string(cke.PhaseUpgrade)},
						value:  0,
					},
					{
						labels: map[string]string{"phase": string(cke.PhaseRivers)},
						value:  0,
					},
				},
			},
		},
		{
			name: "upgrading",
			input: operationPhaseInput{
				isLeader: true,
				phase:    cke.PhaseUpgrade,
			},
			expected: operationPhaseExpected{
				returned: true,
				values: []labeledValue{
					{
						labels: map[string]string{"phase": string(cke.PhaseCompleted)},
						value:  0,
					},
					{
						labels: map[string]string{"phase": string(cke.PhaseUpgrade)},
						value:  1,
					},
					{
						labels: map[string]string{"phase": string(cke.PhaseRivers)},
						value:  0,
					},
				},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			defer ctx.Done()

			collector, _ := newTestCollector()
			handler := GetHandler(collector)

			UpdateLeader(tt.input.isLeader)
			UpdateOperationPhase(tt.input.phase, time.Now().UTC())

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
				for _, exp := range tt.expected.values {
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
					if tt.expected.returned && !metricsFound {
						t.Errorf("metrics cke_operation_phase with labels of %v was not found", exp.labels)
					}
				}
			}
			if tt.expected.returned && !metricsFamilyFound {
				t.Errorf("metrics cke_operation_phase was not found")
			}
			if !tt.expected.returned && metricsFamilyFound {
				t.Errorf("metrics cke_operation_phase should not be returned")
			}
		})
	}
}

func testUpdateSabakanIntegration(t *testing.T) {
	testCases := []updateSabakanIntegrationTestCase{
		{
			name: "not leader",
			input: sabakanInput{
				isLeader: false,
			},
			expected: sabakanExpected{
				returned: false,
			},
		},
		{
			name: "disabled",
			input: sabakanInput{
				isLeader: true,
				enabled:  false,
			},
			expected: sabakanExpected{
				returned: false,
			},
		},
		{
			name: "failed",
			input: sabakanInput{
				isLeader:   true,
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
				isLeader:   true,
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

			collector, storage := newTestCollector()
			handler := GetHandler(collector)

			UpdateLeader(tt.input.isLeader)
			storage.enableSabakan(tt.input.enabled)
			UpdateSabakanIntegration(tt.input.successful, tt.input.workersByRole, tt.input.unusedMachines, time.Now().UTC())

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

func newTestCollector() (*Collector, *testStorage) {
	c := NewCollector(nil)
	s := &testStorage{}
	c.storage = s
	return c, s
}

type testStorage struct {
	sabakanEnabled bool
}

func (s *testStorage) enableSabakan(flag bool) {
	s.sabakanEnabled = flag
}

func (s *testStorage) IsSabakanDisabled(_ context.Context) (bool, error) {
	return !s.sabakanEnabled, nil
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
