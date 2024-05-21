package sabakan

import (
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
)

func Repairer(machines []Machine, repairEntries []*cke.RepairQueueEntry, rebootEntries []*cke.RebootQueueEntry, constraints *cke.Constraints, now time.Time) []*cke.RepairQueueEntry {
	recent := make(map[string]bool)
	for _, entry := range repairEntries {
		// entry.Operation is ignored when checking duplication
		recent[entry.Address] = true
	}

	rebootLimit := now.Add(time.Duration(-constraints.RepairRebootingSeconds) * time.Second)
	rebootingSince := make(map[string]time.Time)
	for _, entry := range rebootEntries {
		if entry.Status == cke.RebootStatusRebooting {
			rebootingSince[entry.Node] = entry.LastTransitionTime // entry.Node denotes IP address
		}
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

		since, ok := rebootingSince[machine.Spec.IPv4[0]]
		if ok && since.After(rebootLimit) && machine.Status.State == StateUnreachable {
			log.Info("ignore rebooting unreachable machine", map[string]interface{}{
				"serial":  machine.Spec.Serial,
				"address": machine.Spec.IPv4[0],
			})
			continue
		}

		newMachines = append(newMachines, machine)
	}

	if len(repairEntries)+len(newMachines) > constraints.MaximumRepairs {
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
