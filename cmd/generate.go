package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/apparentlymart/awsup/cfnjson"
	"github.com/apparentlymart/awsup/eval"
	"github.com/apparentlymart/awsup/schema"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/spf13/cobra"
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

		sch := schema.Builtin()

		inputConstants, constantsDiags := parser.ParseValuesFiles(generateCmdConstantsFiles...)
		diags = append(diags, constantsDiags...)
		exitIfErrors(diags)

		ctx, ctxDiags := eval.NewRootContext(parser, args[0], inputConstants, sch)
		diags = append(diags, ctxDiags...)
		exitIfErrors(diags)

		template, templateDiags := ctx.Build()
		diags = append(diags, templateDiags...)
		exitIfErrors(diags)

		rawTemplate, prepDiags := cfnjson.PrepareStructure(template)
		diags = append(diags, prepDiags...)
		exitIfErrors(diags)

		jsonSrc, _ := json.MarshalIndent(rawTemplate, "", "  ")
		fmt.Printf("%s\n", jsonSrc)

		// If we didn't error out above then we might still have some warnings
		// to print here.
		printDiagnostics(diags)
	},
}

func init() {
	generateCmd.Flags().StringSliceVarP(&generateCmdConstantsFiles, "constants", "c", nil, "pass constants from values files into the root module")
	rootCmd.AddCommand(generateCmd)
}
