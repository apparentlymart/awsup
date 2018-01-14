package cmd

import (
	"fmt"
	"os"

	"github.com/apparentlymart/awsup/eval"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/spf13/cobra"
	"github.com/zclconf/go-cty/cty"
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

		// TODO: Populate this from files provided via the CLI
		inputConstants := make(hcl.Attributes)

		ctx, diags := eval.NewRootContext(parser, args[0], inputConstants)
		printDiagnostics(diags)
		if diags.HasErrors() {
			os.Exit(2)
		}

		// The following is just a placeholder for real context processing,
		// to demonstrate that the config loading is working.
		ctx.VisitModules(func(mctx *eval.ModuleContext) bool {
			descVal, _ := mctx.EvalConstant(mctx.Config.Description, cty.String, eval.NoEachState)
			fmt.Printf("- %s: %#v\n", mctx.Path, descVal)
			return true
		})
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
