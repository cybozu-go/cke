package sabakan

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func newTestMachine(rack int, role string, retireDate time.Time, state State) *Machine {
	m := &Machine{}
	m.Spec.Rack = rack
	m.Spec.Role = role
	m.Spec.RetireDate = retireDate
	m.Status.State = state
	return m
}

var (
	testBaseTS     = time.Date(2018, 12, 2, 0, 0, 0, 0, time.UTC)
	testFuture250  = time.Date(2019, 12, 2, 0, 0, 0, 0, time.UTC)
	testFuture500  = time.Date(2020, 6, 2, 0, 0, 0, 0, time.UTC)
	testFuture1000 = time.Date(2021, 12, 2, 0, 0, 0, 0, time.UTC)
	testPast250    = time.Date(2017, 12, 2, 0, 0, 0, 0, time.UTC)
	testPast500    = time.Date(2017, 6, 2, 0, 0, 0, 0, time.UTC)
	testPast1000   = time.Date(2015, 12, 2, 0, 0, 0, 0, time.UTC)
)

func TestScoreMachine(t *testing.T) {

	testCases := []struct {
		name      string
		machine   *Machine
		rackCount map[int]int
		expect    int
	}{
		{
			"Base",
			newTestMachine(0, "", testBaseTS, StateHealthy),
			nil,
			maxCountPerRack*100 + 10,
		},
		{
			"SameRack",
			newTestMachine(1, "", testBaseTS, StateHealthy),
			map[int]int{0: 2, 1: 3},
			(maxCountPerRack-3)*100 + 10,
		},
		{
			"SameRack2",
			newTestMachine(1, "", testBaseTS, StateHealthy),
			map[int]int{0: 2, 1: 13},
			(maxCountPerRack-13)*100 + 10,
		},
		{
			"Future250",
			newTestMachine(1, "", testFuture250, StateHealthy),
			nil,
			maxCountPerRack*100 + 10 + 1,
		},
		{
			"Future500",
			newTestMachine(1, "", testFuture500, StateHealthy),
			nil,
			maxCountPerRack*100 + 10 + 2,
		},
		{
			"Future1000",
			newTestMachine(1, "", testFuture1000, StateHealthy),
			nil,
			maxCountPerRack*100 + 10 + 3,
		},
		{
			"Past250",
			newTestMachine(1, "", testPast250, StateHealthy),
			nil,
			maxCountPerRack*100 + 10 - 1,
		},
		{
			"Past500",
			newTestMachine(1, "", testPast500, StateHealthy),
			nil,
			maxCountPerRack*100 + 10 - 2,
		},
		{
			"Past1000",
			newTestMachine(1, "", testPast1000, StateHealthy),
			nil,
			maxCountPerRack*100 + 10 - 3,
		},
		{
			"NotHealthy",
			newTestMachine(1, "", testBaseTS, StateRetiring),
			nil,
			maxCountPerRack * 100,
		},
		{
			"Compound",
			newTestMachine(2, "", testFuture500, StateRetiring),
			map[int]int{2: 9},
			(maxCountPerRack-9)*100 + 2,
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			score := scoreMachine(c.machine, c.rackCount, testBaseTS)
			if score != c.expect {
				t.Errorf("unexpected score: expected=%d, actual=%d", c.expect, score)
			}
		})
	}
}

func TestFilterMachine(t *testing.T) {
	testCases := []struct {
		name      string
		machine   *Machine
		role      string
		isHealthy bool
		expect    bool
	}{
		{
			"Base",
			newTestMachine(0, "cs", testBaseTS, StateHealthy),
			"cs",
			true,
			true,
		},
		{
			"FilteredByRole",
			newTestMachine(0, "cs", testBaseTS, StateHealthy),
			"ss",
			true,
			false,
		},
		{
			"DisableRoleFilter",
			newTestMachine(0, "cs", testBaseTS, StateHealthy),
			"",
			true,
			true,
		},
		{
			"FilteredByHealth",
			newTestMachine(0, "", testBaseTS, StateUnhealthy),
			"",
			true,
			false,
		},
		{
			"DisableHealthFilter",
			newTestMachine(0, "", testBaseTS, StateUnhealthy),
			"",
			false,
			true,
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			actual := filterMachine(c.machine, c.role, c.isHealthy)
			if actual != c.expect {
				t.Errorf("unexpected result: expected=%t, actual=%t", c.expect, actual)
			}
		})
	}
}

func TestFilterMachines(t *testing.T) {
	machines := []*Machine{
		newTestMachine(0, "cs", testBaseTS, StateHealthy),     // [0]
		newTestMachine(0, "ss", testBaseTS, StateUnhealthy),   // [1]
		newTestMachine(1, "cs", testBaseTS, StateUnreachable), // [2]
		newTestMachine(1, "ss", testBaseTS, StateHealthy),     // [3]
		newTestMachine(2, "cs", testBaseTS, StateRetired),     // [4]
		newTestMachine(2, "ss", testBaseTS, StateRetired),     // [5]
	}

	testCases := []struct {
		name      string
		role      string
		isHealthy bool
		expect    []*Machine
	}{
		{
			"Base",
			"cs",
			true,
			[]*Machine{machines[0]},
		},
		{
			"FilteredByRole",
			"cs",
			false,
			[]*Machine{machines[0], machines[2], machines[4]},
		},
		{
			"FilteredByHealth",
			"",
			true,
			[]*Machine{machines[0], machines[3]},
		},
		{
			"DisableFilter",
			"",
			false,
			[]*Machine{machines[0], machines[1], machines[2], machines[3], machines[4], machines[5]},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			actual := filterMachines(machines, c.role, c.isHealthy)
			if !cmp.Equal(c.expect, actual) {
				t.Error("unexpected result", cmp.Diff(c.expect, actual))
			}
		})
	}
}
