package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// LeadTraderStatusResponse holds whether the account is a futures lead trader; Code "000000" means success
type LeadTraderStatusResponse struct {
	Code    string           `json:"code"`
	Message string           `json:"message"`
	Data    LeadTraderStatus `json:"data"`
	Success bool             `json:"success"`
}

// LeadTraderStatus is an account's futures lead trader status
type LeadTraderStatus struct {
	IsLeadTrader bool       `json:"isLeadTrader"`
	Time         types.Time `json:"time"`
}

// LeadTradingSymbolWhitelistResponse holds the symbols a futures lead trader may trade; Code "000000" means success
type LeadTradingSymbolWhitelistResponse struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Data    []LeadTradingSymbol `json:"data"`
}

// LeadTradingSymbol is a symbol a futures lead trader may trade
type LeadTradingSymbol struct {
	Symbol     string        `json:"symbol"`
	BaseAsset  currency.Code `json:"baseAsset"`
	QuoteAsset currency.Code `json:"quoteAsset"`
}
