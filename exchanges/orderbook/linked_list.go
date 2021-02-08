package orderbook

import (
	"errors"
	"fmt"
	"time"
)

// linkedList defines a singularly linked list for depth levels
// TODO: Test cross link between bid and ask head nodes **Node ref to head
type linkedList struct {
	length int
	head   *Node
}

var errNoStack = errors.New("cannot load orderbook depth, stack is nil")

// Load iterates across new items and refreshes linked list
func (ll *linkedList) Load(items Items, stack *Stack) error {
	if stack == nil {
		return errNoStack
	}

	// This sets up a pointer to a field variable to a node. This is used so
	// when a node is popped into existance we can reference the current nodes
	// 'next' field and set on next iteration without utilising
	// `prev.next = *Node` it should automatically be referenced.
	var tip = &ll.head
	var prev *Node
	for i := 0; i < len(items); i++ {
		if *tip == nil {
			// Extend node chain
			*tip = stack.Pop()
		}
		// Set item value
		(*tip).value = items[i]
		// Set current node prev to last node
		(*tip).prev = prev
		// Set previous to current node
		prev = (*tip)
		// Set tip to next node
		tip = &(*tip).next
	}

	var push *Node
	// Cleave unused reference chain from main chain
	if prev == nil {
		push = *tip
		ll.head = nil
	} else {
		push = prev.next
		prev.next = nil
	}

	// Push unused pointers back on stack
	for push != nil {
		pending := push.next
		stack.Push(push)
		push = pending
	}
	return nil
}

// byDecision defines functionality for item data
type byDecision func(Item) bool

// RemoveByPrice removes depth level by price and returns the node to be pushed
// onto the stack
func (ll *linkedList) Remove(fn byDecision, stack *Stack) (*Node, error) {
	for tip := ll.head; tip != nil; tip = tip.next {
		if fn(tip.value) {
			if tip.prev == nil { // tip is at head
				ll.head = tip.next
				if tip.next != nil {
					tip.next.prev = nil
				}
				return tip, nil
			}
			if tip.next == nil { // tip is at tail
				tip.prev.next = nil
				return tip, nil
			}
			// Split reference
			tip.prev.next = tip.next
			tip.next.prev = tip.prev
			return tip, nil
		}
	}
	return nil, errors.New("not found cannot remove")
}

// Add adds depth level by decision
func (ll *linkedList) Add(fn byDecision, item Item, stack *Stack) error {
	for tip := &ll.head; ; tip = &(*tip).next {
		if *tip == nil {
			*tip = stack.Pop()
			(*tip).value = item
			return nil
		}

		if fn((*tip).value) {
			n := stack.Pop()
			n.value = item
			n.next = (*tip).next
			n.prev = *tip
			(*tip).next = n
			return nil
		}
	}
}

// updateBidsByPrice ammends, inserts, moves and cleaves length of depth by
// updates in bid linked list
func (ll *linkedList) updateBidsByPrice(updts Items) {}

// updateAsksByPrice ammends, inserts, moves and cleaves length of depth by
// updates in ask linked list
func (ll *linkedList) updateAsksByPrice(updts Items) {}

// updateByID ammends price by corresponding ID and returns an error if not
// found
func (ll *linkedList) updateByID(updts Items) error { return nil }

// insertUpdatesBid inserts new updates for bids based on price level
// attention: bid updates need to be in descending order
func (ll *linkedList) insertUpdatesBid(updts Items, stack *Stack) {
	target := 0
	for tip := &ll.head; ; tip = &(*tip).next {
		if *tip == nil {
			chain(tip, updts[target:], stack)
		}

		if updts[target].Price > (*tip).value.Price {
			insertItem(updts[target], *tip, stack)
		}
		target++
		if len(updts) < target {
			break
		}
	}
}

// insertUpdatesAsk inserts new updates for asks based on price level
// attention: ask updates need to be in ascending order
func (ll *linkedList) insertUpdatesAsk(updts Items, stack *Stack) {
	target := 0
	for tip := &ll.head; ; tip = &(*tip).next {
		if *tip == nil {
			chain(tip, updts[target:], stack)
		}

		if updts[target].Price < (*tip).value.Price {
			insertItem(updts[target], *tip, stack)
		}
		target++
		if len(updts) < target {
			break
		}
	}
}

// chain adds new nodes to the tail
func chain(tip **Node, updts Items, stack *Stack) {
	for i := range updts {
		n := stack.Pop()
		n.value = updts[i]
		n.prev = *tip
		if *tip != nil {
			(*tip).next = n
			tip = &(*tip).next
		} else {
			*tip = n
		}
	}
}

// insertItem inserts an item a specific target level
func insertItem(item Item, target *Node, stack *Stack) {
	n := stack.Pop()
	n.value = item
	n.next = target.next
	n.prev = target
	target.next = n
}

// Ammend changes depth level by decision and item value
func (ll *linkedList) Ammend(fn byDecision, item Item) error {
	for tip := ll.head; tip != nil; tip = tip.next {
		if fn(tip.value) {
			tip.value = item
			return nil
		}
	}
	return errors.New("value could not be changed")
}

// Liquidity returns total depth liquidity
func (ll *linkedList) Liquidity() (liquidity float64) {
	for tip := ll.head; tip != nil; tip = tip.next {
		liquidity += tip.value.Amount
	}
	return
}

// Value returns total value on price.amount on full depth
func (ll *linkedList) Value() (value float64) {
	for tip := ll.head; tip != nil; tip = tip.next {
		value += tip.value.Amount * tip.value.Price
	}
	return
}

// Amount returns total depth liquidity and value
func (ll *linkedList) Amount() (liquidity, value float64) {
	for tip := ll.head; tip != nil; tip = tip.next {
		liquidity += tip.value.Amount
		value += tip.value.Amount * tip.value.Price
	}
	return
}

// Retrieve returns a full slice of contents from the linked list
func (ll *linkedList) Retrieve() (items Items) {
	for tip := ll.head; tip != nil; tip = tip.next {
		items = append(items, tip.value)
	}
	return
}

// Display displays depth content
func (ll *linkedList) Display() {
	for tip := ll.head; tip != nil; tip = tip.next {
		fmt.Printf("NODE: %+v %p \n", tip, tip)
	}
	fmt.Println()
}

// ------------------- Node stuff ------------------------------------------ //

// Node defines a linked list node for an orderbook item
type Node struct {
	value Item
	next  *Node
	prev  *Node

	// Denotes time pushed to stack, this will influence cleanup routine when
	// there is a pause or minimal actions during period
	shelfed time.Time
	// sync.Pool
}

// Stack defines a FIFO list of reusable nodes
type Stack struct {
	nodes []*Node
	s     *uint32
	count int32
}

// NewStack returns a ptr to a new Stack instance
func NewStack() *Stack {
	// create routine that liquidates stack every minute
	return &Stack{}
}

// Push pushes a node pointer into the stack to be reused
func (s *Stack) Push(n *Node) {
	n.shelfed = time.Now()
	n.next = nil
	n.prev = nil
	n.value = Item{}
	s.nodes = append(s.nodes[:s.count], n)
	s.count++
}

// Pop returns the last pointer off the stack and reduces the count and if empty
// will produce a lovely fresh node
func (s *Stack) Pop() *Node {
	if s.count == 0 {
		// Create an empty node
		return &Node{}
	}
	s.count--
	return s.nodes[s.count]
}

// Display wowwwww
func (s *Stack) Display() {
	for i := int32(0); i < s.count; i++ {
		fmt.Printf("NODE IN STACK: %+v %p \n", s.nodes[i], s.nodes[i])
	}
	fmt.Println("Tatal Count:", s.count)
}
