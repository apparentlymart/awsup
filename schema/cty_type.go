package schema

import (
	"github.com/zclconf/go-cty/cty"
)

var ctyPrimitiveTypes = map[PrimitiveType]cty.Type{
	String:    cty.String,
	Long:      cty.Number,
	Integer:   cty.Number,
	Double:    cty.Number,
	Boolean:   cty.Bool,
	Timestamp: cty.String,
}

func (t *Type) CtyType() cty.Type {
	if t.PrimitiveType != "" {
		return ctyPrimitiveTypes[t.PrimitiveType]
	}

	switch t.TypeName {
	case "List":
		if t.ItemPrimitiveType != "" {
			return cty.List(ctyPrimitiveTypes[t.PrimitiveType])
		}
		return cty.List(t.ItemPropertyType.CtyType())

	case "Map":
		if t.ItemPrimitiveType != "" {
			return cty.Map(ctyPrimitiveTypes[t.PrimitiveType])
		}
		return cty.Map(t.ItemPropertyType.CtyType())
	}

	return t.PropertyType.CtyType()
}

func (pt *PropertyType) CtyType() cty.Type {
	atys := map[string]cty.Type{}
	for name, prop := range pt.Properties {
		atys[name] = prop.CtyType()
	}
	return cty.Object(atys)
}
