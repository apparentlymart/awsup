// Code generated by "stringer -type EachType"; DO NOT EDIT.

package addr

import "strconv"

const (
	_EachType_name_0 = "NoEach"
	_EachType_name_1 = "EachTypeInt"
	_EachType_name_2 = "EachTypeString"
)

func (i EachType) String() string {
	switch {
	case i == 0:
		return _EachType_name_0
	case i == 105:
		return _EachType_name_1
	case i == 115:
		return _EachType_name_2
	default:
		return "EachType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}