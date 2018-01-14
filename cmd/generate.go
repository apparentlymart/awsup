package cmd

import (
	"os"

	"github.com/apparentlymart/awsup/config"
	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate [source-dir-or-file]",
	Short: "Generate CloudFormation template JSON",
	Long:  `Generate CloudFormation template JSON from awsup configuration`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			args = []string{"."}
		}

		cfg, diags := config.ParseDirOrFile(args[0])
		printDiagnostics(diags, cfg.FileASTs)
		if diags.HasErrors() {
			os.Exit(2)
		}
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
