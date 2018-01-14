package eval

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// EvalConstant evaluates the given expression to produce a constant value,
// which is then converted to the requested type.
//
// When evaluating in this mode, only constant values and named constants
// can be accessed. If any other scope traversals are detected then
// error diagnostics will be returned and the result will probably be a null
// value.
//
// If EachState is set to anything other than NoEachState then the "Each"
// object is also available for use, exposing the values in the given EachState.
func (mctx *ModuleContext) EvalConstant(expr hcl.Expression, ty cty.Type, each EachState) (cty.Value, hcl.Diagnostics) {
	// This method must be careful in how it uses mctx because it is called
	// during the construction of mctx. The only safe field to use is
	// mctx.Constants.

	var diags hcl.Diagnostics
	scope := make(map[string]cty.Value)

	traversals := expr.Variables()
	for _, traversal := range traversals {
		if rootName := traversal.RootName(); rootName != "Const" {
			if rootName == "Each" {
				if each == NoEachState {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Illegal use of \"Each\" object",
						Detail:   "The \"Each\" object can be accessed only within modules and resources that have ForEach set.",
						Subject:  traversal.SourceRange().Ptr(),
					})
				}
				continue
			}
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Illegal use of non-constant value",
				Detail:   "Only literals and named constants can be used here.",
				Subject:  traversal.SourceRange().Ptr(),
			})
			// Put a placeholder value in the scope anyway, so that
			// we can still complete evaluation but probably end up with
			// an unknown value as the result.
			scope[rootName] = cty.DynamicVal
		}
	}

	scope["Const"] = cty.ObjectVal(mctx.Constants)
	if each != NoEachState {
		scope["Each"] = cty.ObjectVal(map[string]cty.Value{
			"Key":   each.Key.Value(),
			"Value": each.Value,
		})
	}

	ectx := &hcl.EvalContext{
		Variables: scope,
		// TODO: Once we have some functions, we should expose constant-friendly
		// versions of them in here. We will probably also have some functions
		// that are _only_ supported with constants, due to CloudFormation's
		// rather limited repertoire of dynamic functions.
	}

	val, valDiags := expr.Value(ectx)
	diags = append(diags, valDiags...)

	// Constants must never be unknown. This can happen only if there's an
	// error, so the caller will generally detect this case with
	// diags.HasErrors and not look at the result, but we try to produce
	// a reasonable result anyway so that we can support partial analysis
	// of erroneous configuration.
	if !val.IsKnown() {
		val = cty.NullVal(val.Type())
	}

	val, err := convert.Convert(val, ty)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incorrect value type",
			Detail:   fmt.Sprintf("This expression is not of the expected type: %s", err),
			Subject:  expr.Range().Ptr(),
		})
		val = cty.NullVal(ty) // Ensure we still return something semi-valid
	}

	return val, diags
}

// DetectVariables returns all of the traversals in the given expression that
// are not for named constants. The result is nil if there are no such
// traversals.
func DetectVariables(expr hcl.Expression) []hcl.Traversal {
	var ret []hcl.Traversal
	traversals := expr.Variables()
	for _, traversal := range traversals {
		switch traversal.RootName() {
		case "Const", "Each":
			// We allow both "Const" and "Each" because const refers to named
			// constants and "Each" is used in ForEach constructs which
			// require their inputs to be constant.
		default:
			ret = append(ret, traversal)
		}
	}
	return ret
}
