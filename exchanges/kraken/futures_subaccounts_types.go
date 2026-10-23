package kraken

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var errFuturesSubaccountUIDEmpty = errors.New("subaccount UID is empty")

// FuturesSubaccountsResponse holds the master account's subaccounts
type FuturesSubaccountsResponse struct {
	MasterAccountUID string              `json:"masterAccountUid"`
	Subaccounts      []FuturesSubaccount `json:"subaccounts"`
	ServerTime       time.Time           `json:"serverTime"`
}

// FuturesSubaccount holds a subaccount's details and balances
type FuturesSubaccount struct {
	AccountUID string `json:"accountUid"`
	Email      string `json:"email"`
	FullName   string `json:"fullName"`
	// HoldingAccounts holds the balance of each holding account, one per currency
	HoldingAccounts []FuturesSubaccountHoldingBalance `json:"holdingAccounts"`
	// FuturesAccounts holds each single-collateral margin wallet's available margin
	FuturesAccounts []FuturesSubaccountMarginBalance `json:"futuresAccounts"`
	FlexAccount     FuturesSubaccountFlexAccount     `json:"flexAccount"`
}

// FuturesSubaccountHoldingBalance is the balance of a subaccount's holding account
type FuturesSubaccountHoldingBalance struct {
	Currency currency.Code `json:"currency"`
	Amount   float64       `json:"amount"`
}

// FuturesSubaccountMarginBalance is a subaccount's single-collateral margin wallet
type FuturesSubaccountMarginBalance struct {
	// Name is the wallet's name, such as f-xbt:usd
	Name string `json:"name"`
	// AvailableMargin is in the quote currency of the wallet's pair, and can be negative
	AvailableMargin float64 `json:"availableMargin"`
}

// FuturesSubaccountFlexAccount is a subaccount's multi-collateral wallet. Its totals are in USD
type FuturesSubaccountFlexAccount struct {
	Currencies    []FuturesSubaccountFlexCurrency `json:"currencies"`
	InitialMargin float64                         `json:"initialMargin"`
	// InitialMarginWithOrders is documented as the initial margin of open positions without the margin open orders hold
	InitialMarginWithOrders float64 `json:"initialMarginWithOrders"`
	MaintenanceMargin       float64 `json:"maintenanceMargin"`
	BalanceValue            float64 `json:"balanceValue"`
	// PortfolioValue is BalanceValue plus the total unrealised profit and loss
	PortfolioValue float64 `json:"portfolioValue"`
	// CollateralValue is the value of the balances usable as margin, BalanceValue less haircuts
	CollateralValue   float64 `json:"collateralValue"`
	UnrealisedPNL     float64 `json:"pnl"`
	UnrealisedFunding float64 `json:"unrealizedFunding"`
	// TotalUnrealised is the unrealised profit and loss and funding of open positions
	TotalUnrealised float64 `json:"totalUnrealized"`
	// TotalUnrealisedAsMargin is the part of TotalUnrealised usable as margin, after haircuts and conversion fees
	TotalUnrealisedAsMargin float64 `json:"totalUnrealizedAsMargin"`
	// AvailableMargin is the margin available for new orders, MarginEquity less the total initial margin
	AvailableMargin float64 `json:"availableMargin"`
	// MarginEquity is CollateralValue plus TotalUnrealisedAsMargin
	MarginEquity             float64                          `json:"marginEquity"`
	PortfolioMarginBreakdown *FuturesPortfolioMarginBreakdown `json:"portfolioMarginBreakdown"`
}

// FuturesSubaccountFlexCurrency is a currency's balance in a subaccount's multi-collateral wallet
type FuturesSubaccountFlexCurrency struct {
	Currency currency.Code `json:"currency"`
	Quantity float64       `json:"quantity"`
	Value    float64       `json:"value"`
	// Collateral is Value after the currency's haircut
	Collateral float64 `json:"collateral"`
	// Available is Quantity less the margin requirement
	Available float64 `json:"available"`
}

// FuturesSubaccountTradingStatusResponse reports whether a subaccount may trade
type FuturesSubaccountTradingStatusResponse struct {
	TradingEnabled bool `json:"tradingEnabled"`
}
