package orderbook

import (
	"errors"
	"fmt"
	"sort"

	"github.com/shopspring/decimal"
	math "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// WhaleBombResult returns the whale bomb result
type WhaleBombResult struct {
	Amount               decimal.Decimal
	MinimumPrice         decimal.Decimal
	MaximumPrice         decimal.Decimal
	PercentageGainOrLoss decimal.Decimal
	Orders               orderSummary
	Status               string
}

// WhaleBomb finds the amount required to target a price
func (b *Base) WhaleBomb(priceTarget decimal.Decimal, buy bool) (*WhaleBombResult, error) {
	if priceTarget.LessThan(decimal.Zero) {
		return nil, errors.New("price target is invalid")
	}
	direction := "Buying"
	if !buy {
		direction = "Selling"
	}
	a, orders := b.findAmount(priceTarget, buy)
	min, max := orders.MinimumPrice(false), orders.MaximumPrice(true)
	var err error
	if priceTarget.LessThan(min) || priceTarget.GreaterThan(max) {
		err = errors.New("unable to hit price target due to insufficient orderbook items")
	}
	status := fmt.Sprintf("%s %s %s worth of %s will send the price from %s to %s [%.2f%%] and take %d orders.",
		direction,
		a.Round(2),
		b.Pair.Quote,
		b.Pair.Base,
		min,
		max,
		math.CalculatePercentageGainOrLoss(max.InexactFloat64(), min.InexactFloat64()),
		len(orders))
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
func (b *Base) SimulateOrder(amount decimal.Decimal, buy bool) *OrderSimulationResult {
	if buy {
		orders, amt := b.buy(amount)
		min, max := orders.MinimumPrice(false), orders.MaximumPrice(true)
		pct := math.CalculatePercentageGainOrLoss(max.InexactFloat64(), min.InexactFloat64())
		status := fmt.Sprintf("Buying %s %s worth of %s will send the price from %s to %s [%.2f%%] and take %d orders.",
			amount.Round(2),
			b.Pair.Quote,
			b.Pair.Base,
			min,
			max,
			pct,
			len(orders))
		return &OrderSimulationResult{
			Orders:               orders,
			Amount:               amt,
			MinimumPrice:         min,
			MaximumPrice:         max,
			PercentageGainOrLoss: decimal.NewFromFloat(pct),
			Status:               status,
		}
	}
	orders, amt := b.sell(amount)
	min, max := orders.MinimumPrice(false), orders.MaximumPrice(true)
	pct := math.CalculatePercentageGainOrLoss(min.InexactFloat64(), max.InexactFloat64())
	status := fmt.Sprintf("Selling %s %s worth of %s will send the price from %s to %s [%.2f%%] and take %d orders.",
		amount,
		b.Pair.Base,
		b.Pair.Quote,
		max,
		min,
		pct,
		len(orders))
	return &OrderSimulationResult{
		Orders:               orders,
		Amount:               amt,
		MinimumPrice:         min,
		MaximumPrice:         max,
		PercentageGainOrLoss: decimal.NewFromFloat(pct),
		Status:               status,
	}
}

type orderSummary []Item

// Print prints the full order summary to console
func (o orderSummary) Print() {
	for x := range o {
		log.Debugf(log.OrderBook, "Order: Price: %s Amount: %s", o[x].Price, o[x].Amount)
	}
}

func (o orderSummary) MinimumPrice(reverse bool) decimal.Decimal {
	if len(o) != 0 {
		sortOrdersByPrice(&o, reverse)
		return o[0].Price
	}
	return decimal.Zero
}

func (o orderSummary) MaximumPrice(reverse bool) decimal.Decimal {
	if len(o) != 0 {
		sortOrdersByPrice(&o, reverse)
		return o[0].Price
	}
	return decimal.Zero
}

// ByPrice used for sorting orders by order date
type ByPrice orderSummary

func (b ByPrice) Len() int           { return len(b) }
func (b ByPrice) Less(i, j int) bool { return b[i].Price.LessThan(b[j].Price) }
func (b ByPrice) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

// sortOrdersByPrice the caller function to sort orders
func sortOrdersByPrice(o *orderSummary, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByPrice(*o)))
	} else {
		sort.Sort(ByPrice(*o))
	}
}

func (b *Base) findAmount(price decimal.Decimal, buy bool) (decimal.Decimal, orderSummary) {
	var orders orderSummary
	var amt decimal.Decimal

	if buy {
		for x := range b.Asks {
			if b.Asks[x].Price.GreaterThanOrEqual(price) {
				amt = amt.Add(b.Asks[x].Price.Mul(b.Asks[x].Amount))
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
			amt = amt.Add(b.Asks[x].Price.Mul(b.Asks[x].Amount))
		}
		return amt, orders
	}

	for x := range b.Bids {
		if b.Bids[x].Price.LessThanOrEqual(price) {
			amt = amt.Add(b.Bids[x].Amount)
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
		amt = amt.Add(b.Bids[x].Amount)
	}
	return amt, orders
}

func (b *Base) buy(amount decimal.Decimal) (orders orderSummary, baseAmount decimal.Decimal) {
	var processedAmt decimal.Decimal
	for x := range b.Asks {
		subtotal := b.Asks[x].Price.Mul(b.Asks[x].Amount)
		if processedAmt.Add(subtotal).GreaterThanOrEqual(amount) {
			diff := amount.Sub(processedAmt)
			subAmt := diff.Div(b.Asks[x].Price)
			orders = append(orders, Item{
				Price:  b.Asks[x].Price,
				Amount: subAmt,
			})
			baseAmount = baseAmount.Add(subAmt)
			break
		}
		processedAmt = processedAmt.Add(subtotal)
		baseAmount = baseAmount.Add(b.Asks[x].Amount)
		orders = append(orders, Item{
			Price:  b.Asks[x].Price,
			Amount: b.Asks[x].Amount,
		})
	}
	return
}

func (b *Base) sell(amount decimal.Decimal) (orders orderSummary, quoteAmount decimal.Decimal) {
	var processedAmt decimal.Decimal
	for x := range b.Bids {
		if processedAmt.Add(b.Bids[x].Amount).GreaterThanOrEqual(amount) {
			diff := amount.Sub(processedAmt)
			orders = append(orders, Item{
				Price:  b.Bids[x].Price,
				Amount: diff,
			})
			quoteAmount = quoteAmount.Add(diff.Mul(b.Bids[x].Price))
			break
		}
		processedAmt = processedAmt.Add(b.Bids[x].Amount)
		quoteAmount = quoteAmount.Add(b.Bids[x].Amount.Mul(b.Bids[x].Price))
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
func (b *Base) GetAveragePrice(buy bool, amount decimal.Decimal) (decimal.Decimal, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, errAmountInvalid
	}
	var aggNominalAmount, remainingAmount decimal.Decimal
	if buy {
		aggNominalAmount, remainingAmount = b.Asks.FindNominalAmount(amount)
	} else {
		aggNominalAmount, remainingAmount = b.Bids.FindNominalAmount(amount)
	}
	if remainingAmount != decimal.Zero {
		return decimal.Zero, fmt.Errorf("%w for %v on exchange %v to support a buy amount of %v",
			errNotEnoughLiquidity,
			b.Pair,
			b.Exchange,
			amount)
	}
	return aggNominalAmount.Div(amount), nil
}

// FindNominalAmount finds the nominal amount spent in terms of the quote
// If the orderbook doesn't have enough liquidity it returns a non zero
// remaining amount value
func (elem Items) FindNominalAmount(amount decimal.Decimal) (aggNominalAmount, remainingAmount decimal.Decimal) {
	remainingAmount = amount
	for x := range elem {
		if remainingAmount.LessThanOrEqual(elem[x].Amount) {
			aggNominalAmount = aggNominalAmount.Add(elem[x].Price.Mul(remainingAmount))
			remainingAmount = decimal.Zero
			break
		} else {
			aggNominalAmount = aggNominalAmount.Add(elem[x].Price.Mul(elem[x].Amount))
			remainingAmount = remainingAmount.Sub(elem[x].Amount)
		}
	}
	return aggNominalAmount, remainingAmount
}
