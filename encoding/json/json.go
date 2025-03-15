//go:build !sonic_on

package json

import "encoding/json" //nolint:depguard // Acceptable use in gct json wrapper

// Implementation is a constant string that represents the current JSON implementation package
const Implementation = "encoding/json"

var (
	// Marshal returns the JSON encoding of v. See the "encoding/json" documentation for Marshal
	Marshal = json.Marshal
	// Unmarshal parses the JSON-encoded data and stores the result in the value pointed to by v. See the "encoding/json" documentation for Unmarshal
	Unmarshal = json.Unmarshal
	// NewEncoder returns a new encoder that writes to w. See the "encoding/json" documentation for NewEncoder
	NewEncoder = json.NewEncoder
	// NewDecoder returns a new decoder that reads from r. See the "encoding/json" documentation for NewDecoder
	NewDecoder = json.NewDecoder
	// MarshalIndent is like Marshal but applies Indent to format the output. See the "encoding/json" documentation for MarshalIndent
	MarshalIndent = json.MarshalIndent
	// Valid reports whether data is a valid JSON encoding. See the "encoding/json" documentation for Valid
	Valid = json.Valid
)
