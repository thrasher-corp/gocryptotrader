package orderbook

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Depth defines a linked list of orderbook items
type Depth struct {
	ask linkedList
	bid linkedList

	// Unexported stack of nodes
	stack Stack

	// Change of state to re-check depth list
	wait    chan struct{}
	waiting uint32
	wMtx    sync.Mutex
	// -----

	// RestSnapshot defines if the depth was applied via the REST protocol thus
	// an update cannot be applied via websocket mechanics and a resubscription
	// would need to take place to maintain book integrity
	restSnapshot bool

	lastUpdated time.Time
	sync.Mutex
}

// LenAsk returns length of asks
func (d *Depth) LenAsk() int {
	d.Lock()
	defer d.Unlock()
	return d.ask.length
}

// LenBids returns length of bids
func (d *Depth) LenBids() int {
	d.Lock()
	defer d.Unlock()
	return d.bid.length
}

// AddBid adds a bid to the list
func (d *Depth) AddBid(i Item) error {
	d.Lock()
	defer d.Unlock()
	return d.bid.Add(func(i Item) bool { return true }, i, &d.stack)
}

// Retrieve gets stuff
func (d *Depth) Retrieve() (bids, asks Items) {
	d.Lock()
	defer d.Unlock()
	return d.bid.Retrieve(), d.ask.Retrieve()
}

// // AddBids adds a collection of bids to the linked list
// func (d *Depth) AddBids(i Item) error {
// 	d.Lock()
// 	defer d.Unlock()
// 	n := d.stack.Pop()
// 	n.value = i
// 	d.bid.Add(func(i Item) bool { return true }, n)
// 	return nil
// }

// RemoveBidByPrice removes a bid
func (d *Depth) RemoveBidByPrice(price float64) error {
	// d.Lock()
	// defer d.Unlock()
	// n, err := d.bid.Remove(func(i Item) bool { return i.Price == price })
	// if err != nil {
	// 	return err
	// }
	// d.stack.Push(n)
	return nil
}

// DisplayBids does a helpful display!!! YAY!
func (d *Depth) DisplayBids() {
	d.Lock()
	defer d.Unlock()
	d.bid.Display()
}

// alert establishes state change for depth to all waiting routines
func (d *Depth) alert() {
	if !atomic.CompareAndSwapUint32(&d.waiting, 1, 0) {
		// return if no waiting routines
		return
	}
	d.wMtx.Lock()
	close(d.wait)
	d.wait = make(chan struct{})
	d.wMtx.Unlock()
}

type kicker chan struct{}

// timeInForce allows a kick
func timeInForce(t time.Duration) kicker {
	ch := make(chan struct{})
	go func(ch chan<- struct{}) {
		time.Sleep(t)
		close(ch)
	}(ch)
	return ch
}

// Wait pauses routine until depth change has been established
func (d *Depth) Wait(kick <-chan struct{}) {
	d.wMtx.Lock()
	atomic.StoreUint32(&d.waiting, 1)
	d.wMtx.Unlock()
	select {
	case <-d.wait:
	case <-kick:
	}
}

// TotalBidsAmount returns the total amount of bids and the total orderbook
// bids value
func (d *Depth) TotalBidsAmount() (liquidity, value float64) {
	d.Lock()
	defer d.Unlock()
	return d.bid.Amount()
}

// TotalAsksAmount returns the total amount of asks and the total orderbook
// asks value
func (d *Depth) TotalAsksAmount() (liquidity, value float64) {
	d.Lock()
	defer d.Unlock()
	return d.ask.Amount()
}

// // Update updates the bids and asks
// func (d *Depth) Update(bids, asks []Item) error {
// 	d.Lock()
// 	defer d.Unlock()

// 	err := d.bid.Load(bids, &d.stack)
// 	if err != nil {
// 		return err
// 	}

// 	err = d.ask.Load(asks, &d.stack)
// 	if err != nil {
// 		return err
// 	}
// 	// Update occurred, alert routines
// 	d.alert()
// 	return nil
// }

// Process processes incoming orderbook snapshots
func (d *Depth) Process(bids, asks Items) error {
	err := d.bid.Load(bids, &d.stack)
	if err != nil {
		return err
	}
	err = d.ask.Load(asks, &d.stack)
	if err != nil {
		return err
	}
	d.alert()
	return nil
}

// invalidate will pop entire bid and ask node chain onto stack when an error
// occurs, so as to not be able to traverse potential invalid books.
func (d *Depth) invalidate() {

}

// linkedList defines a depth linked list
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

type outOfOrder func(float64, float64) bool

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
