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
	errNoLiquidity                     = errors.New("no liquidity")
)

type linkedList []Item

// comparison defines expected functionality to compare between two reference
// price levels
type comparison func(float64, float64) bool

// load iterates across new items and refreshes linked list. It creates a linked
// list exactly the same as the item slice that is supplied, if items is of nil
// value it will flush entire list.
func (ll *linkedList) load(items Items) {
	if len(items) == 0 {
		*ll = (*ll)[:0] // Flush
		return
	}
	if len(items) <= len(*ll) {
		copy(*ll, items)         // Reuse
		*ll = (*ll)[:len(items)] // Flush excess
		return
	}
	if len(items) > cap(*ll) {
		*ll = make([]Item, len(items)) // Extend
		copy(*ll, items)               // Copy
		return
	}
	*ll = (*ll)[:0]             // Flush
	*ll = append(*ll, items...) // Append
}

// updateByID amends price by corresponding ID and returns an error if not found
func (ll linkedList) updateByID(updts []Item) error {
updates:
	for x := range updts {
		for y := range ll {
			if updts[x].ID != ll[y].ID { // Filter IDs that don't match
				continue
			}
			if updts[x].Price > 0 {
				// Only apply changes when zero values are not present, Bitmex
				// for example sends 0 price values.
				ll[y].Price = updts[x].Price
				ll[y].StrPrice = updts[x].StrPrice
			}
			ll[y].Amount = updts[x].Amount
			ll[y].StrAmount = updts[x].StrAmount
			continue updates
		}
		return fmt.Errorf("update error: %w ID: %d not found",
			errIDCannotBeMatched,
			updts[x].ID)
	}
	return nil
}

// deleteByID deletes reference by ID
func (ll *linkedList) deleteByID(updts Items, bypassErr bool) error {
updates:
	for x := range updts {
		for y := range *ll {
			if updts[x].ID != (*ll)[y].ID {
				continue
			}

			*ll = append((*ll)[:y], (*ll)[y+1:]...)
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

// amount returns total depth liquidity and value
func (ll linkedList) amount() (liquidity, value float64) {
	for x := range ll {
		liquidity += ll[x].Amount
		value += ll[x].Amount * ll[x].Price
	}
	return
}

// retrieve returns a full slice of contents from the linked list
func (ll linkedList) retrieve(count int) Items {
	if count == 0 || len(ll) < count {
		count = len(ll)
	}
	return Items(ll[:count])
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates
func (ll *linkedList) updateInsertByPrice(updts Items, maxChainLength int, compare func(float64, float64) bool) {
updateroo:
	for x := range updts {
		for y := range *ll {
			switch {
			case (*ll)[y].Price == updts[x].Price:
				if updts[x].Amount <= 0 {
					// Delete
					if y+1 == len(*ll) {
						*ll = (*ll)[:y]
					} else {
						*ll = append((*ll)[:y], (*ll)[y+1:]...)
					}
				} else {
					// Update
					(*ll)[y].Amount = updts[x].Amount
					(*ll)[y].StrAmount = updts[x].StrAmount
				}
				continue updateroo
			case compare((*ll)[y].Price, updts[x].Price):
				if updts[x].Amount > 0 {
					*ll = append(*ll, Item{})    // Extend
					copy((*ll)[y+1:], (*ll)[y:]) // Copy elements from index y onwards one position to the right
					(*ll)[y] = updts[x]          // Insert updts[x] at index y
				}
				continue updateroo
			}
		}
		if updts[x].Amount > 0 {
			*ll = append(*ll, updts[x])
		}
	}
	// Reduces length of total linked list chain to a maxChainLength value
	if maxChainLength != 0 && len(*ll) > maxChainLength {
		*ll = (*ll)[:maxChainLength]
	}
}

// updateInsertByID updates or inserts if not found for a bid or ask depth
// 1) node ID found amount amended (best case)
// 2) node ID found amount and price amended and node moved to correct position
// (medium case)
// 3) Update price exceeds traversal node price before ID found, save node
// address for either; node ID matches then re-address node or end of depth pop
// a node from the stack (worst case)
func (ll *linkedList) updateInsertByID(updts Items, compare comparison) error {
updates:
	for x := range updts {
		if updts[x].Amount <= 0 {
			return errAmountCannotBeLessOrEqualToZero
		}
		var popped bool
		for y := 0; y < len(*ll); y++ {
			if (*ll)[y].ID == updts[x].ID {
				if (*ll)[y].Price != updts[x].Price { // Price level change
					if y+1 == len(*ll) { // end of depth
						// no movement needed just a re-adjustment
						(*ll)[y] = updts[x]
						continue updates
					}
					copy((*ll)[y:], (*ll)[y+1:]) // remove him: cya m8
					*ll = (*ll)[:len(*ll)-1]     // keep underlying array
					y--                          // adjust index
					popped = true
					continue // continue through node depth
				}
				// no price change, amend amount and continue update
				(*ll)[y].Amount = updts[x].Amount
				(*ll)[y].StrAmount = updts[x].StrAmount
				continue updates // continue to next update
			}

			if compare((*ll)[y].Price, updts[x].Price) {
				*ll = append(*ll, Item{})    // Extend
				copy((*ll)[y+1:], (*ll)[y:]) // Copy elements from index y onwards one position to the right
				(*ll)[y] = updts[x]          // Insert updts[x] at index y

				if popped { // already found ID and popped
					continue updates
				}

				// search for ID
				for z := y + 1; z < len(*ll); z++ {
					if (*ll)[z].ID == updts[x].ID {
						copy((*ll)[z:], (*ll)[z+1:]) // remove him: cya m8
						*ll = (*ll)[:len(*ll)-1]     // keep underlying array
						break
					}
				}
				continue updates
			}
		}
		*ll = append(*ll, updts[x])
	}
	return nil
}

// insertUpdates inserts new updates for bids or asks based on price level
func (ll *linkedList) insertUpdates(updts Items, comp comparison) error {
updaterino:
	for x := range updts {
		if len(*ll) == 0 { // TODO: Offset this and outline
			*ll = append(*ll, updts[x])
			continue
		}

		for y := range *ll {
			switch {
			case (*ll)[y].Price == updts[x].Price: // Price already found
				return fmt.Errorf("%w for price %f", errCollisionDetected, updts[x].Price)
			case comp((*ll)[y].Price, updts[x].Price): // price at correct spot
				*ll = append((*ll)[:y], append([]Item{updts[x]}, (*ll)[y:]...)...)
				continue updaterino
			}
		}
		*ll = append(*ll, updts[x])
	}
	return nil
}

// getHeadPriceNoLock gets best/head price
func (ll linkedList) getHeadPriceNoLock() (float64, error) {
	if len(ll) == 0 {
		return 0, errNoLiquidity
	}
	return ll[0].Price, nil
}

// getHeadVolumeNoLock gets best/head volume
func (ll linkedList) getHeadVolumeNoLock() (float64, error) {
	if len(ll) == 0 {
		return 0, errNoLiquidity
	}
	return ll[0].Amount, nil
}

// getMovementByQuotation traverses through orderbook liquidity using quotation
// currency as a limiter and returns orderbook movement details. Swap boolean
// allows the swap of sold and purchased to reduce code so it doesn't need to be
// specific to bid or ask.
func (ll linkedList) getMovementByQuotation(quote, refPrice float64, swap bool) (*Movement, error) {
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
	for x := range ll {
		trancheValue := ll[x].Amount * ll[x].Price
		leftover := quote - trancheValue
		if leftover < 0 {
			m.Purchased += quote
			m.Sold += quote / trancheValue * ll[x].Amount
			// This tranche is not consumed so the book shifts to this price.
			m.EndPrice = ll[x].Price
			quote = 0
			break
		}
		// Full tranche consumed
		m.Purchased += ll[x].Price * ll[x].Amount
		m.Sold += ll[x].Amount
		quote = leftover
		if leftover == 0 {
			// Price no longer exists on the book so use next full price tranche
			// to calculate book impact. If available.
			if x+1 < len(ll) {
				m.EndPrice = ll[x+1].Price
			} else {
				m.FullBookSideConsumed = true
			}
			break
		}
	}
	return m.finalizeFields(m.Purchased, m.Sold, head, quote, swap)
}

// getMovementByBase traverses through orderbook liquidity using base currency
// as a limiter and returns orderbook movement details. Swap boolean allows the
// swap of sold and purchased to reduce code so it doesn't need to be specific
// to bid or ask.
func (ll linkedList) getMovementByBase(base, refPrice float64, swap bool) (*Movement, error) {
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
	for x := range ll {
		leftover := base - ll[x].Amount
		if leftover < 0 {
			m.Purchased += ll[x].Price * base
			m.Sold += base
			// This tranche is not consumed so the book shifts to this price.
			m.EndPrice = ll[x].Price
			base = 0
			break
		}
		// Full tranche consumed
		m.Purchased += ll[x].Price * ll[x].Amount
		m.Sold += ll[x].Amount
		base = leftover
		if leftover == 0 {
			// Price no longer exists on the book so use next full price tranche
			// to calculate book impact.
			if x+1 < len(ll) {
				m.EndPrice = ll[x+1].Price
			} else {
				m.FullBookSideConsumed = true
			}
			break
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
func (ll *bids) updateInsertByPrice(updts Items, maxChainLength int) {
	ll.linkedList.updateInsertByPrice(updts, maxChainLength, bidCompare)
}

// updateInsertByID updates or inserts if not found
func (ll *bids) updateInsertByID(updts Items) error {
	return ll.linkedList.updateInsertByID(updts, bidCompare)
}

// insertUpdates inserts new updates for bids based on price level
func (ll *bids) insertUpdates(updts Items) error {
	return ll.linkedList.insertUpdates(updts, bidCompare)
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

	if len(ll.linkedList) == 0 {
		return nil, errNoLiquidity
	}

	nominal := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	var cumulativeValue, cumulativeAmounts float64
	for x := range ll.linkedList {
		totalTrancheValue := ll.linkedList[x].Price * ll.linkedList[x].Amount
		currentFullValue := totalTrancheValue + cumulativeValue
		currentTotalAmounts := cumulativeAmounts + ll.linkedList[x].Amount

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
			trancheTargetPriceDiff := ll.linkedList[x].Price - targetCost
			trancheAmountExpectation := comparativeDiff / trancheTargetPriceDiff
			nominal.NominalPercentage = slippage
			nominal.Sold = cumulativeAmounts + trancheAmountExpectation
			nominal.Purchased += trancheAmountExpectation * ll.linkedList[x].Price
			nominal.AverageOrderCost = nominal.Purchased / nominal.Sold
			nominal.EndPrice = ll.linkedList[x].Price
			return nominal, nil
		}

		nominal.EndPrice = ll.linkedList[x].Price
		cumulativeValue = currentFullValue
		nominal.NominalPercentage = percent
		nominal.Sold += ll.linkedList[x].Amount
		nominal.Purchased += totalTrancheValue
		cumulativeAmounts = currentTotalAmounts
		if slippage == percent {
			nominal.FullBookSideConsumed = x+1 >= len(ll.linkedList)
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

	if len(ll.linkedList) == 0 {
		return nil, errNoLiquidity
	}

	impact := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	for x := range ll.linkedList {
		percent := math.CalculatePercentageGainOrLoss(ll.linkedList[x].Price, refPrice)
		if percent != 0 {
			percent *= -1
		}
		impact.EndPrice = ll.linkedList[x].Price
		impact.ImpactPercentage = percent
		if slippage <= percent {
			// Don't include this tranche amount as this consumes the tranche
			// book price, thus obtaining a higher percentage impact.
			return impact, nil
		}
		impact.Sold += ll.linkedList[x].Amount
		impact.Purchased += ll.linkedList[x].Amount * ll.linkedList[x].Price
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
func (ll *asks) updateInsertByPrice(updts Items, maxChainLength int) {
	ll.linkedList.updateInsertByPrice(updts, maxChainLength, askCompare)
}

// updateInsertByID updates or inserts if not found
func (ll *asks) updateInsertByID(updts Items) error {
	return ll.linkedList.updateInsertByID(updts, askCompare)
}

// insertUpdates inserts new updates for asks based on price level
func (ll *asks) insertUpdates(updts Items) error {
	return ll.linkedList.insertUpdates(updts, askCompare)
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

	if len(ll.linkedList) == 0 {
		return nil, errNoLiquidity
	}

	nominal := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	var cumulativeAmounts float64
	for x := range ll.linkedList {
		totalTrancheValue := ll.linkedList[x].Price * ll.linkedList[x].Amount
		currentValue := totalTrancheValue + nominal.Sold
		currentAmounts := cumulativeAmounts + ll.linkedList[x].Amount

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
			trancheTargetPriceDiff := ll.linkedList[x].Price - targetCost
			trancheAmountExpectation := comparativeDiff / trancheTargetPriceDiff
			nominal.NominalPercentage = slippage
			nominal.Sold += trancheAmountExpectation * ll.linkedList[x].Price
			nominal.Purchased += trancheAmountExpectation
			nominal.AverageOrderCost = nominal.Sold / nominal.Purchased
			nominal.EndPrice = ll.linkedList[x].Price
			return nominal, nil
		}

		nominal.EndPrice = ll.linkedList[x].Price
		nominal.Sold = currentValue
		nominal.Purchased += ll.linkedList[x].Amount
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

	if len(ll.linkedList) == 0 {
		return nil, errNoLiquidity
	}

	impact := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	for x := range ll.linkedList {
		percent := math.CalculatePercentageGainOrLoss(ll.linkedList[x].Price, refPrice)
		impact.ImpactPercentage = percent
		impact.EndPrice = ll.linkedList[x].Price
		if slippage <= percent {
			// Don't include this tranche amount as this consumes the tranche
			// book price, thus obtaining a higher percentage impact.
			return impact, nil
		}
		impact.Sold += ll.linkedList[x].Amount * ll.linkedList[x].Price
		impact.Purchased += ll.linkedList[x].Amount
		impact.AverageOrderCost = impact.Sold / impact.Purchased
	}
	impact.FullBookSideConsumed = true
	impact.ImpactPercentage = FullLiquidityExhaustedPercentage
	return impact, nil
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
