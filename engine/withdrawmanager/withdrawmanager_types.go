package withdrawmanager

import (
	"errors"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var (
	// ErrWithdrawRequestNotFound message to display when no record is found
	ErrWithdrawRequestNotFound = errors.New("request not found")
)

// WithdrawManager is responsible for performing withdrawal requests and
// saving them to the database
type WithdrawManager struct {
	exchangeManager  iExchangeManager
	portfolioManager iPortfolioManager
	isDryRun         bool
}

// iExchangeManager limits exposure of accessible functions to exchange manager
type iExchangeManager interface {
	GetExchangeByName(string) exchange.IBotExchange
}

// iPortfolioManager limits exposure of accessible functions to portfolio manager
type iPortfolioManager interface {
	IsWhiteListed(string) bool
	IsExchangeSupported(string, string) bool
}
