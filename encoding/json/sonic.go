//go:build sonic_on

package json

import (
	"github.com/bytedance/sonic"
)

// Implementation is a constant string that represents the current JSON implementation package
const Implementation = "bytedance/sonic"

var (
	// Marshal returns the JSON encoding of v. See the "github.com/bytedance/sonic" documentation for Marshal
	Marshal = sonic.ConfigStd.Marshal
	// Unmarshal parses the JSON-encoded data and stores the result in the value pointed to by v. See the "github.com/bytedance/sonic" documentation for Unmarshal
	Unmarshal = sonic.ConfigStd.Unmarshal
	// NewEncoder returns a new encoder that writes to w. See the "github.com/bytedance/sonic" documentation for NewEncoder
	NewEncoder = sonic.ConfigStd.NewEncoder
	// NewDecoder returns a new decoder that reads from r. See the "github.com/bytedance/sonic" documentation for NewDecoder
	NewDecoder = sonic.ConfigStd.NewDecoder
	// MarshalIndent is like Marshal but applies Indent to format the output. See the "github.com/bytedance/sonic" documentation for MarshalIndent
	MarshalIndent = sonic.ConfigStd.MarshalIndent
	// Valid reports whether data is a valid JSON encoding. See the "github.com/bytedance/sonic" documentation for Valid
	Valid = sonic.ConfigStd.Valid
)
