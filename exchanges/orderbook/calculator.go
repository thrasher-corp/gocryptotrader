package orderbook

import (
	"errors"
	"fmt"
	"sort"

	math "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// WhaleBombResult returns the whale bomb result
type WhaleBombResult struct {
	Amount               float64
	MinimumPrice         float64
	MaximumPrice         float64
	PercentageGainOrLoss float64
	Orders               orderSummary
	Status               string
}

// WhaleBomb finds the amount required to target a price
func (b *Base) WhaleBomb(priceTarget float64, buy bool) (*WhaleBombResult, error) {
	if priceTarget < 0 {
		return nil, errors.New("price target is invalid")
	}
	if buy {
		a, orders := b.findAmount(priceTarget, true)
		min, max := orders.MinimumPrice(false), orders.MaximumPrice(true)
		var err error
		if max < priceTarget {
			err = errors.New("unable to hit price target due to insufficient orderbook items")
		}
		status := fmt.Sprintf("Buying %.2f %v worth of %v will send the price from %v to %v [%.2f%%] and take %v orders.",
			a, b.Pair.Quote.String(), b.Pair.Base.String(), min, max,
			math.CalculatePercentageGainOrLoss(max, min), len(orders))
		return &WhaleBombResult{
			Amount:       a,
			Orders:       orders,
			MinimumPrice: min,
			MaximumPrice: max,
			Status:       status,
		}, err
	}

	a, orders := b.findAmount(priceTarget, false)
	min, max := orders.MinimumPrice(false), orders.MaximumPrice(true)
	var err error
	if min > priceTarget {
		err = errors.New("unable to hit price target due to insufficient orderbook items")
	}
	status := fmt.Sprintf("Selling %.2f %v worth of %v will send the price from %v to %v [%.2f%%] and take %v orders.",
		a, b.Pair.Base.String(), b.Pair.Quote.String(), max, min,
		math.CalculatePercentageGainOrLoss(min, max), len(orders))
	return &WhaleBombResult{
		Amount:       a,
		Orders:       orders,
		MinimumPrice: min,
		MaximumPrice: max,
		Status:       status,
	}, err
}

// OrderSimulationResult returns the order simulation result
type OrderSimulationResult WhaleBombResult

// SimulateOrder simulates an order
func (b *Base) SimulateOrder(amount float64, buy bool) *OrderSimulationResult {
	if buy {
		orders, amt := b.buy(amount)
		min, max := orders.MinimumPrice(false), orders.MaximumPrice(true)
		pct := math.CalculatePercentageGainOrLoss(max, min)
		status := fmt.Sprintf("Buying %.2f %v worth of %v will send the price from %v to %v [%.2f%%] and take %v orders.",
			amount, b.Pair.Quote.String(), b.Pair.Base.String(), min, max,
			pct, len(orders))
		return &OrderSimulationResult{
			Orders:               orders,
			Amount:               amt,
			MinimumPrice:         min,
			MaximumPrice:         max,
			PercentageGainOrLoss: pct,
			Status:               status,
		}
	}
	orders, amt := b.sell(amount)
	min, max := orders.MinimumPrice(false), orders.MaximumPrice(true)
	pct := math.CalculatePercentageGainOrLoss(min, max)
	status := fmt.Sprintf("Selling %f %v worth of %v will send the price from %v to %v [%.2f%%] and take %v orders.",
		amount, b.Pair.Base.String(), b.Pair.Quote.String(), max, min,
		pct, len(orders))
	return &OrderSimulationResult{
		Orders:               orders,
		Amount:               amt,
		MinimumPrice:         min,
		MaximumPrice:         max,
		PercentageGainOrLoss: pct,
		Status:               status,
	}
}

type orderSummary []Item

func (o orderSummary) Print() {
	for x := range o {
		log.Debugf(log.OrderBook, "Order: Price: %f Amount: %f", o[x].Price, o[x].Amount)
	}
}

func (o orderSummary) MinimumPrice(reverse bool) float64 {
	if len(o) != 0 {
		sortOrdersByPrice(&o, reverse)
		return o[0].Price
	}
	return 0
}

func (o orderSummary) MaximumPrice(reverse bool) float64 {
	if len(o) != 0 {
		sortOrdersByPrice(&o, reverse)
		return o[0].Price
	}
	return 0
}

// ByPrice used for sorting orders by order date
type ByPrice orderSummary

func (b ByPrice) Len() int           { return len(b) }
func (b ByPrice) Less(i, j int) bool { return b[i].Price < b[j].Price }
func (b ByPrice) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

// sortOrdersByPrice the caller function to sort orders
func sortOrdersByPrice(o *orderSummary, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByPrice(*o)))
	} else {
		sort.Sort(ByPrice(*o))
	}
}

func (b *Base) findAmount(price float64, buy bool) (float64, orderSummary) {
	var orders orderSummary
	var amt float64

	if buy {
		for x := range b.Asks {
			if b.Asks[x].Price >= price {
				amt += b.Asks[x].Price * b.Asks[x].Amount
				orders = append(orders, Item{
					Price:  b.Asks[x].Price,
					Amount: b.Asks[x].Amount,
				})
				return amt, orders
			}
			orders = append(orders, Item{
				Price:  b.Asks[x].Price,
				Amount: b.Asks[x].Amount,
			})
			amt += b.Asks[x].Price * b.Asks[x].Amount
		}
		return amt, orders
	}

	for x := range b.Bids {
		if b.Bids[x].Price <= price {
			amt += b.Bids[x].Amount
			orders = append(orders, Item{
				Price:  b.Bids[x].Price,
				Amount: b.Bids[x].Amount,
			})
			break
		}
		orders = append(orders, Item{
			Price:  b.Bids[x].Price,
			Amount: b.Bids[x].Amount,
		})
		amt += b.Bids[x].Amount
	}
	return amt, orders
}

func (b *Base) buy(amount float64) (orders orderSummary, baseAmount float64) {
	var processedAmt float64
	for x := range b.Asks {
		subtotal := b.Asks[x].Price * b.Asks[x].Amount
		if processedAmt+subtotal >= amount {
			diff := amount - processedAmt
			subAmt := diff / b.Asks[x].Price
			orders = append(orders, Item{
				Price:  b.Asks[x].Price,
				Amount: subAmt,
			})
			baseAmount += subAmt
			break
		}
		processedAmt += subtotal
		baseAmount += b.Asks[x].Amount
		orders = append(orders, Item{
			Price:  b.Asks[x].Price,
			Amount: b.Asks[x].Amount,
		})
	}
	return
}

func (b *Base) sell(amount float64) (orders orderSummary, quoteAmount float64) {
	var processedAmt float64
	for x := range b.Bids {
		if processedAmt+b.Bids[x].Amount >= amount {
			diff := amount - processedAmt
			orders = append(orders, Item{
				Price:  b.Bids[x].Price,
				Amount: diff,
			})
			quoteAmount += diff * b.Bids[x].Price
			break
		}
		processedAmt += b.Bids[x].Amount
		quoteAmount += b.Bids[x].Amount * b.Bids[x].Price
		orders = append(orders, Item{
			Price:  b.Bids[x].Price,
			Amount: b.Bids[x].Amount,
		})
	}
	return
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
		} else {
			aggNominalAmount += elem[x].Price * elem[x].Amount
			remainingAmount -= elem[x].Amount
		}
	}
	return aggNominalAmount, remainingAmount
}
