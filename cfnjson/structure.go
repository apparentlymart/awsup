package cfnjson

import (
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
