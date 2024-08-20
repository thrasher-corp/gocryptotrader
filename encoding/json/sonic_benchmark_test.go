//go:build sonic && !(darwin || arm64)

package json

import "testing"

// BenchmarkUnmarshal-16  1900803  631.8 ns/op
// Usage: go test --tags=sonic -bench=BenchmarkUnmarshal -v
func BenchmarkUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Unmarshal([]byte(`{"Name":"Wednesday","Age":6,"Parents":["Gomez","Morticia"]}`), &map[string]interface{}{})
	}
}
