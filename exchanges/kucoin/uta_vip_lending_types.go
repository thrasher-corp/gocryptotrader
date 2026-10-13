package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTAVIPLendingCollateralRatio holds a collateral currency's discount rate tiers
type UTAVIPLendingCollateralRatio struct {
	Currency currency.Code `json:"currency"`
	// USDTLevels are the tiers, by the collateral's value in USDT
	USDTLevels []UTAVIPLendingCollateralLevel `json:"usdtLevels"`
}

// UTAVIPLendingCollateralLevel holds the discount rate for collateral valued above Left and up to Right, in USDT
type UTAVIPLendingCollateralLevel struct {
	Left         uint64       `json:"left"`
	Right        uint64       `json:"right"`
	DiscountRate types.Number `json:"discountRate"`
}

// UTAVIPLendingLoanInfoResponse holds the off-exchange loans of the accounts in a loan and their margin
type UTAVIPLendingLoanInfoResponse struct {
	ParentUID string                   `json:"parentUid"`
	Orders    []UTAVIPLendingLoanOrder `json:"orders"`
	LTV       UTAVIPLendingLTV         `json:"ltv"`
	// TotalMarginAmount and TransferMarginAmount are in USDT; TransferMarginAmount is the maintenance margin that
	// restricts transfers
	TotalMarginAmount    types.Number          `json:"totalMarginAmount"`
	TransferMarginAmount types.Number          `json:"transferMarginAmount"`
	Margins              []UTAVIPLendingMargin `json:"margins"`
}

// UTAVIPLendingLoanOrder holds a loan's outstanding principal and interest
type UTAVIPLendingLoanOrder struct {
	OrderID   string        `json:"orderId"`
	Principal types.Number  `json:"principal"`
	Interest  types.Number  `json:"interest"`
	Currency  currency.Code `json:"currency"`
}

// UTAVIPLendingLTV holds a loan's current loan-to-value ratio and the ratios at which KuCoin restricts and liquidates
type UTAVIPLendingLTV struct {
	// TransferLTV restricts transfers to the funding account
	TransferLTV types.Number `json:"transferLtv"`
	// OnlyClosePositionLTV restricts the accounts to reducing positions
	OnlyClosePositionLTV  types.Number `json:"onlyClosePosLtv"`
	DelayedLiquidationLTV types.Number `json:"delayedLiquidationLtv"`
	InstantLiquidationLTV types.Number `json:"instantLiquidationLtv"`
	CurrentLTV            types.Number `json:"currentLtv"`
}

// UTAVIPLendingMargin holds a margin currency's maintenance quantity
type UTAVIPLendingMargin struct {
	MarginCurrency currency.Code `json:"marginCcy"`
	// MarginQuantity is the quantity after MarginFactor, the currency's real time discount rate to USDT
	MarginQuantity types.Number `json:"marginQty"`
	MarginFactor   types.Number `json:"marginFactor"`
}

// UTAVIPLendingAccount holds an account in an off-exchange loan and its margin
type UTAVIPLendingAccount struct {
	UID            string        `json:"uid"`
	MarginCurrency currency.Code `json:"marginCcy"`
	// MarginQuantity is the quantity after MarginFactor
	MarginQuantity types.Number `json:"marginQty"`
	MarginFactor   types.Number `json:"marginFactor"`
	AccountType    string       `json:"accountType"`
	IsParent       bool         `json:"isParent"`
}
