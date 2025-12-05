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
	ErrorType                int          `json:"error_type"`
}
