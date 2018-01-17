package eval

import (
	"github.com/apparentlymart/awsup/addr"
	"github.com/apparentlymart/awsup/config"
	"github.com/apparentlymart/awsup/schema"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

// TypeCheck verifies the internal type consistency of the given expression and
// then, if successful, returns the expression's own result type.
//
// TypeCheck relies on the type rules as defined by HCL and so does not
// enforce the additional constraints that apply when lowering to CloudFormation
// dynamic expressions using EvalDynamic, which arise from the limitations
// of the CloudFormation expression language.
//
// The result may be cty.DynamicPseudoType if insufficient information is
// available to produce a result, which can occur if there are inconsistencies
// or errors elsewhere in the configuration. For best results, ensure that
// all of the referenceable configuration constructs are correct before
// type-checking references to them. For example, we assume that a prior check
// has detected resources with invalid type names and reported them, and so
// TypeCheck will treat these as being DynamicPseudoType.
//
// If any type inconsistencies are found then they are returned as error
// diagnostics. The returned type is always valid, but may not be accurate
// in the precence of error diagnostics. cty.DynamicPseudoType is returned
// if errors prevent type resolution altogether.
func (mctx *ModuleContext) TypeCheck(expr hcl.Expression, each EachState) (cty.Type, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	traversals := expr.Variables()
	locals := map[string]cty.Value{}
	modules := map[string]cty.Value{}
	resources := map[string]cty.Value{}
	params := map[string]cty.Value{}

	// The methodology here is to actually evaluate the _value_ of the given
	// expression, but to do it in a scope where dynamic expressions are
	// represented as unknown values of a suitable type. That way the type
	// information propagates through the expression, usually resulting in
	// an unknown value of some type as the result.
	// We then discard the actual value and return only the type.

	for _, tr := range traversals {
		switch tr.RootName() {

		case "Const":
			// Allowed but no special action required because the entire constant
			// table is included in the scope below.

		case "Each":
			if each == NoEachState {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Illegal use of \"Each\" object",
					Detail:   "The \"Each\" object can be accessed only within modules and resources that have ForEach set.",
					Subject:  tr.SourceRange().Ptr(),
				})
			}
			// No special value is required other than the check above, because
			// we always place the "Each" object in the scope below.

		case "Local":
			if len(tr) < 2 {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Illegal use of Local object",
					Detail:   "The top-level object \"Local\" requires an attribute to specify which local value to access.",
					Subject:  tr.SourceRange().Ptr(),
				})
				break
			}
			nameStep, ok := tr[1].(hcl.TraverseAttr)
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
				// this resource so that we'll produce our usual message for
				// the attribute not existing.
				break
			}

			// We intentionally discard diagnostics here because we assume
			// that the caller will check the local value expressions
			// individually and report the errors in them, and we don't want
			// to repeat the same error diagnostics multiple times.
			localTy, _ := mctx.TypeCheck(localAttr.Expr, NoEachState)
			locals[localName] = cty.UnknownVal(localTy)

		case "Module":
			if len(tr) < 2 {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Illegal use of Module object",
					Detail:   "The top-level object \"Module\" requires an attribute to specify which module to access.",
					Subject:  tr.SourceRange().Ptr(),
				})
				break
			}
			nameStep, ok := tr[1].(hcl.TraverseAttr)
			if !ok {
				// We'll just fall out here so that we'll later produce our
				// usual message for doing an inappropriate traversal of an
				// object.
				break
			}

			modName := nameStep.Name
			childEach, exists := mctx.Children[modName]
			if !exists {
				// We'll just fall out here without setting a value for
				// this module so that we'll produce our usual message for
				// the attribute not existing.
				break
			}

			switch childEach.EachType {
			case addr.NoEach:
				childMctx := childEach.Single()
				modules[modName] = moduleObjectPlaceholder(childMctx)
			case addr.EachTypeInt:
				// We have a map here but we assume that in EachTypeInt
				// we will always have consecutive indices starting at zero.
				instances := make([]cty.Value, len(childEach.Modules))
				for i := 0; i < len(childEach.Modules); i++ {
					childMctx := childEach.Modules[addr.EachInt(i)]
					if childMctx == nil {
						// Should never happen
						instances[i] = cty.DynamicVal
						continue
					}
					instances[i] = moduleObjectPlaceholder(childMctx)
				}
				modules[modName] = cty.TupleVal(instances)
			case addr.EachTypeString:
				instances := map[string]cty.Value{}
				for key, childMctx := range childEach.Modules {
					instances[string(key.(addr.EachString))] = moduleObjectPlaceholder(childMctx)
				}
				modules[modName] = cty.ObjectVal(instances)
			}

		case "Resource":
			if len(tr) < 2 {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Illegal use of Resource object",
					Detail:   "The top-level object \"Resource\" requires an attribute to specify which resource to access.",
					Subject:  tr.SourceRange().Ptr(),
				})
				break
			}
			nameStep, ok := tr[1].(hcl.TraverseAttr)
			if !ok {
				// We'll just fall out here so that we'll later produce our
				// usual message for doing an inappropriate traversal of an
				// object.
				break
			}

			logicalId := nameStep.Name
			rcfg, exists := mctx.Config.Resources[logicalId]
			if !exists {
				// We'll just fall out here without setting a value for
				// this resource so that we'll produce our usual message for
				// the attribute not existing.
				break
			}

			typeName := rcfg.Type
			rsch, exists := mctx.Global.Schema.ResourceTypes[typeName]
			if !exists {
				// We'll assume that a separate explicit check will detect
				// and report references to non-existant types, so for our
				// purposes here we'll just stub out the object to allow
				// type checking to complete.
				resources[logicalId] = cty.DynamicVal
				break
			}

			resources[logicalId] = resourceObjectPlaceholder(rsch)

		case "Param":
			if len(tr) < 2 {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Illegal use of Param object",
					Detail:   "The top-level object \"Param\" requires an attribute to specify which parameter to access.",
					Subject:  tr.SourceRange().Ptr(),
				})
				break
			}
			nameStep, ok := tr[1].(hcl.TraverseAttr)
			if !ok {
				// We'll just fall out here so that we'll later produce our
				// usual message for doing an inappropriate traversal of an
				// object.
				break
			}

			paramName := nameStep.Name
			param, exists := mctx.Config.Parameters[paramName]
			if !exists {
				// We'll just fall out here without setting a value for
				// this parameter so that we'll produce our usual message for
				// the attribute not existing.
				break
			}

			params[paramName] = paramPlaceholder(param)

		default:
			// We don't take any special action for unrecognized root names,
			// because by omitting them from the scope we'll get good errors
			// for them during evaluation.
		}
	}

	scope := map[string]cty.Value{
		"Const":    cty.ObjectVal(mctx.Constants),
		"Each":     eachObject(each),
		"Local":    cty.ObjectVal(locals),
		"Module":   cty.ObjectVal(modules),
		"Resource": cty.ObjectVal(resources),
		"Param":    cty.ObjectVal(params),
		// TODO: "Condition"
		// TODO: "Mapping"
	}

	ectx := &hcl.EvalContext{
		Variables: scope,
		// TODO: Once we have functions, include those in here too
	}

	val, valDiags := expr.Value(ectx)
	diags = append(diags, valDiags...)

	return val.Type(), diags
}

func resourceObjectPlaceholder(rsch *schema.ResourceType) cty.Value {
	attrs := map[string]cty.Value{}
	for name, attr := range rsch.Attributes {
		attrs[name] = cty.UnknownVal(attr.CtyType())
	}
	return cty.ObjectVal(attrs)
}

func moduleObjectPlaceholder(mctx *ModuleContext) cty.Value {
	outputs := mctx.Config.Outputs
	attrs := map[string]cty.Value{}
	for name, output := range outputs {
		// We ignore diagnostics here because we expect caller will
		// check each output separately and so any errors will already
		// be reported.
		ty, _ := mctx.TypeCheck(output.Value, NoEachState)
		attrs[name] = cty.UnknownVal(ty)
	}
	return cty.ObjectVal(attrs)
}

func paramPlaceholder(param *config.Parameter) cty.Value {
	return cty.UnknownVal(paramTypeCtyType(param.Type))
}

func paramTypeCtyType(name string) cty.Type {
	// Parameters support a weird assortment of special type strings, along
	// with a number of service-specific types that seem to all just be
	// strings of a specific syntax. Therefore we'll cover the weird special
	// ones and then just treat everything else as a string.
	switch name {

	case "String":
		return cty.String

	case "Number":
		// CloudFormation actually converts numbers to strings when returning
		// them, but we'll say Number here to avoid quirky results when
		// we try to use number params in contexts where HCL really expects
		// a number. We expect CloudFormation to be able to convert the
		// stringified number back into a number when needed anyway.
		return cty.Number

	case "List<Number>":
		// Again this actually comes out as a list of strings on the other side,
		// but we treat it as cty number for the same reason as for "Number"
		// above.
		return cty.List(cty.Number)

	case "CommaDelimitedList":
		return cty.List(cty.String)

	default:
		return cty.String

	}
}
