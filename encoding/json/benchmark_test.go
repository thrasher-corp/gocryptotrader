package json

import "testing"

// BenchmarkUnmarshal-16  838503    1282 ns/op   816 B/op  24 allocs/op (encoding/json)
// BenchmarkUnmarshal-16  1859184   653.3 ns/op  900 B/op  18 allocs/op (bytedance/sonic) Usage: go test --tags=sonic -bench=BenchmarkUnmarshal -v
func BenchmarkUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Unmarshal([]byte(`{"Name":"Wednesday","Age":6,"Parents":["Gomez","Morticia"]}`), &map[string]any{})
	}
}
