package cfnjson

import (
	"fmt"

	"github.com/apparentlymart/awsup/eval"
	"github.com/hashicorp/hcl2/hcl"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func PrepareStructure(template *eval.FlatTemplate) (map[string]interface{}, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ret := map[string]interface{}{}

	if template.Description != "" {
		ret["Description"] = template.Description
	}

	if len(template.Parameters) != 0 {
		var paramDiags hcl.Diagnostics
		ret["Parameters"], paramDiags = prepareParameters(template.Parameters)
		diags = append(diags, paramDiags...)
	}

	if len(template.Outputs) != 0 {
		var outputDiags hcl.Diagnostics
		ret["Outputs"], outputDiags = prepareOutputs(template.Outputs)
		diags = append(diags, outputDiags...)
	}

	return ret, diags
}

func prepareParameters(params map[string]*eval.FlatParameter) (map[string]interface{}, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ret := map[string]interface{}{}

	for name, param := range params {
		raw := map[string]interface{}{}

		if param.Type != "" {
			raw["Type"] = param.Type
		}

		if !param.AllowedPattern.IsNull() {
			raw["AllowedPattern"] = ctyjson.SimpleJSONValue{param.AllowedPattern}
		}

		if len(param.AllowedValues) != 0 {
			allowed := make([]interface{}, len(param.AllowedValues))
			for i, val := range param.AllowedValues {
				allowed[i] = ctyjson.SimpleJSONValue{val}
			}
			raw["AllowedValues"] = allowed
		}

		if !param.DefaultValue.IsNull() {
			raw["Default"] = ctyjson.SimpleJSONValue{param.DefaultValue}
		}

		if !param.MinLength.IsNull() {
			raw["MinLength"] = ctyjson.SimpleJSONValue{param.MinLength}
		}
		if !param.MaxLength.IsNull() {
			raw["MaxLength"] = ctyjson.SimpleJSONValue{param.MaxLength}
		}
		if !param.MinValue.IsNull() {
			raw["MinValue"] = ctyjson.SimpleJSONValue{param.MinValue}
		}
		if !param.MaxValue.IsNull() {
			raw["MaxValue"] = ctyjson.SimpleJSONValue{param.MaxValue}
		}

		if !param.NoEcho.IsNull() {
			raw["NoEcho"] = ctyjson.SimpleJSONValue{param.NoEcho}
		}

		ret[name] = raw
	}

	return ret, diags
}

func prepareOutputs(outputs map[string]*eval.FlatOutput) (map[string]interface{}, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ret := map[string]interface{}{}

	for name, output := range outputs {
		raw := map[string]interface{}{}

		var valDiags hcl.Diagnostics
		raw["Value"], valDiags = prepareDynExpr(output.Value)
		diags = append(diags, valDiags...)

		if output.ExportName != nil {
			rawName, nameDiags := prepareDynExpr(output.ExportName)
			diags = append(diags, nameDiags...)
			raw["Export"] = map[string]interface{}{
				"Name": rawName,
			}
		}

		ret[name] = raw
	}

	return ret, diags
}

func prepareDynExpr(expr eval.DynExpr) (interface{}, hcl.Diagnostics) {
	switch te := expr.(type) {

	case *eval.DynLiteral:
		return ctyjson.SimpleJSONValue{te.Value}, nil

	case *eval.DynJoin:
		var diags hcl.Diagnostics
		args := make([]interface{}, 0, len(te.Exprs)+1)
		args = append(args, te.Delimiter)
		for _, se := range te.Exprs {
			subExpr, subDiags := prepareDynExpr(se)
			diags = append(diags, subDiags...)
			args = append(args, subExpr)
		}
		return prepareFuncCall("Fn::Join", args...), diags

	case *eval.DynIf:
		var diags hcl.Diagnostics
		ifRaw, subDiags := prepareDynExpr(te.If)
		diags = append(diags, subDiags...)
		elseRaw, subDiags := prepareDynExpr(te.Else)
		diags = append(diags, subDiags...)
		return prepareFuncCall("Fn::If", ifRaw, elseRaw), diags

	case *eval.DynEquals:
		var diags hcl.Diagnostics
		aRaw, subDiags := prepareDynExpr(te.A)
		diags = append(diags, subDiags...)
		bRaw, subDiags := prepareDynExpr(te.B)
		diags = append(diags, subDiags...)
		return prepareFuncCall("Fn::Equals", aRaw, bRaw), diags

	case *eval.DynLogical:
		panic(fmt.Errorf("DynLogical rendering not yet implemented"))

	case *eval.DynNot:
		panic(fmt.Errorf("DynNot rendering not yet implemented"))

	case *eval.DynSplit:
		strRaw, diags := prepareDynExpr(te.String)
		return prepareFuncCall("Fn::Split", te.Delimiter, strRaw), diags

	case *eval.DynIndex:
		var diags hcl.Diagnostics
		listRaw, subDiags := prepareDynExpr(te.List)
		diags = append(diags, subDiags...)
		indexRaw, subDiags := prepareDynExpr(te.Index)
		diags = append(diags, subDiags...)
		return prepareFuncCall("Fn::Select", indexRaw, listRaw), diags

	case *eval.DynRef:
		return prepareFuncCall("Ref", te.LogicalID), nil

	case *eval.DynGetAttr:
		var diags hcl.Diagnostics
		args := make([]interface{}, 0, len(te.Attrs)+1)
		args = append(args, te.LogicalID)
		for _, se := range te.Attrs {
			subExpr, subDiags := prepareDynExpr(se)
			diags = append(diags, subDiags...)
			args = append(args, subExpr)
		}
		return prepareFuncCall("Fn::GetAtt", args...), diags

	case *eval.DynMappingLookup:
		var diags hcl.Diagnostics
		firstRaw, subDiags := prepareDynExpr(te.FirstKey)
		diags = append(diags, subDiags...)
		secondRaw, subDiags := prepareDynExpr(te.SecondKey)
		diags = append(diags, subDiags...)
		return prepareFuncCall("Fn::FindInMap", te.MappingName, firstRaw, secondRaw), diags

	case *eval.DynBase64:
		strRaw, diags := prepareDynExpr(te.String)
		return prepareFuncCall("Fn::Base64", strRaw), diags

	case *eval.DynAccountAZs:
		regionRaw, diags := prepareDynExpr(te.RegionName)
		return prepareFuncCall("Fn::GetAZs", regionRaw), diags

	default:
		// Should never happen, since the above should be comprehensive
		panic(fmt.Errorf("unsupported dynamic expression type %T", expr))

	}
}

func prepareFuncCall(name string, args ...interface{}) interface{} {
	return map[string]interface{}{name: args}
}
