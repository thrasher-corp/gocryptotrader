package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CrossMarginSymbolsResponse holds the cross margin symbol configurations
type CrossMarginSymbolsResponse struct {
	Timestamp types.Time          `json:"timestamp"`
	Items     []CrossMarginSymbol `json:"items"`
}

// CrossMarginSymbol is a cross margin symbol's trading configuration
type CrossMarginSymbol struct {
	Symbol        string        `json:"symbol"`
	Name          string        `json:"name"`
	EnableTrading bool          `json:"enableTrading"`
	Market        string        `json:"market"`
	BaseCurrency  currency.Code `json:"baseCurrency"`
	QuoteCurrency currency.Code `json:"quoteCurrency"`
	BaseIncrement types.Number  `json:"baseIncrement"`
	BaseMinSize   types.Number  `json:"baseMinSize"`
	// QuoteIncrement and QuoteMinSize apply to the funds of a market order
	QuoteIncrement types.Number  `json:"quoteIncrement"`
	QuoteMinSize   types.Number  `json:"quoteMinSize"`
	BaseMaxSize    types.Number  `json:"baseMaxSize"`
	QuoteMaxSize   types.Number  `json:"quoteMaxSize"`
	PriceIncrement types.Number  `json:"priceIncrement"`
	FeeCurrency    currency.Code `json:"feeCurrency"`
	// PriceLimitRate is the price protection threshold
	PriceLimitRate types.Number `json:"priceLimitRate"`
	MinFunds       types.Number `json:"minFunds"`
}

// IsolatedMarginSymbolResponse is an isolated margin symbol's configuration
type IsolatedMarginSymbolResponse struct {
	Symbol        string        `json:"symbol"`
	SymbolName    string        `json:"symbolName"`
	BaseCurrency  currency.Code `json:"baseCurrency"`
	QuoteCurrency currency.Code `json:"quoteCurrency"`
	MaxLeverage   uint64        `json:"maxLeverage"`
	// ForcedLiquidationDebtRatio is the debt ratio at which the position is liquidated
	ForcedLiquidationDebtRatio types.Number `json:"flDebtRatio"`
	TradeEnable                bool         `json:"tradeEnable"`
	// AutoRenewMaxDebtRatio is the debt ratio below which loans are renewed automatically
	AutoRenewMaxDebtRatio  types.Number `json:"autoRenewMaxDebtRatio"`
	BaseBorrowEnable       bool         `json:"baseBorrowEnable"`
	QuoteBorrowEnable      bool         `json:"quoteBorrowEnable"`
	BaseTransferInEnable   bool         `json:"baseTransferInEnable"`
	QuoteTransferInEnable  bool         `json:"quoteTransferInEnable"`
	BaseBorrowCoefficient  types.Number `json:"baseBorrowCoefficient"`
	QuoteBorrowCoefficient types.Number `json:"quoteBorrowCoefficient"`
}

// MarginMarkPriceResponse is a margin symbol's current mark price
type MarginMarkPriceResponse struct {
	Symbol string `json:"symbol"`
	// Granularity is the mark price's update interval in milliseconds; KuCoin sends it although the docs omit it
	Granularity uint64     `json:"granularity"`
	TimePoint   types.Time `json:"timePoint"`
	Value       float64    `json:"value"`
}

// MarginConfigurationResponse holds the cross margin configuration
type MarginConfigurationResponse struct {
	CurrencyList []currency.Code `json:"currencyList"`
	MaxLeverage  uint64          `json:"maxLeverage"`
	// WarningDebtRatio is the debt ratio at which KuCoin warns of forced liquidation
	WarningDebtRatio types.Number `json:"warningDebtRatio"`
	// LiquidationDebtRatio is the debt ratio at which positions are liquidated
	LiquidationDebtRatio types.Number `json:"liqDebtRatio"`
}

// MarginCollateralRatioResponse holds the collateral ratio tiers shared by a group of currencies
type MarginCollateralRatioResponse struct {
	CurrencyList []currency.Code             `json:"currencyList"`
	Items        []MarginCollateralRatioItem `json:"items"`
}

// MarginCollateralRatioItem is the collateral ratio of a margin range
type MarginCollateralRatioItem struct {
	LowerLimit      types.Number `json:"lowerLimit"`
	UpperLimit      types.Number `json:"upperLimit"`
	CollateralRatio types.Number `json:"collateralRatio"`
}

// MarketAvailableInventoryResponse is the amount of a currency the platform has available to lend
type MarketAvailableInventoryResponse struct {
	Currency         currency.Code `json:"currency"`
	BorrowableAmount types.Number  `json:"borrowableAmount"`
}
