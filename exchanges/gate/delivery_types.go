package gate

import (
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// DeliveryContractResponse represents a delivery contract instance detail.
type DeliveryContractResponse struct {
	Name             string       `json:"name"`
	Underlying       string       `json:"underlying"`
	Cycle            string       `json:"cycle"`
	Type             string       `json:"type"`
	QuantoMultiplier types.Number `json:"quanto_multiplier"`
	// Deprecated: Gate has deprecated it
	MarkType            string                `json:"mark_type"`
	LastPrice           types.Number          `json:"last_price"`
	MarkPrice           types.Number          `json:"mark_price"`
	IndexPrice          types.Number          `json:"index_price"`
	BasisRate           types.Number          `json:"basis_rate"`
	BasisValue          types.Number          `json:"basis_value"`
	BasisImpactValue    types.Number          `json:"basis_impact_value"`
	SettlePrice         types.Number          `json:"settle_price"`
	SettlePriceInterval types.DurationSeconds `json:"settle_price_interval"`
	SettlePriceDuration types.DurationSeconds `json:"settle_price_duration"`
	SettleFeeRate       types.Number          `json:"settle_fee_rate"`
	OrderPriceRound     types.Number          `json:"order_price_round"`
	MarkPriceRound      types.Number          `json:"mark_price_round"`
	LeverageMin         types.Number          `json:"leverage_min"`
	LeverageMax         types.Number          `json:"leverage_max"`
	MaintenanceRate     types.Number          `json:"maintenance_rate"`
	RiskLimitBase       types.Number          `json:"risk_limit_base"`
	RiskLimitStep       types.Number          `json:"risk_limit_step"`
	RiskLimitMax        types.Number          `json:"risk_limit_max"`
	MakerFeeRate        types.Number          `json:"maker_fee_rate"`
	TakerFeeRate        types.Number          `json:"taker_fee_rate"`
	RefDiscountRate     types.Number          `json:"ref_discount_rate"`
	RefRebateRate       types.Number          `json:"ref_rebate_rate"`
	OrderPriceDeviate   types.Number          `json:"order_price_deviate"`
	OrderSizeMin        types.Number          `json:"order_size_min"`
	OrderSizeMax        types.Number          `json:"order_size_max"`
	OrdersLimit         uint64                `json:"orders_limit"`
	OrderbookID         uint64                `json:"orderbook_id"`
	TradeID             uint64                `json:"trade_id"`
	TradeSize           types.Number          `json:"trade_size"`
	PositionSize        types.Number          `json:"position_size"`
	ExpireTime          types.Time            `json:"expire_time"`
	ConfigChangeTime    types.Time            `json:"config_change_time"`
	InDelisting         bool                  `json:"in_delisting"`
}

// DeliveryOrderCreateRequest represents delivery order creation parameters.
type DeliveryOrderCreateRequest struct {
	Contract                  currency.Pair `json:"contract"`
	Size                      types.Number  `json:"size"`    // positive long, negative short
	Iceberg                   int64         `json:"iceberg"` // required; can be zero
	Price                     types.Number  `json:"price"`   // NOTE: Market orders require string "0"
	TimeInForce               string        `json:"tif"`
	Text                      string        `json:"text,omitempty"`  // errors when empty; Either populated or omitted
	ClosePosition             bool          `json:"close,omitempty"` // Size needs to be zero if true
	ReduceOnly                bool          `json:"reduce_only,omitempty"`
	AutoSize                  string        `json:"auto_size,omitempty"` // either close_long or close_short, requires zero in size field
	Settle                    currency.Code `json:"-"`                   // Used in URL. REST Calls only.
	SelfTradePreventionAction string        `json:"stp_act,omitempty"`
}

// MarshalJSON sends the size as a JSON integer and the price as a decimal string, since types.Number
// marshals both as strings and zero as "", whereas Gate expects a zero size to close a position and
// a "0" price for a market order
func (c DeliveryOrderCreateRequest) MarshalJSON() ([]byte, error) { //nolint:gocritic // a value receiver keeps the encoding when a value rather than a pointer is marshalled
	type alias DeliveryOrderCreateRequest
	return json.Marshal(struct {
		*alias
		Size  float64 `json:"size"`
		Price string  `json:"price"`
	}{
		alias: (*alias)(&c),
		Size:  c.Size.Float64(),
		Price: strconv.FormatFloat(c.Price.Float64(), 'f', -1, 64),
	})
}

// SettlementHistoryItemResponse represents a settlement history item
type SettlementHistoryItemResponse struct {
	Time        types.Time    `json:"time"`
	Contract    currency.Pair `json:"contract"`
	Size        int64         `json:"size"` // Negative for a short position
	Leverage    types.Number  `json:"leverage"`
	Margin      types.Number  `json:"margin"`
	EntryPrice  types.Number  `json:"entry_price"`
	SettlePrice types.Number  `json:"settle_price"`
	Profit      types.Number  `json:"profit"`
	Fee         types.Number  `json:"fee"`
}
