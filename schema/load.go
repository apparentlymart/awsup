package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func Load(r io.Reader) (*Schema, error) {
	dec := json.NewDecoder(r)
	var ret Schema
	err := dec.Decode(&ret)
	if err != nil {
		return nil, err
	}

	for name, rsch := range ret.ResourceTypes {
		rsch.Name = name
		for attrName, attr := range rsch.Attributes {
			attr.Name = attrName
			err := finalizeType(&attr.Type, &ret, name)
			if err != nil {
				return nil, err
			}
		}
		for propName, prop := range rsch.Properties {
			prop.Name = propName
			err := finalizeType(&prop.Type, &ret, name)
			if err != nil {
				return nil, err
			}
		}
	}

	for fullName, asch := range ret.PropertyTypes {
		dotIndex := strings.Index(fullName, ".")
		var resourceTypeName string
		if dotIndex != -1 {
			resourceTypeName = fullName[:dotIndex]
			asch.Name = fullName[dotIndex+1:]
		} else {
			// Some names are just bare names, shared across many resource types
			asch.Name = fullName

		}

		if resourceTypeName != "" {
			asch.ResourceType = ret.ResourceTypes[resourceTypeName]
			if asch.ResourceType == nil {
				return nil, fmt.Errorf("property %s declared for non-existent resource type %q", fullName, resourceTypeName)
			}
		}

		for propName, prop := range asch.Properties {
			prop.Name = propName
			err := finalizeType(&prop.Type, &ret, resourceTypeName)
			if err != nil {
				return nil, err
			}
		}
	}

	return &ret, nil
}

func finalizeType(t *Type, sch *Schema, resourceTypeName string) error {
	if t.TypeName != "" && t.TypeName != "List" && t.TypeName != "Map" {
		t.PropertyType = findPropertyType(resourceTypeName, t.TypeName, sch)
		if t.PropertyType == nil {
			return fmt.Errorf("reference to unknown property type %q for resource type %q", t.TypeName, resourceTypeName)
		}
	}
	if t.ItemTypeName != "" {
		t.ItemPropertyType = findPropertyType(resourceTypeName, t.ItemTypeName, sch)
		if t.ItemPropertyType == nil {
			return fmt.Errorf("reference to unknown property type %q for resource type %q", t.ItemTypeName, resourceTypeName)
		}
	}
	return nil
}

func findPropertyType(resourceTypeName, name string, sch *Schema) *PropertyType {
	key := fmt.Sprintf("%s.%s", resourceTypeName, name)
	ret := sch.PropertyTypes[key]

	if ret == nil {
		// We'll fall back on trying for just the bare name, since in some
		// rare cases (such as "Tag") there are types that are shared across
		// many different resource types.
		ret = sch.PropertyTypes[name]
	}

	return ret
}
