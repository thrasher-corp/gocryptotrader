package engine

import (
	"errors"
)

// ErrWithdrawRequestNotFound message to display when no record is found
var ErrWithdrawRequestNotFound = errors.New("request not found")

// WithdrawManager is responsible for performing withdrawal requests and
// saving them to the database
type WithdrawManager struct {
	exchangeManager  iExchangeManager
	portfolioManager iPortfolioManager
	isDryRun         bool
}
