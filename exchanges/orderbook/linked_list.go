package orderbook

import (
	"errors"
	"fmt"
	"time"
)

// linkedList defines a linked list for depth levels, reutilisation of nodes
// to and from a stack.
// TODO: Test cross link between bid and ask head nodes **Node ref to head
type linkedList struct {
	length int
	head   *Node
}

// var errNoStack = errors.New("cannot load orderbook depth, stack is nil")

// Load iterates across new items and refreshes linked list
func (ll *linkedList) Load(items Items, stack *Stack) {
	// This sets up a pointer to a struct field variable to a node. This is used
	// so when a node is popped from the stack we can reference that current
	// nodes' struct 'next' field and set on next iteration without utilising
	// assignment `prev.next = *Node`.
	var tip = &ll.head
	var prev *Node
	for i := 0; i < len(items); i++ {
		if *tip == nil {
			// Extend node chain
			*tip = stack.Pop()
			ll.length++
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

	// Push has references to dangling nodes that need to be removed and pushed
	// back onto stack for re-use
	var push *Node
	// Cleave unused reference chain from main chain
	if prev == nil {
		// The entire chain will need to be pushed back on to stack
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
		ll.length--
		push = pending
	}
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

// updateInsertAsksByPrice ammends, inserts, moves and cleaves length of depth by
// updates in ask linked list
func (ll *linkedList) updateInsertAsksByPrice(updts Items, stack *Stack, maxChainLength int) error {
	defer ll.cleanup(maxChainLength, stack)
updates:
	for x := range updts {
		for tip := &ll.head; *tip != nil; tip = &(*tip).next {
			if (*tip).value.Price == updts[x].Price { // Match check
				if updts[x].Amount <= 0 { // Delete
					old := *tip
					if old.prev == nil { // Head
						*tip = old.next
						old.next.prev = nil
					} else if old.next == nil { // tail
						old.prev.next = nil
					} else {
						old.prev.next = old.next
						old.next.prev = old.prev
					}
					stack.Push(old)
					ll.length--
				} else { // Amend
					(*tip).value.Amount = updts[x].Amount
				}
				continue updates
			}

			if (*tip).value.Price > updts[x].Price { // Insert
				if updts[x].Amount > 0 { // Filter delete, should already be
					// removed
					n := stack.Pop()
					n.value = updts[x]
					n.next = *tip
					n.prev = (*tip).prev
					if (*tip).prev == nil { // Tip is at head; replace
						*tip = n
					} else {
						(*tip).prev.next = n
					}
					n.next.prev = n
					ll.length++
				}
				continue updates
			}

			if (*tip).next == nil { // Tip is at tail so pop and append
				if updts[x].Amount > 0 {
					n := stack.Pop()
					n.value = updts[x]
					(*tip).next = n
					n.prev = *tip
					ll.length++
				}
			}
		}
		if updts[x].Amount > 0 {
			return fmt.Errorf("could not apply update %+v", updts[x])
		}
	}
	return nil
}

// updateInsertBidsByPrice ammends, inserts, moves and cleaves length of depth by
// updates in bid linked list
func (ll *linkedList) updateInsertBidsByPrice(updts Items, stack *Stack, maxChainLength int) error {
	defer ll.cleanup(maxChainLength, stack)
updates:
	for x := range updts {
		for tip := &ll.head; *tip != nil; tip = &(*tip).next {
			if (*tip).value.Price == updts[x].Price { // Match check
				if updts[x].Amount <= 0 { // Delete
					old := *tip
					if old.prev == nil { // Head
						*tip = old.next
						old.next.prev = nil
					} else if old.next == nil { // tail
						old.prev.next = nil
					} else {
						old.prev.next = old.next
						old.next.prev = old.prev
					}
					stack.Push(old)
					ll.length--
				} else { // Amend
					(*tip).value.Amount = updts[x].Amount
				}
				continue updates
			}

			if (*tip).value.Price < updts[x].Price { // Insert
				if updts[x].Amount > 0 { // Filter delete, should be hit at this tranche level, so obviously not accounted for in depth
					n := stack.Pop()
					n.value = updts[x]
					n.next = *tip
					n.prev = (*tip).prev
					if (*tip).prev == nil { // Tip is at head; replace
						*tip = n
					} else {
						(*tip).prev.next = n
					}
					n.next.prev = n
					ll.length++
				}
				continue updates
			}

			if (*tip).next == nil {
				if updts[x].Amount > 0 {
					n := stack.Pop()
					n.value = updts[x]
					(*tip).next = n
					n.prev = *tip
					ll.length++
				}
			}
		}
		if updts[x].Amount > 0 {
			return fmt.Errorf("could not apply update %+v", updts[x])
		}
	}
	return nil
}

// cleanup reduces the max size of the depth length if exceeded. Is used after
// updates have been applied instead of adhoc, reason being its easier to prune
// at the end.
func (ll *linkedList) cleanup(maxChainLength int, stack *Stack) {
	if maxChainLength == 0 || ll.length <= maxChainLength {
		return
	}

	n := ll.head
	for i := 0; i < maxChainLength; i++ {
		n = n.next
		if n.next == nil {
			break
		}
	}

	// cleave reference to current node
	if n.prev != nil {
		n.prev.next = nil
	}

	var pruned int
	for n != nil {
		pruned++
		pending := n.next
		stack.Push(n)
		n = pending
	}
	ll.length -= pruned
}

// deleteUpdate removes update TODO: Benchmark
func deleteUpdate(updts *Items) (finished bool) {
	if len(*updts) < 2 { // Eager check pre-empts deletion
		return true
	}
	(*updts)[0] = (*updts)[len(*updts)-1]
	*updts = (*updts)[:len(*updts)-1]
	return false
}

// updateByID ammends price by corresponding ID and returns an error if not
// found
func (ll *linkedList) updateByID(updts Items) error {
updates:
	for tip := ll.head; tip != nil; tip = tip.next {
		for x := range updts {
			if updts[x].ID == tip.value.ID {
				tip.value = updts[x]
				if deleteUpdate(&updts) {
					return nil
				}
				continue updates
			}
		}
	}
	return fmt.Errorf("update cannot be applied id: %d not found",
		updts[0].ID)
}

// deleteByID deletes refererence by ID
func (ll *linkedList) deleteByID(updts Items, stack *Stack, bypassErr bool) error {
updates:
	for tip := &ll.head; tip != nil; tip = &(*tip).next {
		for x := range updts {
			if updts[x].ID == (*tip).value.ID {
				old := *tip
				*tip = old.next
				if old.prev != nil {
					old.prev.next = *tip
				}
				stack.Push(old)
				ll.length--
				if deleteUpdate(&updts) {
					return nil
				}
				continue updates
			}
		}
	}

	if !bypassErr {
		return fmt.Errorf("update cannot be deleted id: %d not found",
			updts[0].ID)
	}
	return nil
}

var ASC = func(priceTip, priceUpdate float64) bool { return priceTip > priceUpdate }
var DSC = func(priceTip, priceUpdate float64) bool { return priceTip < priceUpdate }

// updateInsertByID updates or inserts if not found
// 1) Node ID found amount amended (best case)
// 2) Node ID found amount and price amended and node moved to correct position
// (medium case)
// 3) Update price exceeds traversal node price before ID found, save node
// address for either; node ID matches then re-address node or end of depth pop
// a node from the stack (worst case)
func (ll *linkedList) updateInsertByIDAsk(updts Items, stack *Stack) {
updates:
	for x := range updts {
		// bookmark allows for saving of a position of a node in the event that
		// an update price exceeds the current node price. We can then match an
		// ID and re-assign that ID's node to that positioning without popping
		// from the stack and then pushing to the stack later for cleanup.
		// If the ID is not found we can pop from stack then insert into that
		// price level
		var bookmark *Node
		for tip := ll.head; tip != nil; tip = tip.next {
			if tip.value.ID == updts[x].ID {
				if tip.value.Price != updts[x].Price { // Price level change
					// bookmark tip to move this node to correct price level
					bookmark = tip
					continue // continue through node depth
				}
				// no price change, ammend amount and conintue update
				tip.value.Amount = updts[x].Amount
				continue updates // continue to next update
			}

			if tip.value.Price > updts[x].Price {
				if bookmark != nil { // shift bookmarked node to current tip
					bookmark.value = updts[x]
					move(&ll.head, bookmark, tip)
					continue updates
				}

				// search for ID
				for n := tip.next; n != nil; n = n.next {
					if n.value.ID == updts[x].ID {
						n.value = updts[x]
						// inserting before the tip
						move(&ll.head, n, tip)
						continue updates
					}
				}
				// ID not matched in depth so add correct level for insert
				bookmark = tip
				break
			}

			if tip.next == nil {
				if bookmark == nil {
					bookmark = tip
				} else {
					bookmark.value = updts[x]
					bookmark.next.prev = bookmark.prev
					bookmark.prev.next = bookmark.next
					tip.next = bookmark
					bookmark.prev = tip
					bookmark.next = nil
					continue updates
				}
			}
		}
		n := stack.Pop()
		n.value = updts[x]
		if bookmark.prev == nil {
			ll.head = n
		} else {
			bookmark.prev.next = n
		}
		n.prev = bookmark.prev
		bookmark.prev = n
		n.next = bookmark
		ll.length++
	}
}

// updateInsertByID updates or inserts if not found
// 1) Node ID found amount amended (best case)
// 2) Node ID found amount and price amended and node moved to correct position
// (medium case)
// 3) Update price exceeds traversal node price before ID found, save node
// address for either; node ID matches then re-address node or end of depth pop
// a node from the stack (worst case)
func (ll *linkedList) updateInsertByIDBid(updts Items, stack *Stack) {
updates:
	for x := range updts {
		// bookmark allows for saving of a position of a node in the event that
		// an update price exceeds the current node price. We can then match an
		// ID and re-assign that ID's node to that positioning without popping
		// from the stack and then pushing to the stack later for cleanup.
		// If the ID is not found we can pop from stack then insert into that
		// price level
		var bookmark *Node
		for tip := ll.head; tip != nil; tip = tip.next {
			if tip.value.ID == updts[x].ID {
				if tip.value.Price != updts[x].Price { // Price level change
					// bookmark tip to move this node to correct price level
					bookmark = tip
					continue // continue through node depth
				}
				// no price change, ammend amount and conintue update
				tip.value.Amount = updts[x].Amount
				continue updates // continue to next update
			}

			if tip.value.Price < updts[x].Price {
				if bookmark != nil { // shift bookmarked node to current tip
					bookmark.value = updts[x]
					move(&ll.head, bookmark, tip)
					continue updates
				}

				// search for ID
				for n := tip.next; n != nil; n = n.next {
					if n.value.ID == updts[x].ID {
						n.value = updts[x]
						// inserting before the tip
						move(&ll.head, n, tip)
						continue updates
					}
				}
				// ID not matched in depth so add correct level for insert
				bookmark = tip
				break
			}

			if tip.next == nil {
				if bookmark == nil {
					bookmark = tip
				} else {
					bookmark.value = updts[x]
					bookmark.next.prev = bookmark.prev
					bookmark.prev.next = bookmark.next
					tip.next = bookmark
					bookmark.prev = tip
					bookmark.next = nil
					continue updates
				}
			}
		}
		n := stack.Pop()
		n.value = updts[x]
		if bookmark.prev == nil {
			ll.head = n
		} else if bookmark.next == nil {
			bookmark.next = n
			n.prev = bookmark
		} else {
			bookmark.prev.next = n
			n.prev = bookmark.prev
			bookmark.prev = n
			n.next = bookmark
		}
		ll.length++
	}
}

// move moves a node from a point in a node chain to another node position,
// this left justified towards head as element zero is the top of the depth
// side.
func move(head **Node, from, to *Node) {
	// remove 'from' node from current position in chain
	from.next.prev = from.prev
	from.prev.next = from.next
	// insert from node next to 'to' node
	if to.prev == nil {
		*head = from
	} else {
		to.prev.next = from
	}
	from.prev = to.prev
	to.prev = from
	from.next = to
}

// insertUpdatesBid inserts new updates for bids based on price level
// attention: bid updates need to be in descending order
func (ll *linkedList) insertUpdatesBid(updts Items, stack *Stack) {
	target := 0
	for tip := &ll.head; ; tip = &(*tip).next {
		if *tip == nil {
			chain(tip, updts[target:], stack)
			ll.length += len(updts[target:]) // TODO: REDO/RETHINK
			return
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
