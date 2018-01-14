package addr

import (
	"bytes"
	"fmt"
)

type ModulePath []ModulePathStep

type ModulePathStep interface {
	modulePathStep()
}

type modulePathName string

func (n modulePathName) modulePathStep() {
}

type modulePathIndex struct {
	EachIndex
}

func (n modulePathIndex) modulePathStep() {
}

// RootModulePath is a ModulePath representing the root module
var RootModulePath ModulePath

func (m ModulePath) String() string {
	var buf bytes.Buffer
	for _, rawStep := range m {
		switch step := rawStep.(type) {
		case modulePathName:
			buf.WriteByte('.')
			buf.WriteString(string(step))
		case modulePathIndex:
			buf.WriteByte('[')
			buf.WriteString(step.String())
			buf.WriteByte(']')
		default:
			// should never happen since we ensure no other step types are
			// included in ParseModulePath
			panic(fmt.Errorf("unsupported %T step in ModulePath traversal", rawStep))
		}
	}
	return buf.String()
}

func (m ModulePath) append(step ModulePathStep) ModulePath {
	new := make(ModulePath, len(m), len(m)+1)
	copy(new, m)
	new = append(new, step)
	return new
}

func (m ModulePath) AppendName(name string) ModulePath {
	return m.append(modulePathName(name))
}

func (m ModulePath) AppendIndex(key EachIndex) ModulePath {
	return m.append(modulePathIndex{key})
}

func (m ModulePath) Parent() ModulePath {
	if len(m) == 0 {
		return RootModulePath
	}
	path := m
	for path = path[:len(path)-1]; len(path) > 0; path = path[:len(path)-1] {
		if _, ok := path[len(path)-1].(modulePathName); ok {
			return path
		}
	}
	return RootModulePath
}

func (m ModulePath) NearestName() ModulePath {
	path := m
	if len(path) == 0 {
		return RootModulePath
	}
	for ; len(path) > 0; path = path[:len(path)-1] {
		if _, ok := path[len(path)-1].(modulePathName); ok {
			return path
		}
	}
	return RootModulePath
}

func (m ModulePath) IsRoot() bool {
	return len(m) == 0
}
