package eval

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// EvalDynamic evaluates the given expression to produce a DynExpr, which
// can then be serialized as a value in CloudFormation JSON.
//
// If EachState is set to anything other than NoEachState then the "Each"
// object is also available for use, exposing the values in the given EachState.
func (mctx *ModuleContext) EvalDynamic(expr hcl.Expression, each EachState) (DynExpr, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	switch te := expr.(type) {

	case *hclsyntax.LiteralValueExpr:
		return &DynLiteral{
			Value:    te.Val,
			SrcRange: te.SrcRange,
		}, diags

	case *hclsyntax.ScopeTraversalExpr:
		return mctx.evalVariableDynamic(te, each)

	case *hclsyntax.RelativeTraversalExpr:
		start, startDiags := mctx.EvalDynamic(te.Source, each)
		diags = append(diags, startDiags...)
		final, finalDiags := mctx.evalTraversalDynamic(start, te.Traversal, each)
		diags = append(diags, finalDiags...)
		return final, diags

	case *hclsyntax.TemplateExpr:
		parts := make([]DynExpr, len(te.Parts))
		for i, partExpr := range te.Parts {
			var partDiags hcl.Diagnostics
			parts[i], partDiags = mctx.EvalDynamic(partExpr, each)
			diags = append(diags, partDiags...)
		}
		if len(parts) == 1 {
			return parts[0], diags
		}
		return &DynJoin{
			Delimiter: "",
			Exprs:     parts,
			SrcRange:  te.SrcRange,
		}, diags

	case *hclsyntax.IndexExpr:
		// TODO: Verify that the collection is a list and error if not,
		// since CloudFormation only supports indexing of lists.
		index, indexDiags := mctx.EvalDynamic(te.Key, each)
		diags = append(diags, indexDiags...)
		coll, collDiags := mctx.EvalDynamic(te.Collection, each)
		diags = append(diags, collDiags...)

		return &DynIndex{
			List:  coll,
			Index: index,

			SrcRange: te.SrcRange,
		}, diags

	case *hclsyntax.BinaryOpExpr:
		switch te.Op {
		case hclsyntax.OpLogicalAnd, hclsyntax.OpLogicalOr:
			var opDiags hcl.Diagnostics
			lhs, opDiags := mctx.EvalDynamic(te.LHS, each)
			diags = append(diags, opDiags...)
			rhs, opDiags := mctx.EvalDynamic(te.RHS, each)
			diags = append(diags, opDiags...)

			var op DynLogicalOp
			if te.Op == hclsyntax.OpLogicalAnd {
				op = DynLogicalAnd
			} else {
				op = DynLogicalOr
			}

			// We're making some additional effort here to flatten down
			// nested expressions into a single node, since that produces
			// a more compact final CloudFormation template.
			var values []DynExpr
			if tlhs, ok := lhs.(*DynLogical); ok && tlhs.Op == op {
				values = append(values, tlhs.Values...)
			} else {
				values = append(values, lhs)
			}
			if trhs, ok := rhs.(*DynLogical); ok && trhs.Op == op {
				values = append(values, trhs.Values...)
			} else {
				values = append(values, lhs)
			}

			return &DynLogical{
				Op:       op,
				Values:   values,
				SrcRange: te.SrcRange,
			}, diags
		case hclsyntax.OpEqual, hclsyntax.OpNotEqual:
			var opDiags hcl.Diagnostics
			lhs, opDiags := mctx.EvalDynamic(te.LHS, each)
			diags = append(diags, opDiags...)
			rhs, opDiags := mctx.EvalDynamic(te.RHS, each)
			diags = append(diags, opDiags...)

			var ret DynExpr
			ret = &DynEquals{
				A:        lhs,
				B:        rhs,
				SrcRange: te.SrcRange,
			}

			if te.Op == hclsyntax.OpNotEqual {
				// CloudFormation doesn't have a "not equal" test, so we'll
				// just wrap a "not" expression around.
				ret = &DynNot{
					Value:    ret,
					SrcRange: te.SrcRange,
				}
			}

			return ret, diags
		}
	}

	// If we encounter an expression we don't know how to deal with then
	// we'll fall out here and try to evaluate it as a constant to see if
	// we can make a DynLiteral to represent it. This means that the full
	// language can be used for constants, even though only a small subset of
	// it can be used for dynamic values, at the expense of a more
	// confusing user experience as we force the user to puzzle out
	// the relationship between an unsupported operation and a
	// deeply-nested refence to a variable.

	variables := mctx.DetectVariables(expr)
	if len(variables) != 0 {
		// We return a specialized error here that marks the whole
		// expression as problematic, as a small way to try to help the
		// user understand what's going on and how to address it.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Illegal use of non-constant value",
			Detail:   fmt.Sprintf("This expression type is not supported by CloudFormation, so only constant values are permitted and the result value will be hard-coded into the generated template. A non-constant value is referenced at %s.", variables[0].SourceRange()),
			Subject:  expr.Range().Ptr(), // Intentionally the whole expression rather than just the erroneous traversal
		})
		return &DynLiteral{
			Value:    cty.NullVal(cty.DynamicPseudoType),
			SrcRange: expr.Range(),
		}, diags
	}

	val, valDiags := mctx.EvalConstant(expr, cty.DynamicPseudoType, each)
	diags = append(diags, valDiags...)
	return &DynLiteral{
		Value:    val,
		SrcRange: expr.Range(),
	}, diags

}

func (mctx *ModuleContext) evalVariableDynamic(expr *hclsyntax.ScopeTraversalExpr, each EachState) (DynExpr, hcl.Diagnostics) {
	traversal := expr.Traversal
	var diags hcl.Diagnostics
	switch traversal.RootName() {

	case "Const", "Each":
		val, valDiags := mctx.EvalConstant(expr, cty.DynamicPseudoType, each)
		diags = append(diags, valDiags...)
		return &DynLiteral{
			Value:    val,
			SrcRange: expr.SrcRange,
		}, diags

	case "Local":
		var nameStepI hcl.Traverser
		if len(traversal) >= 2 {
			nameStepI = traversal[1]
		}
		nameStep, ok := nameStepI.(hcl.TraverseAttr)
		if !ok {
			if len(traversal) < 2 {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Illegal use of Local object",
					Detail:   "The Local object requires an attribute to select a specific named local value.",
					Subject:  traversal.SourceRange().Ptr(),
				})
				return &DynLiteral{
					Value:    cty.DynamicVal,
					SrcRange: traversal.SourceRange(),
				}, diags
			}
		}

		name := nameStep.Name
		local, exists := mctx.Config.Locals[name]
		if !exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unknown local value",
				Detail:   fmt.Sprintf("There is no local value named %q.", name),
				Subject:  &nameStep.SrcRange,
			})
			return &DynLiteral{
				Value:    cty.DynamicVal,
				SrcRange: nameStep.SrcRange,
			}, diags
		}

		subExpr := local.Expr
		vars := mctx.DetectVariables(subExpr)
		if len(vars) == 0 {
			// If the local value is constant-only then we'll evaluate it here
			// and return its literal value.
			val, valDiags := mctx.EvalConstant(subExpr, cty.DynamicPseudoType, each)
			diags = append(diags, valDiags...)
			return &DynLiteral{
				Value:    val,
				SrcRange: expr.SrcRange,
			}, diags
		}

		// If the value contains dynamic-only constructs then we'll fall out
		// here and try to incorporate its expression int ours.
		dynExpr, dynDiags := mctx.EvalDynamic(subExpr, each)
		diags = append(diags, dynDiags...)
		return dynExpr, diags

	default:
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unknown object",
			Detail:   fmt.Sprintf("There is no object named %q.", traversal.RootName()),
			Subject:  traversal[0].SourceRange().Ptr(),
		})
		return &DynLiteral{
			Value:    cty.DynamicVal,
			SrcRange: traversal[0].SourceRange(),
		}, diags
	}
}

func (mctx *ModuleContext) evalTraversalDynamic(start DynExpr, traversal hcl.Traversal, each EachState) (DynExpr, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	expr := start
Steps:
	for _, rawStep := range traversal {
		switch step := rawStep.(type) {
		case hcl.TraverseRoot:
			panic("can't use absolute traversal with evalTraversalDynamic")
		case hcl.TraverseIndex:
			expr = &DynIndex{
				List: expr,
				Index: &DynLiteral{
					Value:    step.Key,
					SrcRange: step.SrcRange,
				},
			}
		case hcl.TraverseAttr:
			// For variables that _do_ have attributes we'll handle them
			// in evalVariableDynamic before we pass off the remaining
			// traversal to this function, so this is always an error here.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsupported attribute",
				Detail:   "This value does not have any attributes.",
				Subject:  &step.SrcRange,
			})
			expr = &DynLiteral{
				Value:    cty.DynamicVal,
				SrcRange: step.SrcRange,
			}
			break Steps
		case hcl.TraverseSplat:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Splat expression not supported",
				Detail:   "This value does not support splat expressions.",
				Subject:  &step.SrcRange,
			})
			expr = &DynLiteral{
				Value:    cty.DynamicVal,
				SrcRange: step.SrcRange,
			}
			break Steps
		default:
			// Shouldn't happen but we'll produce a generic message just in
			// case HCL adds new traversal steps in future without us noticing.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsupported operation",
				Detail:   "This value does not support this operation.",
				Subject:  rawStep.SourceRange().Ptr(),
			})
			expr = &DynLiteral{
				Value:    cty.DynamicVal,
				SrcRange: rawStep.SourceRange(),
			}
			break Steps
		}
	}
	return expr, diags
}
