package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// C2CTradeHistoryRequest holds the parameters of Get C2C Trade History
type C2CTradeHistoryRequest struct {
	// TradeType is BUY or SELL; both are returned when it is empty
	TradeType string
	// StartTime and EndTime are at most 30 days apart
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	// Rows is at most 100, the default
	Rows uint64
}

// C2CTradeHistoryResponse holds a page of C2C trades; Code "000000" means success
type C2CTradeHistoryResponse struct {
	Code    string     `json:"code"`
	Message string     `json:"message"`
	Data    []C2CTrade `json:"data"`
	Total   uint64     `json:"total"`
	Success bool       `json:"success"`
}

// C2CTrade is a C2C trade
type C2CTrade struct {
	OrderNumber         string        `json:"orderNumber"`
	AdvertisementNumber string        `json:"advNo"`
	TradeType           string        `json:"tradeType"`
	Asset               currency.Code `json:"asset"`
	Fiat                currency.Code `json:"fiat"`
	FiatSymbol          string        `json:"fiatSymbol"`
	Amount              types.Number  `json:"amount"`     // In crypto
	TotalPrice          types.Number  `json:"totalPrice"` // In fiat
	UnitPrice           types.Number  `json:"unitPrice"`  // In fiat
	OrderStatus         string        `json:"orderStatus"`
	CreateTime          types.Time    `json:"createTime"`
	Commission          types.Number  `json:"commission"` // In crypto
	CounterPartNickName string        `json:"counterPartNickName"`
	PayMethodName       string        `json:"payMethodName"`
	// AdditionalKYCVerify is 0 not required, 1 not verified or 2 verified
	AdditionalKYCVerify uint64       `json:"additionalKycVerify"`
	TakerCommissionRate types.Number `json:"takerCommissionRate"`
	TakerCommission     types.Number `json:"takerCommission"`
	TakerAmount         types.Number `json:"takerAmount"`
	AdvertisementRole   string       `json:"advertisementRole"`
}
