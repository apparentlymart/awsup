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
	// during the construction of mctx. The only safe fields to use are
	// mctx.Constants and mctx.Config.

	var diags hcl.Diagnostics
	scope := make(map[string]cty.Value)
	locals := make(map[string]cty.Value)

	traversals := expr.Variables()
	for _, traversal := range traversals {
		rootName := traversal.RootName()
		switch rootName {
		case "Const":
			// Allowed, but no special action is required here since we just
			// install the full constant table in the scope below.
		case "Each":
			if each == NoEachState {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Illegal use of \"Each\" object",
					Detail:   "The \"Each\" object can be accessed only within modules and resources that have ForEach set.",
					Subject:  traversal.SourceRange().Ptr(),
				})
			}
		case "Local":
			// Only locals that are fully-constant are allowed in constant
			// expressions.
			if len(traversal) < 2 {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Illegal use of Local object",
					Detail:   "The top-level object \"Local\" requires an attribute to specify which local value to access.",
					Subject:  traversal.SourceRange().Ptr(),
				})
				break
			}
			nameStep, ok := traversal[1].(hcl.TraverseAttr)
			if !ok {
				// We'll just fall out here so that we'll later produce our
				// usual message for doing an inappropriate traversal of an
				// object.
				break
			}

			localName := nameStep.Name
			localAttr, exists := mctx.Config.Locals[localName]
			if !exists {
				// We'll just fall out here without setting a value for
				// this local so that we'll produce our usual message for
				// the attribute not existing.
				break
			}

			if len(mctx.DetectVariables(localAttr.Expr)) != 0 {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Illegal use of non-constant value",
					Detail:   "Can only use locals that have constant values in this context.",
					Subject:  traversal.SourceRange().Ptr(),
				})
				// Put in a placeholder value so we won't generate any more
				// errors for this one during evaluation.
				locals[localName] = cty.DynamicVal
			} else {
				localVal, localDiags := mctx.EvalConstant(localAttr.Expr, cty.DynamicPseudoType, NoEachState)
				diags = append(diags, localDiags...)
				locals[localName] = localVal
			}
		default:
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
	scope["Local"] = cty.ObjectVal(locals)
	if each != NoEachState {
		scope["Each"] = cty.ObjectVal(map[string]cty.Value{
			"Key":   each.Key.Value(),
			"Value": each.Value,
		})
	} else {
		scope["Each"] = cty.UnknownVal(cty.Object(map[string]cty.Type{
			"Key":   cty.DynamicPseudoType,
			"Value": cty.DynamicPseudoType,
		}))
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
			Detail:   fmt.Sprintf("This expression is not acceptable: %s.", err),
			Subject:  expr.Range().Ptr(),
		})
		val = cty.NullVal(ty) // Ensure we still return something semi-valid
	}

	return val, diags
}

// DetectVariables is like the global function of the same name, except that
// it additionally follows references to locals and includes them only if
// they transitively refer to variables.
func (mctx *ModuleContext) DetectVariables(expr hcl.Expression) []hcl.Traversal {
	var ret []hcl.Traversal
	traversals := expr.Variables()
	for _, traversal := range traversals {
		switch traversal.RootName() {
		case "Const", "Each":
			// We allow both "Const" and "Each" because const refers to named
			// constants and "Each" is used in ForEach constructs which
			// require their inputs to be constant.
			continue
		case "Local":
			if len(traversal) < 2 {
				break
			}
			nameStep, ok := traversal[1].(hcl.TraverseAttr)
			if !ok {
				break
			}
			localAttr, exists := mctx.Config.Locals[nameStep.Name]
			if !exists {
				break
			}

			transitive := mctx.DetectVariables(localAttr.Expr)
			if len(transitive) == 0 {
				// If the local's expression doesn't depend on any variables
				// then we'll omit it from what we return.
				continue
			}

			// Otherwise, we'll include just the local reference itself,
			// and discard the transitive variables, so the caller can
			// produce reasonable error diagnostics that don't point to
			// expressions outside of the one that was passed in here.
		}

		ret = append(ret, traversal)
	}
	return ret

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
