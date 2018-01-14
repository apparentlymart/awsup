package config

import (
	"github.com/hashicorp/hcl2/hcl"
)

type Module struct {
	SourcePath string
	Files      map[string]*File
	FileASTs   map[string]*hcl.File

	Description   hcl.Expression
	Conditions    map[string]*hcl.Attribute
	Constants     map[string]*Constant
	Locals        map[string]*hcl.Attribute
	Mappings      map[string]*hcl.Attribute
	Metadata      map[string]*hcl.Attribute
	Modules       map[string]*ModuleCall
	Outputs       map[string]*Output
	Parameters    map[string]*Parameter
	Resources     map[string]*Resource
	UIParamGroups []*UIParamGroup
	UIParamLabels map[string]*hcl.Attribute
}

type File struct {
	SourcePath string
	SourceAST  *hcl.File
	Source     []byte

	Description   *hcl.Attribute
	Conditions    []*hcl.Attribute
	Constants     []*Constant
	Locals        []*hcl.Attribute
	Mappings      []*hcl.Attribute
	Metadata      []*hcl.Attribute
	Modules       []*ModuleCall
	Outputs       []*Output
	Parameters    []*Parameter
	Resources     []*Resource
	UIParamGroups []*UIParamGroup
	UIParamLabels []*hcl.Attribute
}

type Constant struct {
	Name        string
	DeclRange   hcl.Range
	Description hcl.Expression
	Default     hcl.Expression
}

type ModuleCall struct {
	Name       string
	DeclRange  hcl.Range
	Source     hcl.Expression
	Parameters hcl.Attributes
	Constants  hcl.Attributes
	ForEach    hcl.Expression
}

type Output struct {
	Name        string
	DeclRange   hcl.Range
	Description hcl.Expression
	Value       hcl.Expression
	Export      *OutputExport
}

type OutputExport struct {
	Name hcl.Expression
}

type Parameter struct {
	Name                  string
	Type                  string
	DeclRange             hcl.Range
	Description           hcl.Expression
	Default               hcl.Expression
	AllowedPattern        hcl.Expression
	AllowedValues         hcl.Expression
	ConstraintDescription hcl.Expression
	MinLength             hcl.Expression
	MaxLength             hcl.Expression
	MinValue              hcl.Expression
	MaxValue              hcl.Expression
	Obscure               hcl.Expression
}

type Resource struct {
	LogicalID      string
	Type           string
	DeclRange      hcl.Range
	Properties     hcl.Attributes
	Metadata       hcl.Attributes
	DependsOn      []hcl.Traversal
	CreationPolicy *ResourceCreationPolicy
	DeletionPolicy hcl.Expression
	UpdatePolicy   *ResourceUpdatePolicy
	ForEach        hcl.Expression
}

type ResourceCreationPolicy struct {
	AutoScaling *ResourceCreationPolicyAutoScaling
	Signal      *ResourceCreationPolicySignal
}

type ResourceCreationPolicyAutoScaling struct {
	MinSuccessfulInstancesPercent hcl.Expression
}

type ResourceCreationPolicySignal struct {
	Count   hcl.Expression
	Timeout hcl.Expression
}

type ResourceUpdatePolicy struct {
	DeclRange   hcl.Range
	AutoScaling *ResourceUpdatePolicyAutoScaling
}

type ResourceUpdatePolicyAutoScaling struct {
	Replace hcl.Expression
}

type UIParamGroup struct {
	Parameters []hcl.Traversal
}
