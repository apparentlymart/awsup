package eval

import (
	"github.com/apparentlymart/awsup/addr"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

func (ctx *RootContext) Build() (*FlatTemplate, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ret := &FlatTemplate{
		Metadata:   map[string]cty.Value{},
		Parameters: map[string]*FlatParameter{},
		Mappings:   map[string]map[string]cty.Value{},
		Conditions: map[string]DynExpr{},
		Resources:  map[string]*FlatResource{},
		Outputs:    map[string]*FlatOutput{},
	}
	root := ctx.RootModule

	{
		descVal, descDiags := root.EvalConstant(root.Config.Description, cty.String, NoEachState)
		diags = append(diags, descDiags...)
		if descVal.IsKnown() && !descVal.IsNull() && descVal.Type() == cty.String {
			ret.Description = descVal.AsString()
		}
	}

	for name, param := range root.Config.Parameters {
		if !addr.ValidName(name) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid parameter name",
				Detail:   "Parameter names may contain only alphanumeric characters.",
				Subject:  &param.DeclRange,
			})
		}

		flat := &FlatParameter{
			Type: param.Type,
		}

		valType := paramTypeCtyType(param.Type)

		flat.DefaultValue = evalConstantWithDiags(root, param.Default, valType, NoEachState, &diags)
		flat.AllowedPattern = evalConstantWithDiags(root, param.AllowedPattern, cty.String, NoEachState, &diags)
		if valType != cty.String && !flat.AllowedPattern.IsNull() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Default value not permitted",
				Detail:   "DefaultValue may be set only for parameters of string type.",
				Subject:  param.AllowedPattern.Range().Ptr(),
			})
		}

		rawAllowedVals := evalConstantWithDiags(root, param.AllowedValues, cty.List(valType), NoEachState, &diags)
		if rawAllowedVals.Type().IsListType() && rawAllowedVals.IsKnown() && !rawAllowedVals.IsNull() {
			for it := rawAllowedVals.ElementIterator(); it.Next(); {
				_, val := it.Element()
				flat.AllowedValues = append(flat.AllowedValues, val)
			}
		}

		flat.MinLength = evalConstantWithDiags(root, param.MinLength, cty.Number, NoEachState, &diags)
		if valType != cty.String && !flat.MinLength.IsNull() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Length constraint not permitted",
				Detail:   "MinLength may be set only for parameters of string type.",
				Subject:  param.MinLength.Range().Ptr(),
			})
		}
		flat.MaxLength = evalConstantWithDiags(root, param.MaxLength, cty.Number, NoEachState, &diags)
		if valType != cty.String && !flat.MaxLength.IsNull() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Length constraint not permitted",
				Detail:   "MaxLength may be set only for parameters of string type.",
				Subject:  param.MaxLength.Range().Ptr(),
			})
		}
		flat.MinValue = evalConstantWithDiags(root, param.MinValue, cty.Number, NoEachState, &diags)
		if valType != cty.Number && !flat.MinValue.IsNull() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Value constraint not permitted",
				Detail:   "MinValue may be set only for parameters of number type.",
				Subject:  param.MinValue.Range().Ptr(),
			})
		}
		flat.MaxValue = evalConstantWithDiags(root, param.MaxValue, cty.Number, NoEachState, &diags)
		if valType != cty.Number && !flat.MaxValue.IsNull() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Value constraint not permitted",
				Detail:   "MaxValue may be set only for parameters of number type.",
				Subject:  param.MaxValue.Range().Ptr(),
			})
		}

		flat.NoEcho = evalConstantWithDiags(root, param.Obscure, cty.Bool, NoEachState, &diags)

		ret.Parameters[name] = flat
	}

	for name, output := range root.Config.Outputs {
		if !addr.ValidName(name) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid output name",
				Detail:   "Output names may contain only alphanumeric characters.",
				Subject:  &output.DeclRange,
			})
		}

		flat := &FlatOutput{}
		flat.Value = evalDynamicWithDiags(root, output.Value, NoEachState, &diags)
		if output.Export != nil {
			flat.ExportName = evalDynamicWithDiags(root, output.Export.Name, NoEachState, &diags)
		}

		ret.Outputs[name] = flat
	}

	return ret, diags
}

func evalConstantWithDiags(mctx *ModuleContext, expr hcl.Expression, ty cty.Type, each EachState, diags *hcl.Diagnostics) cty.Value {
	val, newDiags := mctx.EvalConstant(expr, ty, each)
	*diags = append(*diags, newDiags...)
	return val
}

func evalDynamicWithDiags(mctx *ModuleContext, expr hcl.Expression, each EachState, diags *hcl.Diagnostics) DynExpr {
	dynExpr, newDiags := mctx.EvalDynamic(expr, each)
	*diags = append(*diags, newDiags...)
	return dynExpr
}
