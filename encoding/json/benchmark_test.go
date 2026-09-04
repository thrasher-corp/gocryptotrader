package json

import "testing"

// BenchmarkUnmarshal-8  1995155  1813 ns/op   648 B/op  18 allocs/op (encoding/json/v2)
// BenchmarkUnmarshal-8  4072055  895.2 ns/op  892 B/op  18 allocs/op (bytedance/sonic) Usage: go test -tags=sonic_on -bench=BenchmarkUnmarshal -v
func BenchmarkUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Unmarshal([]byte(`{"Name":"Wednesday","Age":6,"Parents":["Gomez","Morticia"]}`), &map[string]any{})
	}
}
