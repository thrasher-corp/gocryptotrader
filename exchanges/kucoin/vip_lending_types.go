package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// VIPLendingCollateralRatioResponse holds a currency's gradient collateral ratios
type VIPLendingCollateralRatioResponse struct {
	Currency   currency.Code                    `json:"currency"`
	USDTLevels []VIPLendingCollateralRatioLevel `json:"usdtLevels"`
}

// VIPLendingCollateralRatioLevel is a gradient interval of a collateral ratio, applying to amounts, converted into
// USDT, above Left and up to Right
type VIPLendingCollateralRatioLevel struct {
	Left         uint64       `json:"left"`
	Right        uint64       `json:"right"`
	DiscountRate types.Number `json:"discountRate"`
}

// VIPLendingLoanInfoResponse holds the account's off-exchange loans, loan-to-value ratios and margins
type VIPLendingLoanInfoResponse struct {
	ParentUID string                `json:"parentUid"`
	Orders    []VIPLendingLoanOrder `json:"orders"`
	LTV       VIPLendingLTV         `json:"ltv"`
	// TotalMarginAmount is in USDT
	TotalMarginAmount types.Number `json:"totalMarginAmount"`
	// TransferMarginAmount is the total maintenance margin for restricted transfers, in USDT
	TransferMarginAmount types.Number       `json:"transferMarginAmount"`
	Margins              []VIPLendingMargin `json:"margins"`
}

// VIPLendingLoanOrder is an off-exchange loan order
type VIPLendingLoanOrder struct {
	OrderID string `json:"orderId"`
	// Principal is the principal still to be repaid
	Principal types.Number `json:"principal"`
	// Interest is the interest still to be repaid
	Interest types.Number  `json:"interest"`
	Currency currency.Code `json:"currency"`
}

// VIPLendingLTV holds the loan-to-value ratios that restrict an off-exchange loan account
type VIPLendingLTV struct {
	// TransferLTV restricts transfers to the funding account
	TransferLTV types.Number `json:"transferLtv"`
	// OnlyClosePositionLTV restricts the account to reducing positions
	OnlyClosePositionLTV  types.Number `json:"onlyClosePosLtv"`
	DelayedLiquidationLTV types.Number `json:"delayedLiquidationLtv"`
	InstantLiquidationLTV types.Number `json:"instantLiquidationLtv"`
	CurrentLTV            types.Number `json:"currentLtv"`
}

// VIPLendingMargin is a currency's margin in an off-exchange loan
type VIPLendingMargin struct {
	MarginCurrency currency.Code `json:"marginCcy"`
	// MarginQuantity is the maintenance quantity, calculated with the margin factor
	MarginQuantity types.Number `json:"marginQty"`
	// MarginFactor is the real-time margin discount rate to USDT
	MarginFactor types.Number `json:"marginFactor"`
}

// VIPLendingAccountResponse is an account involved in off-exchange lending
type VIPLendingAccountResponse struct {
	UID            string        `json:"uid"`
	MarginCurrency currency.Code `json:"marginCcy"`
	// MarginQuantity is the maintenance quantity, calculated with the margin factor
	MarginQuantity types.Number `json:"marginQty"`
	MarginFactor   types.Number `json:"marginFactor"`
	// AccountType is TRADE (spot trading) or CONTRACT (futures, for the total futures equity)
	AccountType string `json:"accountType"`
	IsParent    bool   `json:"isParent"`
}
