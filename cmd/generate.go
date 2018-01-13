package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate [source-dir-or-file]",
	Short: "Generate CloudFormation template JSON",
	Long:  `Generate CloudFormation template JSON from awsup configuration`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("'generate' is not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
