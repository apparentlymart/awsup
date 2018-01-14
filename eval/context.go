package eval

import (
	"fmt"

	"github.com/apparentlymart/awsup/addr"
	"github.com/apparentlymart/awsup/config"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

type RootContext struct {
	// RootModule is a reference to the ModuleContext for the root module.
	RootModule *ModuleContext
}

// NewRootContext creates a RootContext by loading a module configuration
// from the given path (which may be either a directory containing .awsup
// files or a single .awsup file) and then loading the tree of descendent
// modules referenced by the root by following Source values given in
// Module blocks.
//
// If the returned hcl.Diagnostics contains errors then the returned
// context may not be complete, but is still returned to allow for cautious
// use by analysis use-cases such as text editor integrations.
func NewRootContext(parser *config.Parser, rootPath string, constants hcl.Attributes) (*RootContext, hcl.Diagnostics) {
	rootModule, diags := newModuleContext(parser, rootPath, addr.RootModulePath, NoEachState, constants, nil, nil, hcl.Range{})
	return &RootContext{
		RootModule: rootModule,
	}, diags
}

func (ctx *RootContext) VisitModules(cb ModuleVisitor) {
	ctx.RootModule.VisitModules(cb)
}

type ModuleContext struct {
	// Path is the absolute path of the module instance that this context
	// belongs to. This can be used as part of identifiers that need to be
	// globally-unique in the resulting flattened CloudFormation JSON.
	Path addr.ModulePath

	// Root is a reference to the ModuleContext for the root module. The
	// root ModuleContext points to itself.
	Root *ModuleContext

	// Parent is a reference to the ModuleContext for the parent module.
	// The root module has a nil Parent.
	Parent *ModuleContext

	// Children contains references to ModuleContexts for child modules,
	// keyed by the module name given in configuration. Since a single
	// Module block can fan out to many instances with ForEach, the children
	// are accessed through a ModuleEach.
	Children map[string]*ModuleEach

	// Config is the configuration for the module that this context
	// belongs to. A Config should not be modified once it is included
	// in a ModuleContext.
	Config *config.Module

	// Constants is a map of values of all of the named constants
	// for the module.
	Constants map[string]cty.Value
}

func (mctx *ModuleContext) IsRootModule() bool {
	return mctx.Parent == nil
}

func (mctx *ModuleContext) VisitModules(cb ModuleVisitor) {
	cont := cb(mctx)
	if !cont {
		return
	}
	mctx.VisitDownstreamModules(cb)
}

func (mctx *ModuleContext) VisitDownstreamModules(cb ModuleVisitor) {
	for _, eacher := range mctx.Children {
		for _, childCtx := range eacher.Modules {
			childCtx.VisitModules(cb)
		}
	}
}

// ModuleEach represents either a single child ModuleContext or the multiple
// indexed ModuleContexts created when ForEach is used in a module block.
//
// Use IsForEach to determine whether ForEach mode is in use, since this
// dictates which methods may be used on a particular instance.
type ModuleEach struct {
	// EachType is the type of index being used for ForEach on this collection
	// of module instances, or addr.NoEach if ForEach is not in use.
	EachType addr.EachType

	// Modules contains a reference to the ModuleContext for each known index.
	// If not in ForEach mode, this map contains only a single member whose
	// key is addr.NoEachIndex.
	//
	// To iterate over all module instances, use the values of this map
	// and disregard the keys.
	Modules map[addr.EachIndex]*ModuleContext
}

type ModuleVisitor func(*ModuleContext) bool

func newModuleEach(ty addr.EachType) *ModuleEach {
	return &ModuleEach{
		EachType: ty,
		Modules:  make(map[addr.EachIndex]*ModuleContext),
	}
}

// IsForEach returns true if
func (e *ModuleEach) IsForEach() bool {
	return e.EachType != addr.NoEach
}

func (e *ModuleEach) Single() *ModuleContext {
	if e.IsForEach() {
		panic("can't use Single on a ModuleEach for a ForEach module block")
	}
	return e.Modules[addr.NoEachIndex]
}

func (e *ModuleEach) Index(key addr.EachIndex) *ModuleContext {
	if !e.IsForEach() {
		panic("can't use Index on a ModuleEach for a non-ForEach module block")
	}
	if key.EachType() != e.EachType {
		panic(fmt.Errorf("this ModuleEach requires %s, but given %s", e.EachType, key.EachType()))
	}
	return e.Modules[key]
}
