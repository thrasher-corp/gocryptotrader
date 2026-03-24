package types

import "errors"

var errInvalidBooleanValue = errors.New("invalid value for Boolean type")

// Boolean represents a boolean value and implements json.UnmarshalJSON
type Boolean bool

// UnmarshalJSON implements json.Unmarshaler and converts the JSON boolean representation into a Boolean type
func (b *Boolean) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "1", `"1"`, "true", `"true"`:
		*b = Boolean(true)
	case "0", `"0"`, "false", `"false"`:
		*b = Boolean(false)
	default:
		return errInvalidBooleanValue
	}
	return nil
}

// Bool returns the underlying bool
func (b Boolean) Bool() bool {
	return bool(b)
}
