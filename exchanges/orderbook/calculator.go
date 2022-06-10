package orderbook

import (
	"errors"
	"fmt"

	math "github.com/thrasher-corp/gocryptotrader/common/math"
)

var (
	errPriceTargetInvalid     = errors.New("price target is invalid")
	errUnableToHitPriceTarget = errors.New("unable to hit price target due to insufficient orderbook items")
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

	var status string
	var percent, min, max, amount float64
	if buy {
		min = action.ReferencePrice
		max = action.TranchePositionPrice
		amount = action.QuoteAmount
		percent = math.CalculatePercentageGainOrLoss(action.TranchePositionPrice, action.ReferencePrice)
		status = fmt.Sprintf("Buying %.2f %s worth of %s will send the price from %v to %v [%.2f%%] and take %v orders.",
			amount, b.Pair.Quote, b.Pair.Base, min, max,
			percent, len(action.Orders))
	} else {
		min = action.TranchePositionPrice
		max = action.ReferencePrice
		amount = action.BaseAmount
		percent = math.CalculatePercentageGainOrLoss(action.TranchePositionPrice, action.ReferencePrice)
		status = fmt.Sprintf("Selling %.2f %s worth of %s will send the price from %v to %v [%.2f%%] and take %v orders.",
			amount, b.Pair.Base, b.Pair.Quote, max, min,
			percent, len(action.Orders))
	}

	return &WhaleBombResult{
		Amount:               amount,
		Orders:               action.Orders,
		MinimumPrice:         min,
		MaximumPrice:         max,
		Status:               status,
		PercentageGainOrLoss: percent,
	}, err
}

// SimulateOrder simulates an order
func (b *Base) SimulateOrder(amount float64, buy bool) *WhaleBombResult {
	if buy {
		action, err := b.buy(amount)
		if err != nil {
			return nil
		}
		pct := math.CalculatePercentageGainOrLoss(action.TranchePositionPrice, action.ReferencePrice)
		status := fmt.Sprintf("Buying %.2f %v worth of %v will send the price from %v to %v [%.2f%%] and take %v orders.",
			amount, b.Pair.Quote.String(), b.Pair.Base.String(), action.ReferencePrice, action.TranchePositionPrice,
			pct, len(action.Orders))
		return &WhaleBombResult{
			Orders:               action.Orders,
			Amount:               action.BaseAmount,
			MinimumPrice:         action.ReferencePrice,
			MaximumPrice:         action.TranchePositionPrice,
			PercentageGainOrLoss: pct,
			Status:               status,
		}
	}
	action, err := b.sell(amount)
	if err != nil {
		return nil
	}
	pct := math.CalculatePercentageGainOrLoss(action.TranchePositionPrice, action.ReferencePrice)
	status := fmt.Sprintf("Selling %f %v worth of %v will send the price from %v to %v [%.2f%%] and take %v orders.",
		amount, b.Pair.Base.String(), b.Pair.Quote.String(), action.ReferencePrice, action.TranchePositionPrice,
		pct, len(action.Orders))
	return &WhaleBombResult{
		Orders:               action.Orders,
		Amount:               action.QuoteAmount,
		MinimumPrice:         action.TranchePositionPrice,
		MaximumPrice:         action.ReferencePrice,
		PercentageGainOrLoss: pct,
		Status:               status,
	}
}

func (b *Base) findAmount(priceTarget float64, buy bool) (*DeploymentAction, error) {
	action := DeploymentAction{}
	if buy {
		if len(b.Asks) == 0 {
			return nil, errNoLiquidity
		}
		action.ReferencePrice = b.Asks[0].Price
		if action.ReferencePrice > priceTarget {
			return nil, errUnableToHitPriceTarget
		}
		for x := range b.Asks {
			if b.Asks[x].Price >= priceTarget {
				action.TranchePositionPrice = b.Asks[x].Price
				return &action, nil
			}
			action.Orders = append(action.Orders, b.Asks[x])
			action.QuoteAmount += b.Asks[x].Price * b.Asks[x].Amount
			action.BaseAmount += b.Asks[x].Amount
		}
		return nil, errNotEnoughLiquidity
	}

	if len(b.Bids) == 0 {
		return nil, errNoLiquidity
	}
	action.ReferencePrice = b.Bids[0].Price
	if action.ReferencePrice < priceTarget {
		return nil, errUnableToHitPriceTarget
	}
	for x := range b.Bids {
		if b.Bids[x].Price <= priceTarget {
			action.TranchePositionPrice = b.Bids[x].Price
			return &action, nil
		}
		action.Orders = append(action.Orders, b.Bids[x])
		action.QuoteAmount += b.Bids[x].Price * b.Bids[x].Amount
		action.BaseAmount += b.Bids[x].Amount
	}
	return nil, errNotEnoughLiquidity
}

// DeploymentAction defines deployment information on a liquidity side.
type DeploymentAction struct {
	ReferencePrice       float64
	TranchePositionPrice float64
	BaseAmount           float64
	QuoteAmount          float64
	Orders               Items
}

func (b *Base) buy(quote float64) (*DeploymentAction, error) {
	if len(b.Asks) == 0 {
		return nil, errNoLiquidity
	}
	action := DeploymentAction{ReferencePrice: b.Asks[0].Price}
	for x := range b.Asks {
		trancheValue := b.Asks[x].Price * b.Asks[x].Amount
		if action.QuoteAmount+trancheValue >= quote {
			diff := quote - action.QuoteAmount
			subAmt := diff / b.Asks[x].Price
			action.Orders = append(action.Orders,
				Item{Price: b.Asks[x].Price, Amount: subAmt})
			action.BaseAmount += subAmt
			if len(b.Asks) == x+1 {
				return nil, errNotEnoughLiquidity
			}
			action.TranchePositionPrice = b.Asks[x+1].Price
			break
		}
		action.QuoteAmount += trancheValue
		action.BaseAmount += b.Asks[x].Amount
		action.Orders = append(action.Orders, b.Asks[x])
	}
	return &action, nil
}

func (b *Base) sell(base float64) (*DeploymentAction, error) {
	if len(b.Bids) == 0 {
		return nil, errNoLiquidity
	}
	action := DeploymentAction{ReferencePrice: b.Bids[0].Price}
	for x := range b.Bids {
		if action.BaseAmount+b.Bids[x].Amount >= base {
			diff := base - action.BaseAmount
			action.Orders = append(action.Orders,
				Item{Price: b.Bids[x].Price, Amount: diff})
			action.QuoteAmount += diff * b.Bids[x].Price
			if len(b.Bids) == x+1 {
				return nil, errNotEnoughLiquidity
			}
			action.TranchePositionPrice = b.Bids[x+1].Price
			break
		}
		action.BaseAmount += b.Bids[x].Amount
		action.QuoteAmount += b.Bids[x].Amount * b.Bids[x].Price
		action.Orders = append(action.Orders, b.Bids[x])
	}
	return &action, nil
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
