package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PMProAccountInfoResponse holds portfolio margin pro account information, with amounts in USD
type PMProAccountInfoResponse struct {
	UniMMR        types.Number `json:"uniMMR"`
	AccountEquity types.Number `json:"accountEquity"`
	ActualEquity  types.Number `json:"actualEquity"`
	// AccountMaintenanceMargin is the portfolio margin account maintenance margin
	AccountMaintenanceMargin types.Number `json:"accountMaintMargin"`
	// AccountInitialMargin and TotalAvailableBalance do not apply to PM PRO and PM PRO SPAN accounts
	AccountInitialMargin types.Number `json:"accountInitialMargin"`
	// TotalAvailableBalance is documented with a thousands separator, which a number type cannot decode
	TotalAvailableBalance string `json:"totalAvailableBalance"`
	AccountStatus         string `json:"accountStatus"`
	// AccountType is PM_1 for PM PRO, PM_2 for PM and PM_3 for PM PRO SPAN
	AccountType string `json:"accountType"`
}

// PMProBankruptcyLoanAmountResponse holds the portfolio margin pro bankruptcy loan amount, which is 0 without a loan
type PMProBankruptcyLoanAmountResponse struct {
	Asset  currency.Code `json:"asset"`
	Amount types.Number  `json:"amount"`
}

// PMProNegativeBalanceInterest holds a portfolio margin pro negative balance interest record
type PMProNegativeBalanceInterest struct {
	Asset               currency.Code `json:"asset"`
	Interest            types.Number  `json:"interest"`
	InterestAccruedTime types.Time    `json:"interestAccruedTime"`
	InterestRate        types.Number  `json:"interestRate"` // daily rate
	Principal           types.Number  `json:"principal"`
}

// PMAssetLeverage holds a portfolio margin asset's leverage
type PMAssetLeverage struct {
	Asset    currency.Code `json:"asset"`
	Leverage uint64        `json:"leverage"`
}

// PMCollateralRate holds a portfolio margin asset's collateral rate
type PMCollateralRate struct {
	Asset          currency.Code `json:"asset"`
	CollateralRate types.Number  `json:"collateralRate"`
}

// PortfolioMarginAssetIndexPrice holds a portfolio margin asset's index price
type PortfolioMarginAssetIndexPrice struct {
	Asset           currency.Code `json:"asset"`
	AssetIndexPrice types.Number  `json:"assetIndexPrice"` // in USD
	Time            types.Time    `json:"time"`
}
