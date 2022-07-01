package signal

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// IsSignal returns whether the event is a signal type
func (s *Signal) IsSignal() bool {
	return true
}

// SetDirection sets the direction
func (s *Signal) SetDirection(st order.Side) {
	s.Direction = st
}

// GetDirection returns the direction
func (s *Signal) GetDirection() order.Side {
	return s.Direction
}

// SetBuyLimit sets the buy limit
func (s *Signal) SetBuyLimit(f decimal.Decimal) {
	s.BuyLimit = f
}

// GetBuyLimit returns the buy limit
func (s *Signal) GetBuyLimit() decimal.Decimal {
	return s.BuyLimit
}

// SetSellLimit sets the sell limit
func (s *Signal) SetSellLimit(f decimal.Decimal) {
	s.SellLimit = f
}

// GetSellLimit returns the sell limit
func (s *Signal) GetSellLimit() decimal.Decimal {
	return s.SellLimit
}

// Pair returns the currency pair
func (s *Signal) Pair() currency.Pair {
	return s.CurrencyPair
}

// GetClosePrice returns the price
func (s *Signal) GetClosePrice() decimal.Decimal {
	return s.ClosePrice
}

// SetPrice sets the price
func (s *Signal) SetPrice(f decimal.Decimal) {
	s.ClosePrice = f
}

// GetAmount retrieves the order amount
func (s *Signal) GetAmount() decimal.Decimal {
	return s.Amount
}

// SetAmount sets the order amount
func (s *Signal) SetAmount(d decimal.Decimal) {
	s.Amount = d
}

// GetUnderlyingPair returns the underlying currency pair
func (s *Signal) GetUnderlyingPair() currency.Pair {
	return s.UnderlyingPair
}

// GetFillDependentEvent returns the fill dependent event
// so it can be added to the event queue
func (s *Signal) GetFillDependentEvent() Event {
	return s.FillDependentEvent
}

// GetCollateralCurrency returns the collateral currency
func (s *Signal) GetCollateralCurrency() currency.Code {
	return s.CollateralCurrency
}

// IsNil says if the event is nil
func (s *Signal) IsNil() bool {
	return s == nil
}

// MatchOrderAmount ensures an order must match
// its set amount or fail
func (s *Signal) MatchOrderAmount() bool {
	return s.MatchesOrderAmount
}
