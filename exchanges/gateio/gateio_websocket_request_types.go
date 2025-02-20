package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
	Status       string     `json:"status"`
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

// WebsocketOrder defines a websocket order
type WebsocketOrder struct {
	Text         string `json:"text"`
	CurrencyPair string `json:"currency_pair,omitempty"`
	Type         string `json:"type,omitempty"`
	Account      string `json:"account,omitempty"`
	Side         string `json:"side,omitempty"`
	Amount       string `json:"amount,omitempty"`
	Price        string `json:"price,omitempty"`
	TimeInForce  string `json:"time_in_force,omitempty"`
	Iceberg      string `json:"iceberg,omitempty"`
	AutoBorrow   bool   `json:"auto_borrow,omitempty"`
	AutoRepay    bool   `json:"auto_repay,omitempty"`
	StpAct       string `json:"stp_act,omitempty"`
}

// WebsocketOrderResponse defines a websocket order response
type WebsocketOrderResponse struct {
	Left               types.Number  `json:"left"`
	UpdateTime         types.Time    `json:"update_time"`
	Amount             types.Number  `json:"amount"`
	CreateTime         types.Time    `json:"create_time"`
	Price              types.Number  `json:"price"`
	FinishAs           string        `json:"finish_as"`
	TimeInForce        string        `json:"time_in_force"`
	CurrencyPair       currency.Pair `json:"currency_pair"`
	Type               string        `json:"type"`
	Account            string        `json:"account"`
	Side               string        `json:"side"`
	AmendText          string        `json:"amend_text"`
	Text               string        `json:"text"`
	Status             string        `json:"status"`
	Iceberg            types.Number  `json:"iceberg"`
	FilledTotal        types.Number  `json:"filled_total"`
	ID                 string        `json:"id"`
	FillPrice          types.Number  `json:"fill_price"`
	UpdateTimeMs       types.Time    `json:"update_time_ms"`
	CreateTimeMs       types.Time    `json:"create_time_ms"`
	Fee                types.Number  `json:"fee"`
	FeeCurrency        currency.Code `json:"fee_currency"`
	PointFee           types.Number  `json:"point_fee"`
	GTFee              types.Number  `json:"gt_fee"`
	GTMakerFee         types.Number  `json:"gt_maker_fee"`
	GTTakerFee         types.Number  `json:"gt_taker_fee"`
	GTDiscount         bool          `json:"gt_discount"`
	RebatedFee         types.Number  `json:"rebated_fee"`
	RebatedFeeCurrency currency.Code `json:"rebated_fee_currency"`
	STPID              int           `json:"stp_id"`
	STPAct             string        `json:"stp_act"`
}

// WebsocketOrderBatchRequest defines a websocket order batch request
type WebsocketOrderBatchRequest struct {
	OrderID string        `json:"id"` // This require id tag not order_id
	Pair    currency.Pair `json:"currency_pair"`
	Account string        `json:"account,omitempty"`
}

// WebsocketOrderRequest defines a websocket order request
type WebsocketOrderRequest struct {
	OrderID string `json:"order_id"` // This requires order_id tag
	Pair    string `json:"pair"`
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
