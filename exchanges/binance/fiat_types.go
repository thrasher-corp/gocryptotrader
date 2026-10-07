package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FiatHistoryRequest holds the parameters of the fiat deposit/withdrawal and payment history queries
type FiatHistoryRequest struct {
	// TransactionType is 0 for deposits or buys and 1 for withdrawals or sells; it is always sent
	TransactionType uint64
	BeginTime       time.Time
	EndTime         time.Time
	Page            uint64
	// Rows is at most 500 and defaults to 100
	Rows uint64
}

// FiatOrdersResponse holds a page of fiat deposits or withdrawals; Code "000000" means success
type FiatOrdersResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    []FiatOrder `json:"data"`
	Total   uint64      `json:"total"`
	Success bool        `json:"success"`
}

// FiatOrder is a fiat deposit or withdrawal
type FiatOrder struct {
	OrderNumber     string        `json:"orderNo"`
	FiatCurrency    currency.Code `json:"fiatCurrency"`
	IndicatedAmount types.Number  `json:"indicatedAmount"`
	Amount          types.Number  `json:"amount"`
	TotalFee        types.Number  `json:"totalFee"`
	Method          string        `json:"method"`
	// Status is Processing, Failed, Successful, Finished, Refunding, Refunded, Refund Failed or Order Partial Credit
	// Stopped
	Status     string     `json:"status"`
	CreateTime types.Time `json:"createTime"`
	UpdateTime types.Time `json:"updateTime"`
}

// FiatPaymentsResponse holds a page of fiat payments for crypto; Code "000000" means success
type FiatPaymentsResponse struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Data    []FiatPayment `json:"data"`
	Total   uint64        `json:"total"`
	Success bool          `json:"success"`
}

// FiatPayment is a crypto buy or sell paid in fiat
type FiatPayment struct {
	OrderNumber    string        `json:"orderNo"`
	SourceAmount   types.Number  `json:"sourceAmount"`
	FiatCurrency   currency.Code `json:"fiatCurrency"`
	ObtainAmount   types.Number  `json:"obtainAmount"`
	CryptoCurrency currency.Code `json:"cryptoCurrency"`
	TotalFee       types.Number  `json:"totalFee"`
	Price          types.Number  `json:"price"`
	Status         string        `json:"status"`
	// PaymentMethod is returned for buys only: Cash Balance, Credit Card, Online Banking or Bank Transfer
	PaymentMethod string     `json:"paymentMethod"`
	CreateTime    types.Time `json:"createTime"`
	UpdateTime    types.Time `json:"updateTime"`
}
