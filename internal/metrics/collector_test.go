package metrics

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStandardMetrics(t *testing.T) {
	collector, _ := newTestCollector()
	handler := GetHandler(collector)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	handler.ServeHTTP(w, req)

	metricsFamily, err := parseMetrics(w.Result())
	if err != nil {
		t.Fatal(err)
	}

	foundStandardMetrics := false
	foundRuntimeMetrics := false
	for _, mf := range metricsFamily {
		fmt.Println(*mf.Name)
		if strings.HasPrefix(*mf.Name, "process_") {
			foundStandardMetrics = true
		}
		if strings.HasPrefix(*mf.Name, "go_") {
			foundRuntimeMetrics = true
		}
	}
	if !foundStandardMetrics {
		t.Errorf("standard metrics was not found")
	}
	if !foundRuntimeMetrics {
		t.Errorf("runtime metrics was not found")
	}
}
