package schema

//go:generate go run generate_builtin.go

type Schema struct {
	ResourceTypes       map[string]*ResourceType `json:"ResourceTypes"`
	ResourceSpecVersion string                   `json:"ResourceSpecificationVersion"`
	PropertyTypes       map[string]*PropertyType `json:"PropertyTypes"`
}

type ResourceType struct {
	Name          string                `json:"-"`
	Documentation string                `json:"Documentation"`
	Attributes    map[string]*Attribute `json:"Attributes"`
	Properties    map[string]*Property  `json:"Properties"`
}

type PropertyType struct {
	Name          string               `json:"-"`
	ResourceType  *ResourceType        `json:"-"`
	Documentation string               `json:"Documentation"`
	Properties    map[string]*Property `json:"Properties"`
}

type Property struct {
	Name              string     `json:"-"`
	Documentation     string     `json:"Documentation"`
	DuplicatesAllowed bool       `json:"DuplicatesAllowed"`
	Required          bool       `json:"Required"`
	UpdateType        UpdateType `json:"UpdateType"`
	Type
}

type Attribute struct {
	Name string `json:"-"`
	Type
}

type Type struct {
	TypeName          string        `json:"Type"`
	PropertyType      *PropertyType `json:"-"`
	PrimitiveType     PrimitiveType `json:"PrimitiveType"`
	ItemTypeName      string        `json:"ItemType"`
	ItemPropertyType  *PropertyType `json:"-"`
	ItemPrimitiveType PrimitiveType `json:"ItemPrimitiveType"`
}

type PrimitiveType string

const (
	String    PrimitiveType = "String"
	Long      PrimitiveType = "Long"
	Integer   PrimitiveType = "Integer"
	Double    PrimitiveType = "Double"
	Boolean   PrimitiveType = "Boolean"
	Timestamp PrimitiveType = "Timestamp"
)

type UpdateType string

const (
	Mutable     UpdateType = "Mutable"
	Immutable   UpdateType = "Immutable"
	Conditional UpdateType = "Conditional"
)
