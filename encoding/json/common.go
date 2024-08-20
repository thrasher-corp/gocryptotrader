package json

import "encoding/json"

type (
	RawMessage         = json.RawMessage
	UnmarshalTypeError = json.UnmarshalTypeError // Assignment as this needs associated methods
)
