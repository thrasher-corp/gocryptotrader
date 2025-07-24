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
	errIDCannotBeMatched               = errors.New("cannot match ID")
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

// Levels defines a slice of orderbook levels
type Levels []Level

// comparison defines expected functionality to compare between two reference
// price levels
type comparison func(float64, float64) bool

// load iterates across new Levels and refreshes stored slice with this
// incoming snapshot.
func (l *Levels) load(incoming Levels) {
	if len(incoming) == 0 {
		*l = (*l)[:0] // Flush
		return
	}
	if len(incoming) <= len(*l) {
		copy(*l, incoming)        // Reuse
		*l = (*l)[:len(incoming)] // Flush excess
		return
	}
	*l = make([]Level, len(incoming)) // Extend
	copy(*l, incoming)                // Copy
}

// updateByID amends price by corresponding ID and returns an error if not found
func (l Levels) updateByID(updts []Level) error {
updates:
	for x := range updts {
		for y := range l {
			if updts[x].ID != l[y].ID { // Filter IDs that don't match
				continue
			}
			if updts[x].Price > 0 {
				// Only apply changes when zero values are not present, Bitmex
				// for example sends 0 price values.
				l[y].Price = updts[x].Price
				l[y].StrPrice = updts[x].StrPrice
			}
			l[y].Amount = updts[x].Amount
			l[y].StrAmount = updts[x].StrAmount
			continue updates
		}
		return fmt.Errorf("update error: %w; ID: %d not found", errIDCannotBeMatched, updts[x].ID)
	}
	return nil
}

// deleteByID deletes reference by ID
func (l *Levels) deleteByID(updts Levels, bypassErr bool) error {
updates:
	for x := range updts {
		for y := range *l {
			if updts[x].ID != (*l)[y].ID {
				continue
			}

			copy((*l)[y:], (*l)[y+1:])
			*l = (*l)[:len(*l)-1]

			continue updates
		}
		if !bypassErr {
			return fmt.Errorf("delete error: %w %d not found", errIDCannotBeMatched, updts[x].ID)
		}
	}
	return nil
}

// amount returns total depth liquidity and value
func (l Levels) amount() (liquidity, value float64) {
	for x := range l {
		liquidity += l[x].Amount
		value += l[x].Amount * l[x].Price
	}
	return
}

// retrieve returns a slice of contents from the stored Levels up to the
// count length. If count is zero or greater than the length of the stored
// Levels, the entire slice is returned.
func (l Levels) retrieve(count int) Levels {
	if count == 0 || count >= len(l) {
		count = len(l)
	}
	result := make(Levels, count)
	copy(result, l)
	return result
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates
func (l *Levels) updateInsertByPrice(updts Levels, maxChainLength int, compare func(float64, float64) bool) {
updates:
	for x := range updts {
		for y := range *l {
			switch {
			case (*l)[y].Price == updts[x].Price:
				if updts[x].Amount <= 0 {
					// Delete
					if y+1 == len(*l) {
						*l = (*l)[:y]
					} else {
						copy((*l)[y:], (*l)[y+1:])
						*l = (*l)[:len(*l)-1]
					}
				} else {
					// Update
					(*l)[y].Amount = updts[x].Amount
					(*l)[y].StrAmount = updts[x].StrAmount
				}
				continue updates
			case compare((*l)[y].Price, updts[x].Price):
				if updts[x].Amount > 0 {
					*l = append(*l, Level{})   // Extend
					copy((*l)[y+1:], (*l)[y:]) // Copy elements from index y onwards one position to the right
					(*l)[y] = updts[x]         // Insert updts[x] at index y
				}
				continue updates
			}
		}
		if updts[x].Amount > 0 {
			*l = append(*l, updts[x])
		}
	}
	// Reduces length of total stored slice length to a maxChainLength value
	if maxChainLength != 0 && len(*l) > maxChainLength {
		*l = (*l)[:maxChainLength]
	}
}

// updateInsertByID updates or inserts if not found for a bid or ask depth
func (l *Levels) updateInsertByID(updts Levels, compare comparison) error {
updates:
	for x := range updts {
		if updts[x].Amount <= 0 {
			return errAmountCannotBeLessOrEqualToZero
		}
		var popped bool
		for y := 0; y < len(*l); y++ {
			if (*l)[y].ID == updts[x].ID {
				if (*l)[y].Price != updts[x].Price { // Price level change
					if y+1 == len(*l) { // end of depth
						// no movement needed just a re-adjustment
						(*l)[y] = updts[x]
						continue updates
					}
					copy((*l)[y:], (*l)[y+1:]) // RM level and shift left
					*l = (*l)[:len(*l)-1]      // Unlink residual element from end of slice
					y--                        // adjust index
					popped = true
					continue // continue through node depth
				}
				// no price change, amend amount and continue update
				(*l)[y].Amount = updts[x].Amount
				(*l)[y].StrAmount = updts[x].StrAmount
				continue updates // continue to next update
			}

			if compare((*l)[y].Price, updts[x].Price) {
				*l = append(*l, Level{})   // Extend
				copy((*l)[y+1:], (*l)[y:]) // Copy elements from index y onwards one position to the right
				(*l)[y] = updts[x]         // Insert updts[x] at index y

				if popped { // already found ID and popped
					continue updates
				}

				// search for ID
				for z := y + 1; z < len(*l); z++ {
					if (*l)[z].ID == updts[x].ID {
						copy((*l)[z:], (*l)[z+1:]) // RM level and shift left
						*l = (*l)[:len(*l)-1]      // Unlink residual element from end of slice
						break
					}
				}
				continue updates
			}
		}
		*l = append(*l, updts[x])
	}
	return nil
}

// insertUpdates inserts new updates for bids or asks based on price level
func (l *Levels) insertUpdates(updts Levels, comp comparison) error {
updates:
	for x := range updts {
		if len(*l) == 0 {
			*l = append(*l, updts[x])
			continue
		}

		for y := range *l {
			switch {
			case (*l)[y].Price == updts[x].Price: // Price already found
				return fmt.Errorf("%w for price %f", errCollisionDetected, updts[x].Price)
			case comp((*l)[y].Price, updts[x].Price): // price at correct spot
				*l = append((*l)[:y], append([]Level{updts[x]}, (*l)[y:]...)...)
				continue updates
			}
		}
		*l = append(*l, updts[x])
	}
	return nil
}

// getHeadPriceNoLock gets best/head price
func (l Levels) getHeadPriceNoLock() (float64, error) {
	if len(l) == 0 {
		return 0, errNoLiquidity
	}
	return l[0].Price, nil
}

// getHeadVolumeNoLock gets best/head volume
func (l Levels) getHeadVolumeNoLock() (float64, error) {
	if len(l) == 0 {
		return 0, errNoLiquidity
	}
	return l[0].Amount, nil
}

// getMovementByQuotation traverses through orderbook liquidity using quotation
// currency as a limiter and returns orderbook movement details. Swap boolean
// allows the swap of sold and purchased to reduce code so it doesn't need to be
// specific to bid or ask.
func (l Levels) getMovementByQuotation(quote, refPrice float64, swap bool) (*Movement, error) {
	if quote <= 0 {
		return nil, errQuoteAmountInvalid
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	head, err := l.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}

	m := Movement{StartPrice: refPrice}
	for x := range l {
		levelValue := l[x].Amount * l[x].Price
		leftover := quote - levelValue
		if leftover < 0 {
			m.Purchased += quote
			m.Sold += quote / levelValue * l[x].Amount
			// This level is not consumed so the book shifts to this price.
			m.EndPrice = l[x].Price
			quote = 0
			break
		}
		// Full level consumed
		m.Purchased += l[x].Price * l[x].Amount
		m.Sold += l[x].Amount
		quote = leftover
		if leftover == 0 {
			// Price no longer exists on the book so use next full price level
			// to calculate book impact. If available.
			if x+1 < len(l) {
				m.EndPrice = l[x+1].Price
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
func (l Levels) getMovementByBase(base, refPrice float64, swap bool) (*Movement, error) {
	if base <= 0 {
		return nil, errBaseAmountInvalid
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	head, err := l.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}

	m := Movement{StartPrice: refPrice}
	for x := range l {
		leftover := base - l[x].Amount
		if leftover < 0 {
			m.Purchased += l[x].Price * base
			m.Sold += base
			// This level is not consumed so the book shifts to this price.
			m.EndPrice = l[x].Price
			base = 0
			break
		}
		// Full level consumed
		m.Purchased += l[x].Price * l[x].Amount
		m.Sold += l[x].Amount
		base = leftover
		if leftover == 0 {
			// Price no longer exists on the book so use next full price level
			// to calculate book impact.
			if x+1 < len(l) {
				m.EndPrice = l[x+1].Price
			} else {
				m.FullBookSideConsumed = true
			}
			break
		}
	}
	return m.finalizeFields(m.Purchased, m.Sold, head, base, swap)
}

// bidLevels bid depth specific functionality
type bidLevels struct{ Levels }

// bidCompare ensures price is in correct descending alignment (can inline)
func bidCompare(left, right float64) bool {
	return left < right
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates
func (bids *bidLevels) updateInsertByPrice(updts Levels, maxChainLength int) {
	bids.Levels.updateInsertByPrice(updts, maxChainLength, bidCompare)
}

// updateInsertByID updates or inserts if not found
func (bids *bidLevels) updateInsertByID(updts Levels) error {
	return bids.Levels.updateInsertByID(updts, bidCompare)
}

// insertUpdates inserts new updates for bids based on price level
func (bids *bidLevels) insertUpdates(updts Levels) error {
	return bids.Levels.insertUpdates(updts, bidCompare)
}

// hitBidsByNominalSlippage hits the bids by the required nominal slippage
// percentage, calculated from the reference price and returns orderbook
// movement details.
func (bids *bidLevels) hitBidsByNominalSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage < 0 {
		return nil, errInvalidNominalSlippage
	}

	if slippage > 100 {
		return nil, errInvalidSlippageCannotExceed100
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if len(bids.Levels) == 0 {
		return nil, errNoLiquidity
	}

	nominal := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	var cumulativeValue, cumulativeAmounts float64
	for x := range bids.Levels {
		totallevelValue := bids.Levels[x].Price * bids.Levels[x].Amount
		currentFullValue := totallevelValue + cumulativeValue
		currentTotalAmounts := cumulativeAmounts + bids.Levels[x].Amount

		nominal.AverageOrderCost = currentFullValue / currentTotalAmounts
		percent := math.PercentageChange(refPrice, nominal.AverageOrderCost)
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
			levelTargetPriceDiff := bids.Levels[x].Price - targetCost
			levelAmountExpectation := comparativeDiff / levelTargetPriceDiff
			nominal.NominalPercentage = slippage
			nominal.Sold = cumulativeAmounts + levelAmountExpectation
			nominal.Purchased += levelAmountExpectation * bids.Levels[x].Price
			nominal.AverageOrderCost = nominal.Purchased / nominal.Sold
			nominal.EndPrice = bids.Levels[x].Price
			return nominal, nil
		}

		nominal.EndPrice = bids.Levels[x].Price
		cumulativeValue = currentFullValue
		nominal.NominalPercentage = percent
		nominal.Sold += bids.Levels[x].Amount
		nominal.Purchased += totallevelValue
		cumulativeAmounts = currentTotalAmounts
		if slippage == percent {
			nominal.FullBookSideConsumed = x+1 >= len(bids.Levels)
			return nominal, nil
		}
	}
	nominal.FullBookSideConsumed = true
	return nominal, nil
}

// hitBidsByImpactSlippage hits the bids by the required impact slippage
// percentage, calculated from the reference price and returns orderbook
// movement details.
func (bids *bidLevels) hitBidsByImpactSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage <= 0 {
		return nil, errInvalidImpactSlippage
	}

	if slippage > 100 {
		return nil, errInvalidSlippageCannotExceed100
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if len(bids.Levels) == 0 {
		return nil, errNoLiquidity
	}

	impact := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	for x := range bids.Levels {
		percent := math.PercentageChange(refPrice, bids.Levels[x].Price)
		if percent != 0 {
			percent *= -1
		}
		impact.EndPrice = bids.Levels[x].Price
		impact.ImpactPercentage = percent
		if slippage <= percent {
			// Don't include this level amount as this consumes the level
			// book price, thus obtaining a higher percentage impact.
			return impact, nil
		}
		impact.Sold += bids.Levels[x].Amount
		impact.Purchased += bids.Levels[x].Amount * bids.Levels[x].Price
		impact.AverageOrderCost = impact.Purchased / impact.Sold
	}
	impact.FullBookSideConsumed = true
	impact.ImpactPercentage = FullLiquidityExhaustedPercentage
	return impact, nil
}

// askLevels ask depth specific functionality
type askLevels struct{ Levels }

// askCompare ensures price is in correct ascending alignment (can inline)
func askCompare(left, right float64) bool {
	return left > right
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates
func (ask *askLevels) updateInsertByPrice(updts Levels, maxChainLength int) {
	ask.Levels.updateInsertByPrice(updts, maxChainLength, askCompare)
}

// updateInsertByID updates or inserts if not found
func (ask *askLevels) updateInsertByID(updts Levels) error {
	return ask.Levels.updateInsertByID(updts, askCompare)
}

// insertUpdates inserts new updates for asks based on price level
func (ask *askLevels) insertUpdates(updts Levels) error {
	return ask.Levels.insertUpdates(updts, askCompare)
}

// liftAsksByNominalSlippage lifts the asks by the required nominal slippage
// percentage, calculated from the reference price and returns orderbook
// movement details.
func (ask *askLevels) liftAsksByNominalSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage < 0 {
		return nil, errInvalidNominalSlippage
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if len(ask.Levels) == 0 {
		return nil, errNoLiquidity
	}

	nominal := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	var cumulativeAmounts float64
	for x := range ask.Levels {
		totallevelValue := ask.Levels[x].Price * ask.Levels[x].Amount
		currentValue := totallevelValue + nominal.Sold
		currentAmounts := cumulativeAmounts + ask.Levels[x].Amount

		nominal.AverageOrderCost = currentValue / currentAmounts
		percent := math.PercentageChange(refPrice, nominal.AverageOrderCost)

		if slippage < percent {
			targetCost := (1 + slippage/100) * refPrice
			if targetCost == refPrice {
				nominal.AverageOrderCost = nominal.Sold / nominal.Purchased
				// Rounding issue on requested nominal percentage
				return nominal, nil
			}

			comparative := targetCost * cumulativeAmounts
			comparativeDiff := comparative - nominal.Sold
			levelTargetPriceDiff := ask.Levels[x].Price - targetCost
			levelAmountExpectation := comparativeDiff / levelTargetPriceDiff
			nominal.NominalPercentage = slippage
			nominal.Sold += levelAmountExpectation * ask.Levels[x].Price
			nominal.Purchased += levelAmountExpectation
			nominal.AverageOrderCost = nominal.Sold / nominal.Purchased
			nominal.EndPrice = ask.Levels[x].Price
			return nominal, nil
		}

		nominal.EndPrice = ask.Levels[x].Price
		nominal.Sold = currentValue
		nominal.Purchased += ask.Levels[x].Amount
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
func (ask *askLevels) liftAsksByImpactSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage <= 0 {
		return nil, errInvalidImpactSlippage
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if len(ask.Levels) == 0 {
		return nil, errNoLiquidity
	}

	impact := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	for x := range ask.Levels {
		percent := math.PercentageChange(refPrice, ask.Levels[x].Price)
		impact.ImpactPercentage = percent
		impact.EndPrice = ask.Levels[x].Price
		if slippage <= percent {
			// Don't include this level amount as this consumes the level
			// book price, thus obtaining a higher percentage impact.
			return impact, nil
		}
		impact.Sold += ask.Levels[x].Amount * ask.Levels[x].Price
		impact.Purchased += ask.Levels[x].Amount
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
	m.NominalPercentage = math.PercentageChange(m.StartPrice, m.AverageOrderCost)
	if m.NominalPercentage < 0 {
		m.NominalPercentage *= -1
	}

	if !m.FullBookSideConsumed && leftover == 0 {
		// Impact percentage is how much the orderbook slips from the reference
		// price to the remaining level price.

		m.ImpactPercentage = math.PercentageChange(m.StartPrice, m.EndPrice)
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
