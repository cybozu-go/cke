package sabakan

import "testing"

func TestPromoteWorker(t *testing.T) {
	op := &updateOp{workers: []*Machine{}}
	ips := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
	for _, ip := range ips {
		m := &Machine{}
		m.Spec.IPv4 = []string{ip}
		op.workers = append(op.workers, m)
	}

	{
		// worker is not found
		m := &Machine{}
		m.Spec.IPv4 = []string{"10.0.0.100"}
		if op.promoteWorker(m) {
			t.Fatal("worker should not be found")
		}
		if len(op.workers) != 3 {
			t.Fatalf("len(op.cps): expect 3 actual %d", len(op.workers))
		}
		if len(op.cps) != 0 {
			t.Fatalf("len(op.cps): expect 0 actual %d", len(op.cps))
		}
	}
	{
		// worker is found
		targetIP := "10.0.0.1"
		m := &Machine{}
		m.Spec.IPv4 = []string{targetIP}
		if !op.promoteWorker(m) {
			t.Fatal("worker should be found")
		}
		if len(op.workers) != 2 {
			t.Fatalf("len(op.cps): expect 2 actual %d", len(op.workers))
		}
		if len(op.cps) != 1 {
			t.Fatalf("len(op.cps): expect 1 actual %d", len(op.cps))
		}
		if op.cps[0].Spec.IPv4[0] != targetIP {
			t.Errorf(
				"op.cps[0].Spec.IPv4[0]: expect %s actual %s",
				targetIP,
				op.cps[0].Spec.IPv4[0],
			)
		}
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
