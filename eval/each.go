package eval

import (
	"github.com/apparentlymart/awsup/addr"
	"github.com/zclconf/go-cty/cty"
)

type EachState struct {
	Key   addr.EachIndex
	Value cty.Value
}

func (s EachState) Enabled() bool {
	return s.Key != addr.NoEachIndex
}

var NoEachState EachState
