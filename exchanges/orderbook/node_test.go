package orderbook

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestPushPop(t *testing.T) {
	s := newStack()
	var nSlice []*node
	for i := 0; i < 100; i++ {
		nSlice = append(nSlice, s.Pop())
	}

	if atomic.LoadInt32(&s.count) != 0 {
		t.Fatalf("incorrect stack count expected %v but received %v", 0, atomic.LoadInt32(&s.count))
	}

	for i := 0; i < 100; i++ {
		s.Push(nSlice[i], getNow())
	}

	if atomic.LoadInt32(&s.count) != 100 {
		t.Fatalf("incorrect stack count expected %v but received %v", 100, atomic.LoadInt32(&s.count))
	}
}

func TestCleaner(t *testing.T) {
	s := newStack()
	var nSlice []*node
	for i := 0; i < 100; i++ {
		nSlice = append(nSlice, s.Pop())
	}

	for i := 0; i < 50; i++ {
		s.Push(nSlice[i], getNow())
	}
	// Makes all the 50 pushed nodes invalid
	time.Sleep(time.Millisecond * 550)
	for i := 50; i < 100; i++ {
		s.Push(nSlice[i], getNow())
	}
	time.Sleep(time.Millisecond * 550)
	if atomic.LoadInt32(&s.count) != 50 {
		t.Fatalf("incorrect stack count expected %v but received %v", 50, atomic.LoadInt32(&s.count))
	}
	time.Sleep(time.Second)
	if atomic.LoadInt32(&s.count) != 0 {
		t.Fatalf("incorrect stack count expected %v but received %v", 0, atomic.LoadInt32(&s.count))
	}
}

// Display nodes for testing purposes
func (s *stack) Display() {
	for i := int32(0); i < atomic.LoadInt32(&s.count); i++ {
		fmt.Printf("NODE IN STACK: %+v %p \n", s.nodes[i], s.nodes[i])
	}
	fmt.Println("Tatal Count:", atomic.LoadInt32(&s.count))
}

//  158	   9,521,717 ns/op	 9600104 B/op	  100001 allocs/op
func BenchmarkWithoutStack(b *testing.B) {
	var n *node
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100000; j++ {
			n = new(node)
			n.value.Price = 1337
		}
	}
}

//  949	   1,427,820 ns/op	       0 B/op	       0 allocs/op
func BenchmarkWithStack(b *testing.B) {
	var n *node
	stack := newStack()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100000; j++ {
			n = stack.Pop()
			n.value.Price = 1337
			stack.Push(n, getNow())
		}
	}
}
