package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AttachAlgoOrderRequest holds a take-profit, stop-loss or trailing stop order
// attached to a placed order
type AttachAlgoOrderRequest struct {
	AttachAlgoClientOrderID    string  `json:"attachAlgoClOrdId,omitempty"`
	TakeProfitTriggerPrice     float64 `json:"tpTriggerPx,omitempty,string"`
	TakeProfitTriggerRatio     float64 `json:"tpTriggerRatio,omitempty,string"`
	TakeProfitOrderPrice       float64 `json:"tpOrdPx,omitempty,string"`
	TakeProfitOrderKind        string  `json:"tpOrdKind,omitempty"`
	StopLossTriggerPrice       float64 `json:"slTriggerPx,omitempty,string"`
	StopLossTriggerRatio       float64 `json:"slTriggerRatio,omitempty,string"`
	StopLossOrderPrice         float64 `json:"slOrdPx,omitempty,string"`
	TakeProfitTriggerPriceType string  `json:"tpTriggerPxType,omitempty"`
	StopLossTriggerPriceType   string  `json:"slTriggerPxType,omitempty"`
	Size                       float64 `json:"sz,omitempty,string"`
	AmendPriceOnTriggerType    string  `json:"amendPxOnTriggerType,omitempty"`
	CallbackRatio              float64 `json:"callbackRatio,omitempty,string"`
	CallbackSpread             float64 `json:"callbackSpread,omitempty,string"`
	ActivePrice                float64 `json:"activePx,omitempty,string"`
}

// CancelOrderRequest represents order parameters to cancel an order
type CancelOrderRequest struct {
	InstrumentID     string `json:"instId,omitempty"`     // Required for REST cancellation
	InstrumentIDCode uint64 `json:"instIdCode,omitempty"` // Required for websocket cancellation
	OrderID          string `json:"ordId,omitempty"`
	ClientOrderID    string `json:"clOrdId,omitempty"`
}

// AmendOrderRequest represents amend order requesting parameters
type AmendOrderRequest struct {
	InstrumentID       string                          `json:"instId,omitempty"`     // Required for REST amendment
	InstrumentIDCode   uint64                          `json:"instIdCode,omitempty"` // Required for websocket amendment
	CancelOnFail       bool                            `json:"cxlOnFail,omitempty"`
	OrderID            string                          `json:"ordId,omitempty"`
	ClientOrderID      string                          `json:"clOrdId,omitempty"`
	ClientRequestID    string                          `json:"reqId,omitempty"`
	NewSize            float64                         `json:"newSz,omitempty,string"`
	NewPrice           float64                         `json:"newPx,omitempty,string"`
	NewPriceUSD        float64                         `json:"newPxUsd,omitempty,string"`
	NewPriceVolatility float64                         `json:"newPxVol,omitempty,string"`
	PriceAmendType     string                          `json:"pxAmendType,omitempty"`
	AttachAlgoOrders   []AmendAttachedAlgoOrderRequest `json:"attachAlgoOrds,omitempty"`
	RPITakerAccess     bool                            `json:"rpiTakerAccess,omitempty"`
	RPIPriceRound      bool                            `json:"rpiPxRound,omitempty"`
}

// AmendAttachedAlgoOrderRequest holds the new values of an attached
// take-profit, stop-loss or trailing stop order
type AmendAttachedAlgoOrderRequest struct {
	AttachAlgoID                  string  `json:"attachAlgoId,omitempty"`
	AttachAlgoClientOrderID       string  `json:"attachAlgoClOrdId,omitempty"`
	NewTakeProfitTriggerPrice     float64 `json:"newTpTriggerPx,omitempty,string"`
	NewTakeProfitTriggerRatio     float64 `json:"newTpTriggerRatio,omitempty,string"`
	NewTakeProfitOrderPrice       float64 `json:"newTpOrdPx,omitempty,string"`
	NewTakeProfitOrderKind        string  `json:"newTpOrdKind,omitempty"`
	NewStopLossTriggerPrice       float64 `json:"newSlTriggerPx,omitempty,string"`
	NewStopLossTriggerRatio       float64 `json:"newSlTriggerRatio,omitempty,string"`
	NewStopLossOrderPrice         float64 `json:"newSlOrdPx,omitempty,string"`
	NewTakeProfitTriggerPriceType string  `json:"newTpTriggerPxType,omitempty"`
	NewStopLossTriggerPriceType   string  `json:"newSlTriggerPxType,omitempty"`
	NewSize                       float64 `json:"sz,omitempty,string"`
	AmendPriceOnTriggerType       string  `json:"amendPxOnTriggerType,omitempty"`
	NewCallbackRatio              float64 `json:"newCallbackRatio,omitempty,string"`
	NewCallbackSpread             float64 `json:"newCallbackSpread,omitempty,string"`
	NewActivePrice                float64 `json:"newActivePx,omitempty,string"`
}

// ClosePositionsRequest holds the parameters to close a position
type ClosePositionsRequest struct {
	InstrumentID          string        `json:"instId"`
	PositionSide          string        `json:"posSide,omitempty"`
	MarginMode            string        `json:"mgnMode"`
	Currency              currency.Code `json:"ccy,omitzero"`
	AutomaticallyCanceled bool          `json:"autoCxl,omitempty"`
	ClientOrderID         string        `json:"clOrdId,omitempty"`
	Tag                   string        `json:"tag,omitempty"`
}

// ClosePositionResponse holds the response data for a close position request
type ClosePositionResponse struct {
	InstrumentID  string `json:"instId"`
	PositionSide  string `json:"posSide"`
	ClientOrderID string `json:"clOrdId"`
	Tag           string `json:"tag"`
}

// OrderDetailRequest holds the parameters to request an order's details
type OrderDetailRequest struct {
	InstrumentID  string
	OrderID       string
	ClientOrderID string
}

// LinkedAlgoOrder holds the stop-loss algo order linked to the take-profit
// limit order of a one-cancels-the-other order
type LinkedAlgoOrder struct {
	AlgoID string `json:"algoId"`
}

// OrderDetailResponse holds detailed information about an order
type OrderDetailResponse struct {
	InstrumentType             string              `json:"instType"`
	InstrumentID               string              `json:"instId"`
	TargetCurrency             string              `json:"tgtCcy"` // base_ccy or quote_ccy
	Currency                   currency.Code       `json:"ccy"`
	OrderID                    string              `json:"ordId"`
	ClientOrderID              string              `json:"clOrdId"`
	Tag                        string              `json:"tag"`
	Price                      types.Number        `json:"px"`
	PriceUSD                   types.Number        `json:"pxUsd"`
	PriceVolatility            types.Number        `json:"pxVol"`
	PriceType                  string              `json:"pxType"`
	Size                       types.Number        `json:"sz"`
	ProfitAndLoss              types.Number        `json:"pnl"`
	OrderType                  string              `json:"ordType"`
	Side                       order.Side          `json:"side"`
	PositionSide               string              `json:"posSide"`
	TradeMode                  string              `json:"tdMode"`
	AccumulatedFillSize        types.Number        `json:"accFillSz"`
	FillPrice                  types.Number        `json:"fillPx"`
	TradeID                    string              `json:"tradeId"`
	FillSize                   types.Number        `json:"fillSz"`
	FillTime                   types.Time          `json:"fillTime"`
	AveragePrice               types.Number        `json:"avgPx"`
	State                      string              `json:"state"`
	SelfTradePreventionID      string              `json:"stpId"`
	SelfTradePreventionMode    string              `json:"stpMode"`
	Leverage                   types.Number        `json:"lever"`
	AttachAlgoClientOrderID    string              `json:"attachAlgoClOrdId"`
	TakeProfitTriggerPrice     types.Number        `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string              `json:"tpTriggerPxType"`
	TakeProfitOrderPrice       types.Number        `json:"tpOrdPx"`
	StopLossTriggerPrice       types.Number        `json:"slTriggerPx"`
	StopLossTriggerPriceType   string              `json:"slTriggerPxType"`
	StopLossOrderPrice         types.Number        `json:"slOrdPx"`
	AttachedAlgoOrders         []AttachedAlgoOrder `json:"attachAlgoOrds"`
	LinkedAlgoOrder            LinkedAlgoOrder     `json:"linkedAlgoOrd"`
	FeeCurrency                currency.Code       `json:"feeCcy"`
	TransactionFee             types.Number        `json:"fee"`
	RebateCurrency             currency.Code       `json:"rebateCcy"`
	RebateAmount               types.Number        `json:"rebate"`
	Source                     string              `json:"source"`
	Category                   string              `json:"category"`
	ReduceOnly                 bool                `json:"reduceOnly,string"`
	IsTakeProfitLimit          bool                `json:"isTpLimit,string"`
	CancelSource               string              `json:"cancelSource"`
	CancelSourceReason         string              `json:"cancelSourceReason"`
	QuickMarginType            string              `json:"quickMgnType"`
	AlgoClientOrderID          string              `json:"algoClOrdId"`
	AlgoID                     string              `json:"algoId"`
	UpdateTime                 types.Time          `json:"uTime"`
	CreationTime               types.Time          `json:"cTime"`
	TradeQuoteCurrency         currency.Code       `json:"tradeQuoteCcy"`
	Outcome                    string              `json:"outcome"`
}

// OrderListRequest holds the parameters to list incomplete orders
type OrderListRequest struct {
	InstrumentType   string
	InstrumentFamily string
	InstrumentID     string
	OrderType        string
	State            string
	After            string
	Before           string
	Limit            uint64
}

// OrderHistoryRequest holds the parameters to list completed orders
type OrderHistoryRequest struct {
	InstrumentType   string
	InstrumentFamily string
	InstrumentID     string
	OrderType        string
	State            string
	Category         string
	After            string
	Before           string
	Begin            time.Time
	End              time.Time
	Limit            uint64
}

// TransactionDetailRequest holds the parameters to list recently filled
// transactions
type TransactionDetailRequest struct {
	InstrumentType   string
	InstrumentFamily string
	InstrumentID     string
	OrderID          string
	SubType          string
	After            string
	Before           string
	Begin            time.Time
	End              time.Time
	Limit            uint64
}

// TransactionDetailResponse holds a filled transaction's details
type TransactionDetailResponse struct {
	InstrumentType      string        `json:"instType"`
	InstrumentID        string        `json:"instId"`
	TradeID             string        `json:"tradeId"`
	OrderID             string        `json:"ordId"`
	ClientOrderID       string        `json:"clOrdId"`
	BillID              string        `json:"billId"`
	TransactionType     string        `json:"subType"`
	Tag                 string        `json:"tag"`
	FillPrice           types.Number  `json:"fillPx"`
	FillSize            types.Number  `json:"fillSz"`
	FillIndexPrice      types.Number  `json:"fillIdxPx"`
	FillProfitAndLoss   types.Number  `json:"fillPnl"`
	FillPriceVolatility types.Number  `json:"fillPxVol"`
	FillPriceUSD        types.Number  `json:"fillPxUsd"`
	FillMarkVolatility  types.Number  `json:"fillMarkVol"`
	FillForwardPrice    types.Number  `json:"fillFwdPx"`
	FillMarkPrice       types.Number  `json:"fillMarkPx"`
	Side                order.Side    `json:"side"`
	PositionSide        string        `json:"posSide"`
	ExecutionType       string        `json:"execType"`
	FeeCurrency         currency.Code `json:"feeCcy"`
	Fee                 types.Number  `json:"fee"`
	Timestamp           types.Time    `json:"ts"`
	FillTime            types.Time    `json:"fillTime"`
	FeeRate             types.Number  `json:"feeRate"`
	TradeQuoteCurrency  currency.Code `json:"tradeQuoteCcy"`
}

// EasyConvertCurrencyListRequest holds the parameters to list the currencies
// easy convert takes
type EasyConvertCurrencyListRequest struct {
	Source string
}

// EasyConvertCurrencyListResponse holds the small currencies easy convert can
// convert from and the mainstream currencies it converts to
type EasyConvertCurrencyListResponse struct {
	FromData     []EasyConvertFromCurrency `json:"fromData"`
	ToCurrencies []currency.Code           `json:"toCcy"`
}

// EasyConvertFromCurrency holds a small currency balance easy convert can
// convert from
type EasyConvertFromCurrency struct {
	FromCurrency currency.Code `json:"fromCcy"`
	FromAmount   types.Number  `json:"fromAmt"`
}

// PlaceEasyConvertRequest holds the parameters to convert small currencies to
// a mainstream currency
type PlaceEasyConvertRequest struct {
	FromCurrencies []currency.Code `json:"fromCcy"`
	ToCurrency     currency.Code   `json:"toCcy"`
	Source         string          `json:"source,omitempty"`
}

// EasyConvertResponse holds an easy convert trade's status
type EasyConvertResponse struct {
	Status       string        `json:"status"`
	FromCurrency currency.Code `json:"fromCcy"`
	ToCurrency   currency.Code `json:"toCcy"`
	FillFromSize types.Number  `json:"fillFromSz"`
	FillToSize   types.Number  `json:"fillToSz"`
	Account      string        `json:"acct"`
	UpdateTime   types.Time    `json:"uTime"`
}

// EasyConvertHistoryRequest holds the parameters to list easy convert trades
type EasyConvertHistoryRequest struct {
	After  time.Time
	Before time.Time
	Limit  uint64
}

// OneClickRepayCurrencyListRequest holds the parameters to list the currencies
// one-click repay takes
type OneClickRepayCurrencyListRequest struct {
	DebtType string
}

// OneClickRepayCurrencyListResponse represents one debt type entry of the
// one-click repay currency list
type OneClickRepayCurrencyListResponse struct {
	Debts    []OneClickRepayCurrencyListDebt  `json:"debtData"`
	DebtType string                           `json:"debtType"`
	Repays   []OneClickRepayCurrencyListRepay `json:"repayData"`
}

// OneClickRepayCurrencyListDebt represents one currency's debt, including
// principal and interest
type OneClickRepayCurrencyListDebt struct {
	Currency currency.Code `json:"debtCcy"`
	Amount   types.Number  `json:"debtAmt"`
}

// OneClickRepayCurrencyListRepay represents one currency's available balance
// amount for one-click repayment
type OneClickRepayCurrencyListRepay struct {
	Currency currency.Code `json:"repayCcy"`
	Amount   types.Number  `json:"repayAmt"`
}

// TradeOneClickRepayRequest holds the parameters to repay cross debts with
// one-click repay
type TradeOneClickRepayRequest struct {
	DebtCurrencies []currency.Code `json:"debtCcy"`
	RepayCurrency  currency.Code   `json:"repayCcy"`
}

// OneClickRepayResponse holds a one-click repay trade's status
type OneClickRepayResponse struct {
	Status        string        `json:"status"`
	DebtCurrency  currency.Code `json:"debtCcy"`
	RepayCurrency currency.Code `json:"repayCcy"`
	FillDebtSize  types.Number  `json:"fillDebtSz"`
	FillRepaySize types.Number  `json:"fillRepaySz"`
	UpdateTime    types.Time    `json:"uTime"`
}

// OneClickRepayHistoryRequest holds the parameters to list one-click repay
// trades
type OneClickRepayHistoryRequest struct {
	After  time.Time
	Before time.Time
	Limit  uint64
}

// OneClickRepayCurrencyListV2Response holds the debt currencies and the
// currencies that can repay them through the new one-click repay
type OneClickRepayCurrencyListV2Response struct {
	Debts  []OneClickRepayCurrencyListDebt  `json:"debtData"`
	Repays []OneClickRepayCurrencyListRepay `json:"repayData"`
}

// TradeOneClickRepayV2Request holds the parameters to repay a debt with the
// new one-click repay
type TradeOneClickRepayV2Request struct {
	DebtCurrency    currency.Code   `json:"debtCcy"`
	RepayCurrencies []currency.Code `json:"repayCcyList"`
}

// TradeOneClickRepayV2Response holds an accepted new one-click repay request
type TradeOneClickRepayV2Response struct {
	DebtCurrency    currency.Code   `json:"debtCcy"`
	RepayCurrencies []currency.Code `json:"repayCcyList"`
	Timestamp       types.Time      `json:"ts"`
}

// OneClickRepayHistoryV2Response holds a new one-click repay trade and the
// orders it placed
type OneClickRepayHistoryV2Response struct {
	DebtCurrency    currency.Code        `json:"debtCcy"`
	RepayCurrencies []currency.Code      `json:"repayCcyList"`
	FillDebtSize    types.Number         `json:"fillDebtSz"`
	Status          string               `json:"status"`
	Orders          []OneClickRepayOrder `json:"ordIdInfo"`
	Timestamp       types.Time           `json:"ts"`
}

// OneClickRepayOrder holds an order a new one-click repay trade placed
type OneClickRepayOrder struct {
	OrderID      string       `json:"ordId"`
	InstrumentID string       `json:"instId"`
	OrderType    string       `json:"ordType"`
	Side         order.Side   `json:"side"`
	Price        types.Number `json:"px"`
	Size         types.Number `json:"sz"`
	FillPrice    types.Number `json:"fillPx"`
	FillSize     types.Number `json:"fillSz"`
	State        string       `json:"state"`
	CreationTime types.Time   `json:"cTime"`
}

// MassCancelOrderResponse holds the result of a mass cancel request
type MassCancelOrderResponse struct {
	Result bool `json:"result"`
}

// CancelAllAfterRequest holds the parameters to cancel all pending orders
// after a countdown
type CancelAllAfterRequest struct {
	// TimeOut is the countdown in seconds; 0 disables Cancel All After
	TimeOut uint64 `json:"timeOut,string"`
	Tag     string `json:"tag,omitempty"`
}

// CancelAllAfterResponse holds the state of a Cancel All After countdown
type CancelAllAfterResponse struct {
	// TriggerTime is zero when Cancel All After is disabled
	TriggerTime types.Time `json:"triggerTime"`
	Tag         string     `json:"tag"`
	Timestamp   types.Time `json:"ts"`
}

// AccountRateLimitResponse holds the account's order rate limit details
type AccountRateLimitResponse struct {
	FillRatio            types.Number `json:"fillRatio"`
	MainFillRatio        types.Number `json:"mainFillRatio"`
	AccountRateLimit     types.Number `json:"accRateLimit"`
	NextAccountRateLimit types.Number `json:"nextAccRateLimit"`
	Timestamp            types.Time   `json:"ts"`
}

// OrderPreCheckRequest holds the parameters of an order to precheck
type OrderPreCheckRequest struct {
	InstrumentID     string                   `json:"instId"`
	TradeMode        string                   `json:"tdMode"`
	ClientOrderID    string                   `json:"clOrdId,omitempty"`
	Side             string                   `json:"side"`
	PositionSide     string                   `json:"posSide,omitempty"`
	OrderType        string                   `json:"ordType"`
	Size             float64                  `json:"sz,string"`
	Price            float64                  `json:"px,omitempty,string"`
	Outcome          string                   `json:"outcome,omitempty"`
	ReduceOnly       bool                     `json:"reduceOnly,omitempty"`
	TargetCurrency   string                   `json:"tgtCcy,omitempty"` // base_ccy or quote_ccy
	AttachAlgoOrders []AttachAlgoOrderRequest `json:"attachAlgoOrds,omitempty"`
}

// OrderPreCheckResponse holds the account information before and after
// placing a prechecked order
type OrderPreCheckResponse struct {
	AdjustedEquity                     types.Number  `json:"adjEq"`
	AdjustedEquityChange               types.Number  `json:"adjEqChg"`
	InitialMarginRequirement           types.Number  `json:"imr"`
	InitialMarginRequirementChange     types.Number  `json:"imrChg"`
	MaintenanceMarginRequirement       types.Number  `json:"mmr"`
	MaintenanceMarginRequirementChange types.Number  `json:"mmrChg"`
	MarginRatio                        types.Number  `json:"mgnRatio"`
	MarginRatioChange                  types.Number  `json:"mgnRatioChg"`
	AvailableBalance                   types.Number  `json:"availBal"`
	AvailableBalanceChange             types.Number  `json:"availBalChg"`
	LiquidationPrice                   types.Number  `json:"liqPx"`
	LiquidationPriceDifference         types.Number  `json:"liqPxDiff"`
	LiquidationPriceDifferenceRatio    types.Number  `json:"liqPxDiffRatio"`
	PositiveAssetBalance               types.Number  `json:"posBal"`
	PositiveAssetBalanceChange         types.Number  `json:"posBalChg"`
	Liabilities                        types.Number  `json:"liab"`
	LiabilitiesChange                  types.Number  `json:"liabChg"`
	LiabilitiesChangeCurrency          currency.Code `json:"liabChgCcy"`
	PositiveAssetUnitType              string        `json:"type"`
}
