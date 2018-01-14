package addr

import (
	"crypto/sha1"
	"fmt"
)

// NameInModule is a tuple of a module path, a name defined within that module,
// an an optional index into the referenced object for objects that support
// ForEach. The Key field is NoEachIndex when no index is selected, or will
// otherwise be an EachInt or EachString value.
type NameInModule struct {
	Module ModulePath
	Name   string
	Key    EachIndex
}

func (n NameInModule) String() string {
	switch {
	case n.Module.IsRoot():
		if n.Key == NoEachIndex {
			return n.Name
		}
		return fmt.Sprintf("%s[%s]", n.Name, n.Key)
	default:
		if n.Key == NoEachIndex {
			return fmt.Sprintf("%s:%s", n.Module, n.Name)
		}
		return fmt.Sprintf("%s:%s[%s]", n.Module, n.Name, n.Key)
	}
}

// ID returns an opaque string that uniquely identifies the recieving
// qualified name using only alphanumeric characters, suitable for use
// as a resource identifier in CloudFormation template JSON.
//
// The result is not intelligible to humans, so objects using such ids
// should generally be annotated with a human-readable form too so that
// users can map generated objects back onto the source construct that
// created them.
func (n NameInModule) ID() string {
	hash := sha1.Sum([]byte(n.String()))
	return string(hash[:])
}
