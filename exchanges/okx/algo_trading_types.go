package okx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CancelAlgoOrderRequest holds the parameters to cancel an algo order
type CancelAlgoOrderRequest struct {
	InstrumentID      string `json:"instId"`
	AlgoID            string `json:"algoId,omitempty"`
	AlgoClientOrderID string `json:"algoClOrdId,omitempty"`
}

// AmendAlgoOrderRequest holds the parameters to amend a stop or trigger algo
// order
type AmendAlgoOrderRequest struct {
	InstrumentID      string  `json:"instId"`
	AlgoID            string  `json:"algoId,omitempty"`
	AlgoClientOrderID string  `json:"algoClOrdId,omitempty"`
	CancelOnFail      bool    `json:"cxlOnFail,omitempty"`
	RequestID         string  `json:"reqId,omitempty"`
	NewSize           float64 `json:"newSz,omitempty,string"`

	// Take profit and stop loss orders; a -1 order price executes at market
	NewTakeProfitTriggerPrice     float64 `json:"newTpTriggerPx,omitempty,string"`
	NewTakeProfitOrderPrice       float64 `json:"newTpOrdPx,omitempty,string"`
	NewStopLossTriggerPrice       float64 `json:"newSlTriggerPx,omitempty,string"`
	NewStopLossOrderPrice         float64 `json:"newSlOrdPx,omitempty,string"`
	NewTakeProfitTriggerPriceType string  `json:"newTpTriggerPxType,omitempty"`
	NewStopLossTriggerPriceType   string  `json:"newSlTriggerPxType,omitempty"`

	// Trigger orders; a -1 order price executes at market
	NewTriggerPrice         float64                         `json:"newTriggerPx,omitempty,string"`
	NewOrderPrice           float64                         `json:"newOrdPx,omitempty,string"`
	NewTriggerPriceType     string                          `json:"newTriggerPxType,omitempty"`
	AttachAlgoOrders        []AmendAttachedAlgoOrderRequest `json:"attachAlgoOrds,omitempty"`
	AdvancedChaseParameters []AmendAdvancedChaseRequest     `json:"advChaseParams,omitempty"`
}

// AmendAdvancedChaseRequest holds the new values of a pending trigger order's
// chase limit order
type AmendAdvancedChaseRequest struct {
	NewChaseValue    float64 `json:"newChaseVal,omitempty,string"`
	NewMaxChaseValue float64 `json:"newMaxChaseVal,omitempty,string"`
}

// AmendAlgoOrderResponse holds the result of amending an algo order
type AmendAlgoOrderResponse struct {
	AlgoID            string `json:"algoId"`
	AlgoClientOrderID string `json:"algoClOrdId"`
	RequestID         string `json:"reqId"`
	StatusCode        uint64 `json:"sCode,string"`
	StatusMessage     string `json:"sMsg"`

	statusCodeMissing bool
}

// UnmarshalJSON decodes an algo order amendment's reply row, noting a row without the sCode stating its execution
// result, whose status code would otherwise read as success
func (a *AmendAlgoOrderResponse) UnmarshalJSON(data []byte) error {
	type row AmendAlgoOrderResponse
	decoded := struct {
		*row
		StatusCode *string `json:"sCode"`
	}{row: (*row)(a)}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var err error
	a.StatusCode, a.statusCodeMissing, err = decodeStatusCode(decoded.StatusCode)
	return err
}

// AlgoOrderDetailRequest holds the parameters to request an algo order's
// details
type AlgoOrderDetailRequest struct {
	AlgoID            string
	AlgoClientOrderID string
}

// AlgoOrderListRequest holds the parameters to list untriggered algo orders
type AlgoOrderListRequest struct {
	OrderType string
	AlgoID    string
	// AlgoClientOrderID is no longer documented, but OKX still filters on it
	AlgoClientOrderID string
	InstrumentType    string
	InstrumentID      string
	After             string
	Before            string
	Limit             uint64
}

// AlgoOrderHistoryRequest holds the parameters to list the algo orders of the
// last 3 months
type AlgoOrderHistoryRequest struct {
	OrderType      string
	State          string
	AlgoID         string
	InstrumentType string
	InstrumentID   string
	After          string
	Before         string
	Limit          uint64
}

// AlgoLinkedOrder represents the linked take-profit order detail of an OCO
// stop-loss order.
type AlgoLinkedOrder struct {
	OrderID string `json:"ordId"`
}

// AlgoOrderResponse holds an algo order's details
type AlgoOrderResponse struct {
	InstrumentType             string                    `json:"instType"`
	InstrumentID               string                    `json:"instId"`
	Currency                   currency.Code             `json:"ccy"`
	OrderID                    string                    `json:"ordId"`
	OrderIDList                []string                  `json:"ordIdList"`
	SubAlgoIDList              []string                  `json:"subAlgoIdList"`
	AlgoID                     string                    `json:"algoId"`
	ClientOrderID              string                    `json:"clOrdId"`
	Size                       types.Number              `json:"sz"`
	CloseFraction              types.Number              `json:"closeFraction"`
	OrderType                  string                    `json:"ordType"`
	Side                       order.Side                `json:"side"`
	PositionSide               string                    `json:"posSide"`
	TradeMode                  string                    `json:"tdMode"`
	TargetCurrency             string                    `json:"tgtCcy"` // base_ccy or quote_ccy
	State                      string                    `json:"state"`
	Leverage                   types.Number              `json:"lever"`
	TakeProfitTriggerPrice     types.Number              `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string                    `json:"tpTriggerPxType"`
	TakeProfitOrderPrice       types.Number              `json:"tpOrdPx"`
	StopLossTriggerPrice       types.Number              `json:"slTriggerPx"`
	StopLossTriggerPriceType   string                    `json:"slTriggerPxType"`
	StopLossOrderPrice         types.Number              `json:"slOrdPx"`
	TriggerPrice               types.Number              `json:"triggerPx"`
	TriggerPriceType           string                    `json:"triggerPxType"`
	OrderPrice                 types.Number              `json:"ordPx"`
	AdvanceOrderType           string                    `json:"advanceOrdType"`
	AdvancedChaseParameters    []AdvancedChaseParameters `json:"advChaseParams"`
	ActualSize                 types.Number              `json:"actualSz"`
	ActualPrice                types.Number              `json:"actualPx"`
	Tag                        string                    `json:"tag"`
	ActualSide                 string                    `json:"actualSide"`
	TriggerTime                types.Time                `json:"triggerTime"`
	PriceVariance              types.Number              `json:"pxVar"`
	PriceSpread                types.Number              `json:"pxSpread"`
	SizeLimit                  types.Number              `json:"szLimit"`
	PriceLimit                 types.Number              `json:"pxLimit"`
	LimitOrderNumber           types.Number              `json:"lmtOrderNumber"`
	Aggressiveness             string                    `json:"aggressiveness"`
	TriggerParameters          []AlgoTriggerRequest      `json:"triggerParams"`
	TimeInterval               types.Number              `json:"timeInterval"`
	CallbackRatio              types.Number              `json:"callbackRatio"`
	CallbackSpread             types.Number              `json:"callbackSpread"`
	ActivePrice                types.Number              `json:"activePx"`
	MoveTriggerPrice           types.Number              `json:"moveTriggerPx"`
	// ReduceOnly is a string because algo order replies send "true", "false" and "", which no bool decoding accepts
	ReduceOnly              string              `json:"reduceOnly"`
	QuickMarginType         string              `json:"quickMgnType"`
	LastPrice               types.Number        `json:"last"`
	FailCode                string              `json:"failCode"`
	AlgoClientOrderID       string              `json:"algoClOrdId"`
	AmendPriceOnTriggerType string              `json:"amendPxOnTriggerType"`
	AttachedAlgoOrders      []AttachedAlgoOrder `json:"attachAlgoOrds"`
	LinkedOrder             AlgoLinkedOrder     `json:"linkedOrd"`
	CreationTime            types.Time          `json:"cTime"`
	UpdateTime              types.Time          `json:"uTime"`
	// IsTradeBorrowMode is a string for the same reason as ReduceOnly
	IsTradeBorrowMode  string        `json:"isTradeBorrowMode"`
	ChaseType          string        `json:"chaseType"`
	ChaseValue         types.Number  `json:"chaseVal"`
	MaxChaseType       string        `json:"maxChaseType"`
	MaxChaseValue      types.Number  `json:"maxChaseVal"`
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy"`
}
