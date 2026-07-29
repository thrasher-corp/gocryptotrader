package json

import "testing"

// BenchmarkUnmarshal measures whichever JSON implementation is compiled in. To measure
// bytedance/sonic rather than encoding/json:
//
//	go test --tags=sonic -bench=BenchmarkUnmarshal -v
func BenchmarkUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Unmarshal([]byte(`{"Name":"Wednesday","Age":6,"Parents":["Gomez","Morticia"]}`), &map[string]any{})
	}
}
