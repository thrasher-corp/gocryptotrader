package engine

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
)

var (
	// ErrWithdrawRequestNotFound message to display when no record is found
	ErrWithdrawRequestNotFound = errors.New("request not found")
)

// WithdrawManager is responsible for performing withdrawal requests and
// saving them to the database
type WithdrawManager struct {
	exchangeManager  subsystem.ExchangeManager
	portfolioManager subsystem.PortfolioManager
	isDryRun         bool
}
