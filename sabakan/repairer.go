package sabakan

import (
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
)

func Repairer(machines []Machine, entries []*cke.RepairQueueEntry, constraints *cke.Constraints) []*cke.RepairQueueEntry {
	recent := make(map[string]bool)
	for _, entry := range entries {
		// entry.Operation is ignored when checking duplication
		recent[entry.Address] = true
	}

	newMachines := make([]Machine, 0, len(machines))
	for _, machine := range machines {
		if len(machine.Spec.IPv4) == 0 {
			log.Warn("ignore non-healthy machine w/o IPv4 address", map[string]interface{}{
				"serial": machine.Spec.Serial,
			})
			continue
		}

		if recent[machine.Spec.IPv4[0]] {
			log.Warn("ignore recently-repaired non-healthy machine", map[string]interface{}{
				"serial":  machine.Spec.Serial,
				"address": machine.Spec.IPv4[0],
			})
			continue
		}

		newMachines = append(newMachines, machine)
	}

	if len(entries)+len(newMachines) > constraints.MaximumRepairs {
		log.Warn("ignore too many repair requests", nil)
		return nil
	}

	ret := make([]*cke.RepairQueueEntry, len(newMachines))
	for i, machine := range newMachines {
		operation := strings.ToLower(string(machine.Status.State))
		typ := machine.Spec.BMC.Type
		address := machine.Spec.IPv4[0]
		entry := cke.NewRepairQueueEntry(operation, typ, address)
		log.Info("initiate sabakan-triggered automatic repair", map[string]interface{}{
			"serial":    machine.Spec.Serial,
			"address":   address,
			"operation": operation,
		})
		ret[i] = entry
	}

	return ret
}
