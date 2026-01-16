package types

import "errors"

var errInvalidBooleanValue = errors.New("invalid value for Boolean type")

// Boolean represents a boolean value, and implements json.Unmarshaller and json.Marshaller
type Boolean bool

// UnmarshalJSON implements json.Unmarshaler, and decerializes boolean representation into bool typed value
func (b *Boolean) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "1", "true", `"true"`:
		*b = Boolean(true)
	case "0", "false", `"false"`:
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
