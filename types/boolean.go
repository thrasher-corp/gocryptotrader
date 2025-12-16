package types

import "errors"

var errInvalidBooleanValue = errors.New("invalid value for Boolean type")

// Boolean represents a boolean value, and implements json.Unmarshaller and json.Marshaller
type Boolean bool

// UnmarshalJSON implements json.Unmarshaler
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

// MarshalJSON implements json.Marshaler
func (b Boolean) MarshalJSON() ([]byte, error) {
	if b {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}

// Bool returns the underlying bool
func (b Boolean) Bool() bool {
	return bool(b)
}
