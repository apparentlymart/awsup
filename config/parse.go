package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func ParseFileSource(src []byte, filename string) (*File, hcl.Diagnostics) {
	astFile, diags := hclsyntax.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})

	file := &File{
		Source:     src,
		SourcePath: filename,
		SourceAST:  astFile,
	}
	if astFile == nil {
		return file, diags
	}

	content, contentDiags := astFile.Body.Content(fileRootSchema)
	diags = append(diags, contentDiags...)

	file.Description = content.Attributes["description"]

	for _, block := range content.Blocks {
		switch block.Type {

		case "Conditions":
			attrs, attrsDiags := block.Body.JustAttributes()
			diags = append(diags, attrsDiags...)

			for _, attr := range attrs {
				file.Conditions = append(file.Conditions, attr)
			}

		case "Constant":
			constant, decDiags := decodeConstant(block)
			diags = append(diags, decDiags...)
			file.Constants = append(file.Constants, constant)

		case "Locals":
			attrs, attrsDiags := block.Body.JustAttributes()
			diags = append(diags, attrsDiags...)

			for _, attr := range attrs {
				file.Locals = append(file.Locals, attr)
			}

		case "Mappings":
			attrs, attrsDiags := block.Body.JustAttributes()
			diags = append(diags, attrsDiags...)

			for _, attr := range attrs {
				file.Mappings = append(file.Mappings, attr)
			}

		case "Metadata":
			attrs, attrsDiags := block.Body.JustAttributes()
			diags = append(diags, attrsDiags...)

			for _, attr := range attrs {
				file.Metadata = append(file.Metadata, attr)
			}

		case "Module":
			module, decDiags := decodeModuleCall(block)
			diags = append(diags, decDiags...)
			file.Modules = append(file.Modules, module)

		case "Output":
			output, decDiags := decodeOutput(block)
			diags = append(diags, decDiags...)
			file.Outputs = append(file.Outputs, output)

		case "Parameter":
			param, decDiags := decodeParameter(block)
			diags = append(diags, decDiags...)
			file.Parameters = append(file.Parameters, param)

		case "Resource":
			resource, decDiags := decodeResource(block)
			diags = append(diags, decDiags...)
			file.Resources = append(file.Resources, resource)

		case "UserInterface":
			// TODO
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "UserInterface block is not yet supported",
				Detail:   "This block is ignored by this version of awsup.",
				Subject:  &block.DefRange,
			})

		default:
			// Should never happen since the above cases should always cover
			// all of the block types in our schema.
			panic(fmt.Errorf("unhandled block type %q", block.Type))
		}
	}

	return file, diags
}

func ParseFile(filename string) (*File, hcl.Diagnostics) {
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Failed to read configuration file",
				Detail:   fmt.Sprintf("There was an error reading %s: %s", filename, err),
			},
		}
	}
	return ParseFileSource(src, filename)
}

func NewModule(path string, files ...*File) (*Module, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	module := &Module{
		SourcePath:    path,
		Files:         make(map[string]*File),
		FileASTs:      make(map[string]*hcl.File),
		Description:   hcl.StaticExpr(cty.NullVal(cty.String), hcl.Range{}),
		Conditions:    make(map[string]*hcl.Attribute),
		Constants:     make(map[string]*Constant),
		Locals:        make(map[string]*hcl.Attribute),
		Mappings:      make(map[string]*hcl.Attribute),
		Metadata:      make(map[string]*hcl.Attribute),
		Modules:       make(map[string]*ModuleCall),
		Outputs:       make(map[string]*Output),
		Parameters:    make(map[string]*Parameter),
		Resources:     make(map[string]*Resource),
		UIParamGroups: make([]*UIParamGroup, 0),
		UIParamLabels: make(map[string]*hcl.Attribute),
	}

	for _, file := range files {
		if file == nil {
			// Should never happen
			panic("nil *File passed to NewModule")
		}
		if _, conflict := module.Files[file.SourcePath]; conflict {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Duplicate file in module",
				Detail:   fmt.Sprintf("Ignored duplicate definition for file %s while building module.", file.SourcePath),
			})
			continue
		}
		module.Files[file.SourcePath] = file
		module.FileASTs[file.SourcePath] = file.SourceAST

		for _, def := range file.Conditions {
			if _, conflict := module.Conditions[def.Name]; conflict {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate condition",
					Detail: fmt.Sprintf(
						"Duplicate definition of condition %q, which was already defined at %s.",
						def.Name, module.Conditions[def.Name].NameRange,
					),
					Subject: &def.NameRange,
				})
			}
			module.Conditions[def.Name] = def
		}

		for _, def := range file.Constants {
			if _, conflict := module.Constants[def.Name]; conflict {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate constant",
					Detail: fmt.Sprintf(
						"Duplicate definition of constant %q, which was already defined at %s.",
						def.Name, module.Constants[def.Name].DeclRange,
					),
					Subject: &def.DeclRange,
				})
			}
			module.Constants[def.Name] = def
		}

		for _, def := range file.Locals {
			if _, conflict := module.Locals[def.Name]; conflict {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate local",
					Detail: fmt.Sprintf(
						"Duplicate definition of local %q, which was already defined at %s.",
						def.Name, module.Locals[def.Name].NameRange,
					),
					Subject: &def.NameRange,
				})
			}
			module.Locals[def.Name] = def
		}

		for _, def := range file.Mappings {
			if _, conflict := module.Mappings[def.Name]; conflict {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate mapping",
					Detail: fmt.Sprintf(
						"Duplicate definition of mapping %q, which was already defined at %s.",
						def.Name, module.Mappings[def.Name].NameRange,
					),
					Subject: &def.NameRange,
				})
			}
			module.Mappings[def.Name] = def
		}

		for _, def := range file.Metadata {
			if _, conflict := module.Mappings[def.Name]; conflict {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate mapping",
					Detail: fmt.Sprintf(
						"Duplicate definition of metadata field %q, which was already defined at %s.",
						def.Name, module.Metadata[def.Name].NameRange,
					),
					Subject: &def.NameRange,
				})
			}
			module.Metadata[def.Name] = def
		}

		for _, def := range file.Modules {
			if _, conflict := module.Modules[def.Name]; conflict {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate module",
					Detail: fmt.Sprintf(
						"Duplicate definition of module %q, which was already defined at %s.",
						def.Name, module.Modules[def.Name].DeclRange,
					),
					Subject: &def.DeclRange,
				})
			}
			module.Modules[def.Name] = def
		}

		for _, def := range file.Outputs {
			if _, conflict := module.Outputs[def.Name]; conflict {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate output",
					Detail: fmt.Sprintf(
						"Duplicate definition of output %q, which was already defined at %s.",
						def.Name, module.Outputs[def.Name].DeclRange,
					),
					Subject: &def.DeclRange,
				})
			}
			module.Outputs[def.Name] = def
		}

		for _, def := range file.Parameters {
			if _, conflict := module.Parameters[def.Name]; conflict {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate parameter",
					Detail: fmt.Sprintf(
						"Duplicate definition of parameter %q, which was already defined at %s.",
						def.Name, module.Parameters[def.Name].DeclRange,
					),
					Subject: &def.DeclRange,
				})
			}
			module.Parameters[def.Name] = def
		}

		for _, def := range file.Resources {
			if _, conflict := module.Resources[def.LogicalID]; conflict {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate resource",
					Detail: fmt.Sprintf(
						"Duplicate definition of resource %q, which was already defined at %s.",
						def.LogicalID, module.Resources[def.LogicalID].DeclRange,
					),
					Subject: &def.DeclRange,
				})
			}
			module.Resources[def.LogicalID] = def
		}

		for _, def := range file.UIParamGroups {
			module.UIParamGroups = append(module.UIParamGroups, def)
		}

		for _, def := range file.UIParamLabels {
			if _, conflict := module.UIParamLabels[def.Name]; conflict {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate parameter label",
					Detail: fmt.Sprintf(
						"Duplicate parameter label for parameter %q, which was already labeled at %s.",
						def.Name, module.UIParamLabels[def.Name].NameRange,
					),
					Subject: &def.NameRange,
				})
			}
			module.UIParamLabels[def.Name] = def
		}

	}

	return module, diags
}

func ParseDir(path string) (*Module, hcl.Diagnostics) {
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Failed to read configuration",
				Detail:   fmt.Sprintf("There was an error reading %s: %s", path, err),
			},
		}
	}

	var files []*File
	var diags hcl.Diagnostics
	for _, info := range infos {
		name := info.Name()

		// Look for files with names ending in ".awsup" while also filtering
		// out things that look like editor temporary files.
		switch {
		case info.IsDir():
			continue
		case !strings.HasSuffix(name, ".awsup"):
			continue
		case strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#"):
			continue
		case strings.HasPrefix(name, "."):
			continue
		}

		filePath := filepath.Join(path, name)
		file, fileDiags := ParseFile(filePath)
		diags = append(diags, fileDiags...)
		files = append(files, file)
	}

	module, modDiags := NewModule(path, files...)
	diags = append(diags, modDiags...)
	return module, diags
}

func ParseDirOrFile(path string) (*Module, hcl.Diagnostics) {
	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		file, diags := ParseFile(path)
		module, modDiags := NewModule(path, file)
		diags = append(diags, modDiags...)
		return module, diags
	}

	return ParseDir(path)
}

func decodeConstant(block *hcl.Block) (*Constant, hcl.Diagnostics) {
	var b struct {
		Description hcl.Expression `hcl:"Description"`
		Default     hcl.Expression `hcl:"Default"`
	}
	diags := gohcl.DecodeBody(block.Body, nil, &b)

	return &Constant{
		Name:        block.Labels[0],
		DeclRange:   block.DefRange,
		Description: b.Description,
		Default:     b.Default,
	}, diags
}

func decodeModuleCall(block *hcl.Block) (*ModuleCall, hcl.Diagnostics) {
	var b struct {
		Source     hcl.Expression  `hcl:"Source"`
		Parameters *hcl.Attributes `hcl:"Parameters,block"`
		Constants  *hcl.Attributes `hcl:"Constants,block"`
		ForEach    hcl.Expression  `hcl:"ForEach"`
	}
	diags := gohcl.DecodeBody(block.Body, nil, &b)

	module := &ModuleCall{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
		Source:    b.Source,
		ForEach:   b.ForEach,
	}

	if b.Parameters != nil {
		module.Parameters = *b.Parameters
	}
	if b.Constants != nil {
		module.Constants = *b.Constants
	}

	return module, diags
}

func decodeOutput(block *hcl.Block) (*Output, hcl.Diagnostics) {
	var b struct {
		Description hcl.Expression `hcl:"Description"`
		Value       hcl.Expression `hcl:"Value"`
		Export      *struct {
			Name hcl.Expression `hcl:"Name"`
		} `hcl:"Export,block"`
	}
	diags := gohcl.DecodeBody(block.Body, nil, &b)

	ret := &Output{
		Name:        block.Labels[0],
		DeclRange:   block.DefRange,
		Description: b.Description,
		Value:       b.Value,
	}

	if b.Export != nil {
		ret.Export = &OutputExport{
			Name: b.Export.Name,
		}
	}

	return ret, diags
}

func decodeParameter(block *hcl.Block) (*Parameter, hcl.Diagnostics) {
	var b struct {
		Type                  string         `hcl:"Type"`
		Description           hcl.Expression `hcl:"Description"`
		Default               hcl.Expression `hcl:"Default"`
		AllowedPattern        hcl.Expression `hcl:"AllowedPattern"`
		AllowedValues         hcl.Expression `hcl:"AllowedValues"`
		ConstraintDescription hcl.Expression `hcl:"ConstraintDescription"`
		MinLength             hcl.Expression `hcl:"MinLength"`
		MaxLength             hcl.Expression `hcl:"MaxLength"`
		MinValue              hcl.Expression `hcl:"MinValue"`
		MaxValue              hcl.Expression `hcl:"MaxValue"`
		Obscure               hcl.Expression `hcl:"Obscure"`
	}
	diags := gohcl.DecodeBody(block.Body, nil, &b)

	return &Parameter{
		Name:                  block.Labels[0],
		DeclRange:             block.DefRange,
		Type:                  b.Type,
		Description:           b.Description,
		Default:               b.Default,
		AllowedPattern:        b.AllowedPattern,
		AllowedValues:         b.AllowedValues,
		ConstraintDescription: b.ConstraintDescription,
		MinLength:             b.MinLength,
		MaxLength:             b.MaxLength,
		MinValue:              b.MinValue,
		MaxValue:              b.MaxValue,
		Obscure:               b.Obscure,
	}, diags
}

func decodeResource(block *hcl.Block) (*Resource, hcl.Diagnostics) {
	var b struct {
		Type           string          `hcl:"Type"`
		Parameters     *hcl.Attributes `hcl:"Parameters,block"`
		Metadata       *hcl.Attributes `hcl:"Metadata,block"`
		DependsOn      hcl.Expression  `hcl:"DependsOn"`
		CreationPolicy *struct {
			AutoScaling *struct {
				MinSuccessfulInstancesPercent hcl.Expression `hcl:"MinSuccessfulInstancesPercent"`
			} `hcl:"AutoScaling"`
			Signal *struct {
				Count   hcl.Expression `hcl:"Count"`
				Timeout hcl.Expression `hcl:"Timeout"`
			} `hcl:"AutoScaling"`
		} `hcl:"CreationPolicy"`
		DeletionPolicy hcl.Expression `hcl:"DeletionPolicy"`
		UpdatePolicy   *struct {
		} `hcl:"UpdatePolicy"`
		ForEach hcl.Expression `hcl:"ForEach"`
	}
	diags := gohcl.DecodeBody(block.Body, nil, &b)

	resource := &Resource{
		LogicalID: block.Labels[0],
		DeclRange: block.DefRange,
	}

	return resource, diags
}

var fileRootSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "Description",
			Required: false,
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "Conditions",
		},
		{
			Type:       "Constant",
			LabelNames: []string{"name"},
		},
		{
			Type: "Locals",
		},
		{
			Type: "Mappings",
		},
		{
			Type: "Metadata",
		},
		{
			Type:       "Module",
			LabelNames: []string{"name"},
		},
		{
			Type:       "Output",
			LabelNames: []string{"name"},
		},
		{
			Type:       "Parameter",
			LabelNames: []string{"name"},
		},
		{
			Type:       "Resource",
			LabelNames: []string{"logical id"},
		},
		{
			Type: "UserInterface",
		},
	},
}
