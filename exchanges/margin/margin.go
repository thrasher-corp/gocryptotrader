package margin

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Valid returns whether the margin type is valid
func (t Type) Valid() bool {
	return t != Unset && supported&t == t
}

// UnmarshalJSON converts json into margin type
func (t *Type) UnmarshalJSON(d []byte) error {
	var marginType string
	err := json.Unmarshal(d, &marginType)
	if err != nil {
		return err
	}
	*t, err = StringToMarginType(marginType)
	return err
}

// String returns the string representation of the margin type in lowercase
// the absence of a lower func should hopefully highlight that String is lower
func (t Type) String() string {
	switch t {
	case Unset:
		return unsetStr
	case Isolated:
		return isolatedStr
	case Multi:
		return multiStr
	case Unknown:
		return unknownStr
	}
	return ""
}

// Upper returns the upper case string representation of the margin type
func (t Type) Upper() string {
	return strings.ToUpper(t.String())
}

// IsValidString checks to see if the supplied string is a valid margin type
func IsValidString(m string) bool {
	switch strings.ToLower(m) {
	case isolatedStr, multiStr, unsetStr, crossedStr, crossStr:
		return true
	}
	return false
}

// StringToMarginType converts a string to a margin type
// doesn't error, just returns unknown if the string is not recognised
func StringToMarginType(m string) (Type, error) {
	switch strings.ToLower(m) {
	case isolatedStr:
		return Isolated, nil
	case multiStr, crossedStr, crossStr:
		return Multi, nil
	case "":
		return Unset, nil
	}
	return Unknown, fmt.Errorf("%w %v", ErrInvalidMarginType, m)
}
