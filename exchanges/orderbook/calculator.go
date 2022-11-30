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
	Orders               Items
	Status               string
}

// WhaleBomb finds the amount required to target a price
func (b *Base) WhaleBomb(priceTarget float64, buy bool) (*WhaleBombResult, error) {
	if priceTarget < 0 {
		return nil, errPriceTargetInvalid
	}
	action, err := b.findAmount(priceTarget, buy)
	if err != nil {
		return nil, err
	}

	var warning string
	if action.FullLiquidityUsed {
		warning = fullLiquidityUsageWarning
	}

	var status string
	var percent, min, max, amount float64
	if buy {
		min = action.ReferencePrice
		max = action.TranchePositionPrice
		amount = action.QuoteAmount
		percent = math.CalculatePercentageGainOrLoss(action.TranchePositionPrice, action.ReferencePrice)
		status = fmt.Sprintf("Buying using %.2f %s worth of %s will send the price from %v to %v [%.2f%%] and impact %d price tranche(s). %s",
			amount, b.Pair.Quote, b.Pair.Base, min, max,
			percent, len(action.Tranches), warning)
	} else {
		min = action.TranchePositionPrice
		max = action.ReferencePrice
		amount = action.BaseAmount
		percent = math.CalculatePercentageGainOrLoss(action.TranchePositionPrice, action.ReferencePrice)
		status = fmt.Sprintf("Selling using %.2f %s worth of %s will send the price from %v to %v [%.2f%%] and impact %d price tranche(s). %s",
			amount, b.Pair.Base, b.Pair.Quote, max, min,
			percent, len(action.Tranches), warning)
	}

	return &WhaleBombResult{
		Amount:               amount,
		Orders:               action.Tranches,
		MinimumPrice:         min,
		MaximumPrice:         max,
		Status:               status,
		PercentageGainOrLoss: percent,
	}, err
}

// SimulateOrder simulates an order
func (b *Base) SimulateOrder(amount float64, buy bool) (*WhaleBombResult, error) {
	var direction string
	var action *DeploymentAction
	var soldAmount, boughtAmount, minimumPrice, maximumPrice float64
	var sold, bought currency.Code
	var err error
	if buy {
		direction = "Buying"
		action, err = b.buy(amount)
		if err != nil {
			return nil, err
		}
		soldAmount = action.QuoteAmount
		boughtAmount = action.BaseAmount
		maximumPrice = action.TranchePositionPrice
		minimumPrice = action.ReferencePrice
		sold = b.Pair.Quote
		bought = b.Pair.Base
	} else {
		direction = "Selling"
		action, err = b.sell(amount)
		if err != nil {
			return nil, err
		}
		soldAmount = action.BaseAmount
		boughtAmount = action.QuoteAmount
		minimumPrice = action.TranchePositionPrice
		maximumPrice = action.ReferencePrice
		sold = b.Pair.Base
		bought = b.Pair.Quote
	}

	var warning string
	if action.FullLiquidityUsed {
		warning = fullLiquidityUsageWarning
	}

	pct := math.CalculatePercentageGainOrLoss(action.TranchePositionPrice, action.ReferencePrice)
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

func (b *Base) findAmount(priceTarget float64, buy bool) (*DeploymentAction, error) {
	action := DeploymentAction{}
	if buy {
		if len(b.Asks) == 0 {
			return nil, errNoLiquidity
		}
		action.ReferencePrice = b.Asks[0].Price
		if action.ReferencePrice > priceTarget {
			return nil, fmt.Errorf("%w to %f as it's below ascending ask prices starting at %f",
				errCannotShiftPrice, priceTarget, action.ReferencePrice)
		}
		for x := range b.Asks {
			if b.Asks[x].Price >= priceTarget {
				action.TranchePositionPrice = b.Asks[x].Price
				return &action, nil
			}
			action.Tranches = append(action.Tranches, b.Asks[x])
			action.QuoteAmount += b.Asks[x].Price * b.Asks[x].Amount
			action.BaseAmount += b.Asks[x].Amount
		}
		action.TranchePositionPrice = b.Asks[len(b.Asks)-1].Price
		action.FullLiquidityUsed = true
		return &action, nil
	}

	if len(b.Bids) == 0 {
		return nil, errNoLiquidity
	}
	action.ReferencePrice = b.Bids[0].Price
	if action.ReferencePrice < priceTarget {
		return nil, fmt.Errorf("%w to %f as it's above descending bid prices starting at %f",
			errCannotShiftPrice, priceTarget, action.ReferencePrice)
	}
	for x := range b.Bids {
		if b.Bids[x].Price <= priceTarget {
			action.TranchePositionPrice = b.Bids[x].Price
			return &action, nil
		}
		action.Tranches = append(action.Tranches, b.Bids[x])
		action.QuoteAmount += b.Bids[x].Price * b.Bids[x].Amount
		action.BaseAmount += b.Bids[x].Amount
	}
	action.TranchePositionPrice = b.Bids[len(b.Bids)-1].Price
	action.FullLiquidityUsed = true
	return &action, nil
}

// DeploymentAction defines deployment information on a liquidity side.
type DeploymentAction struct {
	ReferencePrice       float64
	TranchePositionPrice float64
	BaseAmount           float64
	QuoteAmount          float64
	Tranches             Items
	FullLiquidityUsed    bool
}

func (b *Base) buy(quote float64) (*DeploymentAction, error) {
	if quote <= 0 {
		return nil, errQuoteAmountInvalid
	}
	if len(b.Asks) == 0 {
		return nil, errNoLiquidity
	}
	action := &DeploymentAction{ReferencePrice: b.Asks[0].Price}
	for x := range b.Asks {
		action.TranchePositionPrice = b.Asks[x].Price
		trancheValue := b.Asks[x].Price * b.Asks[x].Amount
		action.QuoteAmount += trancheValue
		remaining := quote - trancheValue
		if remaining <= 0 {
			if remaining == 0 {
				if len(b.Asks)-1 > x {
					action.TranchePositionPrice = b.Asks[x+1].Price
				} else {
					action.FullLiquidityUsed = true
				}
			}
			subAmount := quote / b.Asks[x].Price
			action.Tranches = append(action.Tranches, Item{
				Price:  b.Asks[x].Price,
				Amount: subAmount,
			})
			action.BaseAmount += subAmount
			return action, nil
		}
		if len(b.Asks)-1 <= x {
			action.FullLiquidityUsed = true
		}
		quote = remaining
		action.BaseAmount += b.Asks[x].Amount
		action.Tranches = append(action.Tranches, b.Asks[x])
	}

	return action, nil
}

func (b *Base) sell(base float64) (*DeploymentAction, error) {
	if base <= 0 {
		return nil, errBaseAmountInvalid
	}
	if len(b.Bids) == 0 {
		return nil, errNoLiquidity
	}
	action := &DeploymentAction{ReferencePrice: b.Bids[0].Price}
	for x := range b.Bids {
		action.TranchePositionPrice = b.Bids[x].Price
		remaining := base - b.Bids[x].Amount
		if remaining <= 0 {
			if remaining == 0 {
				if len(b.Bids)-1 > x {
					action.TranchePositionPrice = b.Bids[x+1].Price
				} else {
					action.FullLiquidityUsed = true
				}
			}
			action.Tranches = append(action.Tranches, Item{
				Price:  b.Bids[x].Price,
				Amount: base,
			})
			action.BaseAmount += base
			action.QuoteAmount += base * b.Bids[x].Price
			return action, nil
		}
		if len(b.Bids)-1 <= x {
			action.FullLiquidityUsed = true
		}
		base = remaining
		action.BaseAmount += b.Bids[x].Amount
		action.QuoteAmount += b.Bids[x].Amount * b.Bids[x].Price
		action.Tranches = append(action.Tranches, b.Bids[x])
	}
	return action, nil
}

// GetAveragePrice finds the average buy or sell price of a specified amount.
// It finds the nominal amount spent on the total purchase or sell and uses it
// to find the average price for an individual unit bought or sold
func (b *Base) GetAveragePrice(buy bool, amount float64) (float64, error) {
	if amount <= 0 {
		return 0, errAmountInvalid
	}
	var aggNominalAmount, remainingAmount float64
	if buy {
		aggNominalAmount, remainingAmount = b.Asks.FindNominalAmount(amount)
	} else {
		aggNominalAmount, remainingAmount = b.Bids.FindNominalAmount(amount)
	}
	if remainingAmount != 0 {
		return 0, fmt.Errorf("%w for %v on exchange %v to support a buy amount of %v", errNotEnoughLiquidity, b.Pair, b.Exchange, amount)
	}
	return aggNominalAmount / amount, nil
}

// FindNominalAmount finds the nominal amount spent in terms of the quote
// If the orderbook doesn't have enough liquidity it returns a non zero
// remaining amount value
func (elem Items) FindNominalAmount(amount float64) (aggNominalAmount, remainingAmount float64) {
	remainingAmount = amount
	for x := range elem {
		if remainingAmount <= elem[x].Amount {
			aggNominalAmount += elem[x].Price * remainingAmount
			remainingAmount = 0
			break
		}
		aggNominalAmount += elem[x].Price * elem[x].Amount
		remainingAmount -= elem[x].Amount
	}
	return aggNominalAmount, remainingAmount
}
