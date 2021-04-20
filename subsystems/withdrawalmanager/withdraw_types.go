package withdrawalmanager

import (
	"errors"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var (
	// ErrWithdrawRequestNotFound message to display when no record is found
	ErrWithdrawRequestNotFound = errors.New("request not found")
)

type Manager struct {
	exchangeManager  iExchangeManager
	portfolioManager iPortfolioManager
	isDryRun         bool
}

type iExchangeManager interface {
	GetExchangeByName(string) exchange.IBotExchange
}

type iPortfolioManager interface {
	IsWhiteListed(string) bool
	IsExchangeSupported(string, string) bool
}
