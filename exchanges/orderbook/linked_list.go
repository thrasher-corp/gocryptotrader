package orderbook

import (
	"errors"
	"fmt"
)

var errIDCannotBeMatched = errors.New("cannot match ID on linked list")
var errCollisionDetected = errors.New("cannot insert update collision detected")
var errAmountCannotBeLessOrEqualToZero = errors.New("amount cannot be less or equal to zero")

// linkedList defines a linked list for a depth level, reutilisation of nodes
// to and from a stack.
type linkedList struct {
	length int
	head   *Node
}

// comparison defines expected functionality to compare between two reference
// price levels
type comparison func(float64, float64) bool

// load iterates across new items and refreshes linked list. It creates a linked
// list exactly the same as the item slice that is supplied, if items is of nil
// value it will flush entire list.
func (ll *linkedList) load(items Items, stack *stack) {
	// Tip sets up a pointer to a struct field variable pointer. This is used
	// so when a node is popped from the stack we can reference that current
	// nodes' struct 'next' field and set on next iteration without utilising
	// assignment for example `prev.Next = *node`.
	var tip = &ll.head
	// Prev denotes a place holder to node and all of its next references need
	// to be pushed back onto stack.
	var prev *Node
	for i := range items {
		if *tip == nil {
			// Extend node chain
			*tip = stack.Pop()
			// Set current node prev to last node
			(*tip).Prev = prev
			ll.length++
		}
		// Set item value
		(*tip).Value = items[i]
		// Set previous to current node
		prev = *tip
		// Set tip to next node
		tip = &(*tip).Next
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
		push = prev.Next
		prev.Next = nil
	}

	// Push unused pointers back on stack
	for push != nil {
		pending := push.Next
		stack.Push(push, getNow())
		ll.length--
		push = pending
	}
}

// updateByID amends price by corresponding ID and returns an error if not found
func (ll *linkedList) updateByID(updts []Item) error {
updates:
	for x := range updts {
		for tip := ll.head; tip != nil; tip = tip.Next {
			if updts[x].ID != tip.Value.ID { // Filter IDs that don't match
				continue
			}
			if updts[x].Price > 0 {
				// Only apply changes when zero values are not present, Bitmex
				// for example sends 0 price values.
				tip.Value.Price = updts[x].Price
			}
			tip.Value.Amount = updts[x].Amount
			continue updates
		}
		return fmt.Errorf("update error: %w %d not found",
			errIDCannotBeMatched,
			updts[x].ID)
	}
	return nil
}

// deleteByID deletes reference by ID
func (ll *linkedList) deleteByID(updts Items, stack *stack, bypassErr bool) error {
updates:
	for x := range updts {
		for tip := &ll.head; *tip != nil; tip = &(*tip).Next {
			if updts[x].ID != (*tip).Value.ID {
				continue
			}
			stack.Push(deleteAtTip(ll, tip), getNow())
			continue updates
		}
		if !bypassErr {
			return fmt.Errorf("delete error: %w %d not found",
				errIDCannotBeMatched,
				updts[x].ID)
		}
	}
	return nil
}

// cleanup reduces the max size of the depth length if exceeded. Is used after
// updates have been applied instead of adhoc, reason being its easier to prune
// at the end. (cant inline)
func (ll *linkedList) cleanup(maxChainLength int, stack *stack) {
	// Reduces the max length of total linked list chain, occurs after updates
	// have been implemented as updates can push length out of bounds, if
	// cleaved after that update, new update might not applied correctly.
	n := ll.head
	for i := 0; i < maxChainLength; i++ {
		if n.Next == nil {
			return
		}
		n = n.Next
	}

	// cleave reference to current node
	if n.Prev != nil {
		n.Prev.Next = nil
	} else {
		ll.head = nil
	}

	var pruned int
	for n != nil {
		pruned++
		pending := n.Next
		stack.Push(n, getNow())
		n = pending
	}
	ll.length -= pruned
}

// amount returns total depth liquidity and value
func (ll *linkedList) amount() (liquidity, value float64) {
	for tip := ll.head; tip != nil; tip = tip.Next {
		liquidity += tip.Value.Amount
		value += tip.Value.Amount * tip.Value.Price
	}
	return
}

// retrieve returns a full slice of contents from the linked list
func (ll *linkedList) retrieve() Items {
	depth := make(Items, ll.length)
	iterator := 0
	for tip := ll.head; tip != nil; tip = tip.Next {
		depth[iterator] = tip.Value
		iterator++
	}
	return depth
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates
func (ll *linkedList) updateInsertByPrice(updts Items, stack *stack, maxChainLength int, compare func(float64, float64) bool, tn now) {
	for x := range updts {
		for tip := &ll.head; ; tip = &(*tip).Next {
			if *tip == nil {
				insertHeadSpecific(ll, updts[x], stack)
				break
			}
			if (*tip).Value.Price == updts[x].Price { // Match check
				if updts[x].Amount <= 0 { // Capture delete update
					stack.Push(deleteAtTip(ll, tip), tn)
				} else { // Amend current amount value
					(*tip).Value.Amount = updts[x].Amount
				}
				break // Continue updates
			}

			if compare((*tip).Value.Price, updts[x].Price) { // Insert
				// This check below filters zero values and provides an
				// optimisation for when select exchanges send a delete update
				// to a non-existent price level (OTC/Hidden order) so we can
				// break instantly and reduce the traversal of the entire chain.
				if updts[x].Amount > 0 {
					insertAtTip(ll, tip, updts[x], stack)
				}
				break // Continue updates
			}

			if (*tip).Next == nil { // Tip is at tail
				// This check below is just a catch all in the event the above
				// zero value check fails
				if updts[x].Amount > 0 {
					insertAtTail(ll, tip, updts[x], stack)
				}
				break
			}
		}
	}
	// Reduces length of total linked list chain to a maxChainLength value
	if maxChainLength != 0 && ll.length > maxChainLength {
		ll.cleanup(maxChainLength, stack)
	}
}

// updateInsertByID updates or inserts if not found for a bid or ask depth
// 1) node ID found amount amended (best case)
// 2) node ID found amount and price amended and node moved to correct position
// (medium case)
// 3) Update price exceeds traversal node price before ID found, save node
// address for either; node ID matches then re-address node or end of depth pop
// a node from the stack (worst case)
func (ll *linkedList) updateInsertByID(updts Items, stack *stack, compare comparison) error {
updates:
	for x := range updts {
		if updts[x].Amount <= 0 {
			return errAmountCannotBeLessOrEqualToZero
		}
		// bookmark allows for saving of a position of a node in the event that
		// an update price exceeds the current node price. We can then match an
		// ID and re-assign that ID's node to that positioning without popping
		// from the stack and then pushing to the stack later for cleanup.
		// If the ID is not found we can pop from stack then insert into that
		// price level
		var bookmark *Node
		for tip := ll.head; tip != nil; tip = tip.Next {
			if tip.Value.ID == updts[x].ID {
				if tip.Value.Price != updts[x].Price { // Price level change
					if tip.Next == nil {
						// no movement needed just a re-adjustment
						tip.Value.Price = updts[x].Price
						tip.Value.Amount = updts[x].Amount
						continue updates
					}
					// bookmark tip to move this node to correct price level
					bookmark = tip
					continue // continue through node depth
				}
				// no price change, amend amount and continue update
				tip.Value.Amount = updts[x].Amount
				continue updates // continue to next update
			}

			if compare(tip.Value.Price, updts[x].Price) {
				if bookmark != nil { // shift bookmarked node to current tip
					bookmark.Value = updts[x]
					move(&ll.head, bookmark, tip)
					continue updates
				}

				// search for ID
				for n := tip.Next; n != nil; n = n.Next {
					if n.Value.ID == updts[x].ID {
						n.Value = updts[x]
						// inserting before the tip
						move(&ll.head, n, tip)
						continue updates
					}
				}
				// ID not matched in depth so add correct level for insert
				if tip.Next == nil {
					n := stack.Pop()
					n.Value = updts[x]
					ll.length++
					if tip.Prev == nil {
						tip.Prev = n
						n.Next = tip
						ll.head = n
						continue updates
					}
					tip.Prev.Next = n
					n.Prev = tip.Prev
					tip.Prev = n
					n.Next = tip
					continue updates
				}
				bookmark = tip
				break
			}

			if tip.Next == nil {
				if shiftBookmark(tip, &bookmark, &ll.head, updts[x]) {
					continue updates
				}
			}
		}
		n := stack.Pop()
		n.Value = updts[x]
		insertNodeAtBookmark(ll, bookmark, n) // Won't inline with stack
	}
	return nil
}

// insertUpdates inserts new updates for bids or asks based on price level
func (ll *linkedList) insertUpdates(updts Items, stack *stack, comp comparison) error {
	for x := range updts {
		var prev *Node
		for tip := &ll.head; ; tip = &(*tip).Next {
			if *tip == nil { // Head
				n := stack.Pop()
				n.Value = updts[x]
				n.Prev = prev
				ll.length++
				*tip = n
				break // Continue updates
			}

			if (*tip).Value.Price == updts[x].Price { // Price already found
				return fmt.Errorf("%w for price %f",
					errCollisionDetected,
					updts[x].Price)
			}

			if comp((*tip).Value.Price, updts[x].Price) { // Alignment
				n := stack.Pop()
				n.Value = updts[x]
				n.Prev = prev
				ll.length++
				// Reference current with new node
				(*tip).Prev = n
				// Push tip to the right
				n.Next = *tip
				// This is the same as prev.Next = n
				*tip = n
				break // Continue updates
			}

			if (*tip).Next == nil { // Tail
				insertAtTail(ll, tip, updts[x], stack)
				break // Continue updates
			}
			prev = *tip
		}
	}
	return nil
}

// bids embed a linked list to attach methods for bid depth specific
// functionality
type bids struct {
	linkedList
}

// bidCompare ensures price is in correct descending alignment (can inline)
func bidCompare(left, right float64) bool {
	return left < right
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates
func (ll *bids) updateInsertByPrice(updts Items, stack *stack, maxChainLength int, tn now) {
	ll.linkedList.updateInsertByPrice(updts, stack, maxChainLength, bidCompare, tn)
}

// updateInsertByID updates or inserts if not found
func (ll *bids) updateInsertByID(updts Items, stack *stack) error {
	return ll.linkedList.updateInsertByID(updts, stack, bidCompare)
}

// insertUpdates inserts new updates for bids based on price level
func (ll *bids) insertUpdates(updts Items, stack *stack) error {
	return ll.linkedList.insertUpdates(updts, stack, bidCompare)
}

// asks embed a linked list to attach methods for ask depth specific
// functionality
type asks struct {
	linkedList
}

// askCompare ensures price is in correct ascending alignment (can inline)
func askCompare(left, right float64) bool {
	return left > right
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates
func (ll *asks) updateInsertByPrice(updts Items, stack *stack, maxChainLength int, tn now) {
	ll.linkedList.updateInsertByPrice(updts, stack, maxChainLength, askCompare, tn)
}

// updateInsertByID updates or inserts if not found
func (ll *asks) updateInsertByID(updts Items, stack *stack) error {
	return ll.linkedList.updateInsertByID(updts, stack, askCompare)
}

// insertUpdates inserts new updates for asks based on price level
func (ll *asks) insertUpdates(updts Items, stack *stack) error {
	return ll.linkedList.insertUpdates(updts, stack, askCompare)
}

// move moves a node from a point in a node chain to another node position,
// this left justified towards head as element zero is the top of the depth
// side. (can inline)
func move(head **Node, from, to *Node) {
	if from.Next != nil { // From is at tail
		from.Next.Prev = from.Prev
	}
	if from.Prev == nil { // From is at head
		(*head).Next.Prev = nil
		*head = (*head).Next
	} else {
		from.Prev.Next = from.Next
	}
	// insert from node next to 'to' node
	if to.Prev == nil { // Destination is at head position
		*head = from
	} else {
		to.Prev.Next = from
	}
	from.Prev = to.Prev
	to.Prev = from
	from.Next = to
}

// deleteAtTip removes a node from tip target returns old node (can inline)
func deleteAtTip(ll *linkedList, tip **Node) *Node {
	// Old is a placeholder for current tips node value to push
	// back on to the stack.
	old := *tip
	switch {
	case old.Prev == nil: // At head position
		// shift current tip head to the right
		*tip = old.Next
		// Remove reference to node from chain
		if old.Next != nil { // This is when liquidity hits zero
			old.Next.Prev = nil
		}
	case old.Next == nil: // At tail position
		// Remove reference to node from chain
		old.Prev.Next = nil
	default:
		// Reference prior node in chain to next node in chain
		// bypassing current node
		old.Prev.Next = old.Next
		old.Next.Prev = old.Prev
	}
	ll.length--
	return old
}

// insertAtTip inserts at a tip target (can inline)
func insertAtTip(ll *linkedList, tip **Node, updt Item, stack *stack) {
	n := stack.Pop()
	n.Value = updt
	n.Next = *tip
	n.Prev = (*tip).Prev
	if (*tip).Prev == nil { // Tip is at head
		// Replace head which will push everything to the right
		// when this node will reference new node below
		*tip = n
	} else {
		// Reference new node to previous node
		(*tip).Prev.Next = n
	}
	// Reference next node to new node
	n.Next.Prev = n
	ll.length++
}

// insertAtTail inserts at tail end of node chain (can inline)
func insertAtTail(ll *linkedList, tip **Node, updt Item, stack *stack) {
	n := stack.Pop()
	n.Value = updt
	// Reference tip to new node
	(*tip).Next = n
	// Reference new node with current tip
	n.Prev = *tip
	ll.length++
}

// insertHeadSpecific inserts at head specifically there might be an instance
// where the liquidity on an exchange does fall to zero through a streaming
// endpoint then it comes back online. (can inline)
func insertHeadSpecific(ll *linkedList, updt Item, stack *stack) {
	n := stack.Pop()
	n.Value = updt
	ll.head = n
	ll.length++
}

// insertNodeAtBookmark inserts a new node at a bookmarked node position
// returns if a node needs to replace head (can inline)
func insertNodeAtBookmark(ll *linkedList, bookmark, n *Node) {
	switch {
	case bookmark == nil: // Zero liquidity and we are rebuilding from scratch
		ll.head = n
	case bookmark.Prev == nil:
		n.Prev = bookmark.Prev
		bookmark.Prev = n
		n.Next = bookmark
		ll.head = n
	case bookmark.Next == nil:
		n.Prev = bookmark
		bookmark.Next = n
	default:
		bookmark.Prev.Next = n
		n.Prev = bookmark.Prev
		bookmark.Prev = n
		n.Next = bookmark
	}
	ll.length++
}

// shiftBookmark moves a bookmarked node to the tip's next position or if nil,
// sets tip as bookmark (can inline)
func shiftBookmark(tip *Node, bookmark, head **Node, updt Item) bool {
	if *bookmark == nil { // End of the chain and no bookmark set
		*bookmark = tip // Set tip to bookmark so we can set a new node there
		return false
	}
	(*bookmark).Value = updt
	(*bookmark).Next.Prev = (*bookmark).Prev
	if (*bookmark).Prev == nil { // Bookmark is at head
		*head = (*bookmark).Next
	} else {
		(*bookmark).Prev.Next = (*bookmark).Next
	}
	tip.Next = *bookmark
	(*bookmark).Prev = tip
	(*bookmark).Next = nil
	return true
}
