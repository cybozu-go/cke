package cmd

import (
	"errors"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cybozu-go/cke"
)

var cstrSet func(*cke.Constraints)

// constraintsSetCmd represents the "constraints set" command
var constraintsSetCmd = &cobra.Command{
	Use:   "set NAME VALUE",
	Short: "set a constraint for cluster configuration",
	Long: `Set a constraint for cluster configuration.

NAME is one of:
    control-plane-count
    minimum-workers-rate
    maximum-unreachable-nodes-for-reboot
    maximum-repair-queue-entries
    wait-seconds-to-repair-rebooting

VALUE is an integer.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("wrong number of arguments")
		}

		val, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}

		switch args[0] {
		case "control-plane-count":
			cstrSet = func(cstr *cke.Constraints) {
				cstr.ControlPlaneCount = val
			}
		case "minimum-workers-rate":
			cstrSet = func(cstr *cke.Constraints) {
				cstr.MinimumWorkersRate = val
			}
		case "maximum-unreachable-nodes-for-reboot":
			cstrSet = func(cstr *cke.Constraints) {
				cstr.RebootMaximumUnreachable = val
			}
		case "maximum-repair-queue-entries":
			cstrSet = func(cstr *cke.Constraints) {
				cstr.MaximumRepairs = val
			}
		case "wait-seconds-to-repair-rebooting":
			cstrSet = func(cstr *cke.Constraints) {
				cstr.RepairRebootingSeconds = val
			}
		default:
			return errors.New("no such constraint: " + args[0])
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		cstr, err := storage.GetConstraints(ctx)
		switch err {
		case cke.ErrNotFound:
			cstr = cke.DefaultConstraints()
		case nil:
		default:
			return err
		}

		cstrSet(cstr)
		return storage.PutConstraints(ctx, cstr)
	},
}

func init() {
	constraintsCmd.AddCommand(constraintsSetCmd)
}
