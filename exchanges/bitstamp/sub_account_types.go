package bitstamp

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// TransferRequest holds the parameters for transferring balances between the main account and a sub account
type TransferRequest struct {
	Amount     float64
	Currency   currency.Code
	SubAccount uint64
}
