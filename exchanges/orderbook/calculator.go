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
	Orders               Levels
	Status               string
}

// WhaleBomb finds the amount required to target a price
func (b *Book) WhaleBomb(priceTarget float64, buy bool) (*WhaleBombResult, error) {
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
	var percent, minPrice, maxPrice, amount float64
	if buy {
		minPrice = action.ReferencePrice
		maxPrice = action.LevelPositionPrice
		amount = action.QuoteAmount
		percent = math.PercentageChange(action.ReferencePrice, action.LevelPositionPrice)
		status = fmt.Sprintf("Buying using %.2f %s worth of %s will send the price from %v to %v [%.2f%%] and impact %d price level(s). %s",
			amount, b.Pair.Quote, b.Pair.Base, minPrice, maxPrice,
			percent, len(action.Levels), warning)
	} else {
		minPrice = action.LevelPositionPrice
		maxPrice = action.ReferencePrice
		amount = action.BaseAmount
		percent = math.PercentageChange(action.ReferencePrice, action.LevelPositionPrice)
		status = fmt.Sprintf("Selling using %.2f %s worth of %s will send the price from %v to %v [%.2f%%] and impact %d price level(s). %s",
			amount, b.Pair.Base, b.Pair.Quote, maxPrice, minPrice,
			percent, len(action.Levels), warning)
	}

	return &WhaleBombResult{
		Amount:               amount,
		Orders:               action.Levels,
		MinimumPrice:         minPrice,
		MaximumPrice:         maxPrice,
		Status:               status,
		PercentageGainOrLoss: percent,
	}, err
}

// SimulateOrder simulates an order
func (b *Book) SimulateOrder(amount float64, buy bool) (*WhaleBombResult, error) {
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
		maximumPrice = action.LevelPositionPrice
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
		minimumPrice = action.LevelPositionPrice
		maximumPrice = action.ReferencePrice
		sold = b.Pair.Base
		bought = b.Pair.Quote
	}

	var warning string
	if action.FullLiquidityUsed {
		warning = fullLiquidityUsageWarning
	}

	pct := math.PercentageChange(action.ReferencePrice, action.LevelPositionPrice)
	status := fmt.Sprintf("%s using %f %v worth of %v will send the price from %v to %v [%.2f%%] and impact %v price level(s). %s",
		direction, soldAmount, sold, bought, action.ReferencePrice,
		action.LevelPositionPrice, pct, len(action.Levels), warning)
	return &WhaleBombResult{
		Orders:               action.Levels,
		Amount:               boughtAmount,
		MinimumPrice:         minimumPrice,
		MaximumPrice:         maximumPrice,
		PercentageGainOrLoss: pct,
		Status:               status,
	}, nil
}

func (b *Book) findAmount(priceTarget float64, buy bool) (*DeploymentAction, error) {
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
				action.LevelPositionPrice = b.Asks[x].Price
				return &action, nil
			}
			action.Levels = append(action.Levels, b.Asks[x])
			action.QuoteAmount += b.Asks[x].Price * b.Asks[x].Amount
			action.BaseAmount += b.Asks[x].Amount
		}
		action.LevelPositionPrice = b.Asks[len(b.Asks)-1].Price
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
			action.LevelPositionPrice = b.Bids[x].Price
			return &action, nil
		}
		action.Levels = append(action.Levels, b.Bids[x])
		action.QuoteAmount += b.Bids[x].Price * b.Bids[x].Amount
		action.BaseAmount += b.Bids[x].Amount
	}
	action.LevelPositionPrice = b.Bids[len(b.Bids)-1].Price
	action.FullLiquidityUsed = true
	return &action, nil
}

// DeploymentAction defines deployment information on a liquidity side.
type DeploymentAction struct {
	ReferencePrice     float64
	LevelPositionPrice float64
	BaseAmount         float64
	QuoteAmount        float64
	Levels             Levels
	FullLiquidityUsed  bool
}

func (b *Book) buy(quote float64) (*DeploymentAction, error) {
	if quote <= 0 {
		return nil, errQuoteAmountInvalid
	}
	if len(b.Asks) == 0 {
		return nil, errNoLiquidity
	}
	action := &DeploymentAction{ReferencePrice: b.Asks[0].Price}
	for x := range b.Asks {
		action.LevelPositionPrice = b.Asks[x].Price
		levelValue := b.Asks[x].Price * b.Asks[x].Amount
		action.QuoteAmount += levelValue
		remaining := quote - levelValue
		if remaining <= 0 {
			if remaining == 0 {
				if len(b.Asks)-1 > x {
					action.LevelPositionPrice = b.Asks[x+1].Price
				} else {
					action.FullLiquidityUsed = true
				}
			}
			subAmount := quote / b.Asks[x].Price
			action.Levels = append(action.Levels, Level{Price: b.Asks[x].Price, Amount: subAmount})
			action.BaseAmount += subAmount
			return action, nil
		}
		if len(b.Asks)-1 <= x {
			action.FullLiquidityUsed = true
		}
		quote = remaining
		action.BaseAmount += b.Asks[x].Amount
		action.Levels = append(action.Levels, b.Asks[x])
	}

	return action, nil
}

func (b *Book) sell(base float64) (*DeploymentAction, error) {
	if base <= 0 {
		return nil, errBaseAmountInvalid
	}
	if len(b.Bids) == 0 {
		return nil, errNoLiquidity
	}
	action := &DeploymentAction{ReferencePrice: b.Bids[0].Price}
	for x := range b.Bids {
		action.LevelPositionPrice = b.Bids[x].Price
		remaining := base - b.Bids[x].Amount
		if remaining <= 0 {
			if remaining == 0 {
				if len(b.Bids)-1 > x {
					action.LevelPositionPrice = b.Bids[x+1].Price
				} else {
					action.FullLiquidityUsed = true
				}
			}
			action.Levels = append(action.Levels, Level{Price: b.Bids[x].Price, Amount: base})
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
		action.Levels = append(action.Levels, b.Bids[x])
	}
	return action, nil
}

// GetAveragePrice finds the average buy or sell price of a specified amount.
// It finds the nominal amount spent on the total purchase or sell and uses it
// to find the average price for an individual unit bought or sold
func (b *Book) GetAveragePrice(buy bool, amount float64) (float64, error) {
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
func (l Levels) FindNominalAmount(amount float64) (aggNominalAmount, remainingAmount float64) {
	remainingAmount = amount
	for x := range l {
		if remainingAmount <= l[x].Amount {
			aggNominalAmount += l[x].Price * remainingAmount
			remainingAmount = 0
			break
		}
		aggNominalAmount += l[x].Price * l[x].Amount
		remainingAmount -= l[x].Amount
	}
	return aggNominalAmount, remainingAmount
}
