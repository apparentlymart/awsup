package addr

import (
	"math"
	"math/big"
	"strconv"

	"github.com/zclconf/go-cty/cty"
)

// EachIndex represents an index for an instance of a configuration structure
// that supports ForEach.
//
// EachInt and EachString represent integer and string indices respectively.
// Which type is used depends on whether ForEach is assigned a list or a map.
//
// A nil EachIndex represents no index at all, which us used when ForEach
// is not set.
type EachIndex interface {
	EachType() EachType
	String() string
}

// MakeEachIndex takes a cty.Value of either cty.Number or cty.String and
// produces the equivalent EachIndex.
//
// This function will return NoEachIndex if the given value is not of a
// suitable type or if it not convertable to an int. It will panic
// if the value is unknown or null.
func MakeEachIndex(val cty.Value) EachIndex {
	switch val.Type() {
	case cty.String:
		return EachString(val.AsString())
	case cty.Number:
		bf := val.AsBigFloat()
		i, acc := bf.Int64()
		if acc != big.Exact {
			return NoEachIndex
		}
		if strconv.IntSize == 32 && i > math.MaxInt32 {
			return NoEachIndex
		}
		return EachInt(i)
	default:
		return NoEachIndex
	}
}

// NoIndex is a nil value of type EachIndex that is used when ForEach is
// not in use, to represent the absense of an index.
var NoEachIndex = EachIndex(nil)

type EachInt int

func (i EachInt) EachType() EachType {
	return EachTypeInt
}

func (i EachInt) String() string {
	return strconv.Itoa(int(i))
}

type EachString string

func (s EachString) EachType() EachType {
	return EachTypeString
}

func (s EachString) String() string {
	return strconv.Quote(string(s))
}

type EachType rune

//go:generate stringer -type EachType

const (
	NoEach         EachType = 0
	EachTypeInt    EachType = 'i'
	EachTypeString EachType = 's'
)
