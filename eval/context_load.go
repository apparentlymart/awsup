package eval

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/apparentlymart/awsup/addr"
	"github.com/apparentlymart/awsup/config"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

func newModuleContext(parser *config.Parser, srcPath string, path addr.ModulePath, each EachState, inputConstants hcl.Attributes, root, parent *ModuleContext, callRange hcl.Range) (*ModuleContext, hcl.Diagnostics) {
	cfg, diags := parser.ParseDirOrFile(srcPath)
	mctx := &ModuleContext{
		Path:   path,
		Config: cfg,
	}
	if diags.HasErrors() {
		// If we failed during parsing then we'll just bail altogether,
		// though giving the caller the option to poke around in the
		// returned Config object if desired, since it may include some
		// partial information for valid portions of the configuration.
		return mctx, diags
	}

	if parent == nil {
		// If parent is nil then we're the root module, and so we need to set
		// ourselves as our root.
		mctx.Root = mctx
		root = mctx
	} else {
		mctx.Root = root
		mctx.Parent = parent
	}

	constants, constsDiags := buildConstantsTable(cfg.Constants, inputConstants, parent, each, callRange)
	diags = append(diags, constsDiags...)
	mctx.Constants = constants

	// Now that mctx.Constants is set, we can safely use mctx.EvalConstant from
	// this point forward.

	children := make(map[string]*ModuleEach)
	for name, mcfg := range cfg.Modules {
		forEachVal, valDiags := mctx.EvalConstant(mcfg.ForEach, cty.DynamicPseudoType, NoEachState)
		diags = append(diags, valDiags...)
		if valDiags.HasErrors() {
			// Can't process any further if we can't evaluate ForEach
			continue
		}
		forEachType := forEachVal.Type()

		path := path.AppendName(name)

		switch {
		case forEachVal.IsNull():
			children[name] = newModuleEach(addr.NoEach)
			childCtx, childDiags := mctx.childModuleContext(parser, path, mcfg, NoEachState)
			diags = append(diags, childDiags...)
			children[name].Modules[addr.NoEachIndex] = childCtx
		case forEachType.IsSetType():
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Incorrect value type",
				Detail:   "A set value cannot be used as a ForEach interator.",
				Subject:  mcfg.ForEach.StartRange().Ptr(),
			})
			continue
		case forEachType.IsCollectionType() || forEachType.IsObjectType() || forEachType.IsTupleType():
			switch {
			case forEachType.IsListType() || forEachType.IsTupleType():
				children[name] = newModuleEach(addr.EachTypeInt)
			case forEachType.IsMapType() || forEachType.IsObjectType():
				children[name] = newModuleEach(addr.EachTypeString)
			default:
				// should never happen since we've now covered all of the
				// collection types
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Incorrect value type",
					Detail:   fmt.Sprintf("A %s value cannot be used as a ForEach interator.", forEachType.FriendlyName()),
					Subject:  mcfg.ForEach.StartRange().Ptr(),
				})
				continue
			}

			for it := forEachVal.ElementIterator(); it.Next(); {
				keyVal, val := it.Element()
				key := addr.MakeEachIndex(keyVal)
				path := path.AppendIndex(key)
				each := EachState{
					Key:   key,
					Value: val,
				}
				childCtx, childDiags := mctx.childModuleContext(parser, path, mcfg, each)
				diags = append(diags, childDiags...)
				if childCtx == nil {
					// The content of the config block was so broken that we
					// weren't able to construct any context.
					continue
				}
				children[name].Modules[key] = childCtx
			}
		default:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Incorrect value type",
				Detail:   fmt.Sprintf("A %s value cannot be used as a ForEach interator.", forEachType.FriendlyName()),
				Subject:  mcfg.ForEach.StartRange().Ptr(),
			})
			continue
		}
	}

	mctx.Children = children

	return mctx, diags
}

func (mctx *ModuleContext) childModuleContext(parser *config.Parser, path addr.ModulePath, cfg *config.ModuleCall, each EachState) (*ModuleContext, hcl.Diagnostics) {
	// This method is called while mctx is still being constructed, so
	// mctx.Config, mctx.Root, and mctx.Constantsare the only fields safe to
	// access. mctx.EvalConstant uses only mctx.Constants and so it is also
	// safe to use in here.

	var diags hcl.Diagnostics

	basePath := mctx.Config.SourceDir
	if basePath == "" {
		// An empty basePath indicates that a module was loaded from a
		// synthetic source, such as an in-memory buffer (e.g. for unit testing).
		// basePath should never be empty in normal CLI usage.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Child modules not allowed",
			Detail:   "The current module was not loaded from an on-disk path, so child module references cannot be resolved.",
			Subject:  cfg.Source.Range().Ptr(),
		})
		return nil, diags
	}

	srcPathVal, srcDiags := mctx.EvalConstant(cfg.Source, cty.String, each)
	diags = append(diags, srcDiags...)
	if srcDiags.HasErrors() {
		// We can't proceed any further if we don't have a valid source
		return nil, diags
	}

	if srcPathVal.IsNull() {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unspecified module source",
			Detail:   "Child module declaration is missing the required attribute \"Source\".",
			Subject:  &cfg.DeclRange,
		})
		return nil, diags
	}

	srcPath := srcPathVal.AsString()
	if !(strings.HasPrefix(srcPath, "./") || strings.HasPrefix(srcPath, "../")) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid child module source path",
			Detail:   "A child module source must be a relative path beginning with either \"./\" or \"../\".",
			Subject:  cfg.Source.Range().Ptr(),
		})
		return nil, diags
	}
	srcPath = filepath.Join(basePath, srcPath)

	childCtx, childDiags := newModuleContext(parser, srcPath, path, each, cfg.Constants, mctx.Root, mctx, cfg.DeclRange)
	diags = append(diags, childDiags...)
	return childCtx, diags
}

func buildConstantsTable(cfgs map[string]*config.Constant, input hcl.Attributes, parent *ModuleContext, each EachState, callRange hcl.Range) (map[string]cty.Value, hcl.Diagnostics) {
	table := map[string]cty.Value{}
	var diags hcl.Diagnostics

	// If we're working on the root module, this is signalled by our range
	// being the zero value.
	inRoot := callRange == hcl.Range{}

	for name, cfg := range cfgs {
		attr, isSet := input[name]
		if !isSet {
			val, valDiags := cfg.Default.Value(nil)
			diags = append(diags, valDiags...)
			if val.IsNull() {
				if inRoot {
					// Root constants are expected to come from the CLI and
					// thus a different diagnostic message is warranted.
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Required root constant not set",
						Detail:   fmt.Sprintf("The root module requires a value for its named constant %q. Set it in a file passed with the --constants argument.", name),
					})
				} else {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Missing required constant for module",
						Detail:   fmt.Sprintf("This module requires a value for its named constant %q.", name),
						Subject:  &callRange,
					})
				}
			}
			table[name] = val
			continue
		}

		var val cty.Value
		var valDiags hcl.Diagnostics
		if parent != nil {
			val, valDiags = parent.EvalConstant(attr.Expr, cty.DynamicPseudoType, each)
		} else {
			// For the root module we're evaluating expressions from a
			// constants definition file provided on the command line, so
			// we don't allow any variables here.
			val, valDiags = attr.Expr.Value(nil)
		}
		diags = append(diags, valDiags...)

		// Ensure we don't get unknown values in our table, even if an error
		// causes Value to return DynamicVal; a constant value must always
		// be known.
		if !val.IsKnown() {
			val = cty.NullVal(val.Type())
		}
		table[name] = val
	}

	// Detect any extraneous constants set in the input
	for name, attr := range input {
		if _, isAllowed := cfgs[name]; !isAllowed {
			if inRoot {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported root module constant",
					Detail:   fmt.Sprintf("The root module does not expect a constant named %q.", name),
					Subject:  &attr.NameRange,
				})
			} else {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported module constant",
					Detail:   fmt.Sprintf("This child module does not expect a constant named %q.", name),
					Subject:  &attr.NameRange,
				})
			}
		}
	}

	return table, diags
}
