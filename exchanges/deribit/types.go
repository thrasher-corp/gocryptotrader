package deribit

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Instrument kinds, which select instruments in requests and identify them in responses
const (
	KindFuture      = "future"
	KindOption      = "option"
	KindSpot        = "spot"
	KindFutureCombo = "future_combo"
	KindOptionCombo = "option_combo"
	// KindCombo selects future and option combos, where a method accepts it
	KindCombo = "combo"
	// KindAny selects instruments of every kind, where a method accepts it
	KindAny = "any"
)

// Order and trade directions
const (
	DirectionBuy  = "buy"
	DirectionSell = "sell"
)

// Order types, as orders are placed and reported; a trade's order type may also be liquidation
const (
	OrderTypeLimit        = "limit"
	OrderTypeMarket       = "market"
	OrderTypeMarketLimit  = "market_limit"
	OrderTypeStopLimit    = "stop_limit"
	OrderTypeStopMarket   = "stop_market"
	OrderTypeTakeLimit    = "take_limit"
	OrderTypeTakeMarket   = "take_market"
	OrderTypeTrailingStop = "trailing_stop"
)

// marketPrice is the price Deribit sends in place of a number for an open trigger market order
const marketPrice = "market_price"

var (
	errInvalidOrderPrice     = errors.New("invalid order price")
	errInvalidObscuredUserID = errors.New("invalid obscured user ID")
	errInvalidTuple          = errors.New("invalid tuple")
)

// Exchange implements exchange.IBotExchange and provides Deribit API access
type Exchange struct {
	exchange.Base
	// incrementalTickers holds each incremental_ticker instrument's state, every change merged onto
	// it. The ticker store cannot stand in for it: it holds only what a ticker.Price maps, and the
	// data handler's consumer writes it behind the reader, so it can hold an older state
	incrementalTickers    map[string]*WebsocketIncrementalTicker
	incrementalTickersMtx sync.Mutex
	// coinbaseRoutedSpot holds the spot instruments Deribit routes to Coinbase Exchange, whose trades and candles it does
	// not stream, as of the last spot pairs fetched
	coinbaseRoutedSpot    map[string]struct{}
	coinbaseRoutedSpotMtx sync.RWMutex
}

// JSONRPCRequest is the envelope of every request; requests sent over HTTP leave the ID out, as each HTTP response
// answers its own request
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JSONRPCResponse is the envelope of every response; Error is set in place of Result when a request fails
type JSONRPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      int64     `json:"id"`
	Result  any       `json:"result"`
	Error   *APIError `json:"error"`
	Testnet bool      `json:"testnet"`
}

// APIError is the error object Deribit returns in place of a result; its code is negative for JSON-RPC protocol errors
// and positive for Deribit's own
type APIError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	// Data holds the error's details, usually the parameter at fault and the reason, although some errors carry other
	// fields, so it is kept as sent
	Data json.RawMessage `json:"data"`
}

// Error returns the error's code and message, followed by its details when Deribit sent any
func (a *APIError) Error() string {
	msg := "code " + strconv.FormatInt(a.Code, 10) + ": " + a.Message
	if data := bytes.TrimSpace(a.Data); len(data) > 0 && !bytes.Equal(data, []byte("null")) {
		msg += " " + string(data)
	}
	return msg
}

// OrderPrice is an order's price; Deribit sends "market_price" in place of a number for an open trigger market order,
// which decodes as a zero Value with Market set
type OrderPrice struct {
	Value  float64
	Market bool
}

// UnmarshalJSON decodes a price number, or the market_price string of an open trigger market order
func (o *OrderPrice) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return fmt.Errorf("%w: %w", errInvalidOrderPrice, err)
		}
		if text == marketPrice {
			*o = OrderPrice{Market: true}
			return nil
		}
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return fmt.Errorf("%w: %q", errInvalidOrderPrice, text)
		}
		*o = OrderPrice{Value: value}
		return nil
	}
	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("%w: %w", errInvalidOrderPrice, err)
	}
	*o = OrderPrice{Value: value}
	return nil
}

// Order contains an order, as trading and account methods and the user.orders notifications report it
type Order struct {
	OrderID string `json:"order_id"`
	// OrderState is open, filled, rejected, cancelled, untriggered or triggered
	OrderState string `json:"order_state"`
	// OrderType is limit, market, stop_limit, stop_market, take_limit, take_market or trailing_stop
	OrderType string `json:"order_type"`
	// OriginalOrderType is the order type placed, market or market_limit, when Deribit holds the order as a limit order
	OriginalOrderType string `json:"original_order_type"`
	// TimeInForce is good_til_cancelled, good_til_day, fill_or_kill or immediate_or_cancel
	TimeInForce string `json:"time_in_force"`
	// IsRebalance is set for a spot order created to restore a cross collateral balance
	IsRebalance bool `json:"is_rebalance"`
	// IsLiquidation is set for an order created by a liquidation; Deribit omits it for spot
	IsLiquidation  bool   `json:"is_liquidation"`
	InstrumentName string `json:"instrument_name"`
	// CreationTimestamp and LastUpdateTimestamp are in milliseconds; StarbaseLastUpdateTimestamp, the Starbase causal
	// time of the last book update affecting a Starbase order, is in nanoseconds
	CreationTimestamp           types.Time `json:"creation_timestamp"`
	LastUpdateTimestamp         types.Time `json:"last_update_timestamp"`
	StarbaseLastUpdateTimestamp types.Time `json:"starbase_last_update_timestamp"`
	StarbaseClientOrderID       string     `json:"starbase_client_order_id"`
	Direction                   string     `json:"direction"`
	Price                       OrderPrice `json:"price"`
	Label                       string     `json:"label"`
	PostOnly                    bool       `json:"post_only"`
	RejectPostOnly              bool       `json:"reject_post_only"`
	ReduceOnly                  bool       `json:"reduce_only"`
	API                         bool       `json:"api"`
	Web                         bool       `json:"web"`
	Mobile                      bool       `json:"mobile"`
	// RefreshAmount is the display amount an iceberg order refreshes to, and DisplayAmount its current display amount
	RefreshAmount float64 `json:"refresh_amount"`
	DisplayAmount float64 `json:"display_amount"`
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures
	Amount       float64 `json:"amount"`
	Contracts    float64 `json:"contracts"`
	FilledAmount float64 `json:"filled_amount"`
	AveragePrice float64 `json:"average_price"`
	// Advanced is usd or implv for an option order priced in USD or implied volatility
	Advanced          string  `json:"advanced"`
	ImpliedVolatility float64 `json:"implv"`
	USDPrice          float64 `json:"usd"`
	Triggered         bool    `json:"triggered"`
	// Trigger is index_price, mark_price or last_price for a trigger order
	Trigger               string  `json:"trigger"`
	TriggerPrice          float64 `json:"trigger_price"`
	TriggerOffset         float64 `json:"trigger_offset"`
	TriggerReferencePrice float64 `json:"trigger_reference_price"`
	BlockTrade            bool    `json:"block_trade"`
	MMP                   bool    `json:"mmp"`
	RiskReducing          bool    `json:"risk_reducing"`
	Replaced              bool    `json:"replaced"`
	AutoReplaced          bool    `json:"auto_replaced"`
	Quote                 bool    `json:"quote"`
	MMPGroup              string  `json:"mmp_group"`
	QuoteSetID            string  `json:"quote_set_id"`
	QuoteID               string  `json:"quote_id"`
	TriggerOrderID        string  `json:"trigger_order_id"`
	ComboOrderID          string  `json:"combo_order_id"`
	StarbaseOrderID       uint64  `json:"starbase_order_id"`
	AppName               string  `json:"app_name"`
	MMPCancelled          bool    `json:"mmp_cancelled"`
	CancelReason          string  `json:"cancel_reason"`
	// OTOOrderIDs are the orders a fill of this order triggers
	OTOOrderIDs []string `json:"oto_order_ids"`
	// TriggerFillCondition is first_hit, complete_fill or incremental for a linked order
	TriggerFillCondition string `json:"trigger_fill_condition"`
	OCOReference         string `json:"oco_ref"`
	PrimaryOrderID       string `json:"primary_order_id"`
	IsSecondaryOTO       bool   `json:"is_secondary_oto"`
	IsPrimaryOTOCO       bool   `json:"is_primary_otoco"`
}

// UserTrade contains one of the account's trades, as trading, block trade and Block RFQ methods and the user.trades
// notifications report it
type UserTrade struct {
	TradeID string `json:"trade_id"`
	// TradeSequence is the trade's sequence number within its instrument
	TradeSequence  uint64 `json:"trade_seq"`
	InstrumentName string `json:"instrument_name"`
	// Timestamp is in milliseconds, and StarbaseTimestamp, the Starbase causal time of a trade matched in Starbase, in
	// nanoseconds
	Timestamp             types.Time `json:"timestamp"`
	StarbaseTimestamp     types.Time `json:"starbase_timestamp"`
	StarbaseOrderID       uint64     `json:"starbase_order_id"`
	StarbaseClientOrderID string     `json:"starbase_client_order_id"`
	StarbaseMatchID       uint64     `json:"starbase_match_id"`
	// OrderType is limit, market or liquidation
	OrderType         string `json:"order_type"`
	OriginalOrderType string `json:"original_order_type"`
	Advanced          string `json:"advanced"`
	OrderID           string `json:"order_id"`
	Direction         string `json:"direction"`
	// TickDirection is 0 for a plus tick, 1 for a zero plus tick, 2 for a minus tick and 3 for a zero minus tick
	TickDirection uint64  `json:"tick_direction"`
	IndexPrice    float64 `json:"index_price"`
	Price         float64 `json:"price"`
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures
	Amount            float64 `json:"amount"`
	Contracts         float64 `json:"contracts"`
	ImpliedVolatility float64 `json:"iv"`
	UnderlyingPrice   float64 `json:"underlying_price"`
	// Liquidation is M, T or MT when the maker, the taker or both were being liquidated
	Liquidation string `json:"liquidation"`
	// Liquidity is M when the account's order was the maker and T when it was the taker
	Liquidity   string        `json:"liquidity"`
	Fee         float64       `json:"fee"`
	FeeCurrency currency.Code `json:"fee_currency"`
	Label       string        `json:"label"`
	// State is the order's state: open, filled, rejected, cancelled, untriggered or archive
	State              string `json:"state"`
	BlockTradeID       string `json:"block_trade_id"`
	BlockTradeLegCount uint64 `json:"block_trade_leg_count"`
	BlockRFQID         uint64 `json:"block_rfq_id"`
	BlockRFQQuoteID    uint64 `json:"block_rfq_quote_id"`
	// ReduceOnly and PostOnly are documented as strings, so either form is accepted
	ReduceOnly   types.Boolean `json:"reduce_only"`
	PostOnly     types.Boolean `json:"post_only"`
	MMP          bool          `json:"mmp"`
	RiskReducing bool          `json:"risk_reducing"`
	SelfTrade    bool          `json:"self_trade"`
	API          bool          `json:"api"`
	ProfitLoss   float64       `json:"profit_loss"`
	MarkPrice    float64       `json:"mark_price"`
	// Legs holds a combo trade's leg trades, which Deribit includes only when combo trades alone are requested
	Legs             []UserTrade       `json:"legs"`
	ComboID          string            `json:"combo_id"`
	ComboTradeID     string            `json:"combo_trade_id"`
	QuoteSetID       string            `json:"quote_set_id"`
	QuoteID          string            `json:"quote_id"`
	TradeAllocations []TradeAllocation `json:"trade_allocations"`
}

// ObscuredUserID is a user ID Deribit obscures for brokers, such as ***009; Deribit documents it as a string or an
// integer, so either is accepted
type ObscuredUserID string

// UnmarshalJSON decodes an obscured user ID from a string or an integer
func (o *ObscuredUserID) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var userID string
		if err := json.Unmarshal(data, &userID); err != nil {
			return fmt.Errorf("%w: %w", errInvalidObscuredUserID, err)
		}
		*o = ObscuredUserID(userID)
		return nil
	}
	var userID uint64
	if err := json.Unmarshal(data, &userID); err != nil {
		return fmt.Errorf("%w: %w", errInvalidObscuredUserID, err)
	}
	*o = ObscuredUserID(strconv.FormatUint(userID, 10))
	return nil
}

// TradeAllocation contains the part of a Block RFQ trade allocated to one user
type TradeAllocation struct {
	// UserID is obscured for brokers
	UserID     ObscuredUserID             `json:"user_id"`
	Amount     float64                    `json:"amount"`
	Fee        float64                    `json:"fee"`
	ClientInfo *TradeAllocationClientInfo `json:"client_info"`
}

// TradeAllocationClientInfo identifies the broker client a trade allocation belongs to
type TradeAllocationClientInfo struct {
	ClientID     uint64 `json:"client_id"`
	ClientLinkID uint64 `json:"client_link_id"`
	Name         string `json:"name"`
}

// BlockTrade contains a block trade and the account's trades in it
type BlockTrade struct {
	ID         string      `json:"id"`
	Timestamp  types.Time  `json:"timestamp"`
	Trades     []UserTrade `json:"trades"`
	AppName    string      `json:"app_name"`
	BrokerCode string      `json:"broker_code"`
	BrokerName string      `json:"broker_name"`
}

// SettlementsResponse contains a page of settlement events and the continuation token of the next page
type SettlementsResponse struct {
	Settlements  []Settlement `json:"settlements"`
	Continuation string       `json:"continuation"`
}

// Settlement contains a settlement, delivery, exercise or bankruptcy event
type Settlement struct {
	// Type is settlement, delivery, exercise or bankruptcy; Deribit's documentation omits exercise, which it sends for
	// expired options
	Type           string     `json:"type"`
	Timestamp      types.Time `json:"timestamp"`
	InstrumentName string     `json:"instrument_name"`
	// Funding is in the base currency, and only perpetual settlements carry it
	Funding    float64 `json:"funding"`
	Funded     float64 `json:"funded"`
	IndexPrice float64 `json:"index_price"`
	MarkPrice  float64 `json:"mark_price"`
	Position   float64 `json:"position"`
	// ProfitLoss and SessionProfitLoss aggregate every position holder's profit and loss, in the base currency
	ProfitLoss        float64 `json:"profit_loss"`
	SessionProfitLoss float64 `json:"session_profit_loss"`
	SessionBankruptcy float64 `json:"session_bankruptcy"`
	SessionTax        float64 `json:"session_tax"`
	SessionTaxRate    float64 `json:"session_tax_rate"`
	Socialised        float64 `json:"socialized"`
}

// currencyParams contains the currency of a request
type currencyParams struct {
	Currency currency.Code `json:"currency"`
}

// instrumentNameParams contains the instrument of a request
type instrumentNameParams struct {
	InstrumentName string `json:"instrument_name"`
}

// orderIDParams contains the order of a request
type orderIDParams struct {
	OrderID string `json:"order_id"`
}

// unmarshalTuple decodes a JSON array into exactly as many targets as are supplied; null leaves the targets unchanged,
// as the json package does
func unmarshalTuple(data []byte, targets ...any) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	var elements []json.RawMessage
	if err := json.Unmarshal(data, &elements); err != nil {
		return fmt.Errorf("%w %s: %w", errInvalidTuple, data, err)
	}
	if len(elements) != len(targets) {
		return fmt.Errorf("%w %s: expected %d elements, got %d", errInvalidTuple, data, len(targets), len(elements))
	}
	for i := range elements {
		if err := json.Unmarshal(elements[i], targets[i]); err != nil {
			return fmt.Errorf("%w %s: element %d: %w", errInvalidTuple, data, i, err)
		}
	}
	return nil
}
