//go:build !sonic

package json

import "testing"

// 827232	1318 ns/op	816 B/op	24 allocs/op
func BenchmarkUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Unmarshal([]byte(`{"Name":"Wednesday","Age":6,"Parents":["Gomez","Morticia"]}`), &map[string]interface{}{})
	}
}
