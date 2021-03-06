package orderbook

import (
	"sync/atomic"
	"time"
)

var (
	defaultInterval  = time.Minute
	defaultAllowance = time.Second * 30
)

// node defines a linked list node for an orderbook item
type node struct {
	value Item
	next  *node
	prev  *node
	// Denotes time pushed to stack, this will influence cleanup routine when
	// there is a pause or minimal actions during period
	shelfed time.Time
}

// stack defines a FIFO list of reusable nodes
type stack struct {
	nodes []*node
	s     uint32
	count int32
}

// newstack returns a ptr to a new stack instance, also starts the cleaning
// serbvice
func newStack() *stack {
	s := &stack{}
	go s.cleaner()
	return s
}

// Push pushes a node pointer into the stack to be reused
func (s *stack) Push(n *node) {
	if atomic.LoadUint32(&s.s) != 0 {
		// Cleaner is activated, for now we can derefence pointer
		n = nil
		return
	}
	// Adds a time when its placed back on to stack.
	n.shelfed = time.Now()
	n.next = nil
	n.prev = nil
	n.value = Item{}
	// Allows for resize when overflow TODO: rethink this
	s.nodes = append(s.nodes[:s.count], n)
	s.count++
}

// Pop returns the last pointer off the stack and reduces the count and if empty
// will produce a lovely fresh node
func (s *stack) Pop() *node {
	if atomic.LoadUint32(&s.s) != 0 || s.count == 0 {
		// Create an empty node when no nodes are in slice or when cleaning
		// service is running
		return &node{}
	}
	s.count--
	return s.nodes[s.count]
}

// cleaner (POC) runs to the defaultTimer to clean excess nodes (nodes not being
// utilised) TODO: Couple time parameters to chec for a reduction in activity.
// Add in counter per second function (?) so if there is a lot of activity don't
// inhibit stack performance.
func (s *stack) cleaner() {
	tt := time.NewTimer(defaultInterval)
sleeperino:
	for {
		select {
		case <-tt.C:
			atomic.StoreUint32(&s.s, 1)
			// We are going to iterate through slice running man styles
			// As the old nodes are going to be left justified on this slice we
			// should just be able to shift the nodes that are still within time
			// allowance all the way to the left. Not going to resize capacity
			// because if it can get this big, it might as well stay this big.
			// TODO: Test and rethink if sizing is an issue
			for x := int32(0); x < s.count; x++ {
				// find the first good one, everything to the left can be
				// reassigned
				if time.Since(s.nodes[x].shelfed) < defaultAllowance {
					// Go through good nodes
					var counter int32
					for y := int32(0); y+x < s.count; y++ {
						// Reassign
						s.nodes[y] = s.nodes[y+x]
						// Add to the changed counter to remove from main
						// counter
						counter++
					}
					s.count -= counter
					atomic.StoreUint32(&s.s, 0)
					tt.Reset(defaultInterval)
					continue sleeperino
				}
			}
			// All the nodes were old af, slightly upsetting
			s.count = 0
			atomic.StoreUint32(&s.s, 0)
			tt.Reset(defaultInterval)
		}
	}
}
