package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/hcl2/hcl"
)

func (p *Parser) ParseValuesFiles(filenames ...string) (hcl.Attributes, hcl.Diagnostics) {
	attrs := make(hcl.Attributes)
	var diags hcl.Diagnostics

	for _, filename := range filenames {
		src, err := ioutil.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, hcl.Diagnostics{
					{
						Severity: hcl.DiagError,
						Summary:  "Failed to read values from file",
						Detail:   fmt.Sprintf("The requested file %s does not exist.", filename),
					},
				}
			} else {
				return nil, hcl.Diagnostics{
					{
						Severity: hcl.DiagError,
						Summary:  "Failed to read values from file",
						Detail:   fmt.Sprintf("There was an error reading %s: %s", filename, err),
					},
				}
			}
		}
		thisAttrs, thisDiags := p.ParseValuesSource(src, filename)
		diags = append(diags, thisDiags...)
		for k, v := range thisAttrs {
			attrs[k] = v
		}
	}

	return attrs, diags
}

func (p *Parser) ParseValuesSource(src []byte, filename string) (hcl.Attributes, hcl.Diagnostics) {
	astFile, diags := p.HCLParser.ParseHCL(src, filename)
	if astFile == nil {
		return make(hcl.Attributes), diags
	}

	attrs, decDiags := astFile.Body.JustAttributes()
	diags = append(diags, decDiags...)
	return attrs, diags
}
