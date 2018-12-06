package cmd

import (
	"fmt"

	"github.com/cybozu-go/cke"
	"github.com/spf13/cobra"
)

// imagesCmd represents the images command
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "list container image names used by cke",
	Long:  `List container image names used by cke.`,

	// Override rootCmd.PersistentPreRunE.
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		for _, img := range cke.AllImages() {
			fmt.Println(img)
		}
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd)
}
