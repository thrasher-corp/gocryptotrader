package okx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PlaceRecurringBuyOrderRequest holds the parameters to place a recurring buy order
type PlaceRecurringBuyOrderRequest struct {
	StrategyName  string              `json:"stgyName"`
	RecurringList []RecurringListItem `json:"recurringList"`
	Period        string              `json:"period"`
	RecurringDay  uint64              `json:"recurringDay,string,omitempty"`  // Day of the month or week; monthly and weekly periods only
	RecurringHour uint64              `json:"recurringHour,string,omitempty"` // Hours between buys; hourly periods only
	RecurringTime uint64              `json:"recurringTime,string"`           // Hour of the day, an integer between 0 and 23
	// TimeZone is the UTC offset in hours, an integer between -12 and 14
	TimeZone           int64         `json:"timeZone,string"`
	Amount             float64       `json:"amt,string"`
	InvestmentCurrency currency.Code `json:"investmentCcy"`
	TradeMode          string        `json:"tdMode"`
	AlgoClientOrderID  string        `json:"algoClOrdId,omitempty"`
	Tag                string        `json:"tag,omitempty"`
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy,omitzero"`
	Source             []string      `json:"source,omitempty"`
	RecurringTimeType  string        `json:"recurringTimeType,omitempty"`
}

// RecurringListItem holds a currency a recurring buy order buys and its share of each investment; a zero price
// bound means no limit
type RecurringListItem struct {
	Currency     currency.Code `json:"ccy"`
	Ratio        types.Number  `json:"ratio"`
	MinimumPrice types.Number  `json:"minPx,omitempty"`
	MaximumPrice types.Number  `json:"maxPx,omitempty"`
}

// RecurringListItemDetailed holds a currency a recurring buy order buys with the amount bought so far
type RecurringListItemDetailed struct {
	AveragePrice types.Number  `json:"avgPx"`
	Currency     currency.Code `json:"ccy"`
	MinimumPrice types.Number  `json:"minPx"`
	MaximumPrice types.Number  `json:"maxPx"`
	Profit       types.Number  `json:"profit"`
	Price        types.Number  `json:"px"`
	Ratio        types.Number  `json:"ratio"`
	TotalAmount  types.Number  `json:"totalAmt"`
}

// AmendRecurringBuyOrderRequest holds a recurring buy order's new strategy name
type AmendRecurringBuyOrderRequest struct {
	AlgoID       string `json:"algoId"`
	StrategyName string `json:"stgyName"`
}

// RecurringBuyOrderListRequest holds the filters for listing pending or historic recurring buy orders
type RecurringBuyOrderListRequest struct {
	AlgoID string
	After  string // Returns records earlier than this algo ID
	Before string // Returns records newer than this algo ID
	Limit  uint64
}

// RecurringBuyOrderResponse holds a recurring buy order
type RecurringBuyOrderResponse struct {
	AlgoID               string              `json:"algoId"`
	AlgoClientOrderID    string              `json:"algoClOrdId"`
	InstrumentType       string              `json:"instType"`
	CreationTime         types.Time          `json:"cTime"`
	UpdateTime           types.Time          `json:"uTime"`
	AlgoOrderType        string              `json:"algoOrdType"`
	State                string              `json:"state"`
	StrategyName         string              `json:"stgyName"`
	RecurringList        []RecurringListItem `json:"recurringList"`
	Period               string              `json:"period"`
	RecurringDay         types.Number        `json:"recurringDay"`
	RecurringHour        types.Number        `json:"recurringHour"`
	RecurringTime        types.Number        `json:"recurringTime"`
	TimeZone             types.Number        `json:"timeZone"`
	Amount               types.Number        `json:"amt"`
	InvestmentAmount     types.Number        `json:"investmentAmt"`
	InvestmentCurrency   currency.Code       `json:"investmentCcy"`
	TotalPNL             types.Number        `json:"totalPnl"`
	TotalAnnualisedRate  types.Number        `json:"totalAnnRate"`
	PNLRatio             types.Number        `json:"pnlRatio"`
	MarketCap            types.Number        `json:"mktCap"`
	Cycles               types.Number        `json:"cycles"`
	Tag                  string              `json:"tag"`
	TradeQuoteCurrency   currency.Code       `json:"tradeQuoteCcy"`
	Source               []string            `json:"source"`
	RecurringTimeType    string              `json:"recurringTimeType"`
	RecurringTimeMinutes types.Number        `json:"recurringTimeMinutes"`
}

// RecurringOrderDetailResponse holds a recurring buy order's details
type RecurringOrderDetailResponse struct {
	RecurringBuyOrderResponse
	// RecurringList shadows the promoted RecurringBuyOrderResponse.RecurringList: the details endpoint returns the
	// detailed item type
	RecurringList  []RecurringListItemDetailed `json:"recurringList"`
	NextInvestTime types.Time                  `json:"nextInvestTime"`
}

// RecurringBuySubOrdersRequest holds the filters for listing a recurring buy order's sub orders
type RecurringBuySubOrdersRequest struct {
	AlgoID  string
	OrderID string
	After   string // Returns records earlier than this ID
	Before  string // Returns records newer than this ID
	Limit   uint64
}

// RecurringBuySubOrderResponse holds a recurring buy order's sub order
type RecurringBuySubOrderResponse struct {
	AlgoID              string        `json:"algoId"`
	InstrumentType      string        `json:"instType"`
	InstrumentID        string        `json:"instId"`
	AlgoOrderType       string        `json:"algoOrdType"`
	OrderID             string        `json:"ordId"`
	CreationTime        types.Time    `json:"cTime"`
	UpdateTime          types.Time    `json:"uTime"`
	TradeMode           string        `json:"tdMode"`
	OrderType           string        `json:"ordType"`
	Size                types.Number  `json:"sz"`
	State               string        `json:"state"`
	Side                string        `json:"side"`
	Price               types.Number  `json:"px"` // -1 for market orders
	Fee                 types.Number  `json:"fee"`
	FeeCurrency         currency.Code `json:"feeCcy"`
	AveragePrice        types.Number  `json:"avgPx"`
	AccumulatedFillSize types.Number  `json:"accFillSz"`
	Tag                 string        `json:"tag"`
	AlgoClientOrderID   string        `json:"algoClOrdId"`
}

// AmendRecurringBuyTimeRequest holds a recurring buy order's new schedule. RecurringTime is required for custom
// times and a nil RecurringTime is left unsent
type AmendRecurringBuyTimeRequest struct {
	AlgoID            string `json:"algoId"`
	RecurringTimeType string `json:"recurringTimeType"`
	// TimeZone is the UTC offset in hours, an integer between -12 and 14
	TimeZone      int64   `json:"timeZone,string"`
	Period        string  `json:"period"`
	RecurringHour uint64  `json:"recurringHour,string,omitempty"`
	RecurringDay  uint64  `json:"recurringDay,string,omitempty"`
	RecurringTime *uint64 `json:"recurringTime,string,omitempty"` // Hour of the day, an integer between 0 and 23
}

// RecurringBuyAmountRequest holds an amount in a recurring buy order's investment currency
type RecurringBuyAmountRequest struct {
	AlgoID string  `json:"algoId"`
	Amount float64 `json:"amount,string"`
}

// AmendRecurringBuyPriceRangeRequest holds the price ranges a recurring buy order buys its currencies within
type AmendRecurringBuyPriceRangeRequest struct {
	AlgoID        string                   `json:"algoId"`
	RecurringList []RecurringBuyPriceRange `json:"recurringList"`
}

// RecurringBuyPriceRange holds the price range a recurring buy order buys a currency within; a zero bound sends "",
// which means no limit
type RecurringBuyPriceRange struct {
	Currency     currency.Code `json:"ccy"`
	MinimumPrice types.Number  `json:"minPx"`
	MaximumPrice types.Number  `json:"maxPx"`
}
