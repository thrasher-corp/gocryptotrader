package orderbook

import (
	"errors"
	"fmt"
)

var errIDCannotBeMatched = errors.New("cannot match ID on linked list")
var errCollisionDetected = errors.New("cannot insert update collision detected")

// linkedList defines a linked list for a depth level, reutilisation of nodes
// to and from a stack.
// TODO: Test cross link between bid and ask head nodes **node ref to head for
// future strategy assymetric traversal between two books
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
	for i := 0; i < len(items); i++ {
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
		prev = (*tip)
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
		stack.Push(push)
		ll.length--
		push = pending
	}
}

// updateByID ammends price by corresponding ID and returns an error if not
// found
func (ll *linkedList) updateByID(updts []Item) error {
updates:
	for x := range updts {
		for tip := ll.head; tip != nil; tip = tip.next {
			if updts[x].ID == tip.value.ID { // Match ID
				tip.value.Price = updts[x].Price
				tip.value.Amount = updts[x].Amount
				continue updates
			}
		}
		return fmt.Errorf("update error: %w %d not found",
			errIDCannotBeMatched,
			updts[x].ID)
	}
	return nil
}

// deleteByID deletes refererence by ID
func (ll *linkedList) deleteByID(updts Items, stack *stack, bypassErr bool) error {
updates:
	for x := range updts {
		for tip := &ll.head; *tip != nil; tip = &(*tip).next {
			if updts[x].ID == (*tip).value.ID {
				old := *tip
				if old.prev == nil { // Tip is at head
					// Shift everything to the left by setting the next node
					*tip = old.next
					// Dereference old node from current chain
					if old.next != nil { // This is when liquidity hits zero
						old.next.prev = nil
					}
				} else if old.next == nil { // Tip is at tail
					// Remove old node
					*tip = old.prev
					old.prev.next = nil
				} else {
					// Bypass old node with its prev and next
					old.prev.next = old.next
					old.next.prev = old.prev
				}
				stack.Push(old)
				ll.length--
				continue updates
			}
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
// at the end.
func (ll *linkedList) cleanup(maxChainLength int, stack *stack) {
	// Reduces the max length of total linked list chain, occurs after updates
	// have been implemented as updates can push length out of bounds, if
	// cleaved after that update, new update might not applied correcly.
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

// Liquidity returns total depth liquidity
func (ll *linkedList) liquidity() (liquidity float64) {
	for tip := ll.head; tip != nil; tip = tip.next {
		liquidity += tip.value.Amount
	}
	return
}

// Value returns total value on price.amount on full depth
func (ll *linkedList) value() (value float64) {
	for tip := ll.head; tip != nil; tip = tip.next {
		value += tip.value.Amount * tip.value.Price
	}
	return
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

// bids imbed a linked list to attach methods for bid depth specific
// functionaliy
type bids struct {
	linkedList
}

// updateInsertByPrice ammends, inserts, moves and cleaves length of depth by
// updates in bid linked list
func (ll *bids) updateInsertByPrice(updts Items, stack *stack, maxChainLength int) {
updates:
	for x := range updts {
		for tip := &ll.head; *tip != nil; tip = &(*tip).next {
			if (*tip).value.Price == updts[x].Price { // Match check
				if updts[x].Amount <= 0 { // Capture delete update
					// Old is a placeholder for current tips node value to push
					// back on to the stack.
					old := *tip
					if old.prev == nil { // At head position
						// shift current tip head to the right
						*tip = old.next
						// Remove reference to node from chain
						if old.next != nil { // This is when liquidity hits zero
							old.next.prev = nil
						}
					} else if old.next == nil { // At tail position
						// Remove reference to node from chain
						old.prev.next = nil
					} else {
						// Reference prior node in chain to next node in chain
						// bypassing current node
						old.prev.next = old.next
						old.next.prev = old.prev
					}
					stack.Push(old)
					ll.length--
				} else { // Amend current amount value
					(*tip).value.Amount = updts[x].Amount
				}
				continue updates
			}

			if (*tip).value.Price < updts[x].Price { // Insert
				if updts[x].Amount > 0 { // Filter delete, should already be
					// removed
					n := stack.Pop()
					n.value = updts[x]
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
				continue updates
			}

			if (*tip).next == nil { // Tip is at tail
				if updts[x].Amount > 0 {
					n := stack.Pop()
					n.value = updts[x]
					// Reference tip to new node
					(*tip).next = n
					// Reference new node with current tip
					n.prev = *tip
					ll.length++
				}
			}
		}
	}
	// Reduces length of total linked list chain to a maxChainLength value
	ll.cleanup(maxChainLength, stack)
}

// updateInsertByID updates or inserts if not found
// 1) node ID found amount amended (best case)
// 2) node ID found amount and price amended and node moved to correct position
// (medium case)
// 3) Update price exceeds traversal node price before ID found, save node
// address for either; node ID matches then re-address node or end of depth pop
// a node from the stack (worst case)
func (ll *bids) updateInsertByID(updts Items, stack *stack) {
updates:
	for x := range updts {
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
					if bookmark.prev == nil { // Bookmark is at head
						ll.head = bookmark.next
					} else {
						bookmark.prev.next = bookmark.next
					}
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
			n.next = ll.head
			ll.head.prev = n
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

// insertUpdates inserts new updates for bids based on price level
func (ll *bids) insertUpdates(updts Items, stack *stack) error {
updates:
	for x := range updts {
		var prev *node
		for tip := &ll.head; ; tip = &(*tip).next {
			if *tip == nil { // Head
				n := stack.Pop()
				n.value = updts[x]
				n.prev = prev
				ll.length++
				*tip = n
				continue updates
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
				if (*tip).prev == nil {
					// Place new node in front of current node
					(*tip).prev = n
					n.next = *tip
					// Replace head entry
					*tip = n
				} else {
					// Reference current with new node
					(*tip).prev = n
					// Push tip to the right
					n.next = *tip
					// This is the same as prev.next = n
					*tip = n

				}
				continue updates
			}

			if (*tip).next == nil { // Tail
				n := stack.Pop()
				n.value = updts[x]
				n.prev = prev
				ll.length++
				(*tip).next = n
				n.prev = *tip
				continue updates
			}
			prev = *tip
		}
	}
	return nil
}

// asks imbed a linked list to attach methods for ask depth specific
// functionaliy
type asks struct {
	linkedList
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates in an ask linked list
func (ll *asks) updateInsertByPrice(updts Items, stack *stack, maxChainLength int) {
updates:
	for x := range updts {
		for tip := &ll.head; *tip != nil; tip = &(*tip).next {
			if (*tip).value.Price == updts[x].Price { // Match check
				if updts[x].Amount <= 0 { // Capture delete update
					// Old is a placeholder for current tips node value to push
					// back on to the stack.
					old := *tip
					if old.prev == nil { // At head position
						// shift current tip head to the right
						*tip = old.next
						// Remove reference to node from chain
						if old.next != nil { // This is when liquidity hits zero
							old.next.prev = nil
						}
					} else if old.next == nil { // At tail position
						// Remove reference to node from chain
						old.prev.next = nil
					} else {
						// Reference prior node in chain to next node in chain
						// bypassing current node
						old.prev.next = old.next
						old.next.prev = old.prev
					}
					stack.Push(old)
					ll.length--
				} else { // Amend current amount value
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
				continue updates
			}

			if (*tip).next == nil { // Tip is at tail
				if updts[x].Amount > 0 {
					n := stack.Pop()
					n.value = updts[x]
					// Reference tip to new node
					(*tip).next = n
					// Reference new node with current tip
					n.prev = *tip
					ll.length++
				}
			}
		}
	}
	// Reduces length of total linked list chain to a maxChainLength value
	ll.cleanup(maxChainLength, stack)
}

// updateInsertByID updates or inserts if not found
// 1) node ID found amount amended (best case)
// 2) node ID found amount and price amended and node moved to correct position
// (medium case)
// 3) Update price exceeds traversal node price before ID found, save node
// address for either; node ID matches then re-address node or end of depth pop
// a node from the stack (worst case)
func (ll *asks) updateInsertByID(updts Items, stack *stack) {
updates:
	for x := range updts {
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
				// no price change, ammend amount and conintue updates
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
					if bookmark.prev == nil { // Bookmark is at head
						ll.head = bookmark.next
					} else {
						bookmark.prev.next = bookmark.next
					}
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
			n.prev = bookmark.prev
			bookmark.prev = n
			n.next = bookmark
		} else if bookmark.next == nil {
			n.prev = bookmark
			bookmark.next = n
		} else {
			bookmark.prev.next = n
			n.prev = bookmark.prev
			bookmark.prev = n
			n.next = bookmark
		}
		ll.length++
	}
}

// insertUpdates inserts new updates for asks based on price level
func (ll *asks) insertUpdates(updts Items, stack *stack) error {
updates:
	for x := range updts {
		var prev *node
		for tip := &ll.head; ; tip = &(*tip).next {
			if *tip == nil { // Head is empty
				// This is here because there might be an instance where the
				// liquidity on an exchange does fall to zero through a
				// streaming endpoint then it comes back online.
				n := stack.Pop()
				n.value = updts[x]
				n.prev = prev
				ll.length++
				*tip = n
				continue updates
			}

			if (*tip).value.Price == updts[x].Price { // Price already found
				return fmt.Errorf("%w for price %f",
					errCollisionDetected,
					updts[x].Price)
			}

			// Correct position/allignment found for price level
			if (*tip).value.Price > updts[x].Price {
				n := stack.Pop()
				n.value = updts[x]
				n.prev = prev
				ll.length++
				if (*tip).prev == nil {
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
				continue updates
			}

			if (*tip).next == nil { // Tail
				n := stack.Pop()
				n.value = updts[x]
				n.prev = prev
				ll.length++
				(*tip).next = n
				n.prev = *tip
				continue updates
			}
			prev = *tip
		}
	}
	return nil
}

// move moves a node from a point in a node chain to another node position,
// this left justified towards head as element zero is the top of the depth
// side.
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
