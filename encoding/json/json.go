//go:build !sonic

package json

import "encoding/json"

var (
	Marshal       = json.Marshal
	Unmarshal     = json.Unmarshal
	NewEncoder    = json.NewEncoder
	NewDecoder    = json.NewDecoder
	MarshalIndent = json.MarshalIndent
	Valid         = json.Valid
)
