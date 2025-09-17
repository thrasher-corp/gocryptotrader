package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketAPIResponse defines a general websocket response for api calls
type WebsocketAPIResponse struct {
	Header Header          `json:"header"`
	Data   json.RawMessage `json:"data"`
}

// Header defines a websocket header
type Header struct {
	ResponseTime types.Time `json:"response_time"`
	Status       int64      `json:"status,string"`
	Channel      string     `json:"channel"`
	Event        string     `json:"event"`
	ClientID     string     `json:"client_id"`
	ConnectionID string     `json:"conn_id"`
	TraceID      string     `json:"trace_id"`
}

// WebsocketRequest defines a websocket request
type WebsocketRequest struct {
	Time    int64            `json:"time,omitempty"`
	ID      int64            `json:"id,omitempty"`
	Channel string           `json:"channel"`
	Event   string           `json:"event"`
	Payload WebsocketPayload `json:"payload"`
}

// WebsocketPayload defines an individualised websocket payload
type WebsocketPayload struct {
	RequestID string `json:"req_id,omitempty"`
	// APIKey and signature are only required in the initial login request
	// which is done when the connection is established.
	APIKey       string          `json:"api_key,omitempty"`
	Timestamp    string          `json:"timestamp,omitempty"`
	Signature    string          `json:"signature,omitempty"`
	RequestParam json.RawMessage `json:"req_param,omitempty"`
}

// WebsocketErrors defines a websocket error
type WebsocketErrors struct {
	Errors struct {
		Label   string `json:"label"`
		Message string `json:"message"`
	} `json:"errs"`
}

// WebsocketOrderResponse defines a websocket order response
type WebsocketOrderResponse struct {
	Left                      types.Number  `json:"left"`
	UpdateTime                types.Time    `json:"update_time"`
	Amount                    types.Number  `json:"amount"`
	CreateTime                types.Time    `json:"create_time"`
	Price                     types.Number  `json:"price"`
	FinishAs                  string        `json:"finish_as"`
	TimeInForce               string        `json:"time_in_force"`
	CurrencyPair              currency.Pair `json:"currency_pair"`
	Type                      string        `json:"type"`
	Account                   asset.Item    `json:"account"`
	Side                      string        `json:"side"`
	AmendText                 string        `json:"amend_text"`
	Text                      string        `json:"text"`
	Status                    string        `json:"status"`
	Iceberg                   types.Number  `json:"iceberg"`
	FilledTotal               types.Number  `json:"filled_total"`
	ID                        string        `json:"id"`
	FillPrice                 types.Number  `json:"fill_price"`
	UpdateTimeMs              types.Time    `json:"update_time_ms"`
	CreateTimeMs              types.Time    `json:"create_time_ms"`
	Fee                       types.Number  `json:"fee"`
	FeeCurrency               currency.Code `json:"fee_currency"`
	PointFee                  types.Number  `json:"point_fee"`
	GTFee                     types.Number  `json:"gt_fee"`
	GTMakerFee                types.Number  `json:"gt_maker_fee"`
	GTTakerFee                types.Number  `json:"gt_taker_fee"`
	GTDiscount                bool          `json:"gt_discount"`
	RebatedFee                types.Number  `json:"rebated_fee"`
	RebatedFeeCurrency        currency.Code `json:"rebated_fee_currency"`
	SelfTradePreventionID     int           `json:"stp_id"`
	SelfTradePreventionAction string        `json:"stp_act"`
	AverageDealPrice          types.Number  `json:"avg_deal_price"`
	Label                     string        `json:"label"`
	Message                   string        `json:"message"`
}

// WebsocketFuturesOrderResponse defines a websocket futures order response
type WebsocketFuturesOrderResponse struct {
	ID                        int64         `json:"id"`
	User                      int64         `json:"user"`
	CreateTime                types.Time    `json:"create_time"`
	FinishTime                types.Time    `json:"finish_time"`
	FinishAs                  string        `json:"finish_as"`
	Status                    string        `json:"status"`
	Contract                  currency.Pair `json:"contract"`
	Size                      float64       `json:"size"`
	Iceberg                   int64         `json:"iceberg"`
	Price                     types.Number  `json:"price"`
	IsClose                   bool          `json:"is_close"`
	IsReduceOnly              bool          `json:"is_reduce_only"`
	IsForLiquidation          bool          `json:"is_liq"`
	TimeInForce               string        `json:"tif"`
	Left                      float64       `json:"left"`
	FillPrice                 types.Number  `json:"fill_price"`
	Text                      string        `json:"text"`
	TakerFee                  types.Number  `json:"tkfr"`
	MakerFee                  types.Number  `json:"mkfr"`
	ReferenceUserID           int64         `json:"refu"`
	SelfTradePreventionID     int64         `json:"stp_id"`
	SelfTradePreventionAction string        `json:"stp_act"`
	AmendText                 string        `json:"amend_text"`
	BizInfo                   string        `json:"biz_info"`
	UpdateTime                types.Time    `json:"update_time"`
	Succeeded                 *bool         `json:"succeeded"` // Nil if not present in returned response.
}

// WebsocketOrderBatchRequest defines a websocket order batch request
type WebsocketOrderBatchRequest struct {
	OrderID string        `json:"id"` // This requires id tag not order_id
	Pair    currency.Pair `json:"currency_pair"`
	Account string        `json:"account,omitempty"`
}

// WebsocketOrderRequest defines a websocket order request
type WebsocketOrderRequest struct {
	OrderID string `json:"order_id"` // This requires order_id tag
	Pair    string `json:"currency_pair"`
	Account string `json:"account,omitempty"`
}

// WebsocketCancellAllResponse defines a websocket order cancel response
type WebsocketCancellAllResponse struct {
	Pair      currency.Pair `json:"currency_pair"`
	Label     string        `json:"label"`
	Message   string        `json:"message"`
	Succeeded bool          `json:"succeeded"`
}

// WebsocketCancelParam is a struct to hold the parameters for cancelling orders
type WebsocketCancelParam struct {
	Pair    currency.Pair `json:"pair"`
	Side    string        `json:"side"`
	Account string        `json:"account,omitempty"`
}

// WebsocketAmendOrder defines a websocket amend order
type WebsocketAmendOrder struct {
	OrderID   string        `json:"order_id"`
	Pair      currency.Pair `json:"currency_pair"`
	Account   string        `json:"account,omitempty"`
	AmendText string        `json:"amend_text,omitempty"`
	Price     string        `json:"price,omitempty"`
	Amount    string        `json:"amount,omitempty"`
}

// WebsocketFuturesAmendOrder defines a websocket amend order
type WebsocketFuturesAmendOrder struct {
	OrderID   string        `json:"order_id"`
	Contract  currency.Pair `json:"-"` // Only used internally for routing
	Asset     asset.Item    `json:"-"` // Only used internally for routing
	AmendText string        `json:"amend_text,omitempty"`
	Price     string        `json:"price,omitempty"`
	Size      int64         `json:"size,omitempty"`
}

// WebsocketFutureOrdersList defines a websocket future orders list
type WebsocketFutureOrdersList struct {
	Contract currency.Pair `json:"contract,omitzero"`
	Asset    asset.Item    `json:"-"` // Only used internally for routing
	Status   string        `json:"status"`
	Limit    int64         `json:"limit,omitempty"`
	Offset   int64         `json:"offset,omitempty"`
	LastID   string        `json:"last_id,omitempty"`
}
