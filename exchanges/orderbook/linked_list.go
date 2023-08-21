package orderbook

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common/math"
)

// FullLiquidityExhaustedPercentage defines when a book has been completely
// wiped out of potential liquidity.
const FullLiquidityExhaustedPercentage = -100

var (
	errIDCannotBeMatched               = errors.New("cannot match ID on linked list")
	errCollisionDetected               = errors.New("cannot insert update, collision detected")
	errAmountCannotBeLessOrEqualToZero = errors.New("amount cannot be less than or equal to zero")
	errInvalidNominalSlippage          = errors.New("invalid slippage amount, its value must be greater than or equal to zero")
	errInvalidImpactSlippage           = errors.New("invalid slippage amount, its value must be greater than zero")
	errInvalidSlippageCannotExceed100  = errors.New("invalid slippage amount, its value cannot exceed 100%")
	errBaseAmountInvalid               = errors.New("invalid base amount")
	errInvalidReferencePrice           = errors.New("invalid reference price")
	errQuoteAmountInvalid              = errors.New("quote amount invalid")
	errInvalidCost                     = errors.New("invalid cost amount")
	errInvalidAmount                   = errors.New("invalid amount")
	errInvalidHeadPrice                = errors.New("invalid head price")
)

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
		return fmt.Errorf("update error: %w ID: %d not found",
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
// at the end. (can't inline)
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
func (ll *linkedList) retrieve(count int) Items {
	if count == 0 || ll.length < count {
		count = ll.length
	}
	depth := make(Items, count)
	for i, tip := 0, ll.head; i < count && tip != nil; i, tip = i+1, tip.Next {
		depth[i] = tip.Value
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

// getHeadPriceNoLock gets best/head price
func (ll *linkedList) getHeadPriceNoLock() (float64, error) {
	if ll.head == nil {
		return 0, errNoLiquidity
	}
	return ll.head.Value.Price, nil
}

// getHeadVolumeNoLock gets best/head volume
func (ll *linkedList) getHeadVolumeNoLock() (float64, error) {
	if ll.head == nil {
		return 0, errNoLiquidity
	}
	return ll.head.Value.Amount, nil
}

// getMovementByQuotation traverses through orderbook liquidity using quotation
// currency as a limiter and returns orderbook movement details. Swap boolean
// allows the swap of sold and purchased to reduce code so it doesn't need to be
// specific to bid or ask.
func (ll *linkedList) getMovementByQuotation(quote, refPrice float64, swap bool) (*Movement, error) {
	if quote <= 0 {
		return nil, errQuoteAmountInvalid
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	head, err := ll.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}

	m := Movement{StartPrice: refPrice}
	for tip := ll.head; tip != nil; tip = tip.Next {
		trancheValue := tip.Value.Amount * tip.Value.Price
		leftover := quote - trancheValue
		if leftover < 0 {
			m.Purchased += quote
			m.Sold += quote / trancheValue * tip.Value.Amount
			// This tranche is not consumed so the book shifts to this price.
			m.EndPrice = tip.Value.Price
			quote = 0
			break
		}
		// Full tranche consumed
		m.Purchased += tip.Value.Price * tip.Value.Amount
		m.Sold += tip.Value.Amount
		quote = leftover
		if leftover == 0 {
			// Price no longer exists on the book so use next full price tranche
			// to calculate book impact. If available.
			if tip.Next != nil {
				m.EndPrice = tip.Next.Value.Price
			} else {
				m.FullBookSideConsumed = true
			}
		}
	}
	return m.finalizeFields(m.Purchased, m.Sold, head, quote, swap)
}

// getMovementByBase traverses through orderbook liquidity using base currency
// as a limiter and returns orderbook movement details. Swap boolean allows the
// swap of sold and purchased to reduce code so it doesn't need to be specific
// to bid or ask.
func (ll *linkedList) getMovementByBase(base, refPrice float64, swap bool) (*Movement, error) {
	if base <= 0 {
		return nil, errBaseAmountInvalid
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	head, err := ll.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}

	m := Movement{StartPrice: refPrice}
	for tip := ll.head; tip != nil; tip = tip.Next {
		leftover := base - tip.Value.Amount
		if leftover < 0 {
			m.Purchased += tip.Value.Price * base
			m.Sold += base
			// This tranche is not consumed so the book shifts to this price.
			m.EndPrice = tip.Value.Price
			base = 0
			break
		}
		// Full tranche consumed
		m.Purchased += tip.Value.Price * tip.Value.Amount
		m.Sold += tip.Value.Amount
		base = leftover
		if leftover == 0 {
			// Price no longer exists on the book so use next full price tranche
			// to calculate book impact.
			if tip.Next != nil {
				m.EndPrice = tip.Next.Value.Price
			} else {
				m.FullBookSideConsumed = true
			}
		}
	}
	return m.finalizeFields(m.Purchased, m.Sold, head, base, swap)
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

// hitBidsByNominalSlippage hits the bids by the required nominal slippage
// percentage, calculated from the reference price and returns orderbook
// movement details.
func (ll *bids) hitBidsByNominalSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage < 0 {
		return nil, errInvalidNominalSlippage
	}

	if slippage > 100 {
		return nil, errInvalidSlippageCannotExceed100
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if ll.head == nil {
		return nil, errNoLiquidity
	}

	nominal := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	var cumulativeValue, cumulativeAmounts float64
	for tip := ll.head; tip != nil; tip = tip.Next {
		totalTrancheValue := tip.Value.Price * tip.Value.Amount
		currentFullValue := totalTrancheValue + cumulativeValue
		currentTotalAmounts := cumulativeAmounts + tip.Value.Amount

		nominal.AverageOrderCost = currentFullValue / currentTotalAmounts
		percent := math.CalculatePercentageGainOrLoss(nominal.AverageOrderCost, refPrice)
		if percent != 0 {
			percent *= -1
		}

		if slippage < percent {
			targetCost := (1 - slippage/100) * refPrice
			if targetCost == refPrice {
				nominal.AverageOrderCost = cumulativeValue / cumulativeAmounts
				// Rounding issue on requested nominal percentage
				return nominal, nil
			}
			comparative := targetCost * cumulativeAmounts
			comparativeDiff := comparative - cumulativeValue
			trancheTargetPriceDiff := tip.Value.Price - targetCost
			trancheAmountExpectation := comparativeDiff / trancheTargetPriceDiff
			nominal.NominalPercentage = slippage
			nominal.Sold = cumulativeAmounts + trancheAmountExpectation
			nominal.Purchased += trancheAmountExpectation * tip.Value.Price
			nominal.AverageOrderCost = nominal.Purchased / nominal.Sold
			nominal.EndPrice = tip.Value.Price
			return nominal, nil
		}

		nominal.EndPrice = tip.Value.Price
		cumulativeValue = currentFullValue
		nominal.NominalPercentage = percent
		nominal.Sold += tip.Value.Amount
		nominal.Purchased += totalTrancheValue
		cumulativeAmounts = currentTotalAmounts
		if slippage == percent {
			nominal.FullBookSideConsumed = tip.Next == nil
			return nominal, nil
		}
	}
	nominal.FullBookSideConsumed = true
	return nominal, nil
}

// hitBidsByImpactSlippage hits the bids by the required impact slippage
// percentage, calculated from the reference price and returns orderbook
// movement details.
func (ll *bids) hitBidsByImpactSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage <= 0 {
		return nil, errInvalidImpactSlippage
	}

	if slippage > 100 {
		return nil, errInvalidSlippageCannotExceed100
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if ll.head == nil {
		return nil, errNoLiquidity
	}

	impact := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	for tip := ll.head; tip != nil; tip = tip.Next {
		percent := math.CalculatePercentageGainOrLoss(tip.Value.Price, refPrice)
		if percent != 0 {
			percent *= -1
		}
		impact.EndPrice = tip.Value.Price
		impact.ImpactPercentage = percent
		if slippage <= percent {
			// Don't include this tranche amount as this consumes the tranche
			// book price, thus obtaining a higher percentage impact.
			return impact, nil
		}
		impact.Sold += tip.Value.Amount
		impact.Purchased += tip.Value.Amount * tip.Value.Price
		impact.AverageOrderCost = impact.Purchased / impact.Sold
	}
	impact.FullBookSideConsumed = true
	impact.ImpactPercentage = FullLiquidityExhaustedPercentage
	return impact, nil
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

// liftAsksByNominalSlippage lifts the asks by the required nominal slippage
// percentage, calculated from the reference price and returns orderbook
// movement details.
func (ll *asks) liftAsksByNominalSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage < 0 {
		return nil, errInvalidNominalSlippage
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if ll.head == nil {
		return nil, errNoLiquidity
	}

	nominal := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	var cumulativeAmounts float64
	for tip := ll.head; tip != nil; tip = tip.Next {
		totalTrancheValue := tip.Value.Price * tip.Value.Amount
		currentValue := totalTrancheValue + nominal.Sold
		currentAmounts := cumulativeAmounts + tip.Value.Amount

		nominal.AverageOrderCost = currentValue / currentAmounts
		percent := math.CalculatePercentageGainOrLoss(nominal.AverageOrderCost, refPrice)

		if slippage < percent {
			targetCost := (1 + slippage/100) * refPrice
			if targetCost == refPrice {
				nominal.AverageOrderCost = nominal.Sold / nominal.Purchased
				// Rounding issue on requested nominal percentage
				return nominal, nil
			}

			comparative := targetCost * cumulativeAmounts
			comparativeDiff := comparative - nominal.Sold
			trancheTargetPriceDiff := tip.Value.Price - targetCost
			trancheAmountExpectation := comparativeDiff / trancheTargetPriceDiff
			nominal.NominalPercentage = slippage
			nominal.Sold += trancheAmountExpectation * tip.Value.Price
			nominal.Purchased += trancheAmountExpectation
			nominal.AverageOrderCost = nominal.Sold / nominal.Purchased
			nominal.EndPrice = tip.Value.Price
			return nominal, nil
		}

		nominal.EndPrice = tip.Value.Price
		nominal.Sold = currentValue
		nominal.Purchased += tip.Value.Amount
		nominal.NominalPercentage = percent
		if slippage == percent {
			return nominal, nil
		}
		cumulativeAmounts = currentAmounts
	}
	nominal.FullBookSideConsumed = true
	return nominal, nil
}

// liftAsksByImpactSlippage lifts the asks by the required impact slippage
// percentage, calculated from the reference price and returns orderbook
// movement details.
func (ll *asks) liftAsksByImpactSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage <= 0 {
		return nil, errInvalidImpactSlippage
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if ll.head == nil {
		return nil, errNoLiquidity
	}

	impact := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	for tip := ll.head; tip != nil; tip = tip.Next {
		percent := math.CalculatePercentageGainOrLoss(tip.Value.Price, refPrice)
		impact.ImpactPercentage = percent
		impact.EndPrice = tip.Value.Price
		if slippage <= percent {
			// Don't include this tranche amount as this consumes the tranche
			// book price, thus obtaining a higher percentage impact.
			return impact, nil
		}
		impact.Sold += tip.Value.Amount * tip.Value.Price
		impact.Purchased += tip.Value.Amount
		impact.AverageOrderCost = impact.Sold / impact.Purchased
	}
	impact.FullBookSideConsumed = true
	impact.ImpactPercentage = FullLiquidityExhaustedPercentage
	return impact, nil
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

// finalizeFields sets average order costing, percentages, slippage cost and
// preserves existing fields.
func (m *Movement) finalizeFields(cost, amount, headPrice, leftover float64, swap bool) (*Movement, error) {
	if cost <= 0 {
		return nil, errInvalidCost
	}

	if amount <= 0 {
		return nil, errInvalidAmount
	}

	if headPrice <= 0 {
		return nil, errInvalidHeadPrice
	}

	if m.StartPrice != m.EndPrice {
		// Average order cost defines the actual cost price as capital is
		// deployed through the orderbook liquidity.
		m.AverageOrderCost = cost / amount
	} else {
		// Edge case rounding issue for float64 with small numbers.
		m.AverageOrderCost = m.StartPrice
	}

	// Nominal percentage is the difference from the reference price to average
	// order cost.
	m.NominalPercentage = math.CalculatePercentageGainOrLoss(m.AverageOrderCost, m.StartPrice)
	if m.NominalPercentage < 0 {
		m.NominalPercentage *= -1
	}

	if !m.FullBookSideConsumed && leftover == 0 {
		// Impact percentage is how much the orderbook slips from the reference
		// price to the remaining tranche price.

		m.ImpactPercentage = math.CalculatePercentageGainOrLoss(m.EndPrice, m.StartPrice)
		if m.ImpactPercentage < 0 {
			m.ImpactPercentage *= -1
		}
	} else {
		// Full liquidity exhausted by request amount
		m.ImpactPercentage = FullLiquidityExhaustedPercentage
		m.FullBookSideConsumed = true
	}

	// Slippage cost is the difference in quotation terms between the actual
	// cost and the amounts at head price e.g.
	// Let P(n)=Price A(n)=Amount and iterate through a descending bid order example;
	// Cost: $270 (P1:100 x A1:1 + P2:90 x A2:1 + P3:80 x A3:1)
	// No slippage cost: $300 (P1:100 x A1:1 + P1:100 x A2:1 + P1:100 x A3:1)
	// $300 - $270 = $30 of slippage.
	m.SlippageCost = cost - (headPrice * amount)
	if m.SlippageCost < 0 {
		m.SlippageCost *= -1
	}

	// Swap saves on code duplication for difference in ask or bid amounts.
	if swap {
		m.Sold, m.Purchased = m.Purchased, m.Sold
	}

	return m, nil
}
