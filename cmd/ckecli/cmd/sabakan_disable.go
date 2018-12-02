package cmd

import (
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// sabakanDisableCmd represents the "sabakan disable" command
var sabakanDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "disable sabakan integration",
	Long:  `Disable sabakan integration by removing stored sabakan URL.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(storage.DeleteSabakanURL)
		well.Stop()
		return well.Wait()
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanDisableCmd)
}
