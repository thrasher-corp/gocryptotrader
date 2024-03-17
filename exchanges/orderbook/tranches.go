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

// Tranches defines a slice of orderbook Tranche
type Tranches []Tranche

// comparison defines expected functionality to compare between two reference
// price levels
type comparison func(float64, float64) bool

// load iterates across new tranches and refreshes stored slice with this
// incoming snapshot.
func (ts *Tranches) load(incoming Tranches) {
	if len(incoming) == 0 {
		*ts = (*ts)[:0] // Flush
		return
	}
	if len(incoming) <= len(*ts) {
		copy(*ts, incoming)         // Reuse
		*ts = (*ts)[:len(incoming)] // Flush excess
		return
	}
	*ts = make([]Tranche, len(incoming)) // Extend
	copy(*ts, incoming)                  // Copy
}

// updateByID amends price by corresponding ID and returns an error if not found
func (ts Tranches) updateByID(updts []Tranche) error {
updates:
	for x := range updts {
		for y := range ts {
			if updts[x].ID != ts[y].ID { // Filter IDs that don't match
				continue
			}
			if updts[x].Price > 0 {
				// Only apply changes when zero values are not present, Bitmex
				// for example sends 0 price values.
				ts[y].Price = updts[x].Price
				ts[y].StrPrice = updts[x].StrPrice
			}
			ts[y].Amount = updts[x].Amount
			ts[y].StrAmount = updts[x].StrAmount
			continue updates
		}
		return fmt.Errorf("update error: %w ID: %d not found",
			errIDCannotBeMatched,
			updts[x].ID)
	}
	return nil
}

// deleteByID deletes reference by ID
func (ts *Tranches) deleteByID(updts Tranches, bypassErr bool) error {
updates:
	for x := range updts {
		for y := range *ts {
			if updts[x].ID != (*ts)[y].ID {
				continue
			}

			if y < len(*ts) {
				copy((*ts)[y:], (*ts)[y+1:])
				*ts = (*ts)[:len(*ts)-1]
			} else {
				*ts = append((*ts)[:y], (*ts)[y+1:]...)
			}
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
func (ts Tranches) amount() (liquidity, value float64) {
	for x := range ts {
		liquidity += ts[x].Amount
		value += ts[x].Amount * ts[x].Price
	}
	return
}

// retrieve returns a slice of contents from the stored Tranches up to the
// count length. If count is zero or greater than the length of the stored
// Tranches, the entire slice is returned.
func (ts Tranches) retrieve(count int) Tranches {
	if count == 0 || count >= len(ts) {
		count = len(ts)
	}
	return append(Tranches{}, ts[:count]...)
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates
func (ts *Tranches) updateInsertByPrice(updts Tranches, maxChainLength int, compare func(float64, float64) bool) {
updates:
	for x := range updts {
		for y := range *ts {
			switch {
			case (*ts)[y].Price == updts[x].Price:
				if updts[x].Amount <= 0 {
					// Delete
					if y+1 == len(*ts) {
						*ts = (*ts)[:y]
					} else {
						copy((*ts)[y:], (*ts)[y+1:])
						*ts = (*ts)[:len(*ts)-1]
					}
				} else {
					// Update
					(*ts)[y].Amount = updts[x].Amount
					(*ts)[y].StrAmount = updts[x].StrAmount
				}
				continue updates
			case compare((*ts)[y].Price, updts[x].Price):
				if updts[x].Amount > 0 {
					*ts = append(*ts, Tranche{}) // Extend
					copy((*ts)[y+1:], (*ts)[y:]) // Copy elements from index y onwards one position to the right
					(*ts)[y] = updts[x]          // Insert updts[x] at index y
				}
				continue updates
			}
		}
		if updts[x].Amount > 0 {
			*ts = append(*ts, updts[x])
		}
	}
	// Reduces length of total stored slice length to a maxChainLength value
	if maxChainLength != 0 && len(*ts) > maxChainLength {
		*ts = (*ts)[:maxChainLength]
	}
}

// updateInsertByID updates or inserts if not found for a bid or ask depth
func (ts *Tranches) updateInsertByID(updts Tranches, compare comparison) error {
updates:
	for x := range updts {
		if updts[x].Amount <= 0 {
			return errAmountCannotBeLessOrEqualToZero
		}
		var popped bool
		for y := 0; y < len(*ts); y++ {
			if (*ts)[y].ID == updts[x].ID {
				if (*ts)[y].Price != updts[x].Price { // Price level change
					if y+1 == len(*ts) { // end of depth
						// no movement needed just a re-adjustment
						(*ts)[y] = updts[x]
						continue updates
					}
					copy((*ts)[y:], (*ts)[y+1:]) // RM tranche and shift left
					*ts = (*ts)[:len(*ts)-1]     // Unlink residual element from end of slice
					y--                          // adjust index
					popped = true
					continue // continue through node depth
				}
				// no price change, amend amount and continue update
				(*ts)[y].Amount = updts[x].Amount
				(*ts)[y].StrAmount = updts[x].StrAmount
				continue updates // continue to next update
			}

			if compare((*ts)[y].Price, updts[x].Price) {
				*ts = append(*ts, Tranche{}) // Extend
				copy((*ts)[y+1:], (*ts)[y:]) // Copy elements from index y onwards one position to the right
				(*ts)[y] = updts[x]          // Insert updts[x] at index y

				if popped { // already found ID and popped
					continue updates
				}

				// search for ID
				for z := y + 1; z < len(*ts); z++ {
					if (*ts)[z].ID == updts[x].ID {
						copy((*ts)[z:], (*ts)[z+1:]) // RM tranche and shift left
						*ts = (*ts)[:len(*ts)-1]     // Unlink residual element from end of slice
						break
					}
				}
				continue updates
			}
		}
		*ts = append(*ts, updts[x])
	}
	return nil
}

// insertUpdates inserts new updates for bids or asks based on price level
func (ts *Tranches) insertUpdates(updts Tranches, comp comparison) error {
updates:
	for x := range updts {
		if len(*ts) == 0 {
			*ts = append(*ts, updts[x])
			continue
		}

		for y := range *ts {
			switch {
			case (*ts)[y].Price == updts[x].Price: // Price already found
				return fmt.Errorf("%w for price %f", errCollisionDetected, updts[x].Price)
			case comp((*ts)[y].Price, updts[x].Price): // price at correct spot
				*ts = append((*ts)[:y], append([]Tranche{updts[x]}, (*ts)[y:]...)...)
				continue updates
			}
		}
		*ts = append(*ts, updts[x])
	}
	return nil
}

// getHeadPriceNoLock gets best/head price
func (ts Tranches) getHeadPriceNoLock() (float64, error) {
	if len(ts) == 0 {
		return 0, errNoLiquidity
	}
	return ts[0].Price, nil
}

// getHeadVolumeNoLock gets best/head volume
func (ts Tranches) getHeadVolumeNoLock() (float64, error) {
	if len(ts) == 0 {
		return 0, errNoLiquidity
	}
	return ts[0].Amount, nil
}

// getMovementByQuotation traverses through orderbook liquidity using quotation
// currency as a limiter and returns orderbook movement details. Swap boolean
// allows the swap of sold and purchased to reduce code so it doesn't need to be
// specific to bid or ask.
func (ts Tranches) getMovementByQuotation(quote, refPrice float64, swap bool) (*Movement, error) {
	if quote <= 0 {
		return nil, errQuoteAmountInvalid
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	head, err := ts.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}

	m := Movement{StartPrice: refPrice}
	for x := range ts {
		trancheValue := ts[x].Amount * ts[x].Price
		leftover := quote - trancheValue
		if leftover < 0 {
			m.Purchased += quote
			m.Sold += quote / trancheValue * ts[x].Amount
			// This tranche is not consumed so the book shifts to this price.
			m.EndPrice = ts[x].Price
			quote = 0
			break
		}
		// Full tranche consumed
		m.Purchased += ts[x].Price * ts[x].Amount
		m.Sold += ts[x].Amount
		quote = leftover
		if leftover == 0 {
			// Price no longer exists on the book so use next full price tranche
			// to calculate book impact. If available.
			if x+1 < len(ts) {
				m.EndPrice = ts[x+1].Price
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
func (ts Tranches) getMovementByBase(base, refPrice float64, swap bool) (*Movement, error) {
	if base <= 0 {
		return nil, errBaseAmountInvalid
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	head, err := ts.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}

	m := Movement{StartPrice: refPrice}
	for x := range ts {
		leftover := base - ts[x].Amount
		if leftover < 0 {
			m.Purchased += ts[x].Price * base
			m.Sold += base
			// This tranche is not consumed so the book shifts to this price.
			m.EndPrice = ts[x].Price
			base = 0
			break
		}
		// Full tranche consumed
		m.Purchased += ts[x].Price * ts[x].Amount
		m.Sold += ts[x].Amount
		base = leftover
		if leftover == 0 {
			// Price no longer exists on the book so use next full price tranche
			// to calculate book impact.
			if x+1 < len(ts) {
				m.EndPrice = ts[x+1].Price
			} else {
				m.FullBookSideConsumed = true
			}
			break
		}
	}
	return m.finalizeFields(m.Purchased, m.Sold, head, base, swap)
}

// bidTranches bid depth specific functionality
type bidTranches struct{ Tranches }

// bidCompare ensures price is in correct descending alignment (can inline)
func bidCompare(left, right float64) bool {
	return left < right
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates
func (bids *bidTranches) updateInsertByPrice(updts Tranches, maxChainLength int) {
	bids.Tranches.updateInsertByPrice(updts, maxChainLength, bidCompare)
}

// updateInsertByID updates or inserts if not found
func (bids *bidTranches) updateInsertByID(updts Tranches) error {
	return bids.Tranches.updateInsertByID(updts, bidCompare)
}

// insertUpdates inserts new updates for bids based on price level
func (bids *bidTranches) insertUpdates(updts Tranches) error {
	return bids.Tranches.insertUpdates(updts, bidCompare)
}

// hitBidsByNominalSlippage hits the bids by the required nominal slippage
// percentage, calculated from the reference price and returns orderbook
// movement details.
func (bids *bidTranches) hitBidsByNominalSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage < 0 {
		return nil, errInvalidNominalSlippage
	}

	if slippage > 100 {
		return nil, errInvalidSlippageCannotExceed100
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if len(bids.Tranches) == 0 {
		return nil, errNoLiquidity
	}

	nominal := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	var cumulativeValue, cumulativeAmounts float64
	for x := range bids.Tranches {
		totalTrancheValue := bids.Tranches[x].Price * bids.Tranches[x].Amount
		currentFullValue := totalTrancheValue + cumulativeValue
		currentTotalAmounts := cumulativeAmounts + bids.Tranches[x].Amount

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
			trancheTargetPriceDiff := bids.Tranches[x].Price - targetCost
			trancheAmountExpectation := comparativeDiff / trancheTargetPriceDiff
			nominal.NominalPercentage = slippage
			nominal.Sold = cumulativeAmounts + trancheAmountExpectation
			nominal.Purchased += trancheAmountExpectation * bids.Tranches[x].Price
			nominal.AverageOrderCost = nominal.Purchased / nominal.Sold
			nominal.EndPrice = bids.Tranches[x].Price
			return nominal, nil
		}

		nominal.EndPrice = bids.Tranches[x].Price
		cumulativeValue = currentFullValue
		nominal.NominalPercentage = percent
		nominal.Sold += bids.Tranches[x].Amount
		nominal.Purchased += totalTrancheValue
		cumulativeAmounts = currentTotalAmounts
		if slippage == percent {
			nominal.FullBookSideConsumed = x+1 >= len(bids.Tranches)
			return nominal, nil
		}
	}
	nominal.FullBookSideConsumed = true
	return nominal, nil
}

// hitBidsByImpactSlippage hits the bids by the required impact slippage
// percentage, calculated from the reference price and returns orderbook
// movement details.
func (bids *bidTranches) hitBidsByImpactSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage <= 0 {
		return nil, errInvalidImpactSlippage
	}

	if slippage > 100 {
		return nil, errInvalidSlippageCannotExceed100
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if len(bids.Tranches) == 0 {
		return nil, errNoLiquidity
	}

	impact := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	for x := range bids.Tranches {
		percent := math.CalculatePercentageGainOrLoss(bids.Tranches[x].Price, refPrice)
		if percent != 0 {
			percent *= -1
		}
		impact.EndPrice = bids.Tranches[x].Price
		impact.ImpactPercentage = percent
		if slippage <= percent {
			// Don't include this tranche amount as this consumes the tranche
			// book price, thus obtaining a higher percentage impact.
			return impact, nil
		}
		impact.Sold += bids.Tranches[x].Amount
		impact.Purchased += bids.Tranches[x].Amount * bids.Tranches[x].Price
		impact.AverageOrderCost = impact.Purchased / impact.Sold
	}
	impact.FullBookSideConsumed = true
	impact.ImpactPercentage = FullLiquidityExhaustedPercentage
	return impact, nil
}

// askTranches ask depth specific functionality
type askTranches struct{ Tranches }

// askCompare ensures price is in correct ascending alignment (can inline)
func askCompare(left, right float64) bool {
	return left > right
}

// updateInsertByPrice amends, inserts, moves and cleaves length of depth by
// updates
func (ask *askTranches) updateInsertByPrice(updts Tranches, maxChainLength int) {
	ask.Tranches.updateInsertByPrice(updts, maxChainLength, askCompare)
}

// updateInsertByID updates or inserts if not found
func (ask *askTranches) updateInsertByID(updts Tranches) error {
	return ask.Tranches.updateInsertByID(updts, askCompare)
}

// insertUpdates inserts new updates for asks based on price level
func (ask *askTranches) insertUpdates(updts Tranches) error {
	return ask.Tranches.insertUpdates(updts, askCompare)
}

// liftAsksByNominalSlippage lifts the asks by the required nominal slippage
// percentage, calculated from the reference price and returns orderbook
// movement details.
func (ask *askTranches) liftAsksByNominalSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage < 0 {
		return nil, errInvalidNominalSlippage
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if len(ask.Tranches) == 0 {
		return nil, errNoLiquidity
	}

	nominal := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	var cumulativeAmounts float64
	for x := range ask.Tranches {
		totalTrancheValue := ask.Tranches[x].Price * ask.Tranches[x].Amount
		currentValue := totalTrancheValue + nominal.Sold
		currentAmounts := cumulativeAmounts + ask.Tranches[x].Amount

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
			trancheTargetPriceDiff := ask.Tranches[x].Price - targetCost
			trancheAmountExpectation := comparativeDiff / trancheTargetPriceDiff
			nominal.NominalPercentage = slippage
			nominal.Sold += trancheAmountExpectation * ask.Tranches[x].Price
			nominal.Purchased += trancheAmountExpectation
			nominal.AverageOrderCost = nominal.Sold / nominal.Purchased
			nominal.EndPrice = ask.Tranches[x].Price
			return nominal, nil
		}

		nominal.EndPrice = ask.Tranches[x].Price
		nominal.Sold = currentValue
		nominal.Purchased += ask.Tranches[x].Amount
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
func (ask *askTranches) liftAsksByImpactSlippage(slippage, refPrice float64) (*Movement, error) {
	if slippage <= 0 {
		return nil, errInvalidImpactSlippage
	}

	if refPrice <= 0 {
		return nil, errInvalidReferencePrice
	}

	if len(ask.Tranches) == 0 {
		return nil, errNoLiquidity
	}

	impact := &Movement{StartPrice: refPrice, EndPrice: refPrice}
	for x := range ask.Tranches {
		percent := math.CalculatePercentageGainOrLoss(ask.Tranches[x].Price, refPrice)
		impact.ImpactPercentage = percent
		impact.EndPrice = ask.Tranches[x].Price
		if slippage <= percent {
			// Don't include this tranche amount as this consumes the tranche
			// book price, thus obtaining a higher percentage impact.
			return impact, nil
		}
		impact.Sold += ask.Tranches[x].Amount * ask.Tranches[x].Price
		impact.Purchased += ask.Tranches[x].Amount
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
