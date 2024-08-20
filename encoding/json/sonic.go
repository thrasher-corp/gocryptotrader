//go:build sonic

package json

import (
	"github.com/bytedance/sonic"
)

var (
	Marshal       = sonic.ConfigStd.Marshal
	Unmarshal     = sonic.ConfigStd.Unmarshal
	NewEncoder    = sonic.ConfigStd.NewEncoder
	NewDecoder    = sonic.ConfigStd.NewDecoder
	MarshalIndent = sonic.ConfigStd.MarshalIndent
	Valid         = sonic.ConfigStd.Valid
)
