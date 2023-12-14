package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
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

type updateRebootQueueEntriesTestCase struct {
	name            string
	enabled         bool
	running         bool
	input           []*cke.RebootQueueEntry
	expectedEnabled float64
	expectedRunning float64
	expectedEntries float64
}

type updateRebootQueueItemsTestCase struct {
	name     string
	input    []*cke.RebootQueueEntry
	expected map[string]float64
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
	t.Run("UpdateRebootQueueEntries", testUpdateRebootQueueEntries)
	t.Run("UpdateRebootQueueItems", testUpdateRebootQueueItems)
	t.Run("UpdateNodeRebootStatus", testUpdateNodeRebootStatus)
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

func testUpdateRebootQueueEntries(t *testing.T) {
	testCases := []updateRebootQueueEntriesTestCase{
		{
			name:            "zero",
			enabled:         true,
			running:         false,
			input:           nil,
			expectedEnabled: 1,
			expectedRunning: 0,
			expectedEntries: 0,
		},
		{
			name:    "one",
			enabled: true,
			running: true,
			input: []*cke.RebootQueueEntry{
				{Status: cke.RebootStatusQueued},
			},
			expectedEnabled: 1,
			expectedRunning: 1,
			expectedEntries: 1,
		},
		{
			name:    "two",
			enabled: true,
			running: true,
			input: []*cke.RebootQueueEntry{
				{Status: cke.RebootStatusQueued},
				{Status: cke.RebootStatusRebooting},
			},
			expectedEnabled: 1,
			expectedRunning: 1,
			expectedEntries: 2,
		},
		{
			name:    "two-stopping",
			enabled: false,
			running: true,
			input: []*cke.RebootQueueEntry{
				{Status: cke.RebootStatusQueued},
				{Status: cke.RebootStatusRebooting},
			},
			expectedEnabled: 0,
			expectedRunning: 1,
			expectedEntries: 2,
		},
		{
			name:    "two-disabled",
			enabled: false,
			running: false,
			input: []*cke.RebootQueueEntry{
				{Status: cke.RebootStatusQueued},
				{Status: cke.RebootStatusRebooting},
			},
			expectedEnabled: 0,
			expectedRunning: 0,
			expectedEntries: 2,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			defer ctx.Done()

			collector, storage := newTestCollector()
			storage.enableRebootQueue(tt.enabled)
			storage.setRebootQueueRunning(tt.running)
			storage.setRebootsEntries(tt.input)
			handler := GetHandler(collector)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/metrics", nil)
			handler.ServeHTTP(w, req)

			metricsFamily, err := parseMetrics(w.Result())
			if err != nil {
				t.Fatal(err)
			}

			metricsEnabledFound := false
			metricsRunningFound := false
			metricsEntriesFound := false
			for _, mf := range metricsFamily {
				switch *mf.Name {
				case "cke_reboot_queue_enabled":
					for _, m := range mf.Metric {
						metricsEnabledFound = true
						if *m.Gauge.Value != tt.expectedEnabled {
							t.Errorf("value for cke_reboot_queue_enabled is wrong.  expected: %f, actual: %f", tt.expectedEnabled, *m.Gauge.Value)
						}
					}
				case "cke_reboot_queue_running":
					for _, m := range mf.Metric {
						metricsRunningFound = true
						if *m.Gauge.Value != tt.expectedRunning {
							t.Errorf("value for cke_reboot_queue_running is wrong.  expected: %f, actual: %f", tt.expectedRunning, *m.Gauge.Value)
						}
					}
				case "cke_reboot_queue_entries":
					for _, m := range mf.Metric {
						metricsEntriesFound = true
						if *m.Gauge.Value != tt.expectedEntries {
							t.Errorf("value for cke_reboot_queue_entries is wrong.  expected: %f, actual: %f", tt.expectedEntries, *m.Gauge.Value)
						}
					}
				}
			}
			if !metricsEnabledFound {
				t.Errorf("metrics reboot_queue_enabled was not found")
			}
			if !metricsRunningFound {
				t.Errorf("metrics reboot_queue_running was not found")
			}
			if !metricsEntriesFound {
				t.Errorf("metrics reboot_queue_entries was not found")
			}
		})
	}
}

func testUpdateRebootQueueItems(t *testing.T) {
	// UpdateRebootQueueItems does not take care of nonexistent keys. Those must be cared by CountRebootQueueEntries.
	testCases := []updateRebootQueueItemsTestCase{
		{
			name: "zero",
			input: []*cke.RebootQueueEntry{
				{Status: cke.RebootStatusQueued},
				{Status: cke.RebootStatusDraining},
				{Status: cke.RebootStatusDraining},
				{Status: cke.RebootStatusRebooting},
				{Status: cke.RebootStatusRebooting},
				{Status: cke.RebootStatusRebooting},
			},
			expected: map[string]float64{
				"queued":    1.0,
				"draining":  2.0,
				"rebooting": 3.0,
				"cancelled": 0.0,
			},
		},
		{
			name: "one",
			input: []*cke.RebootQueueEntry{
				{Status: cke.RebootStatusQueued},
				{Status: cke.RebootStatusQueued},
				{Status: cke.RebootStatusQueued},
				{Status: cke.RebootStatusQueued},
				{Status: cke.RebootStatusDraining},
				{Status: cke.RebootStatusDraining},
				{Status: cke.RebootStatusDraining},
				{Status: cke.RebootStatusDraining},
				{Status: cke.RebootStatusDraining},
				{Status: cke.RebootStatusCancelled},
				{Status: cke.RebootStatusCancelled},
				{Status: cke.RebootStatusCancelled},
				{Status: cke.RebootStatusCancelled},
				{Status: cke.RebootStatusCancelled},
				{Status: cke.RebootStatusCancelled},
			},
			expected: map[string]float64{
				"queued":    4.0,
				"draining":  5.0,
				"rebooting": 0.0,
				"cancelled": 6.0,
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			defer ctx.Done()

			collector, storage := newTestCollector()
			storage.setRebootsEntries(tt.input)
			handler := GetHandler(collector)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/metrics", nil)
			handler.ServeHTTP(w, req)

			metricsFamily, err := parseMetrics(w.Result())
			if err != nil {
				t.Fatal(err)
			}

			metricsFound := map[string]float64{}
			for _, mf := range metricsFamily {
				if *mf.Name != "cke_reboot_queue_items" {
					continue
				}
				for _, m := range mf.Metric {
					if len(m.Label) != 1 {
						t.Errorf("value for cke_reboot_queue_items should have exactly one label. actual: %d", len(m.Label))
					}
					metricsFound[*m.GetLabel()[0].Value] = *m.Gauge.Value
				}
			}
			if !cmp.Equal(metricsFound, tt.expected) {
				t.Errorf("value for cke_reboot_queue_items is wrong.  expected: %v, actual: %v", tt.expected, metricsFound)
			}
		})
	}
}

func testUpdateNodeRebootStatus(t *testing.T) {
	inputCluster := &cke.Cluster{
		Nodes: []*cke.Node{
			{
				Address:  "192.168.1.11",
				Hostname: "node1",
			},
			{
				Address:  "192.168.1.12",
				Hostname: "node2",
			},
		},
	}
	inputEntries := []*cke.RebootQueueEntry{
		{
			Node:   "192.168.1.11",
			Status: cke.RebootStatusRebooting,
		},
	}
	expected := map[string]map[string]bool{
		"node1": {
			"queued":    false,
			"draining":  false,
			"rebooting": true,
			"cancelled": false,
		},
		"node2": {
			"queued":    false,
			"draining":  false,
			"rebooting": false,
			"cancelled": false,
		},
	}

	ctx := context.Background()
	defer ctx.Done()

	collector, storage := newTestCollector()
	storage.setCluster(inputCluster)
	storage.setRebootsEntries(inputEntries)
	handler := GetHandler(collector)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	handler.ServeHTTP(w, req)

	metricsFamily, err := parseMetrics(w.Result())
	if err != nil {
		t.Fatal(err)
	}

	actual := make(map[string]map[string]bool)
	for _, mf := range metricsFamily {
		if *mf.Name != "cke_node_reboot_status" {
			continue
		}
		for _, m := range mf.Metric {
			labels := labelToMap(m.Label)
			node := labels["node"]
			status := labels["status"]
			if _, ok := actual[node]; !ok {
				actual[node] = make(map[string]bool)
			}
			actual[node][status] = *m.Gauge.Value != 0
		}
	}

	if !cmp.Equal(actual, expected) {
		t.Errorf("unexpected map was build from cke_node_reboot_status.  expected: %v, actual: %v", expected, actual)
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

func newTestCollector() (prometheus.Collector, *testStorage) {
	s := &testStorage{
		cluster: new(cke.Cluster),
	}
	c := NewCollector(s)
	return c, s
}

type testStorage struct {
	sabakanEnabled     bool
	rebootQueueEnabled bool
	rebootQueueRunning bool
	rebootEntries      []*cke.RebootQueueEntry
	cluster            *cke.Cluster
}

func (s *testStorage) enableSabakan(flag bool) {
	s.sabakanEnabled = flag
}

func (s *testStorage) IsSabakanDisabled(_ context.Context) (bool, error) {
	return !s.sabakanEnabled, nil
}

func (s *testStorage) IsRebootQueueDisabled(_ context.Context) (bool, error) {
	return !s.rebootQueueEnabled, nil
}

func (s *testStorage) enableRebootQueue(flag bool) {
	s.rebootQueueEnabled = flag
}

func (s *testStorage) IsRebootQueueRunning(_ context.Context) (bool, error) {
	return s.rebootQueueRunning, nil
}

func (s *testStorage) setRebootQueueRunning(flag bool) {
	s.rebootQueueRunning = flag
}

func (s *testStorage) setRebootsEntries(entries []*cke.RebootQueueEntry) {
	s.rebootEntries = entries
}

func (s *testStorage) GetRebootsEntries(ctx context.Context) ([]*cke.RebootQueueEntry, error) {
	return s.rebootEntries, nil
}

func (s *testStorage) setCluster(cluster *cke.Cluster) {
	s.cluster = cluster
}

func (s *testStorage) GetCluster(ctx context.Context) (*cke.Cluster, error) {
	return s.cluster, nil
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
