package sabakan

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cybozu-go/cke/sabakan/mock"
	"github.com/cybozu-go/well"
	"github.com/google/go-cmp/cmp"
)

func testMachine(t *testing.T, m Machine) {
	if m.Spec.Serial != "m1" {
		return
	}

	if len(m.Spec.Labels) != 2 {
		t.Error("wrong number of labels")
	} else {
		label := m.Spec.Labels[0]
		if label.Name != "label1" {
			t.Error("wrong label name:", label.Name)
		}
		if label.Value != "value1" {
			t.Error("wrong label value:", label.Value)
		}
	}

	if m.Spec.Rack != 1 {
		t.Error("wrong rack number:", m.Spec.Rack)
	}

	if m.Spec.IndexInRack != 3 {
		t.Error("wrong index in rack:", m.Spec.IndexInRack)
	}

	if m.Spec.Role != "boot" {
		t.Error("wrong role:", m.Spec.Role)
	}

	if !cmp.Equal(m.Spec.IPv4, []string{"10.0.1.3"}) {
		t.Error("wrong addresses:", m.Spec.IPv4)
	}

	if !m.Spec.RegisterDate.Equal(time.Date(2018, 12, 2, 0, 0, 0, 0, time.UTC)) {
		t.Error("wrong register date:", m.Spec.RetireDate.Format(time.RFC3339Nano))
	}

	if !m.Spec.RetireDate.Equal(time.Date(2019, 1, 2, 0, 0, 0, 0, time.UTC)) {
		t.Error("wrong retire date:", m.Spec.RetireDate.Format(time.RFC3339Nano))
	}

	if m.Status.State != StateHealthy {
		t.Error("wrong machine state:", m.Status.State)
	}

	if int(m.Status.Duration) != 100 {
		t.Error("wrong duration:", m.Status.Duration)
	}
}

func matchMachines(t *testing.T, machines []Machine, serials []string) {
	ms := make([]string, len(machines))
	for i, m := range machines {
		if !m.Status.State.IsValid() {
			t.Errorf("invalid machine state: %v, serial=%s", m.Status.State, m.Spec.Serial)
			return
		}
		testMachine(t, m)
		ms[i] = m.Spec.Serial
	}
	if !cmp.Equal(ms, serials) {
		t.Errorf("not match: expected %#v, actual %#v", serials, ms)
	}
}

func parseVars(s string) *QueryVariables {
	vars := new(QueryVariables)
	err := json.Unmarshal([]byte(s), vars)
	if err != nil {
		panic(err)
	}
	return vars
}

func TestQuery(t *testing.T) {
	s := mock.Server()
	hc := &well.HTTPClient{Client: s.Client()}
	ctx := context.Background()

	testCases := []struct {
		name          string
		vars          *QueryVariables
		expectError   bool
		expectSerials []string
	}{
		{
			"nil",
			nil,
			false,
			[]string{"m1", "m3"},
		},
		{
			"notHaving",
			parseVars(`{"notHaving": {"states": ["RETIRED"]}}`),
			false,
			[]string{"m1", "m3"},
		},
		{
			"empty",
			parseVars(`{}`),
			false,
			[]string{"m1", "m2", "m3"},
		},
		{
			"labels",
			parseVars(`{"having": {"labels": [{"name": "foo", "value": "bar"}]}}`),
			false,
			[]string{"m3"},
		},
		{
			"racks",
			parseVars(`{"having": {"racks": [1]}}`),
			false,
			[]string{"m1"},
		},
		{
			"roles",
			parseVars(`{"having": {"roles": ["worker"]}}`),
			false,
			[]string{"m2"},
		},
		{
			"badState",
			parseVars(`{"having": {"states": ["bad"]}}`),
			true,
			nil,
		},
		{
			"states",
			parseVars(`{"having": {"states": ["UNINITIALIZED"]}}`),
			false,
			[]string{"m3"},
		},
		{
			"minDays",
			parseVars(`{"having": {"minDaysBeforeRetire": 90}}`),
			false,
			[]string{"m2"},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			machines, err := doQuery(ctx, s.URL, c.vars, hc)
			if err != nil {
				if !c.expectError {
					t.Error(err)
				}
				return
			}

			matchMachines(t, machines, c.expectSerials)
		})
	}
}
