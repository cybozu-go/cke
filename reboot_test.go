package cke

import (
	"reflect"
	"testing"
)

func TestDedupRebootQueueEntries(t *testing.T) {
	if DedupRebootQueueEntries(nil) != nil {
		t.Errorf("DedupRebootQueueEntries(nil) must return nil")
	}

	e0 := &RebootQueueEntry{Node: "1.2.3.4"}
	e1 := &RebootQueueEntry{Node: "5.6.7.8"}
	e2 := &RebootQueueEntry{Node: "1.2.3.4"}
	input := []*RebootQueueEntry{e0, e1, e2}
	expected := []*RebootQueueEntry{e0, e1}
	actual := DedupRebootQueueEntries(input)

	// must be shallow comparison
	if len(actual) != len(expected) || actual[0] != expected[0] || actual[1] != expected[1] {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}

func TestCountRebootQueueEntries(t *testing.T) {
	input := []*RebootQueueEntry{
		{Status: RebootStatusQueued},
		{Status: RebootStatusDraining},
		{Status: RebootStatusRebooting},
		{Status: RebootStatusDraining},
		{Status: RebootStatusRebooting},
		{Status: RebootStatusRebooting},
	}
	expected := map[string]int{
		"queued":    1,
		"draining":  2,
		"rebooting": 3,
		"cancelled": 0,
	}
	actual := CountRebootQueueEntries(input)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}
