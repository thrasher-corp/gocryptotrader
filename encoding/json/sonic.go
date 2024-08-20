//go:build sonic && !(darwin || arm64)

package json

import "github.com/bytedance/sonic"

var (
	Marshal       = sonic.ConfigStd.Marshal
	Unmarshal     = sonic.ConfigStd.Unmarshal
	NewEncoder    = sonic.ConfigStd.NewEncoder
	NewDecoder    = sonic.ConfigStd.NewDecoder
	MarshalIndent = sonic.ConfigStd.MarshalIndent
	Valid         = sonic.ConfigStd.Valid
)
