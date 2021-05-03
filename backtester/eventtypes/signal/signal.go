package signal

import (
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
func (s *Signal) SetBuyLimit(f float64) {
	s.BuyLimit = f
}

// GetBuyLimit returns the buy limit
func (s *Signal) GetBuyLimit() float64 {
	return s.BuyLimit
}

// SetSellLimit sets the sell limit
func (s *Signal) SetSellLimit(f float64) {
	s.SellLimit = f
}

// GetSellLimit returns the sell limit
func (s *Signal) GetSellLimit() float64 {
	return s.SellLimit
}

// Pair returns the currency pair
func (s *Signal) Pair() currency.Pair {
	return s.CurrencyPair
}

// GetPrice returns the price
func (s *Signal) GetPrice() float64 {
	return s.ClosePrice
}

// SetPrice sets the price
func (s *Signal) SetPrice(f float64) {
	s.ClosePrice = f
}
