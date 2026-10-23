package gate

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// alphaStatusError holds the error label and message embedded in alpha endpoint responses
type alphaStatusError struct {
	Label   string `json:"Label"`
	Message string `json:"Message"`
}

// AsError returns the embedded alpha status as an error, or nil when the response carries no error label
func (a *alphaStatusError) AsError() error {
	if a.Label != "" {
		return fmt.Errorf("label: %s message: %s", a.Label, a.Message)
	}
	return nil
}

// AlphaAccountResponse represents an alpha account information
type AlphaAccountResponse struct {
	Currency     string       `json:"currency"` // Case sensitive, such as memeboxTUT, which Gate no longer finds once currency.Code upper-cases it
	Available    types.Number `json:"available"`
	Locked       types.Number `json:"locked"`
	TokenAddress string       `json:"token_address"`
	Chain        string       `json:"chain"`
}

// AlphaAccountTransactionItemResponse holds alpha account transaction item
type AlphaAccountTransactionItemResponse struct {
	ID       string       `json:"id"`
	Time     types.Time   `json:"time"`
	Currency string       `json:"currency"` // Case sensitive, such as memeboxTUT, which Gate no longer finds once currency.Code upper-cases it
	Change   types.Number `json:"change"`
	Balance  types.Number `json:"balance"`
}

// AlphaCurrencyQuoteInfoRequest represents a currency quote information request parameters
type AlphaCurrencyQuoteInfoRequest struct {
	Currency string       `json:"currency"` // Case sensitive, such as memeboxTUT, which Gate no longer finds once currency.Code upper-cases it
	Side     order.Side   `json:"side"`
	Amount   types.Number `json:"amount"`   // Quote currency (USDT) amount for a buy and base currency amount for a sell
	GasMode  string       `json:"gas_mode"` // Trading mode affects slippage selection: "speed" uses smart mode; "custom" uses the slippage parameter.
	Slippage types.Number `json:"slippage,omitempty"`
	QuoteID  string       `json:"quote_id,omitempty"`
}

// MarshalJSON lowercases the order side, which order.Side otherwise marshals as "BUY"/"SELL"
// while Gate only accepts "buy"/"sell"
func (a *AlphaCurrencyQuoteInfoRequest) MarshalJSON() ([]byte, error) {
	type Alias AlphaCurrencyQuoteInfoRequest
	return json.Marshal(&struct {
		Side string `json:"side"`
		*Alias
	}{
		Side:  a.Side.Lower(),
		Alias: (*Alias)(a),
	})
}

// AlphaCurrencyQuoteDetailResponse holds a currency quote information detail
type AlphaCurrencyQuoteDetailResponse struct {
	alphaStatusError
	QuoteID                  string        `json:"quote_id"`
	MinAmount                types.Number  `json:"min_amount"`
	MaxAmount                string        `json:"max_amount"` // Gate documents a non-numeric form, such as 1000:0.0
	Price                    types.Number  `json:"price"`
	Slippage                 types.Number  `json:"slippage"`
	TargetTokenMinAmount     types.Number  `json:"target_token_min_amount"`
	TargetTokenMaxAmount     types.Number  `json:"target_token_max_amount"`
	ErrorType                uint64        `json:"error_type"` // 0 success, 1 above the maximum or 2 below the minimum
	EstimateGasFeeAmountUSDT DisplayNumber `json:"estimate_gas_fee_amount_usdt"`
	OrderFee                 DisplayNumber `json:"order_fee"`
}

// AlphaPlaceOrderResponse represents response details returned after placing alpha orders
type AlphaPlaceOrderResponse struct {
	alphaStatusError
	OrderID      string       `json:"order_id"`
	Status       uint8        `json:"status"`
	Side         string       `json:"side"`
	GasMode      string       `json:"gas_mode"` // Trading mode affects slippage selection: "speed" uses smart mode; "custom" uses the slippage parameter.
	CreateTime   types.Time   `json:"create_time"`
	Amount       types.Number `json:"amount"`
	TokenAddress string       `json:"token_address"`
	Chain        string       `json:"chain"`
}

// GetAlphaOrdersRequest holds the optional filters for listing alpha orders
type GetAlphaOrdersRequest struct {
	Currency string     // Case sensitive, such as memeboxTUT, which Gate no longer finds once currency.Code upper-cases it
	Side     order.Side // Buy or Sell
	// Status is 1 processing, 2 successful, 3 failed, 4 cancelled, 5 buy order placed but transfer not completed or
	// 6 order cancelled but transfer not completed, with 0 returning every status
	Status    uint8
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	Limit     uint64
}

// AlphaOrderDetailResponse holds a alpha order details
type AlphaOrderDetailResponse struct {
	alphaStatusError
	OrderID         string       `json:"order_id"`
	TransactionHash string       `json:"tx_hash"`
	Side            string       `json:"side"`
	USDTAmount      types.Number `json:"usdt_amount"`
	Currency        string       `json:"currency"` // Case sensitive, such as memeboxTUT, which Gate no longer finds once currency.Code upper-cases it
	CurrencyAmount  types.Number `json:"currency_amount"`
	Status          uint8        `json:"status"`
	GasMode         string       `json:"gas_mode"` // Trading mode affects slippage selection: "speed" uses smart mode; "custom" uses the slippage parameter.
	Chain           string       `json:"chain"`
	GasFee          types.Number `json:"gas_fee"`
	TransactionFee  types.Number `json:"transaction_fee"`
	CreateTime      types.Time   `json:"create_time"`
	FailedReason    string       `json:"failed_reason"`
}

// AlphaCurrencyDetailResponse holds an alpha currency detail
type AlphaCurrencyDetailResponse struct {
	Currency        string `json:"currency"` // Case sensitive, such as memeboxTUT, which Gate no longer finds once currency.Code upper-cases it
	Name            string `json:"name"`
	Chain           string `json:"chain"`
	Address         string `json:"address"`
	Status          uint8  `json:"status"` // 1 trading, 2 suspended or 3 delisted
	Precision       uint8  `json:"precision"`
	AmountPrecision uint8  `json:"amount_precision"`
}

// AlphaCurrencyTickerInfoResponse represents an alpha currency ticker detail
type AlphaCurrencyTickerInfoResponse struct {
	Currency               string       `json:"currency"` // Case sensitive, such as memeboxTUT, which Gate no longer finds once currency.Code upper-cases it
	PercentageChange24Hour types.Number `json:"change"`
	Last                   types.Number `json:"last"`
	USDTVolume24Hour       types.Number `json:"volume"`
	MarketCap              types.Number `json:"market_cap"`
}
