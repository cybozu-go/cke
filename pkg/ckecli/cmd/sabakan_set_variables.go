package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/cybozu-go/cke/sabakan"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// sabakanSetVariablesCmd represents the "sabakan set-variables" command
var sabakanSetVariablesCmd = &cobra.Command{
	Use:   "set-variables FILE",
	Short: "set the query variables to search machines in sabakan",
	Long: `Set the query variables to search machines in sabakan.

FILE should contain a JSON object like this:

    {
        "having": {
            "labels": [{"name": "foo", "value": "bar"}],
            "racks": [0, 1, 2],
            "roles": ["worker"],
            "states": ["HEALTHY"],
            "minDaysBeforeRetire": 90
        },
        "notHaving": {
        }
    }
`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}

		vars := new(sabakan.QueryVariables)
		err = json.Unmarshal(data, vars)
		if err != nil {
			return err
		}
		err = vars.IsValid()
		if err != nil {
			return err
		}

		well.Go(func(ctx context.Context) error {
			return storage.SetSabakanQueryVariables(ctx, string(data))
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanSetVariablesCmd)
}
