package bitstamp

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AccountBalanceResponse holds the balance of a currency
type AccountBalanceResponse struct {
	Currency  currency.Code `json:"currency"`
	Total     types.Number  `json:"total"`
	Available types.Number  `json:"available"`
	Reserved  types.Number  `json:"reserved"`
}
