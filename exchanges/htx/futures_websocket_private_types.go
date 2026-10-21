package htx

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

type wsFuturesAuthRequest struct {
	Operation        string `json:"op"`
	AuthType         string `json:"type"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
}

type wsFuturesSubscriptionRequest struct {
	Operation string `json:"op"`
	Topic     string `json:"topic"`
}

type wsV5FuturesSubscriptionRequest struct {
	Operation    string `json:"op"`
	Topic        string `json:"topic"`
	ContractCode string `json:"contract_code,omitempty"`
}

// wsFuturesTimestamp validates the heartbeat while preserving its JSON representation for the reply.
type wsFuturesTimestamp string

// UnmarshalJSON accepts HTX's numeric and quoted Unix timestamps.
func (t *wsFuturesTimestamp) UnmarshalJSON(data []byte) error {
	var parsed types.Time
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	if parsed.Time().IsZero() {
		return fmt.Errorf("%w: missing futures websocket timestamp", common.ErrParsingWSField)
	}
	*t = wsFuturesTimestamp(data)
	return nil
}

// MarshalJSON echoes the validated timestamp without converting its JSON type.
func (t wsFuturesTimestamp) MarshalJSON() ([]byte, error) {
	return []byte(t), nil
}

type wsFuturesPong struct {
	Operation string             `json:"op"`
	Timestamp wsFuturesTimestamp `json:"ts"`
}

// legacyFuturesWSOrder holds the common order fields shared by delivery and coin-margined notifications.
type legacyFuturesWSOrder struct {
	asset          asset.Item
	contractCode   string
	direction      string
	orderPriceType string
	feeAsset       currency.Code
	status         uint64
	orderID        uint64
	clientOrderID  uint64
	orderType      uint64
	orderIDString  string
	volume         float64
	price          float64
	tradeVolume    float64
	tradeTurnover  float64
	averagePrice   float64
	trades         []LegacyFuturesWSTrade
	fee            float64
	leverage       float64
	createdAt      types.Time
	cancelledAt    types.Time
	reduceOnly     bool
}

// FuturesWebsocketSubOrder stores subscribed order data for futures websocket
type FuturesWebsocketSubOrder struct {
	Operation      string                 `json:"op"`
	Topic          string                 `json:"topic"`
	UID            string                 `json:"uid"`
	Timestamp      types.Time             `json:"ts"`
	Symbol         string                 `json:"symbol"`
	ContractType   string                 `json:"contract_type"`
	ContractCode   string                 `json:"contract_code"`
	Volume         float64                `json:"volume"`
	Price          float64                `json:"price"`
	OrderPriceType string                 `json:"order_price_type"`
	Direction      string                 `json:"direction"`
	Offset         string                 `json:"offset"`
	Status         uint64                 `json:"status"`
	LeverageRate   uint64                 `json:"lever_rate"`
	OrderID        uint64                 `json:"order_id"`
	OrderIDString  string                 `json:"order_id_str"`
	ClientOrderID  uint64                 `json:"client_order_id"`
	OrderSource    string                 `json:"order_source"`
	OrderType      uint64                 `json:"order_type"`
	CreatedAt      types.Time             `json:"created_at"`
	TradeVolume    float64                `json:"trade_volume"`
	TradeTurnover  float64                `json:"trade_turnover"`
	Fee            float64                `json:"fee"`
	TradeAvgPrice  float64                `json:"trade_avg_price"`
	MarginFrozen   float64                `json:"margin_frozen"`
	Profit         float64                `json:"profit"`
	FeeAsset       currency.Code          `json:"fee_asset"`
	CancelledAt    types.Time             `json:"canceled_at"`
	Trade          []LegacyFuturesWSTrade `json:"trade"`
}

// FuturesWebsocketSubMatchOrder stores subscribed match order data for futures websocket
type FuturesWebsocketSubMatchOrder struct {
	Operation      string                 `json:"op"`
	Topic          string                 `json:"topic"`
	UID            string                 `json:"uid"`
	Timestamp      types.Time             `json:"ts"`
	Symbol         string                 `json:"symbol"`
	ContractType   string                 `json:"contract_type"`
	ContractCode   string                 `json:"contract_code"`
	Status         uint64                 `json:"status"`
	OrderID        uint64                 `json:"order_id"`
	OrderIDString  string                 `json:"order_id_str"`
	OrderType      uint64                 `json:"order_type"`
	Direction      string                 `json:"direction"`
	Offset         string                 `json:"offset"`
	OrderPriceType string                 `json:"order_price_type"`
	Price          float64                `json:"price"`
	LeverageRate   uint64                 `json:"lever_rate"`
	CreatedAt      types.Time             `json:"created_at"`
	Volume         float64                `json:"volume"`
	TradeVolume    float64                `json:"trade_volume"`
	ClientOrderID  uint64                 `json:"client_order_id"`
	Trade          []LegacyFuturesWSTrade `json:"trade"`
}

// FuturesWebsocketSubEquityUpdates stores account equity updates data for futures websocket
type FuturesWebsocketSubEquityUpdates struct {
	Operation string                   `json:"op"`
	Topic     string                   `json:"topic"`
	UID       string                   `json:"uid"`
	Timestamp types.Time               `json:"ts"`
	Event     string                   `json:"event"`
	Data      []FuturesWebsocketEquity `json:"data"`
}

// FuturesWebsocketSubPositionUpdates stores subscribed position updates data for futures websocket
type FuturesWebsocketSubPositionUpdates struct {
	Operation     string                     `json:"op"`
	Topic         string                     `json:"topic"`
	UID           string                     `json:"uid"`
	Timestamp     types.Time                 `json:"ts"`
	Event         string                     `json:"event"`
	PositionsData []FuturesWebsocketPosition `json:"data"`
}

// FuturesWebsocketSubTriggerOrderUpdates stores subscribed trigger order updates data for futures websocket
type FuturesWebsocketSubTriggerOrderUpdates struct {
	Operation string                         `json:"op"`
	Topic     string                         `json:"topic"`
	UID       string                         `json:"uid"`
	Event     string                         `json:"event"`
	Data      []FuturesWebsocketTriggerOrder `json:"data"`
}

// LegacyFuturesWSTrade contains one fill in a delivery or coin-margined private notification.
type LegacyFuturesWSTrade struct {
	ID            string        `json:"id"`
	TradeID       uint64        `json:"trade_id"`
	TradeVolume   float64       `json:"trade_volume"`
	TradePrice    float64       `json:"trade_price"`
	TradeFee      float64       `json:"trade_fee"`
	TradeTurnover float64       `json:"trade_turnover"`
	CreatedAt     types.Time    `json:"created_at"`
	Role          string        `json:"role"`
	FeeAsset      currency.Code `json:"fee_asset"`
}

// FuturesWebsocketEquity reports balance, margin and profit changes for a delivery settlement currency.
type FuturesWebsocketEquity struct {
	Symbol            string  `json:"symbol"`
	MarginBalance     float64 `json:"margin_balance"`
	MarginStatic      float64 `json:"margin_static"`
	MarginPosition    float64 `json:"margin_position"`
	MarginFrozen      float64 `json:"margin_frozen"`
	MarginAvailable   float64 `json:"margin_available"`
	ProfitReal        float64 `json:"profit_real"`
	ProfitUnreal      float64 `json:"profit_unreal"`
	WithdrawAvailable float64 `json:"withdraw_available"`
	RiskRate          float64 `json:"risk_rate"`
	LiquidationPrice  float64 `json:"liquidation_price"`
	LeverageRate      float64 `json:"lever_rate"`
	AdjustFactor      float64 `json:"adjust_factor"`
}

// FuturesWebsocketPosition reports the held and available delivery contracts for one position direction.
type FuturesWebsocketPosition struct {
	Symbol         string  `json:"symbol"`
	ContractCode   string  `json:"contract_code"`
	ContractType   string  `json:"contract_type"`
	Volume         float64 `json:"volume"`
	Available      float64 `json:"available"`
	Frozen         float64 `json:"frozen"`
	CostOpen       float64 `json:"cost_open"`
	CostHold       float64 `json:"cost_hold"`
	ProfitUnreal   float64 `json:"profit_unreal"`
	ProfitRate     float64 `json:"profit_rate"`
	Profit         float64 `json:"profit"`
	PositionMargin float64 `json:"position_margin"`
	LeverageRate   float64 `json:"lever_rate"`
	Direction      string  `json:"direction"`
	LastPrice      float64 `json:"last_price"`
}

// FuturesWebsocketTriggerOrder reports trigger execution or cancellation, including any resulting order identifier.
type FuturesWebsocketTriggerOrder struct {
	Symbol          string     `json:"symbol"`
	ContractCode    string     `json:"contract_code"`
	ContractType    string     `json:"contract_type"`
	TriggerType     string     `json:"trigger_type"`
	Volume          float64    `json:"volume"`
	OrderType       uint64     `json:"order_type"`
	Direction       string     `json:"direction"`
	Offset          string     `json:"offset"`
	LeverageRate    uint64     `json:"lever_rate"`
	OrderID         uint64     `json:"order_id"`
	OrderIDString   string     `json:"order_id_str"`
	RelationOrderID string     `json:"relation_order_id"`
	OrderPriceType  string     `json:"order_price_type"`
	Status          uint64     `json:"status"`
	OrderSource     string     `json:"order_source"`
	TriggerPrice    float64    `json:"trigger_price"`
	TriggeredPrice  float64    `json:"triggered_price"`
	OrderPrice      float64    `json:"order_price"`
	CreatedAt       types.Time `json:"created_at"`
	TriggeredAt     types.Time `json:"triggered_at"`
	OrderInsertAt   types.Time `json:"order_insert_at"`
	CancelledAt     types.Time `json:"canceled_at"`
	FailCode        int64      `json:"fail_code"`
	FailReason      string     `json:"fail_reason"`
}
