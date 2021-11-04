package signal

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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

// GetPrice returns the price
func (s *Signal) GetPrice() decimal.Decimal {
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

// SetCloseOrderID links an existing order id
// for a futures order set to be closed
func (s *Signal) SetCloseOrderID(id string) {
	if s.AssetType == asset.Futures {
		s.CloseOrderID = id
	}
}

// GetLinkedOrderID returns the order ID of a 
// linked futures order
func (s *Signal) GetLinkedOrderID() string {
	return s.CloseOrderID
}
