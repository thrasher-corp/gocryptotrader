package kraken

import (
	"errors"
	"time"
)

var (
	errFuturesSelfTradeStrategyEmpty = errors.New("self-trade strategy is empty")
	errFuturesInvalidMaxLeverage     = errors.New("invalid off-book max leverage")
	errFuturesNegativeLimit          = errors.New("limit cannot be negative")
)

// FuturesSelfTradeStrategyResponse holds the account-wide self-trade matching strategy
type FuturesSelfTradeStrategyResponse struct {
	// Strategy is REJECT_TAKER, the default, which rejects a taker order that would match a maker order from any
	// subaccount; CANCEL_MAKER_SELF, which cancels the maker order only when the taker order's account sent it;
	// CANCEL_MAKER_CHILD, which lets only a master account cancel maker orders, its own and its subaccounts'; or
	// CANCEL_MAKER_ANY, which lets master accounts and their subaccounts cancel maker orders
	Strategy   string    `json:"strategy"`
	ServerTime time.Time `json:"serverTime"`
}

// FuturesOffBookMaxLeverageResponse holds the master account's off-book max leverage cap
type FuturesOffBookMaxLeverageResponse struct {
	// MaxLeverage caps the effective leverage, gross open position notional over margin equity, that liquidation
	// assignments and RFQ offer fills may take the account to; book orders are never capped. It is nil when no cap is
	// set, and a cap of 0 blocks every fill that would increase exposure
	MaxLeverage *float64  `json:"maxLeverage"`
	ServerTime  time.Time `json:"serverTime"`
}
