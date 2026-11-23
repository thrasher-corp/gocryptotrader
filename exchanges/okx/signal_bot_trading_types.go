package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CreateSignalRequest holds a signal channel to create
type CreateSignalRequest struct {
	SignalChannelName        string `json:"signalChanName"`
	SignalChannelDescription string `json:"signalChanDesc,omitempty"`
}

// CreateSignalResponse holds a created signal channel
type CreateSignalResponse struct {
	SignalChannelID    string `json:"signalChanId"`
	SignalChannelToken string `json:"signalChanToken"`
}

// SignalsRequest holds the filters for listing signal channels
type SignalsRequest struct {
	SignalSourceType string
	SignalChannelID  string
	After            time.Time // Returns records earlier than this time
	Before           time.Time // Returns records newer than this time
	Limit            uint64
}

// SignalResponse holds a signal channel
type SignalResponse struct {
	SignalChannelID          string `json:"signalChanId"`
	SignalChannelName        string `json:"signalChanName"`
	SignalChannelDescription string `json:"signalChanDesc"`
	SignalChannelToken       string `json:"signalChanToken"`
	SignalSourceType         string `json:"signalSourceType"`
}

// SignalBotEntrySetting holds how a signal bot enters positions. A nil AllowMultipleEntry keeps OKX's default of
// allowing them.
type SignalBotEntrySetting struct {
	AllowMultipleEntry *bool        `json:"allowMultipleEntry,omitempty"`
	EntryType          string       `json:"entryType,omitempty"`
	Amount             types.Number `json:"amt,omitempty"`
	Ratio              types.Number `json:"ratio,omitempty"`
}

// SignalBotExitSetting holds a signal bot's take profit and stop loss settings
type SignalBotExitSetting struct {
	TakeProfitStopLossType string       `json:"tpSlType,omitempty"`
	TakeProfitPercentage   types.Number `json:"tpPct,omitempty"`
	StopLossPercentage     types.Number `json:"slPct,omitempty"`
}

// CreateSignalBotRequest holds the parameters to create a signal bot
type CreateSignalBotRequest struct {
	SignalChannelID  string                 `json:"signalChanId"`
	Leverage         float64                `json:"lever,string"`
	InvestmentAmount float64                `json:"investAmt,string"`
	SubOrderType     string                 `json:"subOrdType"`
	IncludeAll       bool                   `json:"includeAll,omitempty"`
	InstrumentIDs    []string               `json:"instIds,omitempty"`
	Ratio            float64                `json:"ratio,string,omitempty"`
	EntrySetting     *SignalBotEntrySetting `json:"entrySettingParam,omitempty"`
	ExitSetting      *SignalBotExitSetting  `json:"exitSettingParam,omitempty"`
}

// AdjustSignalBotMarginBalanceRequest holds the parameters to add or reduce a signal bot's margin balance
type AdjustSignalBotMarginBalanceRequest struct {
	AlgoID                  string  `json:"algoId"`
	AdjustMarginBalanceType string  `json:"type"`
	Amount                  float64 `json:"amt,string"`
	AllowReinvest           bool    `json:"allowReinvest,omitempty"`
}

// AmendSignalBotTakeProfitStopLossRequest holds a signal bot's new take profit and stop loss settings
type AmendSignalBotTakeProfitStopLossRequest struct {
	AlgoID      string               `json:"algoId"`
	ExitSetting SignalBotExitSetting `json:"exitSettingParam"`
}

// SetSignalBotInstrumentsRequest holds the instruments a signal bot trades
type SetSignalBotInstrumentsRequest struct {
	AlgoID        string   `json:"algoId"`
	InstrumentIDs []string `json:"instIds,omitempty"`
	IncludeAll    bool     `json:"includeAll"`
}

// SignalBotOrderRequest identifies a signal bot
type SignalBotOrderRequest struct {
	AlgoOrderType string
	AlgoID        string
}

// SignalBotOrderResponse holds a signal bot
type SignalBotOrderResponse struct {
	AlgoID            string                `json:"algoId"`
	AlgoClientOrderID string                `json:"algoClOrdId"`
	InstrumentType    string                `json:"instType"`
	InstrumentIDs     []string              `json:"instIds"`
	CreationTime      types.Time            `json:"cTime"`
	UpdateTime        types.Time            `json:"uTime"`
	AlgoOrderType     string                `json:"algoOrdType"`
	State             string                `json:"state"`
	CancelType        string                `json:"cancelType"`
	TotalPNL          types.Number          `json:"totalPnl"`
	TotalPNLRatio     types.Number          `json:"totalPnlRatio"`
	TotalEquity       types.Number          `json:"totalEq"`
	FloatPNL          types.Number          `json:"floatPnl"`
	RealisedPNL       types.Number          `json:"realizedPnl"`
	FrozenBalance     types.Number          `json:"frozenBal"`
	AvailableBalance  types.Number          `json:"availBal"`
	Leverage          types.Number          `json:"lever"`
	InvestmentAmount  types.Number          `json:"investAmt"`
	SubOrderType      string                `json:"subOrdType"`
	Ratio             types.Number          `json:"ratio"`
	EntrySetting      SignalBotEntrySetting `json:"entrySettingParam"`
	ExitSetting       SignalBotExitSetting  `json:"exitSettingParam"`
	SignalChannelID   string                `json:"signalChanId"`
	SignalChannelName string                `json:"signalChanName"`
	SignalSourceType  string                `json:"signalSourceType"`
}

// SignalBotListRequest holds the filters for listing active or historic signal bots
type SignalBotListRequest struct {
	AlgoOrderType string
	AlgoID        string    // Required for the history
	After         time.Time // Returns records earlier than this time
	Before        time.Time // Returns records newer than this time
	Limit         uint64
}

// SignalBotPositionHistoryRequest holds the filters for listing a signal bot's position history
type SignalBotPositionHistoryRequest struct {
	AlgoID       string
	InstrumentID string
	After        time.Time // Returns records updated earlier than this time
	Before       time.Time // Returns records updated later than this time
	Limit        uint64
}

// SignalBotPositionHistoryResponse holds a signal bot's historic position
type SignalBotPositionHistoryResponse struct {
	InstrumentID      string       `json:"instId"`
	MarginMode        string       `json:"mgnMode"`
	CreationTime      types.Time   `json:"cTime"`
	UpdateTime        types.Time   `json:"uTime"`
	OpenAveragePrice  types.Number `json:"openAvgPx"`
	CloseAveragePrice types.Number `json:"closeAvgPx"`
	PNL               types.Number `json:"pnl"`
	PNLRatio          types.Number `json:"pnlRatio"`
	Leverage          types.Number `json:"lever"`
	Direction         string       `json:"direction"`
	Underlying        string       `json:"uly"`
}

// CloseSignalBotPositionRequest holds the signal bot position to close at market
type CloseSignalBotPositionRequest struct {
	AlgoID       string `json:"algoId"`
	InstrumentID string `json:"instId"`
}

// PlaceSignalBotSubOrderRequest holds a sub order to place for a signal bot
type PlaceSignalBotSubOrderRequest struct {
	InstrumentID string  `json:"instId"`
	AlgoID       string  `json:"algoId"`
	Side         string  `json:"side"`
	OrderType    string  `json:"ordType"`
	Size         float64 `json:"sz,string"`
	Price        float64 `json:"px,string,omitempty"`
	ReduceOnly   bool    `json:"reduceOnly,omitempty"`
}

// CancelSignalBotSubOrderRequest holds a signal bot sub order to cancel
type CancelSignalBotSubOrderRequest struct {
	AlgoID        string `json:"algoId"`
	InstrumentID  string `json:"instId"`
	SignalOrderID string `json:"signalOrdId"`
}

// CancelSignalBotSubOrderResponse holds the result of cancelling a signal bot sub order
type CancelSignalBotSubOrderResponse struct {
	SignalOrderID string `json:"signalOrdId"`
	// StatusCode is never negative on the wire; it stays int64 to match getStatusError
	StatusCode    uint64 `json:"sCode,string"`
	StatusMessage string `json:"sMsg"`
}

// SignalBotSubOrdersRequest holds the filters for listing a signal bot's sub orders. Either State or SignalOrderID
// is required; SubOrderType and ClientOrderID are older equivalents OKX still documents
type SignalBotSubOrdersRequest struct {
	AlgoID        string
	AlgoOrderType string
	State         string
	SignalOrderID string
	After         string // Returns records earlier than this order ID
	Before        string // Returns records newer than this order ID
	Begin         time.Time
	End           time.Time
	Limit         uint64
	SubOrderType  string
	ClientOrderID string
}

// SignalBotSubOrderResponse holds a signal bot's sub order
type SignalBotSubOrderResponse struct {
	AlgoID              string        `json:"algoId"`
	AlgoClientOrderID   string        `json:"algoClOrdId"`
	InstrumentType      string        `json:"instType"`
	InstrumentID        string        `json:"instId"`
	AlgoOrderType       string        `json:"algoOrdType"`
	OrderID             string        `json:"ordId"`
	ClientOrderID       string        `json:"clOrdId"`
	CreationTime        types.Time    `json:"cTime"`
	UpdateTime          types.Time    `json:"uTime"`
	TradeMode           string        `json:"tdMode"`
	Currency            currency.Code `json:"ccy"`
	OrderType           string        `json:"ordType"`
	Size                types.Number  `json:"sz"`
	State               string        `json:"state"`
	Side                string        `json:"side"`
	Price               types.Number  `json:"px"`
	Fee                 types.Number  `json:"fee"`
	FeeCurrency         currency.Code `json:"feeCcy"`
	AveragePrice        types.Number  `json:"avgPx"`
	AccumulatedFillSize types.Number  `json:"accFillSz"`
	PositionSide        string        `json:"posSide"`
	PNL                 types.Number  `json:"pnl"`
	ContractValue       types.Number  `json:"ctVal"`
	Leverage            types.Number  `json:"lever"`
	Tag                 string        `json:"tag"`
}

// SignalBotEventHistoryRequest holds the filters for listing a signal bot's events
type SignalBotEventHistoryRequest struct {
	AlgoID string
	After  time.Time // Returns events created earlier than this time
	Before time.Time // Returns events created later than this time
	Limit  uint64
}

// SignalBotTriggeredOrder holds the client order ID of a sub order a signal bot event placed
type SignalBotTriggeredOrder struct {
	ClientOrderID string `json:"clOrdId"`
}

// SignalBotEventHistoryResponse holds a signal bot event
type SignalBotEventHistoryResponse struct {
	AlertMessage        string                    `json:"alertMsg"`
	AlgoID              string                    `json:"algoId"`
	EventType           string                    `json:"eventType"`
	EventCreationTime   types.Time                `json:"eventCtime"`
	EventUpdateTime     types.Time                `json:"eventUtime"`
	EventProcessMessage string                    `json:"eventProcessMsg"`
	EventStatus         string                    `json:"eventStatus"`
	TriggeredOrderData  []SignalBotTriggeredOrder `json:"triggeredOrdData"`
}
