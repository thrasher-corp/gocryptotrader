package coinbase

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Gateway instrument kinds, which select instruments in requests and identify them in responses
const (
	DerivativesKindFuture      = "future"
	DerivativesKindOption      = "option"
	DerivativesKindSpot        = "spot"
	DerivativesKindFutureCombo = "future_combo"
	DerivativesKindOptionCombo = "option_combo"
	// DerivativesKindCombo selects future and option combos, where a method accepts it
	DerivativesKindCombo = "combo"
	// DerivativesKindAny selects instruments of every kind, where a method accepts it
	DerivativesKindAny = "any"
)

// DerivativesCurrencyAny selects every currency, where a gateway method accepts it
const DerivativesCurrencyAny = "any"

// Gateway order and trade directions
const (
	DerivativesDirectionBuy  = "buy"
	DerivativesDirectionSell = "sell"
)

// Gateway order types, as orders are placed and reported; a trade's order type may also be liquidation
const (
	DerivativesOrderTypeLimit        = "limit"
	DerivativesOrderTypeMarket       = "market"
	DerivativesOrderTypeMarketLimit  = "market_limit"
	DerivativesOrderTypeStopLimit    = "stop_limit"
	DerivativesOrderTypeStopMarket   = "stop_market"
	DerivativesOrderTypeTakeLimit    = "take_limit"
	DerivativesOrderTypeTakeMarket   = "take_market"
	DerivativesOrderTypeTrailingStop = "trailing_stop"
)

// Gateway order times in force; the gateway has no good-til-time
const (
	DerivativesTimeInForceGoodTilCancelled  = "good_til_cancelled"
	DerivativesTimeInForceGoodTilDay        = "good_til_day"
	DerivativesTimeInForceFillOrKill        = "fill_or_kill"
	DerivativesTimeInForceImmediateOrCancel = "immediate_or_cancel"
)

// Gateway trigger order price sources
const (
	DerivativesTriggerIndexPrice = "index_price"
	DerivativesTriggerMarkPrice  = "mark_price"
	DerivativesTriggerLastPrice  = "last_price"
)

// Gateway order states
const (
	DerivativesOrderStateOpen        = "open"
	DerivativesOrderStateFilled      = "filled"
	DerivativesOrderStateRejected    = "rejected"
	DerivativesOrderStateCancelled   = "cancelled"
	DerivativesOrderStateUntriggered = "untriggered"
	DerivativesOrderStateTriggered   = "triggered"
)

// Gateway result sorting
const (
	DerivativesSortingAscending  = "asc"
	DerivativesSortingDescending = "desc"
	DerivativesSortingDefault    = "default"
)

// derivativesMarketPrice is the price the gateway sends in place of a number for an open trigger market order
const derivativesMarketPrice = "market_price"

var (
	errInvalidDerivativesOrderPrice = errors.New("invalid gateway order price")
	errInvalidTuple                 = errors.New("invalid tuple")
)

// DerivativesOrderPrice is a gateway order's price; the gateway sends "market_price" in place of a number for an open
// trigger market order, which decodes as a zero Value with Market set
type DerivativesOrderPrice struct {
	Value  float64
	Market bool
}

// UnmarshalJSON decodes a price number, or the market_price string of an open trigger market order
func (o *DerivativesOrderPrice) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return fmt.Errorf("%w: %w", errInvalidDerivativesOrderPrice, err)
		}
		if text == derivativesMarketPrice {
			*o = DerivativesOrderPrice{Market: true}
			return nil
		}
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return fmt.Errorf("%w: %q", errInvalidDerivativesOrderPrice, text)
		}
		*o = DerivativesOrderPrice{Value: value}
		return nil
	}
	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("%w: %w", errInvalidDerivativesOrderPrice, err)
	}
	*o = DerivativesOrderPrice{Value: value}
	return nil
}

// DerivativesOrder is a gateway order, as trading and account methods and the user.orders notifications report it
type DerivativesOrder struct {
	OrderID string `json:"order_id"`
	// OrderState is open, filled, rejected, cancelled, untriggered or triggered
	OrderState string `json:"order_state"`
	// OrderType is limit, market, stop_limit, stop_market, take_limit, take_market or trailing_stop
	OrderType string `json:"order_type"`
	// OriginalOrderType is the order type placed, market or market_limit, when the gateway holds the order as a limit
	// order
	OriginalOrderType string `json:"original_order_type"`
	// TimeInForce is good_til_cancelled, good_til_day, fill_or_kill or immediate_or_cancel
	TimeInForce string `json:"time_in_force"`
	// IsRebalance is set for a spot order created to restore a cross collateral balance
	IsRebalance bool `json:"is_rebalance"`
	// IsLiquidation is set for an order created by a liquidation; the gateway omits it for spot
	IsLiquidation  bool   `json:"is_liquidation"`
	InstrumentName string `json:"instrument_name"`
	// CreationTimestamp and LastUpdateTimestamp are in milliseconds; StarbaseLastUpdateTimestamp, the Starbase causal
	// time of the last book update affecting a Starbase order, is in nanoseconds
	CreationTimestamp           types.Time            `json:"creation_timestamp"`
	LastUpdateTimestamp         types.Time            `json:"last_update_timestamp"`
	StarbaseLastUpdateTimestamp types.Time            `json:"starbase_last_update_timestamp"`
	StarbaseClientOrderID       string                `json:"starbase_client_order_id"`
	Direction                   string                `json:"direction"`
	Price                       DerivativesOrderPrice `json:"price"`
	// Label is the order's client label, which the gateway does not require to be unique
	Label          string `json:"label"`
	PostOnly       bool   `json:"post_only"`
	RejectPostOnly bool   `json:"reject_post_only"`
	ReduceOnly     bool   `json:"reduce_only"`
	API            bool   `json:"api"`
	Web            bool   `json:"web"`
	Mobile         bool   `json:"mobile"`
	// RefreshAmount is the display amount an iceberg order refreshes to, and DisplayAmount its current display amount
	RefreshAmount float64 `json:"refresh_amount"`
	DisplayAmount float64 `json:"display_amount"`
	// Amount is in the base currency for spot, options and linear instruments such as the USDC perpetuals, and in USD
	// for inverse futures and perpetuals
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
	// Isolated, MainUserID and UserID are set for an order of an isolated margin subaccount, whose user ID is the
	// subaccount's
	Isolated   bool   `json:"isolated"`
	MainUserID uint64 `json:"main_uid"`
	UserID     uint64 `json:"user_id"`
}

// DerivativesSettlement is a settlement, delivery or bankruptcy event
type DerivativesSettlement struct {
	// Type is settlement, delivery or bankruptcy
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

// DerivativesSettlementsResponse is a page of settlement events and the continuation token of the next page
type DerivativesSettlementsResponse struct {
	Settlements  []DerivativesSettlement `json:"settlements"`
	Continuation string                  `json:"continuation"`
}

// derivativesCurrencyParams are the params of a request for a currency
type derivativesCurrencyParams struct {
	Currency string `json:"currency"`
}

// derivativesInstrumentNameParams are the params of a request for an instrument
type derivativesInstrumentNameParams struct {
	InstrumentName string `json:"instrument_name"`
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
