package sabakan

import (
	"testing"
	"time"
)

func newTestMachine(rack int, retireDate time.Time, state State) *Machine {
	m := &Machine{}
	m.Spec.Rack = rack
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
			newTestMachine(0, testBaseTS, StateHealthy),
			nil,
			0,
		},
		{
			"SameRack",
			newTestMachine(1, testBaseTS, StateHealthy),
			map[int]int{0: 2, 1: 3},
			-3,
		},
		{
			"SameRack2",
			newTestMachine(1, testBaseTS, StateHealthy),
			map[int]int{0: 2, 1: 13},
			-baseScore,
		},
		{
			"Future250",
			newTestMachine(1, testFuture250, StateHealthy),
			nil,
			baseScore,
		},
		{
			"Future500",
			newTestMachine(1, testFuture500, StateHealthy),
			nil,
			baseScore * 2,
		},
		{
			"Future1000",
			newTestMachine(1, testFuture1000, StateHealthy),
			nil,
			baseScore * 3,
		},
		{
			"Past250",
			newTestMachine(1, testPast250, StateHealthy),
			nil,
			baseScore * -1,
		},
		{
			"Past500",
			newTestMachine(1, testPast500, StateHealthy),
			nil,
			baseScore * -2,
		},
		{
			"Past1000",
			newTestMachine(1, testPast1000, StateHealthy),
			nil,
			baseScore * -3,
		},
		{
			"NotHealthy",
			newTestMachine(1, testBaseTS, StateRetiring),
			nil,
			unhealthyScore,
		},
		{
			"Compound",
			newTestMachine(2, testFuture500, StateRetiring),
			map[int]int{2: 9},
			baseScore*2 + unhealthyScore - 9,
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
