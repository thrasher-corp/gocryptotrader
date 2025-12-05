package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

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
	OrderID         string       `json:"order_id"`
	TransactionHash string       `json:"tx_hash"`
	Side            string       `json:"side"`
	USDTAmount      types.Number `json:"usdt_amount"`
	Currency        string       `json:"currency"`
	CurrencyAmount  types.Number `json:"currency_amount"`
	Status          int64        `json:"status"`
	GasMode         string       `json:"gas_mode"`
	Chain           string       `json:"chain"`
	GasFee          types.Number `json:"gas_fee"`
	TransactionFee  string       `json:"transaction_fee"`
	CreateTime      types.Time   `json:"create_time"`
	FailedReason    string       `json:"failed_reason"`
}
