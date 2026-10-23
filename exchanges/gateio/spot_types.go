package gateio

import (
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// serverTimeReply holds Gate's server time
type serverTimeReply struct {
	ServerTime types.Time `json:"server_time"`
}

// CurrencyInfoResponse represents currency details with permission.
type CurrencyInfoResponse struct {
	Currency currency.Code `json:"currency"`
	Name     string        `json:"name"`
	Delisted bool          `json:"delisted"`
	// Deprecated: use the per chain flags in Chains
	WithdrawDisabled bool `json:"withdraw_disabled"`
	// Deprecated: use the per chain flags in Chains
	WithdrawDelayed bool `json:"withdraw_delayed"`
	// Deprecated: use the per chain flags in Chains
	DepositDisabled bool                 `json:"deposit_disabled"`
	TradeDisabled   bool                 `json:"trade_disabled"`
	FixedFeeRate    types.Number         `json:"fixed_rate,omitempty"`
	Chain           string               `json:"chain"`
	Chains          []*SpotCurrencyChain `json:"chains"`
	TotalSupply     string               `json:"total_supply"` // Gate reports ∞ for currencies without a supply cap
	MarketCap       types.Number         `json:"market_cap"`
	Category        []string             `json:"category"`
}

// SpotCurrencyChain holds the deposit and withdrawal status of a currency on one of its chains
type SpotCurrencyChain struct {
	Name             string `json:"name"`
	Address          string `json:"addr"`
	WithdrawDisabled bool   `json:"withdraw_disabled"`
	WithdrawDelayed  bool   `json:"withdraw_delayed"`
	DepositDisabled  bool   `json:"deposit_disabled"`
}

// CurrencyPairDetailResponse represents a single currency pair detail.
type CurrencyPairDetailResponse struct {
	ID          currency.Pair   `json:"id"`
	Base        currency.Code   `json:"base"`
	BaseName    string          `json:"base_name"`
	Quote       currency.Code   `json:"quote"`
	QuoteName   string          `json:"quote_name"`
	TradeQuotes []currency.Code `json:"trade_quotes"` // Quote currencies accepted by a unified market; empty when unsupported
	// Deprecated: Gate has deprecated the pair's trading fee
	Fee                           types.Number `json:"fee"`
	MinBaseAmount                 types.Number `json:"min_base_amount"`
	MinQuoteAmount                types.Number `json:"min_quote_amount"`
	MaxBaseAmount                 types.Number `json:"max_base_amount"`
	MaxQuoteAmount                types.Number `json:"max_quote_amount"`
	AmountPrecision               uint8        `json:"amount_precision"`
	PricePrecision                uint8        `json:"precision"`
	TradeStatus                   string       `json:"trade_status"` // e.g. "untradable", "buyable", "sellable", "tradable"
	SellStart                     types.Time   `json:"sell_start"`
	BuyStart                      types.Time   `json:"buy_start"`
	DelistingTime                 types.Time   `json:"delisting_time"`
	Type                          string       `json:"type"` // normal or premarket
	TradeURL                      string       `json:"trade_url"`
	STTag                         bool         `json:"st_tag"`
	MaximumQuoteRisePercentage    types.Number `json:"up_rate"`
	MaximumQuoteDeclinePercentage types.Number `json:"down_rate"`
	Slippage                      types.Number `json:"slippage"`
	MarketOrderMaxStock           types.Number `json:"market_order_max_stock"`
	MarketOrderMaxMoney           types.Number `json:"market_order_max_money"`
}

// SpotOrderbookResponse holds orderbook ask and bid datas.
type SpotOrderbookResponse struct {
	ID           int64                            `json:"id"` // int64 to match the order book API it feeds
	ResponseTime types.Time                       `json:"current"`
	UpdateTime   types.Time                       `json:"update"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
}

// MakeOrderbook converts SpotOrderbookResponse into an OrderbookResponse
func (a *SpotOrderbookResponse) MakeOrderbook() *OrderbookResponse {
	return &OrderbookResponse{ID: a.ID, ResponseTime: a.ResponseTime, UpdateTime: a.UpdateTime, Asks: OrderbookLevels(a.Asks.Levels()), Bids: OrderbookLevels(a.Bids.Levels())}
}

// TradeResponse represents market trade.
type TradeResponse struct {
	ID                string        `json:"id"`
	CreateTimeSeconds types.Time    `json:"create_time"`
	CreateTime        types.Time    `json:"create_time_ms"`
	CurrencyPair      string        `json:"currency_pair"`
	OrderID           string        `json:"order_id"`
	Side              string        `json:"side"`
	Role              string        `json:"role"`
	Amount            types.Number  `json:"amount"`
	Price             types.Number  `json:"price"`
	Fee               types.Number  `json:"fee"`
	FeeCurrency       currency.Code `json:"fee_currency"`
	PointFee          types.Number  `json:"point_fee"`
	GTFee             types.Number  `json:"gt_fee"`
	AmendText         string        `json:"amend_text"`
	SequenceID        string        `json:"sequence_id"`
	Text              string        `json:"text"`
	Deal              types.Number  `json:"deal"`
	TradeQuote        currency.Code `json:"trade_quote"`
}

// CandlestickResponse represents candlestick data point detail.
type CandlestickResponse struct {
	Timestamp           types.Time
	QuoteCurrencyVolume types.Number
	ClosePrice          types.Number
	HighestPrice        types.Number
	LowestPrice         types.Number
	OpenPrice           types.Number
	BaseCurrencyAmount  types.Number
	WindowClosed        bool
}

// UnmarshalJSON parses kline data from a JSON array into CandlestickResponse fields.
func (c *CandlestickResponse) UnmarshalJSON(data []byte) error {
	var windowClosed string
	err := json.Unmarshal(data, &[8]any{&c.Timestamp, &c.QuoteCurrencyVolume, &c.ClosePrice, &c.HighestPrice, &c.LowestPrice, &c.OpenPrice, &c.BaseCurrencyAmount, &windowClosed})
	if err != nil {
		return err
	}
	c.WindowClosed, err = strconv.ParseBool(windowClosed)
	return err
}

// SpotTradingFeeRateResponse holds the account's spot fee rates for a currency pair
type SpotTradingFeeRateResponse struct {
	CurrencyPair                   currency.Pair `json:"currency_pair"`
	UserID                         uint64        `json:"user_id"`
	TakerFee                       types.Number  `json:"taker_fee"`
	MakerFee                       types.Number  `json:"maker_fee"`
	RetailPriceImprovementMakerFee types.Number  `json:"rpi_maker_fee"`
	GTDiscount                     bool          `json:"gt_discount"`
	GTTakerFee                     types.Number  `json:"gt_taker_fee"`
	GTMakerFee                     types.Number  `json:"gt_maker_fee"`
	LoanFee                        types.Number  `json:"loan_fee"`
	PointType                      string        `json:"point_type"`
	DebitFee                       uint64        `json:"debit_fee"` // 1 GT deduction, 2 point card deduction, 3 VIP rates
	RPIMMLevel                     uint64        `json:"rpi_mm"`
}

// SpotAccountResponse represents spot account
type SpotAccountResponse struct {
	Currency  currency.Code `json:"currency"`
	Available types.Number  `json:"available"`
	Locked    types.Number  `json:"locked"`
	UpdateID  uint64        `json:"update_id"`
}

// GetSpotAccountBookRequest holds the filters for retrieving the spot account book
type GetSpotAccountBookRequest struct {
	Currency currency.Code
	From     time.Time
	To       time.Time
	Page     uint64
	Limit    uint64
	Type     string
	Code     string // Account change code, which takes priority over Type
}

// SpotAccountBookResponse holds a single spot account balance change
type SpotAccountBookResponse struct {
	ID       string        `json:"id"`
	Time     types.Time    `json:"time"`
	Currency currency.Code `json:"currency"`
	Change   types.Number  `json:"change"`
	Balance  types.Number  `json:"balance"`
	// Deprecated: use Code
	Type string `json:"type"`
	Code string `json:"code"`
	Text string `json:"text"`
}

// CreateOrderRequest represents a single order creation param.
type CreateOrderRequest struct {
	Text                      string        `json:"text,omitempty"`
	CurrencyPair              currency.Pair `json:"currency_pair,omitzero"`
	TradeQuote                currency.Code `json:"trade_quote,omitzero"` // Quote currency actually traded; unified markets only
	Type                      string        `json:"type,omitempty"`
	Account                   asset.Item    `json:"account,omitempty"`
	Side                      string        `json:"side,omitempty"`
	Iceberg                   string        `json:"iceberg,omitempty"`
	Amount                    types.Number  `json:"amount,omitempty"`
	Price                     types.Number  `json:"price,omitempty"`
	TimeInForce               string        `json:"time_in_force,omitempty"`
	AutoBorrow                bool          `json:"auto_borrow,omitempty"`
	AutoRepay                 bool          `json:"auto_repay,omitempty"`
	SelfTradePreventionAction string        `json:"stp_act,omitempty"`
	// ActionMode specifies the processing mode for an order request, determining the fields returned in the response.
	// Valid only during the request and omitted from the response. Options:
	// - ACK: Asynchronous mode, returns only key order fields
	// - RESULT: Excludes clearing information
	// - FULL: Full mode (default)
	ActionMode string `json:"action_mode,omitempty"`
	// Slippage caps a market order's price deviation from the latest price as a ratio, e.g. 0.03 for 3%
	Slippage   types.Number   `json:"slippage,omitempty"`
	StopProfit *SpotOrderTPSL `json:"stop_profit,omitempty"`
	StopLoss   *SpotOrderTPSL `json:"stop_loss,omitempty"`
}

// SpotOrderTPSL holds the take profit or stop loss attached to a limit order
// When amending an order, an empty non-nil value cancels it while nil leaves it unchanged
type SpotOrderTPSL struct {
	TriggerPrice types.Number `json:"trigger_price,omitempty"`
	OrderPrice   types.Number `json:"order_price,omitempty"`
}

// SpotOrderResponse holds a spot order, or one order's result within a batch or cancel all response
type SpotOrderResponse struct {
	OrderID           string        `json:"id,omitempty"`
	BatchOrderID      string        `json:"order_id,omitempty"` // Only returned by batch create and amend responses
	Text              string        `json:"text,omitempty"`
	AmendText         string        `json:"amend_text,omitempty"`
	Succeeded         bool          `json:"succeeded"`
	ErrorLabel        string        `json:"label,omitempty"`
	Message           string        `json:"message,omitempty"`
	CreateTimeSeconds types.Time    `json:"create_time,omitzero"`
	UpdateTimeSeconds types.Time    `json:"update_time,omitzero"`
	CreateTime        types.Time    `json:"create_time_ms,omitzero"`
	UpdateTime        types.Time    `json:"update_time_ms,omitzero"`
	CurrencyPair      string        `json:"currency_pair,omitempty"`
	TradeQuote        currency.Code `json:"trade_quote,omitzero"`
	Status            string        `json:"status,omitempty"`
	Type              string        `json:"type,omitempty"`
	Account           string        `json:"account,omitempty"`
	Side              string        `json:"side,omitempty"`
	Amount            types.Number  `json:"amount,omitempty"`
	Price             types.Number  `json:"price,omitempty"`
	TimeInForce       string        `json:"time_in_force,omitempty"`
	Iceberg           types.Number  `json:"iceberg,omitempty"`
	AutoRepay         bool          `json:"auto_repay"`
	AutoBorrow        bool          `json:"auto_borrow"`
	RemainingAmount   types.Number  `json:"left"`
	FilledAmount      types.Number  `json:"filled_amount"`
	AverageFillPrice  types.Number  `json:"avg_deal_price"`
	FeeDeducted       types.Number  `json:"fee"`
	FeeCurrency       currency.Code `json:"fee_currency"`
	// Deprecated: use FilledTotal
	FillPrice          types.Number   `json:"fill_price"`
	FilledTotal        types.Number   `json:"filled_total"` // Total filled in quote currency
	PointFee           types.Number   `json:"point_fee"`
	GTFee              types.Number   `json:"gt_fee,omitempty"`
	GTDiscount         bool           `json:"gt_discount"`
	GTMakerFee         types.Number   `json:"gt_maker_fee"`
	GTTakerFee         types.Number   `json:"gt_taker_fee"`
	RebatedFee         types.Number   `json:"rebated_fee"`
	RebatedFeeCurrency currency.Code  `json:"rebated_fee_currency"`
	STPID              uint64         `json:"stp_id"`
	STPAction          string         `json:"stp_act"`
	FinishAs           string         `json:"finish_as"`
	Slippage           types.Number   `json:"slippage"`
	StopProfit         *SpotOrderTPSL `json:"stop_profit"`
	StopLoss           *SpotOrderTPSL `json:"stop_loss"`
}

// SpotOrdersDetailResponse represents list of orders for specific currency pair
type SpotOrdersDetailResponse struct {
	CurrencyPair currency.Pair        `json:"currency_pair"`
	Total        uint64               `json:"total"`
	Orders       []*SpotOrderResponse `json:"orders"`
}

// GetSpotOrdersRequest holds the filters for listing spot orders
type GetSpotOrdersRequest struct {
	CurrencyPair currency.Pair // Required for open orders
	Status       string        // open or finished
	Page         uint64
	Limit        uint64     // At most 100 for open orders
	Account      string     // spot, margin, cross_margin or unified
	From         time.Time  // Finished orders only, matched against the order finish time
	To           time.Time  // Finished orders only, matched against the order finish time
	Side         order.Side // Finished orders only
}

// ClosePositionRequest holds an order closing a position whose cross-currency trading is disabled
type ClosePositionRequest struct {
	Text         string        `json:"text,omitempty"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Amount       types.Number  `json:"amount"`
	Price        types.Number  `json:"price"`
	ActionMode   string        `json:"action_mode,omitempty"` // ACK, RESULT or FULL, which is the default
}

// CancelAllSpotOrdersRequest holds the filters for cancelling all open spot orders
type CancelAllSpotOrdersRequest struct {
	CurrencyPair currency.Pair // Every currency pair when empty
	Side         order.Side    // Both sides when unset
	Account      string        // spot, margin, cross_margin or unified; unified accounts must set unified
	TradeQuote   currency.Code // Quote currency to cancel in a unified market; every quote currency when empty
	ActionMode   string        // ACK, RESULT or FULL, which is the default
}

// CancelOrderByIDRequest represents cancel order by id request param.
type CancelOrderByIDRequest struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	ID           string        `json:"id"`
	Account      string        `json:"account,omitempty"`     // Must be unified when cancelling with a unified account API key
	ActionMode   string        `json:"action_mode,omitempty"` // ACK, RESULT or FULL, which is the default
}

// CancelOrderByIDResponse holds the result of cancelling one order by ID
type CancelOrderByIDResponse struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	OrderID      string        `json:"id"`
	Text         string        `json:"text"`
	Succeeded    bool          `json:"succeeded"`
	Label        string        `json:"label"`
	Message      string        `json:"message"`
	// Deprecated: Gate has deprecated it
	Account string `json:"account"`
}

// AmendSpotOrderRequest holds the changes to an open spot order; either Amount or Price must be set
type AmendSpotOrderRequest struct {
	CurrencyPair currency.Pair  `json:"currency_pair"`
	Account      string         `json:"account,omitempty"` // spot, margin, cross_margin or unified
	Amount       types.Number   `json:"amount,omitempty"`
	Price        types.Number   `json:"price,omitempty"`
	AmendText    string         `json:"amend_text,omitempty"`
	ActionMode   string         `json:"action_mode,omitempty"` // ACK, RESULT or FULL, which is the default
	StopProfit   *SpotOrderTPSL `json:"stop_profit,omitempty"`
	StopLoss     *SpotOrderTPSL `json:"stop_loss,omitempty"`
}

// BatchAmendSpotOrderRequest holds the changes to one order of a batch amendment
type BatchAmendSpotOrderRequest struct {
	OrderID string `json:"order_id"` // Order ID or custom text ID
	AmendSpotOrderRequest
}

// GetMySpotTradingHistoryRequest holds the filters for retrieving the account's spot trades
type GetMySpotTradingHistoryRequest struct {
	CurrencyPair currency.Pair // Required when OrderID is set
	OrderID      string
	Page         uint64
	Limit        uint64 // At most 1000
	From         time.Time
	To           time.Time
}

// SpotPersonalTradeHistoryResponse represents personal trading history.
type SpotPersonalTradeHistoryResponse struct {
	TradeID           string        `json:"id"`
	CreateTimeSeconds types.Time    `json:"create_time"`
	CreateTime        types.Time    `json:"create_time_ms"`
	CurrencyPair      currency.Pair `json:"currency_pair"`
	OrderID           string        `json:"order_id"`
	Side              string        `json:"side"`
	Role              string        `json:"role"`
	Amount            types.Number  `json:"amount"`
	Price             types.Number  `json:"price"`
	Fee               types.Number  `json:"fee"`
	FeeCurrency       currency.Code `json:"fee_currency"`
	PointFee          types.Number  `json:"point_fee"`
	GTFee             types.Number  `json:"gt_fee"`
	AmendText         string        `json:"amend_text"`
	SequenceID        string        `json:"sequence_id"`
	Text              string        `json:"text"`
	Deal              types.Number  `json:"deal"`
	TradeQuote        currency.Code `json:"trade_quote"`
}

// CountdownCancelOrderRequest represents countdown cancel order params
type CountdownCancelOrderRequest struct {
	CurrencyPair currency.Pair         `json:"currency_pair,omitzero"` // Every currency pair when empty
	Countdown    types.DurationSeconds `json:"timeout"`                // Whole seconds of at least 5; zero stops the countdown
}

// GetSpotInsuranceHistoryRequest holds the filters for retrieving spot insurance fund history
type GetSpotInsuranceHistoryRequest struct {
	Business string // margin or unified
	Currency currency.Code
	From     time.Time
	To       time.Time
	Page     uint64
	Limit    uint64
}

// SpotInsuranceHistoryResponse holds an insurance fund balance at a point in time
type SpotInsuranceHistoryResponse struct {
	Currency currency.Code `json:"currency"`
	Balance  types.Number  `json:"balance"`
	Time     types.Time    `json:"time"`
}

// PriceTriggeredOrderRequest represents price triggered order request.
type PriceTriggeredOrderRequest struct {
	Trigger TriggerPriceInfo `json:"trigger"`
	Put     PutOrder         `json:"put"`
	Market  currency.Pair    `json:"market"`
}

// TriggerPriceInfo represents a trigger price and related information for Price triggered order
type TriggerPriceInfo struct {
	Price       types.Number          `json:"price"`
	Rule        string                `json:"rule"`
	ExpireAfter types.DurationSeconds `json:"expiration,omitempty"` // Whole seconds to wait for the trigger before cancelling
}

// PutOrder represents order detail for price triggered order request
type PutOrder struct {
	Type        string       `json:"type,omitempty"` // limit, the default, or market
	Side        string       `json:"side"`
	Price       types.Number `json:"price"`
	Amount      types.Number `json:"amount"`
	Account     string       `json:"account"` // normal, margin or unified
	TimeInForce string       `json:"time_in_force"`
	AutoBorrow  bool         `json:"auto_borrow,omitempty"`
	AutoRepay   bool         `json:"auto_repay,omitempty"`
	Text        string       `json:"text,omitempty"`
}

// SpotPriceTriggeredOrderResponse represents spot price triggered order response data.
type SpotPriceTriggeredOrderResponse struct {
	Trigger      TriggerPriceInfo `json:"trigger"`
	Put          PutOrder         `json:"put"`
	AutoOrderID  uint64           `json:"id"`
	UserID       uint64           `json:"user"`
	CreationTime types.Time       `json:"ctime"`
	FireTime     types.Time       `json:"ftime"`
	FiredOrderID uint64           `json:"fired_order_id"`
	Status       string           `json:"status"`
	Reason       string           `json:"reason"`
	Market       currency.Pair    `json:"market"`
}

// SpotPOVOrderRequest holds a spot percentage of volume (POV) order
type SpotPOVOrderRequest struct {
	CurrencyPair                currency.Pair `json:"currency_pair"`
	Side                        string        `json:"side"`
	Amount                      types.Number  `json:"amount"`
	ParticipationRatePercentage uint64        `json:"participation_rate"` // 5, 10, 20 or 40
	TimeToLive                  string        `json:"ttl"`                // 1h, 6h, 12h, 1d, 2d, 3d, 4d, 5d, 6d or 7d
	LimitPrice                  types.Number  `json:"limit_price,omitempty"`
	TriggerPrice                types.Number  `json:"trigger_price,omitempty"`
	Text                        string        `json:"text,omitempty"`
}

// GetSpotPOVOrdersRequest holds the filters for listing spot POV orders
type GetSpotPOVOrdersRequest struct {
	CurrencyPair currency.Pair
	Status       string // open or finished
	Side         order.Side
	Page         uint64 // At most 100
	Limit        uint64
}

// SpotPOVOrderResponse holds a spot POV order
type SpotPOVOrderResponse struct {
	ID                          string        `json:"id"`
	CurrencyPair                currency.Pair `json:"currency_pair"`
	Side                        string        `json:"side"`
	Amount                      types.Number  `json:"amount"`
	ParticipationRatePercentage uint64        `json:"participation_rate"`
	TimeToLive                  string        `json:"ttl"`
	LimitPrice                  types.Number  `json:"limit_price"`
	TriggerPrice                types.Number  `json:"trigger_price"`
	Status                      string        `json:"status"`
	TerminatedAs                string        `json:"terminated_as"`
	StartTime                   types.Time    `json:"start_time_ms"`
	EndTime                     types.Time    `json:"end_time_ms"`
	ExpireTime                  types.Time    `json:"expire_time_ms"`
	CreateTime                  types.Time    `json:"create_time_ms"`
	UpdateTime                  types.Time    `json:"update_time_ms"`
	Text                        string        `json:"text"`
}
