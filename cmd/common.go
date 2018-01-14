package cmd

import (
	"os"

	"github.com/hashicorp/hcl2/hcl"
	"golang.org/x/crypto/ssh/terminal"
)

func printDiagnostics(diags hcl.Diagnostics, fileASTs map[string]*hcl.File) {
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
	printer := hcl.NewDiagnosticTextWriter(os.Stderr, fileASTs, uint(width), isTTY)
	printer.WriteDiagnostics(diags)
}
