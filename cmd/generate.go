package cmd

import (
	"fmt"

	"github.com/apparentlymart/awsup/eval"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/spf13/cobra"
	"github.com/zclconf/go-cty/cty"
)

var generateCmdConstantsFiles []string

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

		var diags hcl.Diagnostics

		inputConstants, constantsDiags := parser.ParseValuesFiles(generateCmdConstantsFiles...)
		diags = append(diags, constantsDiags...)
		exitIfErrors(diags)

		ctx, ctxDiags := eval.NewRootContext(parser, args[0], inputConstants)
		diags = append(diags, ctxDiags...)
		exitIfErrors(diags)

		// The following is just a placeholder for real context processing,
		// to demonstrate that the config loading is working.
		ctx.VisitModules(func(mctx *eval.ModuleContext) bool {
			descVal, _ := mctx.EvalConstant(mctx.Config.Description, cty.String, eval.NoEachState)
			fmt.Printf("- %s: %#v\n", mctx.Path, descVal)
			return true
		})

		// If we didn't error out above then we might still have some warnings
		// to print here.
		printDiagnostics(diags)
	},
}

func init() {
	generateCmd.Flags().StringSliceVarP(&generateCmdConstantsFiles, "constants", "c", nil, "pass constants from values files into the root module")
	rootCmd.AddCommand(generateCmd)
}
