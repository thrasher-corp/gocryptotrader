package json

import "testing"

// BenchmarkUnmarshal measures the JSON implementation compiled in; -tags sonic_on selects sonic
func BenchmarkUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Unmarshal([]byte(`{"Name":"Wednesday","Age":6,"Parents":["Gomez","Morticia"]}`), &map[string]any{})
	}
}
