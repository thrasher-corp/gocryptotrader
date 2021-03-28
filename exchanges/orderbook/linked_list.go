package orderbook

import (
	"errors"
	"fmt"
	"time"
)

var errIDCannotBeMatched = errors.New("cannot match ID on linked list")
var errCollisionDetected = errors.New("cannot insert update collision detected")
var errAmountCannotBeZero = errors.New("amount cannot be zero")

// linkedList defines a linked list for a depth level, reutilisation of nodes
// to and from a stack.
// TODO: Test cross link between bid and ask head nodes **node ref to head for
// future strategy asymmetric traversal between two books
type linkedList struct {
	length int
	head   *node
}

// Load iterates across new items and refreshes linked list. It creates a linked
// list exactly the same as the item slice that is supplied, if items is of nil
// value it will flush entire list.
func (ll *linkedList) load(items Items, stack *stack) {
	// Tip sets up a pointer to a struct field variable pointer. This is used
	// so when a node is popped from the stack we can reference that current
	// nodes' struct 'next' field and set on next iteration without utilising
	// assignment for example `prev.next = *node`.
	var tip = &ll.head
	// Prev denotes a place holder to node and all of its next references need
	// to be pushed back onto stack.
	var prev *node
	for i := range items {
		if *tip == nil {
			// Extend node chain
			*tip = stack.Pop()
			// Set current node prev to last node
			(*tip).prev = prev
			ll.length++
		}
		// Set item value
		(*tip).value = items[i]
		// Set previous to current node
		prev = *tip
		// Set tip to next node
		tip = &(*tip).next
	}

	// Push has references to dangling nodes that need to be removed and pushed
	// back onto stack for re-use
	var push *node
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
		stack.Push(push, time.Now())
		ll.length--
		push = pending
	}
}

// updateByID amends price by corresponding ID and returns an error if not found
func (ll *linkedList) updateByID(updts []Item) error {
updates:
	for x := range updts {
		for tip := ll.head; tip != nil; tip = tip.next {
			if updts[x].ID != tip.value.ID { // Filter IDs that don't match
				continue
			}
			tip.value.Price = updts[x].Price
			tip.value.Amount = updts[x].Amount
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
		for tip := &ll.head; *tip != nil; tip = &(*tip).next {
			if updts[x].ID != (*tip).value.ID {
				continue
			}
			stack.Push(deleteAtTip(ll, tip), time.Now())
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
		if n.next == nil {
			return
		}
		n = n.next
	}

	// cleave reference to current node
	if n.prev != nil {
		n.prev.next = nil
	} else {
		ll.head = nil
	}

	var pruned int
	for n != nil {
		pruned++
		pending := n.next
		stack.Push(n, time.Now())
		n = pending
	}
	ll.length -= pruned
}

// Amount returns total depth liquidity and value
func (ll *linkedList) amount() (liquidity, value float64) {
	for tip := ll.head; tip != nil; tip = tip.next {
		liquidity += tip.value.Amount
		value += tip.value.Amount * tip.value.Price
	}
	return
}

// Retrieve returns a full slice of contents from the linked list
func (ll *linkedList) retrieve() Items {
	depth := make(Items, ll.length)
	iterator := 0
	for tip := ll.head; tip != nil; tip = tip.next {
		depth[iterator] = tip.value
		iterator++
	}
	return depth
}

// bids embed a linked list to attach methods for bid depth specific
// functionality
type bids struct {
	linkedList
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates in bid linked list
func (ll *bids) updateInsertByPrice(updts Items, stack *stack, maxChainLength int) {
	for x := range updts {
		for tip := &ll.head; ; tip = &(*tip).next {
			if *tip == nil {
				insertHeadSpecific(&ll.linkedList, updts[x], stack)
				break
			}
			if (*tip).value.Price == updts[x].Price { // Match check
				if updts[x].Amount <= 0 { // Capture delete update
					stack.Push(deleteAtTip(&ll.linkedList, tip), time.Now())
				} else { // Amend current amount value
					(*tip).value.Amount = updts[x].Amount
				}
				break // Continue updates
			}

			if (*tip).value.Price < updts[x].Price { // Insert
				// This check below filters zero values and provides an
				// optimisation for when select exchanges send a delete update
				// to a non-existent price level (OTC/Hidden order) so we can
				// break instantly and reduce the traversal of the entire chain.
				if updts[x].Amount > 0 {
					insertAtTip(&ll.linkedList, tip, updts[x], stack)
				}
				break // Continue updates
			}

			if (*tip).next == nil { // Tip is at tail
				// This check below is just a catch all in the event the above
				// zero value check fails
				if updts[x].Amount > 0 {
					insertAtTail(&ll.linkedList, tip, updts[x], stack)
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

// updateInsertByID updates or inserts if not found
// 1) node ID found amount amended (best case)
// 2) node ID found amount and price amended and node moved to correct position
// (medium case)
// 3) Update price exceeds traversal node price before ID found, save node
// address for either; node ID matches then re-address node or end of depth pop
// a node from the stack (worst case)
func (ll *bids) updateInsertByID(updts Items, stack *stack) error {
updates:
	for x := range updts {
		if updts[x].Amount == 0 {
			return errAmountCannotBeZero
		}
		// bookmark allows for saving of a position of a node in the event that
		// an update price exceeds the current node price. We can then match an
		// ID and re-assign that ID's node to that positioning without popping
		// from the stack and then pushing to the stack later for cleanup.
		// If the ID is not found we can pop from stack then insert into that
		// price level
		var bookmark *node
		for tip := ll.head; tip != nil; tip = tip.next {
			if tip.value.ID == updts[x].ID {
				if tip.value.Price != updts[x].Price { // Price level change
					// bookmark tip to move this node to correct price level
					bookmark = tip
					continue // continue through node depth
				}
				// no price change, amend amount and continue update
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
				if shiftBookmark(tip, &bookmark, &ll.head, updts[x]) {
					continue updates
				}
			}
		}
		n := stack.Pop()
		n.value = updts[x]
		insertNodeAtBookmark(&ll.linkedList, bookmark, n) // Won't inline with stack
	}
	return nil
}

// insertUpdates inserts new updates for bids based on price level
func (ll *bids) insertUpdates(updts Items, stack *stack) error {
	for x := range updts {
		var prev *node
		for tip := &ll.head; ; tip = &(*tip).next {
			if *tip == nil { // Head
				n := stack.Pop()
				n.value = updts[x]
				n.prev = prev
				ll.length++
				*tip = n
				break // Continue updates
			}

			if (*tip).value.Price == updts[x].Price { // Price already found
				return fmt.Errorf("%w for price %f",
					errCollisionDetected,
					updts[x].Price)
			}

			if (*tip).value.Price < updts[x].Price { // Alignment
				n := stack.Pop()
				n.value = updts[x]
				n.prev = prev
				ll.length++
				// Reference current with new node
				(*tip).prev = n
				// Push tip to the right
				n.next = *tip
				// This is the same as prev.next = n
				*tip = n
				break // Continue updates
			}

			if (*tip).next == nil { // Tail
				insertAtTail(&ll.linkedList, tip, updts[x], stack)
				break // Continue updates
			}
			prev = *tip
		}
	}
	return nil
}

// asks embed a linked list to attach methods for ask depth specific
// functionality
type asks struct {
	linkedList
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates in an ask linked list
func (ll *asks) updateInsertByPrice(updts Items, stack *stack, maxChainLength int) {
	for x := range updts {
		for tip := &ll.head; ; tip = &(*tip).next {
			if *tip == nil {
				insertHeadSpecific(&ll.linkedList, updts[x], stack)
				break
			}
			if (*tip).value.Price == updts[x].Price { // Match check
				if updts[x].Amount <= 0 { // Capture delete update
					stack.Push(deleteAtTip(&ll.linkedList, tip), time.Now())
				} else { // Amend current amount value
					(*tip).value.Amount = updts[x].Amount
				}
				break // Continue updates
			}

			if (*tip).value.Price > updts[x].Price { // Insert
				// This check below filters zero values and provides an
				// optimisation for when select exchanges send a delete update
				// to a non-existent price level (OTC/Hidden order) so we can
				// break instantly and reduce the traversal of the entire chain.
				if updts[x].Amount > 0 {
					insertAtTip(&ll.linkedList, tip, updts[x], stack)
				}
				break // Continue updates
			}

			if (*tip).next == nil { // Tip is at tail
				// This check below is just a catch all in the event the above
				// zero value check fails
				if updts[x].Amount > 0 {
					insertAtTail(&ll.linkedList, tip, updts[x], stack)
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

// updateInsertByID updates or inserts if not found
// 1) node ID found amount amended (best case)
// 2) node ID found amount and price amended and node moved to correct position
// (medium case)
// 3) Update price exceeds traversal node price before ID found, save node
// address for either; node ID matches then re-address node or end of depth pop
// a node from the stack (worst case)
func (ll *asks) updateInsertByID(updts Items, stack *stack) error {
updates:
	for x := range updts {
		if updts[x].Amount == 0 {
			return errAmountCannotBeZero
		}
		// bookmark allows for saving of a position of a node in the event that
		// an update price exceeds the current node price. We can then match an
		// ID and re-assign that ID's node to that positioning without popping
		// from the stack and then pushing to the stack later for cleanup.
		// If the ID is not found we can pop from stack then insert into that
		// price level
		var bookmark *node
		for tip := ll.head; tip != nil; tip = tip.next {
			if tip.value.ID == updts[x].ID {
				if tip.value.Price != updts[x].Price { // Price level change
					// bookmark tip to move this node to correct price level
					bookmark = tip
					continue // continue through node depth
				}
				// no price change, amend amount and continue updates
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
				if shiftBookmark(tip, &bookmark, &ll.head, updts[x]) {
					continue updates
				}
			}
		}
		n := stack.Pop()
		n.value = updts[x]
		insertNodeAtBookmark(&ll.linkedList, bookmark, n) // Won't inline with stack
	}
	return nil
}

// insertUpdates inserts new updates for asks based on price level
func (ll *asks) insertUpdates(updts Items, stack *stack) error {
	for x := range updts {
		var prev *node
		for tip := &ll.head; ; tip = &(*tip).next {
			if *tip == nil { // Head is empty
				insertHeadSpecific(&ll.linkedList, updts[x], stack)
				break // Continue updates
			}

			if (*tip).value.Price == updts[x].Price { // Price already found
				return fmt.Errorf("%w for price %f",
					errCollisionDetected,
					updts[x].Price)
			}

			// Correct position/alignment found for price level
			if (*tip).value.Price > updts[x].Price {
				n := stack.Pop()
				n.value = updts[x]
				n.prev = prev
				ll.length++
				if prev == nil {
					// Place new node in front of current node
					(*tip).prev = n
					n.next = *tip
					// Replace head entry
					*tip = n
				} else {
					old := *tip
					prev.next = n
					n.next = old
					old.prev = n
				}
				break // Continue updates
			}

			if (*tip).next == nil { // Tail
				insertAtTail(&ll.linkedList, tip, updts[x], stack)
				break // Continue updates
			}
			prev = *tip
		}
	}
	return nil
}

// move moves a node from a point in a node chain to another node position,
// this left justified towards head as element zero is the top of the depth
// side. (can inline)
func move(head **node, from, to *node) {
	if from.next != nil { // From is at tail
		from.next.prev = from.prev
	}
	if from.prev == nil { // From is at head
		(*head).next.prev = nil
		*head = (*head).next
	} else {
		from.prev.next = from.next
	}
	// insert from node next to 'to' node
	if to.prev == nil { // Destination is at head position
		*head = from
	} else {
		to.prev.next = from
	}
	from.prev = to.prev
	to.prev = from
	from.next = to
}

// deleteAtTip removes a node from tip target returns old node (can inline)
func deleteAtTip(ll *linkedList, tip **node) *node {
	// Old is a placeholder for current tips node value to push
	// back on to the stack.
	old := *tip
	switch {
	case old.prev == nil: // At head position
		// shift current tip head to the right
		*tip = old.next
		// Remove reference to node from chain
		if old.next != nil { // This is when liquidity hits zero
			old.next.prev = nil
		}
	case old.next == nil: // At tail position
		// Remove reference to node from chain
		old.prev.next = nil
	default:
		// Reference prior node in chain to next node in chain
		// bypassing current node
		old.prev.next = old.next
		old.next.prev = old.prev
	}
	ll.length--
	return old
}

// insertAtTip inserts at a tip target (can inline)
func insertAtTip(ll *linkedList, tip **node, updt Item, stack *stack) {
	n := stack.Pop()
	n.value = updt
	n.next = *tip
	n.prev = (*tip).prev
	if (*tip).prev == nil { // Tip is at head
		// Replace head which will push everything to the right
		// when this node will reference new node below
		*tip = n
	} else {
		// Reference new node to previous node
		(*tip).prev.next = n
	}
	// Reference next node to new node
	n.next.prev = n
	ll.length++
}

// insertAtTail inserts at tail end of node chain (can inline)
func insertAtTail(ll *linkedList, tip **node, updt Item, stack *stack) {
	n := stack.Pop()
	n.value = updt
	// Reference tip to new node
	(*tip).next = n
	// Reference new node with current tip
	n.prev = *tip
	ll.length++
}

// insertHeadSpecific inserts at head specifically there might be an instance
// where the liquidity on an exchange does fall to zero through a streaming
// endpoint then it comes back online. (can inline)
func insertHeadSpecific(ll *linkedList, updt Item, stack *stack) {
	n := stack.Pop()
	n.value = updt
	ll.head = n
	ll.length++
}

// insertNodeAtBookmark inserts a new node at a bookmarked node position
// returns if a node needs to replace head (can inline)
func insertNodeAtBookmark(ll *linkedList, bookmark, n *node) {
	switch {
	case bookmark == nil: // Zero liquidity and we are rebuilding from scratch
		ll.head = n
	case bookmark.prev == nil:
		n.prev = bookmark.prev
		bookmark.prev = n
		n.next = bookmark
		ll.head = n
	case bookmark.next == nil:
		n.prev = bookmark
		bookmark.next = n
	default:
		bookmark.prev.next = n
		n.prev = bookmark.prev
		bookmark.prev = n
		n.next = bookmark
	}
	ll.length++
}

// shiftBookmark moves a bookmarked node to the tip position or if nil sets
// tip as bookmark (can inline)
func shiftBookmark(tip *node, bookmark, head **node, updt Item) bool {
	if *bookmark == nil { // End of the chain and no bookmark set
		*bookmark = tip // Set tip to bookmark so we can set a new node there
		return false
	}
	(*bookmark).value = updt
	(*bookmark).next.prev = (*bookmark).prev
	if (*bookmark).prev == nil { // Bookmark is at head
		*head = (*bookmark).next
	} else {
		(*bookmark).prev.next = (*bookmark).next
	}
	tip.next = *bookmark
	(*bookmark).prev = tip
	(*bookmark).next = nil
	return true
}
