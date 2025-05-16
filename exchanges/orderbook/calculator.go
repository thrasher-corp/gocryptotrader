package orderbook

import (
	"errors"
	"fmt"

	math "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

const fullLiquidityUsageWarning = "[WARNING]: Full liquidity exhausted."

var (
	errPriceTargetInvalid = errors.New("price target is invalid")
	errCannotShiftPrice   = errors.New("cannot shift price")
)

// WhaleBombResult returns the whale bomb result
type WhaleBombResult struct {
	Amount               float64
	MinimumPrice         float64
	MaximumPrice         float64
	PercentageGainOrLoss float64
	Orders               Tranches
	Status               string
}

// WhaleBomb finds the amount required to target a price
func (s *Snapshot) WhaleBomb(priceTarget float64, buy bool) (*WhaleBombResult, error) {
	if priceTarget < 0 {
		return nil, errPriceTargetInvalid
	}
	action, err := s.findAmount(priceTarget, buy)
	if err != nil {
		return nil, err
	}

	var warning string
	if action.FullLiquidityUsed {
		warning = fullLiquidityUsageWarning
	}

	var status string
	var percent, minPrice, maxPrice, amount float64
	if buy {
		minPrice = action.ReferencePrice
		maxPrice = action.TranchePositionPrice
		amount = action.QuoteAmount
		percent = math.PercentageChange(action.ReferencePrice, action.TranchePositionPrice)
		status = fmt.Sprintf("Buying using %.2f %s worth of %s will send the price from %v to %v [%.2f%%] and impact %d price tranche(s). %s",
			amount, s.Pair.Quote, s.Pair.Base, minPrice, maxPrice,
			percent, len(action.Tranches), warning)
	} else {
		minPrice = action.TranchePositionPrice
		maxPrice = action.ReferencePrice
		amount = action.BaseAmount
		percent = math.PercentageChange(action.ReferencePrice, action.TranchePositionPrice)
		status = fmt.Sprintf("Selling using %.2f %s worth of %s will send the price from %v to %v [%.2f%%] and impact %d price tranche(s). %s",
			amount, s.Pair.Base, s.Pair.Quote, maxPrice, minPrice,
			percent, len(action.Tranches), warning)
	}

	return &WhaleBombResult{
		Amount:               amount,
		Orders:               action.Tranches,
		MinimumPrice:         minPrice,
		MaximumPrice:         maxPrice,
		Status:               status,
		PercentageGainOrLoss: percent,
	}, err
}

// SimulateOrder simulates an order
func (s *Snapshot) SimulateOrder(amount float64, buy bool) (*WhaleBombResult, error) {
	var direction string
	var action *DeploymentAction
	var soldAmount, boughtAmount, minimumPrice, maximumPrice float64
	var sold, bought currency.Code
	var err error
	if buy {
		direction = "Buying"
		action, err = s.buy(amount)
		if err != nil {
			return nil, err
		}
		soldAmount = action.QuoteAmount
		boughtAmount = action.BaseAmount
		maximumPrice = action.TranchePositionPrice
		minimumPrice = action.ReferencePrice
		sold = s.Pair.Quote
		bought = s.Pair.Base
	} else {
		direction = "Selling"
		action, err = s.sell(amount)
		if err != nil {
			return nil, err
		}
		soldAmount = action.BaseAmount
		boughtAmount = action.QuoteAmount
		minimumPrice = action.TranchePositionPrice
		maximumPrice = action.ReferencePrice
		sold = s.Pair.Base
		bought = s.Pair.Quote
	}

	var warning string
	if action.FullLiquidityUsed {
		warning = fullLiquidityUsageWarning
	}

	pct := math.PercentageChange(action.ReferencePrice, action.TranchePositionPrice)
	status := fmt.Sprintf("%s using %f %v worth of %v will send the price from %v to %v [%.2f%%] and impact %v price tranche(s). %s",
		direction, soldAmount, sold, bought, action.ReferencePrice,
		action.TranchePositionPrice, pct, len(action.Tranches), warning)
	return &WhaleBombResult{
		Orders:               action.Tranches,
		Amount:               boughtAmount,
		MinimumPrice:         minimumPrice,
		MaximumPrice:         maximumPrice,
		PercentageGainOrLoss: pct,
		Status:               status,
	}, nil
}

func (s *Snapshot) findAmount(priceTarget float64, buy bool) (*DeploymentAction, error) {
	action := DeploymentAction{}
	if buy {
		if len(s.Asks) == 0 {
			return nil, errNoLiquidity
		}
		action.ReferencePrice = s.Asks[0].Price
		if action.ReferencePrice > priceTarget {
			return nil, fmt.Errorf("%w to %f as it's below ascending ask prices starting at %f",
				errCannotShiftPrice, priceTarget, action.ReferencePrice)
		}
		for x := range s.Asks {
			if s.Asks[x].Price >= priceTarget {
				action.TranchePositionPrice = s.Asks[x].Price
				return &action, nil
			}
			action.Tranches = append(action.Tranches, s.Asks[x])
			action.QuoteAmount += s.Asks[x].Price * s.Asks[x].Amount
			action.BaseAmount += s.Asks[x].Amount
		}
		action.TranchePositionPrice = s.Asks[len(s.Asks)-1].Price
		action.FullLiquidityUsed = true
		return &action, nil
	}

	if len(s.Bids) == 0 {
		return nil, errNoLiquidity
	}
	action.ReferencePrice = s.Bids[0].Price
	if action.ReferencePrice < priceTarget {
		return nil, fmt.Errorf("%w to %f as it's above descending bid prices starting at %f",
			errCannotShiftPrice, priceTarget, action.ReferencePrice)
	}
	for x := range s.Bids {
		if s.Bids[x].Price <= priceTarget {
			action.TranchePositionPrice = s.Bids[x].Price
			return &action, nil
		}
		action.Tranches = append(action.Tranches, s.Bids[x])
		action.QuoteAmount += s.Bids[x].Price * s.Bids[x].Amount
		action.BaseAmount += s.Bids[x].Amount
	}
	action.TranchePositionPrice = s.Bids[len(s.Bids)-1].Price
	action.FullLiquidityUsed = true
	return &action, nil
}

// DeploymentAction defines deployment information on a liquidity side.
type DeploymentAction struct {
	ReferencePrice       float64
	TranchePositionPrice float64
	BaseAmount           float64
	QuoteAmount          float64
	Tranches             Tranches
	FullLiquidityUsed    bool
}

func (s *Snapshot) buy(quote float64) (*DeploymentAction, error) {
	if quote <= 0 {
		return nil, errQuoteAmountInvalid
	}
	if len(s.Asks) == 0 {
		return nil, errNoLiquidity
	}
	action := &DeploymentAction{ReferencePrice: s.Asks[0].Price}
	for x := range s.Asks {
		action.TranchePositionPrice = s.Asks[x].Price
		trancheValue := s.Asks[x].Price * s.Asks[x].Amount
		action.QuoteAmount += trancheValue
		remaining := quote - trancheValue
		if remaining <= 0 {
			if remaining == 0 {
				if len(s.Asks)-1 > x {
					action.TranchePositionPrice = s.Asks[x+1].Price
				} else {
					action.FullLiquidityUsed = true
				}
			}
			subAmount := quote / s.Asks[x].Price
			action.Tranches = append(action.Tranches, Tranche{
				Price:  s.Asks[x].Price,
				Amount: subAmount,
			})
			action.BaseAmount += subAmount
			return action, nil
		}
		if len(s.Asks)-1 <= x {
			action.FullLiquidityUsed = true
		}
		quote = remaining
		action.BaseAmount += s.Asks[x].Amount
		action.Tranches = append(action.Tranches, s.Asks[x])
	}

	return action, nil
}

func (s *Snapshot) sell(base float64) (*DeploymentAction, error) {
	if base <= 0 {
		return nil, errSnapshotAmountInvalid
	}
	if len(s.Bids) == 0 {
		return nil, errNoLiquidity
	}
	action := &DeploymentAction{ReferencePrice: s.Bids[0].Price}
	for x := range s.Bids {
		action.TranchePositionPrice = s.Bids[x].Price
		remaining := base - s.Bids[x].Amount
		if remaining <= 0 {
			if remaining == 0 {
				if len(s.Bids)-1 > x {
					action.TranchePositionPrice = s.Bids[x+1].Price
				} else {
					action.FullLiquidityUsed = true
				}
			}
			action.Tranches = append(action.Tranches, Tranche{
				Price:  s.Bids[x].Price,
				Amount: base,
			})
			action.BaseAmount += base
			action.QuoteAmount += base * s.Bids[x].Price
			return action, nil
		}
		if len(s.Bids)-1 <= x {
			action.FullLiquidityUsed = true
		}
		base = remaining
		action.BaseAmount += s.Bids[x].Amount
		action.QuoteAmount += s.Bids[x].Amount * s.Bids[x].Price
		action.Tranches = append(action.Tranches, s.Bids[x])
	}
	return action, nil
}

// GetAveragePrice finds the average buy or sell price of a specified amount.
// It finds the nominal amount spent on the total purchase or sell and uses it
// to find the average price for an individual unit bought or sold
func (s *Snapshot) GetAveragePrice(buy bool, amount float64) (float64, error) {
	if amount <= 0 {
		return 0, errAmountInvalid
	}
	var aggNominalAmount, remainingAmount float64
	if buy {
		aggNominalAmount, remainingAmount = s.Asks.FindNominalAmount(amount)
	} else {
		aggNominalAmount, remainingAmount = s.Bids.FindNominalAmount(amount)
	}
	if remainingAmount != 0 {
		return 0, fmt.Errorf("%w for %v on exchange %v to support a buy amount of %v", errNotEnoughLiquidity, s.Pair, s.Exchange, amount)
	}
	return aggNominalAmount / amount, nil
}

// FindNominalAmount finds the nominal amount spent in terms of the quote
// If the orderbook doesn't have enough liquidity it returns a non zero
// remaining amount value
func (ts Tranches) FindNominalAmount(amount float64) (aggNominalAmount, remainingAmount float64) {
	remainingAmount = amount
	for x := range ts {
		if remainingAmount <= ts[x].Amount {
			aggNominalAmount += ts[x].Price * remainingAmount
			remainingAmount = 0
			break
		}
		aggNominalAmount += ts[x].Price * ts[x].Amount
		remainingAmount -= ts[x].Amount
	}
	return aggNominalAmount, remainingAmount
}
