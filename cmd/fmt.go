package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var fmtCheckOnly bool

// fmtCmd represents the fmt command
var fmtCmd = &cobra.Command{
	Use:   "fmt [source-dirs-or-files...]",
	Short: "Rewrite configuration files to canonical formatting",
	Long: `Rewrite configuration files so that they use the canonical layout for block
nesting and attribute alignment.

- If a specific configuration file is given, that file is rewritten in-place.
- If a directory is given, .awsup files in that directory are rewritten
  in-place.
- If no arguments are given, all .awsup files in the current directory are
  rewritten in-place.
- If the arguments are literally "-" then configuration will be read from stdin
  and the result will be written to stdout. This cannot be mixed with any other
  arguments.

No output is produced on stdout unless the argument is literally "-".

If any of the inputs contain syntax errors then diagnostic information will be
printed to stderr and exit status is 2. If multiple inputs are provided, some
may already have been updated by the time errors are returned.
`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("'fmt' is not yet implemented")
	},
}

func init() {
	fmtCmd.Flags().BoolVarP(
		&fmtCheckOnly,
		"check-only", "c",
		false,
		"don't modify any files; instead, exit status 1 if non-canonical",
	)
	rootCmd.AddCommand(fmtCmd)
}
