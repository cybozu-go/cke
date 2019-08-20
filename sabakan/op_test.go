package sabakan

import "testing"

func TestPromoteWorker(t *testing.T) {
	op := &updateOp{workers: []*Machine{}}
	if op.promoteWorker() {
		t.Error("should return false")
	}
	m := &Machine{}
	m.Status.State = StateRetired
	m.Spec.Serial = "0"
	m.Spec.IPv4 = []string{"10.0.0.1"}
	op.workers = append(op.workers, m)
	if op.promoteWorker() {
		t.Error("should return false")
	}

	m = &Machine{}
	m.Status.State = StateHealthy
	m.Spec.Serial = "1"
	m.Spec.IPv4 = []string{"10.0.0.2"}
	op.workers = append(op.workers, m)
	if !op.promoteWorker() {
		t.Fatal("should return true")
	}
	if len(op.cps) != 1 {
		t.Fatal("len(op.cps) != 1")
	}
	if op.cps[0].Spec.Serial != "1" {
		t.Error(`op.cps[0].Spec.Serial != "1"`)
	}
	if len(op.workers) != 1 {
		t.Fatal("len(op.workers) != 1")
	}
	if op.workers[0].Spec.Serial != "0" {
		t.Error(`op.workers[0].Spec.Serial != "0"`)
	}
	if len(op.changes) != 1 {
		t.Error("len(op.changes) != 1")
	}
}

func TestCountMachinesByRack(t *testing.T) {
	{
		// worker
		op := &updateOp{workers: []*Machine{}}
		racks := []int{0, 0, 1}
		for _, r := range racks {
			m := &Machine{}
			m.Spec.Rack = r
			op.workers = append(op.workers, m)
		}

		bin := op.countMachinesByRack(false)
		if bin[0] != 2 || bin[1] != 1 {
			t.Errorf(
				"rack0: expect 2 actual %d rack1: expect 1 actual %d",
				bin[0],
				bin[1],
			)
		}
	}

	{
		// control plane
		op := &updateOp{cps: []*Machine{}}
		racks := []int{0, 0, 1}
		for _, r := range racks {
			m := &Machine{}
			m.Spec.Rack = r
			op.cps = append(op.cps, m)
		}

		bin := op.countMachinesByRack(true)
		if bin[0] != 2 || bin[1] != 1 {
			t.Errorf(
				"rack0: expect 2 actual %d, rack1: expect 1 actual %d",
				bin[0],
				bin[1],
			)
		}
	}

	{
		// empty
		op := &updateOp{cps: []*Machine{}}
		bin := op.countMachinesByRack(true)
		if len(bin) != 0 {
			t.Errorf("len(bin): expect 0 actual %d", len(bin))
		}
	}
}
