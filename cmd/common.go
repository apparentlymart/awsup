package cmd

import (
	"os"

	"github.com/apparentlymart/awsup/config"
	"github.com/hashicorp/hcl2/hcl"
	"golang.org/x/crypto/ssh/terminal"
)

var parser = config.NewParser()

func printDiagnostics(diags hcl.Diagnostics) {
	if len(diags) == 0 {
		return
	}

	width := 78
	isTTY := terminal.IsTerminal(int(os.Stderr.Fd()))
	if isTTY {
		newWidth, _, err := terminal.GetSize(int(os.Stderr.Fd()))
		if err != nil {
			width = newWidth
		}
	}
	printer := hcl.NewDiagnosticTextWriter(os.Stderr, parser.Files(), uint(width), isTTY)
	printer.WriteDiagnostics(diags)
}

func exitIfErrors(diags hcl.Diagnostics) {
	if diags.HasErrors() {
		printDiagnostics(diags)
		os.Exit(2)
	}
}
