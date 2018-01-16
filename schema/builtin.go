package schema

import (
	"strings"
)

func Builtin() *Schema {
	r := strings.NewReader(builtinSource)
	schema, err := Load(r)
	if err != nil {
		// Should never happen, since builtinSource should always be valid
		panic(err)
	}

	return schema
}
