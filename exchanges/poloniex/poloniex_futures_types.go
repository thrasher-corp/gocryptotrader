package poloniex

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ServerTimeResponse represents a server time response.
type ServerTimeResponse struct {
	Code string     `json:"code"`
	Msg  string     `json:"msg"`
	Data types.Time `json:"data"`
}

// FuturesAccountBalance represents a futures account balance detail
type FuturesAccountBalance struct {
	State                   string       `json:"state"`
	Equity                  types.Number `json:"eq"`
	IsoEquity               types.Number `json:"isoEq"`
	InitialMargin           types.Number `json:"im"`
	MaintenanceMargin       types.Number `json:"mm"`
	MaintenanceMarginRate   types.Number `json:"mmr"`
	UnrealizedProfitAndLoss types.Number `json:"upl"`
	AvailMargin             types.Number `json:"availMgn"`
	CreationTime            types.Time   `json:"cTime"`
	UpdateTime              types.Time   `json:"uTime"`
	Details                 []struct {
		Currency              currency.Code `json:"ccy"`
		Equity                types.Number  `json:"eq"`
		IsoEquity             types.Number  `json:"isoEq"`
		Available             types.Number  `json:"avail"`
		TrdHold               types.Number  `json:"trdHold"`
		UnrealisedPNL         types.Number  `json:"upl"`
		IsoAvailable          types.Number  `json:"isoAvail"`
		IsoHold               string        `json:"isoHold"`
		IsoUpl                string        `json:"isoUpl"`
		InitialMargin         types.Number  `json:"im"`
		MaintenanceMargin     types.Number  `json:"mm"`
		MaintenanceMarginRate types.Number  `json:"mmr"`
		InitialMarginRate     types.Number  `json:"imr"`
		CreationTime          types.Time    `json:"cTime"`
		UpdateTime            types.Time    `json:"uTime"`
	} `json:"details"`
}

// BillDetail represents a bill type detail information
type BillDetail struct {
	ID           string        `json:"id"`
	AccountType  string        `json:"actType"`
	BillType     string        `json:"type"`
	Currency     currency.Code `json:"ccy"`
	Symbol       string        `json:"symbol"`
	MarginMode   string        `json:"mgnMode"`
	PositionSide string        `json:"posSide"`
	CreationTime types.Time    `json:"cTime"`
	Size         types.Number  `json:"sz"`
}

// FuturesOrderRequest represents a futures order parameters
type FuturesOrderRequest struct {
	Symbol                  string  `json:"symbol"`
	Side                    string  `json:"side"`
	MarginMode              string  `json:"mgnMode"`
	PositionSide            string  `json:"posSide"`
	OrderType               string  `json:"type,omitempty"`
	ClientOrderID           string  `json:"clOrdId,omitempty"`
	Price                   float64 `json:"px,omitempty,string"`
	Size                    float64 `json:"sz,omitempty,string"`
	ReduceOnly              bool    `json:"reduceOnly,omitempty"`
	TimeInForce             string  `json:"timeInForce,omitempty"`
	SelfTradePreventionMode string  `json:"stpMode,omitempty"`
}

// FuturesOrderIDResponse represents a futures order creation response
type FuturesOrderIDResponse struct {
	ClientOrderID string `json:"clOrdId"`
	OrderID       string `json:"ordId"`
	Code          int64  `json:"code"`
	Message       string `json:"msg"`
}

// CancelOrderRequest represents a single order cancellation parameters
type CancelOrderRequest struct {
	Symbol        string `json:"symbol"`
	OrderID       string `json:"ordId,omitempty"`
	ClientOrderID string `json:"clOrdId,omitempty"`
}

// FuturesTradeFill represents a trade executions
type FuturesTradeFill struct {
	Symbol         string       `json:"symbol"`
	Side           string       `json:"side"`
	OrderID        string       `json:"ordId"`
	ClientOrderID  string       `json:"clOrdId"`
	Role           string       `json:"role"`
	TradeID        string       `json:"trdId"`
	FeeCurrency    string       `json:"feeCcy"`
	FeeAmount      types.Number `json:"feeAmt"`
	DeductCurrency string       `json:"deductCcy"`
	DeductAmount   types.Number `json:"deductAmt"`
	FillPrice      types.Number `json:"fpx"`
	FillQuantity   types.Number `json:"fqty"`
	UpdateTime     types.Time   `json:"uTime"`
}

// FuturesInfo represents a sample futures order item info
type FuturesInfo struct {
	Symbol              string            `json:"symbol"`
	Side                string            `json:"side"`
	MarginMode          string            `json:"mgnMode"`
	PositionSide        string            `json:"posSide"`
	OrderType           string            `json:"type"`
	Price               types.Number      `json:"px"`
	Size                types.Number      `json:"sz"`
	TimeInForce         order.TimeInForce `json:"timeInForce"`
	SelfTradePrevention string            `json:"stpMode"`
	ReduceOnly          bool              `json:"reduceOnly"`
	ClientOrderID       string            `json:"clOrdId"`
}

// FuturesOrderDetail represents a futures v3 order detail
type FuturesOrderDetail struct {
	Symbol                     string            `json:"symbol"`
	Side                       string            `json:"side"`
	MarginMode                 string            `json:"mgnMode"`
	PositionSide               string            `json:"posSide"`
	OrderType                  string            `json:"type"`
	Price                      types.Number      `json:"px"`
	Size                       types.Number      `json:"sz"`
	TimeInForce                order.TimeInForce `json:"timeInForce"`
	OrderID                    string            `json:"ordId"`
	AveragePrice               types.Number      `json:"avgPx"`
	CreationTime               types.Time        `json:"cTime"`
	ClientOrderID              string            `json:"clOrdId"`
	DeductAmount               types.Number      `json:"deductAmt"`
	ExecutedAmount             types.Number      `json:"execAmt"`
	DeductCurrency             string            `json:"deductCcy"`
	ExecQuantity               types.Number      `json:"execQty"`
	FeeAmount                  types.Number      `json:"feeAmt"`
	FeeCurrency                string            `json:"feeCcy"`
	Leverage                   types.Number      `json:"lever"`
	ReduceOnly                 bool              `json:"reduceOnly"`
	StopLossPrice              types.Number      `json:"slPx"`
	StopLossTriggerPrice       string            `json:"slTrgPx"`
	StopLossTriggerPriceType   string            `json:"slTrgPxType"`
	Source                     string            `json:"source"`
	State                      string            `json:"state"`
	SelfTradePreventionMode    string            `json:"stpMode"`
	TakeProfitPrice            types.Number      `json:"tpPx"`
	TakeProfitTriggerPrice     types.Number      `json:"tpTrgPx"`
	TakeProfitTriggerPriceType string            `json:"tpTrgPxType"`
	UpdateTime                 types.Time        `json:"uTime"`
	FeeRate                    types.Number      `json:"feeRate"`
	ID                         string            `json:"id"`
	Quantity                   types.Number      `json:"qty"`
	Role                       string            `json:"role"`
	TradeID                    string            `json:"trdId"`
	CancelReason               string            `json:"cancelReason"`
	OrdType                    string            `json:"ordType"`
}

// FuturesPosition represents a v3 futures position detail
type FuturesPosition struct {
	AutoDeleveraging       string       `json:"adl"`
	AvailQuantity          types.Number `json:"availQty"`
	CreationTime           types.Time   `json:"cTime"`
	InitialMargin          types.Number `json:"im"`
	Leverage               types.Number `json:"lever"`
	LiqudiationPrice       types.Number `json:"liqPx"`
	MarkPrice              types.Number `json:"markPx"`
	IsolatedPositionMargin string       `json:"mgn"`
	MarginMode             string       `json:"mgnMode"`
	PositionSide           string       `json:"posSide"`
	MarginRatio            types.Number `json:"mgnRatio"`
	MaintenanceMargin      string       `json:"mm"`
	OpenAveragePrice       types.Number `json:"openAvgPx"`
	ProfitAndLoss          types.Number `json:"pnl"`
	Quantity               types.Number `json:"qty"`
	Side                   string       `json:"side"`
	State                  string       `json:"state"`
	Symbol                 string       `json:"symbol"`
	UpdateTime             types.Time   `json:"uTime"`
	UnrealizedPNL          types.Number `json:"upl"`
	UnrealizedPNLRatio     types.Number `json:"uplRatio"`
	CloseAvgPx             string       `json:"closeAvgPx"`
	ClosedQty              string       `json:"closedQty"`
	FFee                   string       `json:"fFee"`
	Fee                    string       `json:"fee"`
	ID                     string       `json:"id"`
}

// AdjustFuturesMarginResponse represents a response data after adjusting futures margin positions
type AdjustFuturesMarginResponse struct {
	Amount       types.Number `json:"amt"`
	Leverage     types.Number `json:"lever"`
	Symbol       string       `json:"symbol"`
	PositionSide string       `json:"posSide"`
	OrderType    string       `json:"type"`
}

// FuturesLeverage represents futures symbols leverage information
type FuturesLeverage struct {
	Leverage     types.Number `json:"lever"`
	MarginMode   string       `json:"mgnMode"`
	PositionSide string       `json:"posSide"`
	Symbol       string       `json:"symbol"`
}

// FuturesOrderbook represents an orderbook data for v3 futures instruments
type FuturesOrderbook struct {
	Asks          [][]types.Number `json:"asks"`
	Bids          [][]types.Number `json:"bids"`
	Timestamp     types.Time       `json:"ts"`
	LastVersionID int64            `json:"lid"`
	ID            types.Number     `json:"id"`
	Symbol        string           `json:"s"`
	CreationTime  types.Time       `json:"cT"`
}

// FuturesCandle represents a kline data for v3 futures instrument
type FuturesCandle struct {
	LowestPrice  types.Number
	HighestPrice types.Number
	OpeningPrice types.Number
	ClosingPrice types.Number
	QuoteAmount  types.Number
	BaseAmount   types.Number
	Trades       types.Number
	StartTime    types.Time
	EndTime      types.Time
}

// UnmarshalJSON deserializes JSON data into a kline.Candle instance
func (v *FuturesCandle) UnmarshalJSON(data []byte) error {
	target := [9]any{&v.LowestPrice, &v.HighestPrice, &v.OpeningPrice, &v.ClosingPrice, &v.QuoteAmount, &v.BaseAmount, &v.Trades, &v.StartTime, &v.EndTime}
	return json.Unmarshal(data, &target)
}

// FuturesExecutionInfo represents a V3 futures instruments execution information
type FuturesExecutionInfo struct {
	ID           int64        `json:"id"`
	Price        types.Number `json:"px"`
	Quantity     types.Number `json:"qty"`
	Amount       types.Number `json:"amt"`
	Side         string       `json:"side"`
	CreationTime types.Time   `json:"cT"`
}

// LiquidiationPrice represents a liquidiation price detail for an instrument
type LiquidiationPrice struct {
	Symbol                         string       `json:"symbol"`
	PositionSide                   string       `json:"posSide"`
	Side                           string       `json:"side"`
	Size                           types.Number `json:"sz"`
	PriceOfCommissionedTransaction types.Number `json:"bkPx"`
	UpdateTime                     types.Time   `json:"uTime"`
}

// FuturesTickerDetail represents a v3 futures instrument ticker detail
type FuturesTickerDetail struct {
	Symbol       string       `json:"s"`
	OpeningPrice types.Number `json:"o"`
	LowPrice     types.Number `json:"l"`
	HighPrice    types.Number `json:"h"`
	ClosingPrice types.Number `json:"c"`
	Quantity     types.Number `json:"qty"`
	Amount       types.Number `json:"amt"`
	TradeCount   int64        `json:"tC"`
	StartTime    types.Time   `json:"sT"`
	EndTime      types.Time   `json:"cT"`
	DailyPrice   types.Number `json:"dC"`
	BestBidPrice types.Number `json:"bPx"`
	BestBidSize  types.Number `json:"bSz"`
	BestAskPrice types.Number `json:"aPx"`
	BestAskSize  types.Number `json:"aSz"`
	MarkPrice    types.Number `json:"mPx"`
	Timestamp    types.Time   `json:"ts"`
}

// InstrumentIndexPrice represents a symbols index price
type InstrumentIndexPrice struct {
	Symbol     string       `json:"symbol"`
	Timestamp  types.Time   `json:"ts"`
	IndexPrice types.Number `json:"iPx"`
}

// IndexPriceComponent represents an index price component detail
type IndexPriceComponent []struct {
	Symbol string       `json:"s"`
	Price  types.Number `json:"px"`
	Cs     []struct {
		Exchange              string       `json:"e"`
		WeightFactor          types.Number `json:"w"`
		TradingPairPrice      types.Number `json:"sPx"`
		TradingPairIndexPrice types.Number `json:"cPx"`
	} `json:"cs"`
}

// FuturesIndexPriceData represents a futures index price data detail
type FuturesIndexPriceData struct {
	OpenPrice    types.Number
	HighPrice    types.Number
	LowestPrice  types.Number
	ClosingPrice types.Number
	StartTime    types.Time
	EndTime      types.Time
}

// UnmarshalJSON deserializes candlestick data into a FuturesIndexPriceData instance
func (v *FuturesIndexPriceData) UnmarshalJSON(data []byte) error {
	target := [6]any{&v.OpenPrice, &v.HighPrice, &v.LowestPrice, &v.ClosingPrice, &v.StartTime, &v.EndTime}
	return json.Unmarshal(data, &target)
}

// FuturesMarkPrice represents a mark price instance
type FuturesMarkPrice struct {
	MarkPrice types.Number `json:"mPx"`
	Symbol    string       `json:"symbol"`
	Timestamp types.Time   `json:"ts"`
}

// FuturesMarkPriceCandle represents a k-line data for mark price
type FuturesMarkPriceCandle struct {
	OpeningPrice types.Number
	HighestPrice types.Number
	LowestPrice  types.Number
	ClosingPrice types.Number
	StartTime    types.Time
	EndTime      types.Time
}

// UnmarshalJSON deserializes byte data into FuturesMarkPriceCandle instance
func (v *FuturesMarkPriceCandle) UnmarshalJSON(data []byte) error {
	target := [6]any{&v.OpeningPrice, &v.HighestPrice, &v.LowestPrice, &v.ClosingPrice, &v.StartTime, &v.EndTime}
	return json.Unmarshal(data, &target)
}

// ProductInfo represents basic information of the all product
type ProductInfo struct {
	Alias                 string       `json:"alias"`
	BaseAsset             string       `json:"bAsset"`
	BaseCurrency          string       `json:"bCcy"`
	ContractType          string       `json:"ctType"`
	ContractValue         types.Number `json:"ctVal"`
	InitialMarginRate     types.Number `json:"iM"`
	Leverage              types.Number `json:"lever"`
	LotSize               types.Number `json:"lotSz"`
	MaintenanceMarginRate types.Number `json:"mM"`
	MaximumRiskLimit      types.Number `json:"mR"`
	MaxLeverage           string       `json:"maxLever"`
	MaxPrice              types.Number `json:"maxPx"`
	MaxQuantity           types.Number `json:"maxQty"`
	MinPrice              types.Number `json:"minPx"`
	MinQuantity           types.Number `json:"minQty"`
	MinSize               types.Number `json:"minSz"`
	ListingDate           types.Time   `json:"oDate"`
	PriceScale            string       `json:"pxScale"`
	QuoteCurrency         string       `json:"qCcy"`
	SettlementCurrency    string       `json:"sCcy"`
	Status                string       `json:"status"`
	Symbol                string       `json:"symbol"`
	TickSize              types.Number `json:"tSz"`
	TradableStartTime     types.Time   `json:"tradableStartTime"`
	VisibleStartTime      types.Time   `json:"visibleStartTime"`
}

// FuturesFundingRate represents symbols funding rate information
type FuturesFundingRate struct {
	Symbol                   string       `json:"s"`
	FundingRate              types.Number `json:"fR"`
	FundingRateSettleTime    types.Time   `json:"fT"`
	NextPredictedFundingRate types.Number `json:"nFR"`
	NextFundingTime          types.Time   `json:"nFT"`
	Timestamp                types.Time   `json:"ts"`
}

// OpenInterestData represents an open interest data
type OpenInterestData struct {
	CurrentOpenInterest types.Number `json:"oInterest"`
	Symbol              string       `json:"s"`
}

// InsuranceFundInfo represents an insurance fund information of a currency
type InsuranceFundInfo struct {
	Amount     types.Number  `json:"amt"`
	Currency   currency.Code `json:"ccy"`
	UpdateTime types.Time    `json:"uTime"`
}

// RiskLimit represents a risk limit of futures instrument
type RiskLimit struct {
	NotionalCap types.Number `json:"notionalCap"`
	Symbol      string       `json:"symbol"`
}

// WsFuturesCandlesctick represents a kline data for futures instrument
type WsFuturesCandlesctick struct {
	Symbol       string
	LowestPrice  types.Number
	HighestPrice types.Number
	OpenPrice    types.Number
	ClosePrice   types.Number
	Amount       types.Number
	Quantity     types.Number
	Trades       types.Number
	StartTime    types.Time
	EndTime      types.Time
	PushTime     types.Time
}

// UnmarshalJSON deserializes byte data into futures candlesticks into *WsFuturesCandlesctick
func (o *WsFuturesCandlesctick) UnmarshalJSON(data []byte) error {
	target := [11]any{&o.Symbol, &o.LowestPrice, &o.HighestPrice, &o.OpenPrice, &o.ClosePrice, &o.Amount, &o.Quantity, &o.Trades, &o.StartTime, &o.EndTime, &o.PushTime}
	return json.Unmarshal(data, &target)
}

// FuturesTrades represents a futures trades detail
type FuturesTrades struct {
	ID           int64        `json:"id"`
	Timestamp    types.Time   `json:"ts"`
	Symbol       string       `json:"s"`
	Price        types.Number `json:"px"`
	Quantity     types.Number `json:"qty"`
	Amount       types.Number `json:"amt"`
	Side         string       `json:"side"`
	CreationTime types.Time   `json:"cT"`
}

// WsFuturesMarkAndIndexPriceCandle represents a websocket k-line data for mark/index candlestick data
type WsFuturesMarkAndIndexPriceCandle struct {
	OpeningPrice  types.Number
	HighestPrice  types.Number
	LowestPrice   types.Number
	ClosingPrice  types.Number
	StartTime     types.Time
	EndTime       types.Time
	Symbol        string
	PushTimestamp types.Time
}

// UnmarshalJSON deserializes byte data into WsFuturesMarkAndIndexPriceCandle instance
func (v *WsFuturesMarkAndIndexPriceCandle) UnmarshalJSON(data []byte) error {
	target := [8]any{&v.Symbol, &v.LowestPrice, &v.HighestPrice, &v.OpeningPrice, &v.ClosingPrice, &v.StartTime, &v.EndTime, &v.PushTimestamp}
	return json.Unmarshal(data, &target)
}

// MarginModeSwitchResponse represents a response detail after switching margin mode for a symbol
type MarginModeSwitchResponse struct {
	MarginMode string `json:"mgnMode"`
	Symbol     string `json:"symbol"`
}
