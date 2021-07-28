package orderbook

import (
	"sync/atomic"
	"time"
)

const (
	neutral uint32 = iota
	active
)

var (
	defaultInterval  = time.Minute
	defaultAllowance = time.Second * 30
)

// Node defines a linked list node for an orderbook item
type Node struct {
	Value Item
	Next  *Node
	Prev  *Node
	// Denotes time pushed to stack, this will influence cleanup routine when
	// there is a pause or minimal actions during period
	shelved time.Time
}

// stack defines a FILO list of reusable nodes
type stack struct {
	nodes []*Node
	sema  uint32
	count int32
}

// newStack returns a ptr to a new stack instance, also starts the cleaning
// service
func newStack() *stack {
	s := &stack{}
	go s.cleaner()
	return s
}

// now defines a time which is now to ensure no other values get passed in
type now time.Time

// getNow returns the time at which it is called
func getNow() now {
	return now(time.Now())
}

// Push pushes a node pointer into the stack to be reused the time is passed in
// to allow for inlining which sets the time at which the node is theoretically
// pushed to a stack.
func (s *stack) Push(n *Node, tn now) {
	if !atomic.CompareAndSwapUint32(&s.sema, neutral, active) {
		// Stack is in use, for now we can dereference pointer
		n = nil
		return
	}
	// Adds a time when its placed back on to stack.
	n.shelved = time.Time(tn)
	n.Next = nil
	n.Prev = nil
	n.Value = Item{}

	// Allows for resize when overflow TODO: rethink this
	s.nodes = append(s.nodes[:s.count], n)
	s.count++
	atomic.StoreUint32(&s.sema, neutral)
}

// Pop returns the last pointer off the stack and reduces the count and if empty
// will produce a lovely fresh node
func (s *stack) Pop() *Node {
	if !atomic.CompareAndSwapUint32(&s.sema, neutral, active) {
		// Stack is in use, for now we can allocate a new node pointer
		return &Node{}
	}

	if s.count == 0 {
		// Create an empty node when no nodes are in slice or when cleaning
		// service is running
		atomic.StoreUint32(&s.sema, neutral)
		return &Node{}
	}
	s.count--
	n := s.nodes[s.count]
	atomic.StoreUint32(&s.sema, neutral)
	return n
}

// cleaner (POC) runs to the defaultTimer to clean excess nodes (nodes not being
// utilised) TODO: Couple time parameters to check for a reduction in activity.
// Add in counter per second function (?) so if there is a lot of activity don't
// inhibit stack performance.
func (s *stack) cleaner() {
	tt := time.NewTimer(defaultInterval)
sleeperino:
	for range tt.C {
		if !atomic.CompareAndSwapUint32(&s.sema, neutral, active) {
			// Stack is in use, reset timer to zero to recheck for neutral state.
			tt.Reset(0)
			continue
		}
		// As the old nodes are going to be left justified on this slice we
		// should just be able to shift the nodes that are still within time
		// allowance all the way to the left. Not going to resize capacity
		// because if it can get this big, it might as well stay this big.
		// TODO: Test and rethink if sizing is an issue
		for x := int32(0); x < s.count; x++ {
			if time.Since(s.nodes[x].shelved) > defaultAllowance {
				// Old node found continue
				continue
			}
			// First good node found, everything to the left of this on the
			// slice can be reassigned
			var counter int32
			for y := int32(0); y+x < s.count; y++ { // Go through good nodes
				// Reassign
				s.nodes[y] = s.nodes[y+x]
				// Add to the changed counter to remove from main
				// counter
				counter++
			}
			s.count -= counter
			atomic.StoreUint32(&s.sema, neutral)
			tt.Reset(defaultInterval)
			continue sleeperino
		}
		// Nodes are old, flush entirety.
		s.count = 0
		atomic.StoreUint32(&s.sema, neutral)
		tt.Reset(defaultInterval)
	}
}
