package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CustomerToCustomerTradeHistoryRequest holds the parameters of Get C2C Trade History
type CustomerToCustomerTradeHistoryRequest struct {
	// TradeType is BUY or SELL; both are returned when it is empty
	TradeType string
	// StartTime and EndTime are at most 30 days apart
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	// Rows is at most 100, the default
	Rows uint64
}

// CustomerToCustomerTradeHistoryResponse holds a page of C2C trades; Code "000000" means success
type CustomerToCustomerTradeHistoryResponse struct {
	Code    string                    `json:"code"`
	Message string                    `json:"message"`
	Data    []CustomerToCustomerTrade `json:"data"`
	Total   uint64                    `json:"total"`
	Success bool                      `json:"success"`
}

// CustomerToCustomerTrade is a C2C trade
type CustomerToCustomerTrade struct {
	OrderNumber         string        `json:"orderNumber"`
	AdvertisementNumber string        `json:"advNo"`
	TradeType           string        `json:"tradeType"`
	Asset               currency.Code `json:"asset"`
	Fiat                currency.Code `json:"fiat"`
	FiatSymbol          string        `json:"fiatSymbol"`
	AmountInCrypto      types.Number  `json:"amount"`
	TotalPriceInFiat    types.Number  `json:"totalPrice"`
	UnitPriceInFiat     types.Number  `json:"unitPrice"`
	OrderStatus         string        `json:"orderStatus"`
	CreateTime          types.Time    `json:"createTime"`
	CommissionInCrypto  types.Number  `json:"commission"`
	CounterPartNickName string        `json:"counterPartNickName"`
	PayMethodName       string        `json:"payMethodName"`
	// AdditionalKYCVerify is 0 not required, 1 not verified or 2 verified
	AdditionalKYCVerify uint64       `json:"additionalKycVerify"`
	TakerCommissionRate types.Number `json:"takerCommissionRate"`
	TakerCommission     types.Number `json:"takerCommission"`
	TakerAmount         types.Number `json:"takerAmount"`
	AdvertisementRole   string       `json:"advertisementRole"`
}
