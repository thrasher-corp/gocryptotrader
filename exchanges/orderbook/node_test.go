package orderbook

import (
	"fmt"
	"testing"
)

// Display nodes for testing purposes
func (s *Stack) Display() {
	for i := int32(0); i < s.count; i++ {
		fmt.Printf("NODE IN STACK: %+v %p \n", s.nodes[i], s.nodes[i])
	}
	fmt.Println("Tatal Count:", s.count)
}

//  158	   9,521,717 ns/op	 9600104 B/op	  100001 allocs/op
func BenchmarkWithoutStack(b *testing.B) {
	var n *Node
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100000; j++ {
			n = new(Node)
			n.value.Price = 1337
		}
	}
}

//  949	   1,427,820 ns/op	       0 B/op	       0 allocs/op
func BenchmarkWithStack(b *testing.B) {
	var n *Node
	stack := NewStack()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100000; j++ {
			n = stack.Pop()
			n.value.Price = 1337
			stack.Push(n)
		}
	}
}
