package json

import "testing"

// BenchmarkUnmarshal measures whichever JSON implementation is compiled in: encoding/json/v2 by
// default. To measure bytedance/sonic instead:
//
//	go test -tags sonic_on -bench=BenchmarkUnmarshal -v
func BenchmarkUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Unmarshal([]byte(`{"Name":"Wednesday","Age":6,"Parents":["Gomez","Morticia"]}`), &map[string]any{})
	}
}
