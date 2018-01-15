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
