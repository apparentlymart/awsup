package eval

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

// DynExpr represents an expression that can be evaluated dynamically
// by CloudFormation when applying a template.
//
// This type represents the subset of operations that we can encode as
// expressions into CloudFormation JSON for dynamic evaluation. DynExpr
// instances are produced by translating hclsyntax.Expression nodes that
// have analogs in the CloudFormation language.
type DynExpr interface {
	dynamicExpr() isDynamicExpr
}

// DynLiteral represents a literal value within our dynamic expression
// intermediate language.
//
// The name is of course a bit of a misnomer since a literal can't be dynamic,
// but this type allows dynamic expressions to make use of literals.
type DynLiteral struct {
	Value cty.Value

	SrcRange hcl.Range
	isDynamicExpr
}

// DynJoin joins several expressions together with a delimiter.
type DynJoin struct {
	Delimiter string
	Exprs     []DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

// DynIf returns one of two values depending on the result of a named
// condition defined in the template.
type DynIf struct {
	ConditionName string
	If, Else      DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

// DynEquals is a boolean expression (to be used in named conditionals only)
// that returns true if the two given values are equal.
type DynEquals struct {
	// A and B must both be either DynLiteral or DynRef.
	A, B DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

// DynLogical is a boolean expression (to be used in named conditionals only) that
// performs the logical AND or OR operation on a number of other boolean expressions.
type DynLogical struct {
	Op     DynLogicalOp
	Values []DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

type DynLogicalOp rune

//go:generate stringer -type DynLogicalOp

const (
	invalidLogicalOp DynLogicalOp = 0
	DynLogicalAnd    DynLogicalOp = '&'
	DynLogicalOr     DynLogicalOp = '|'
)

// DynOr is a boolean expression (to be used in named conditionals only) that
// performs the logical OR operation on a number of other boolean expressions.
type DynOr struct {
	Values []DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

// DynNot is a boolean expression (to be used in named conditionals only) that
// returns the boolean inverse of another boolean expression.
type DynNot struct {
	Value DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

// DynSplit splits a string by a given delimiter to produce a list.
type DynSplit struct {
	Delimiter string
	String    DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

// DynIndex looks up a single item from a list expression by its index.
type DynIndex struct {
	List DynExpr

	// Index may only be DynLiteral, DynRef or DynMappingLookup.
	Index DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

// DynRef represents the name or id returned by a particular reference
// or pseudo-reference in the CloudFormation language.
type DynRef struct {
	LogicalID string

	SrcRange hcl.Range
	isDynamicExpr
}

// DynGetAttr represents an attribute exported by a particular resource.
type DynGetAttr struct {
	LogicalID string
	Attrs     []DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

// DynMappingLookup looks up a value from a named mapping table defined
// within the template.
type DynMappingLookup struct {
	MappingName string

	// FirstKey and SecondKey may only use DynLiteral, DynRef, and nested DynMappingLookup.
	FirstKey, SecondKey DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

// DynBase64 returns a base64-encoded version of a given string.
type DynBase64 struct {
	String DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

// DynAccountAZs returns a list of availability zones that are supported in
// the region where the CloudFormation template is being applied, for the
// AWS account that is applying the template.
type DynAccountAZs struct {
	// RegionName must be either a DynLiteral or a DynRef
	RegionName DynExpr

	SrcRange hcl.Range
	isDynamicExpr
}

type isDynamicExpr struct {
	// embed this to mark a struct as being a DynamicExpr
}

func (i isDynamicExpr) dynamicExpr() isDynamicExpr {
	return i
}
