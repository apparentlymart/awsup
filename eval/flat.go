package eval

import (
	"github.com/zclconf/go-cty/cty"
)

type FlatTemplate struct {
	Description string
	Metadata    map[string]cty.Value
	Parameters  map[string]*FlatParameter
	Mappings    map[string]map[string]cty.Value
	Conditions  map[string]DynExpr
	Resources   map[string]*FlatResource
	Outputs     map[string]*FlatOutput
}

type FlatParameter struct {
	Type           string
	Description    string
	DefaultValue   cty.Value
	AllowedPattern cty.Value
	AllowedValues  []cty.Value
	MinLength      cty.Value
	MaxLength      cty.Value
	MinValue       cty.Value
	MaxValue       cty.Value
	NoEcho         cty.Value
}

type FlatResource struct {
	Type       string
	Properties map[string]*DynExpr
	Metadata   map[string]cty.Value
	DependsOn  []string

	// TODO: CreationPolicy, DeletionPolicy, UpdatePolicy
}

type FlatOutput struct {
	Value      DynExpr
	ExportName DynExpr
}
