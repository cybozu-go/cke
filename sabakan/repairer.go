package sabakan

import (
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
)

func Repairer(machines []Machine, repairEntries []*cke.RepairQueueEntry, rebootEntries []*cke.RebootQueueEntry, nodeStatuses map[string]*cke.NodeStatus, constraints *cke.Constraints) []*cke.RepairQueueEntry {
	recent := make(map[string]bool)
	for _, entry := range repairEntries {
		// entry.Operation is ignored when checking duplication
		recent[entry.Address] = true
	}

	rebooting := make(map[string]bool)
	for _, entry := range rebootEntries {
		if entry.Status == cke.RebootStatusRebooting {
			rebooting[entry.Node] = true // entry.Node denotes IP address
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

		serial := machine.Spec.Serial
		address := machine.Spec.IPv4[0]

		if recent[address] {
			log.Warn("ignore recently-repaired non-healthy machine", map[string]interface{}{
				"serial":  serial,
				"address": address,
			})
			continue
		}

		if machine.Status.State == StateUnreachable && machine.Status.Duration < float64(constraints.RepairRebootingSeconds) {
			if rebooting[address] {
				log.Info("ignore rebooting unreachable machine", map[string]interface{}{
					"serial":  serial,
					"address": address,
				})
				continue
			}

			if _, ok := nodeStatuses[address]; !ok {
				log.Info("ignore out-of-cluster unreachable machine", map[string]interface{}{
					"serial":  serial,
					"address": address,
				})
				continue
			}
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
