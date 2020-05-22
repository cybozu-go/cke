package sabakan

import (
	"time"
)

const (
	// maxCountPerRack should be max count of machines per rack + 1
	maxCountPerRack = 100
)

func scoreByDays(days int) int {
	var score int
	if days > 250 {
		score++
	}
	if days > 500 {
		score++
	}
	if days > 1000 {
		score++
	}
	if days < -250 {
		score--
	}
	if days < -500 {
		score--
	}
	if days < -1000 {
		score--
	}
	return score
}

func scoreMachine(m *Machine, rackCount map[int]int, ts time.Time) int {
	rackScore := maxCountPerRack - rackCount[m.Spec.Rack]

	var isHealthy int
	if m.Status.State == StateHealthy {
		isHealthy = 1
	}

	days := int(m.Spec.RetireDate.Sub(ts).Hours() / 24)
	daysScore := scoreByDays(days)

	score := rackScore*100 + isHealthy*10 + daysScore

	return score
}

func filterMachine(m *Machine, role string, isHealthy bool) bool {
	if role != "" && m.Spec.Role != role {
		return false
	}

	if isHealthy && m.Status.State != StateHealthy {
		return false
	}

	return true
}

func filterMachines(ms []*Machine, role string, isHealthy bool) []*Machine {
	var filtered []*Machine
	for _, m := range ms {
		if !filterMachine(m, role, isHealthy) {
			continue
		}
		filtered = append(filtered, m)
	}

	return filtered
}
