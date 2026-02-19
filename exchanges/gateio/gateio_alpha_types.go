package gateio

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// alphaStatusError implements an interface with a method Error()
type alphaStatusError struct {
	Label   string `json:"Label"`
	Message string `json:"Message"`
}

func (a *alphaStatusError) Error() error {
	if a.Label != "" {
		return fmt.Errorf("label: %s message: %s", a.Label, a.Message)
	}
	return nil
}

// AlphaAccount represents an alpha account information
type AlphaAccount struct {
	Currency     currency.Code `json:"currency"`
	Available    types.Number  `json:"available"`
	Locked       types.Number  `json:"locked"`
	TokenAddress string        `json:"token_address"`
	Chain        string        `json:"chain"`
}

// AlphaAccountTransactionItem holds alpha account transaction item
type AlphaAccountTransactionItem struct {
	ID       string        `json:"id"`
	Time     types.Time    `json:"time"`
	Currency currency.Code `json:"currency"`
	Change   string        `json:"change"`
	Balance  types.Number  `json:"balance"`
}

// AlphaCurrencyQuoteInfoRequest represents a currency quote information request parameters
type AlphaCurrencyQuoteInfoRequest struct {
	Currency currency.Code `json:"currency"`
	Side     order.Side    `json:"side"`
	Amount   float64       `json:"amount,string"`
	GasMode  string        `json:"gas_mode"`
	Slippage float64       `json:"slippage,omitempty,string"`
	QuoteID  string        `json:"quote_id,omitempty"`
}

// AlphaCurrencyQuoteDetail holds a currency quote information detail
type AlphaCurrencyQuoteDetail struct {
	alphaStatusError
	QuoteID                  string       `json:"quote_id"`
	MinAmount                types.Number `json:"min_amount"`
	MaxAmount                types.Number `json:"max_amount"`
	Price                    types.Number `json:"price"`
	Slippage                 types.Number `json:"slippage"`
	EstimateGasFeeAmountUSDT types.Number `json:"estimate_gas_fee_amount_usdt"`
	OrderFee                 types.Number `json:"order_fee"`
	TargetTokenMinAmount     types.Number `json:"target_token_min_amount"`
	TargetTokenMaxAmount     types.Number `json:"target_token_max_amount"`
	ErrorType                int64        `json:"error_type"`
}

// AlphaPlaceOrderResponse represents response details returned after placing alpha orders
type AlphaPlaceOrderResponse struct {
	alphaStatusError
	OrderID      string       `json:"order_id"`
	Status       int64        `json:"status"`
	Side         string       `json:"side"`
	GasMode      string       `json:"gas_mode"`
	CreateTime   types.Time   `json:"create_time"`
	Amount       types.Number `json:"amount"`
	TokenAddress string       `json:"token_address"`
	Chain        string       `json:"chain"`
}

// AlphaOrderDetail holds a alpha order details
type AlphaOrderDetail struct {
	alphaStatusError
	OrderID         string        `json:"order_id"`
	TransactionHash string        `json:"tx_hash"`
	Side            string        `json:"side"`
	USDTAmount      types.Number  `json:"usdt_amount"`
	Currency        currency.Code `json:"currency"`
	CurrencyAmount  types.Number  `json:"currency_amount"`
	Status          int64         `json:"status"`
	GasMode         string        `json:"gas_mode"`
	Chain           string        `json:"chain"`
	GasFee          types.Number  `json:"gas_fee"`
	TransactionFee  string        `json:"transaction_fee"`
	CreateTime      types.Time    `json:"create_time"`
	FailedReason    string        `json:"failed_reason"`
}

// AlphaCurrencyDetail holds an alpha currency detail
type AlphaCurrencyDetail struct {
	Currency        currency.Code `json:"currency"`
	Name            string        `json:"name"`
	Chain           string        `json:"chain"`
	Address         string        `json:"address"`
	Status          int32         `json:"status"`
	Precision       int32         `json:"precision"`
	AmountPrecision int32         `json:"amount_precision"`
}

// AlphaCurrencyTickerInfo represents an alpha currency ticker detail
type AlphaCurrencyTickerInfo struct {
	Currency  currency.Code `json:"currency"`
	Change    string        `json:"change"`
	Last      types.Number  `json:"last"`
	Volume    types.Number  `json:"volume"`
	MarketCap string        `json:"market_cap"`
}
