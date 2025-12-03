package poloniex

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

type marginMode margin.Type

func (m marginMode) MarshalText() ([]byte, error) {
	switch margin.Type(m) {
	case margin.Multi:
		return []byte("CROSS"), nil
	case margin.Isolated:
		return []byte("ISOLATED"), nil
	case margin.Unset:
		return []byte(""), nil
	}
	return nil, fmt.Errorf("%w: %q", margin.ErrMarginTypeUnsupported, m)
}

// ServerTimeResponse represents a server time response.
type ServerTimeResponse struct {
	V3ResponseWrapper
	Data types.Time `json:"data"`
}

// FuturesAccountBalance represents a futures account balance detail
type FuturesAccountBalance struct {
	State                   string                           `json:"state"`
	Equity                  types.Number                     `json:"eq"`
	IsoEquity               types.Number                     `json:"isoEq"`
	InitialMargin           types.Number                     `json:"im"`
	MaintenanceMargin       types.Number                     `json:"mm"`
	MaintenanceMarginRate   types.Number                     `json:"mmr"`
	UnrealizedProfitAndLoss types.Number                     `json:"upl"`
	AvailMargin             types.Number                     `json:"availMgn"`
	CreationTime            types.Time                       `json:"cTime"`
	UpdateTime              types.Time                       `json:"uTime"`
	Details                 []*FuturesCurrencyAccountBalance `json:"details"`
}

// FuturesCurrencyAccountBalance holds a futures account currency balance detail
type FuturesCurrencyAccountBalance struct {
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
}

// BillDetails represents a bill type detail information
type BillDetails struct {
	ID           string        `json:"id"`
	AccountType  string        `json:"actType"`
	BillType     string        `json:"type"`
	Currency     currency.Code `json:"ccy"`
	Symbol       currency.Pair `json:"symbol"`
	MarginMode   string        `json:"mgnMode"`
	PositionSide string        `json:"posSide"`
	CreationTime types.Time    `json:"cTime"`
	Size         types.Number  `json:"sz"`
}

// FuturesOrderRequest represents a futures order parameters
type FuturesOrderRequest struct {
	Symbol                  string      `json:"symbol"`
	Side                    string      `json:"side"`
	MarginMode              marginMode  `json:"mgnMode"`
	PositionSide            order.Side  `json:"posSide"`
	OrderType               OrderType   `json:"type,omitempty"`
	ClientOrderID           string      `json:"clOrdId,omitempty"`
	Price                   float64     `json:"px,omitempty,string"`
	Size                    float64     `json:"sz,omitempty,string"`
	ReduceOnly              bool        `json:"reduceOnly,omitempty"`
	TimeInForce             TimeInForce `json:"timeInForce,omitempty"`
	SelfTradePreventionMode string      `json:"stpMode,omitempty"`
}

// FuturesOrderIDResponse represents a futures order creation response
type FuturesOrderIDResponse struct {
	ClientOrderID string `json:"clOrdId"`
	OrderID       string `json:"ordId"`
	V3ResponseWrapper
}

// CancelOrderRequest represents a single order cancellation parameters
type CancelOrderRequest struct {
	Symbol        string `json:"symbol"`
	OrderID       string `json:"ordId,omitempty"`
	ClientOrderID string `json:"clOrdId,omitempty"`
}

// CancelFuturesOrdersRequest represents cancel futures order request parameters
type CancelFuturesOrdersRequest struct {
	Symbol         currency.Pair `json:"symbol"`
	OrderIDs       []string      `json:"ordIds,omitempty"`
	ClientOrderIDs []string      `json:"clOrdIds,omitempty"`
}

// FuturesTradeFill represents a trade executions
type FuturesTradeFill struct {
	Symbol         currency.Pair `json:"symbol"`
	Side           string        `json:"side"`
	OrderID        string        `json:"ordId"`
	ClientOrderID  string        `json:"clOrdId"`
	Role           string        `json:"role"`
	TradeID        string        `json:"trdId"`
	FeeCurrency    currency.Code `json:"feeCcy"`
	FeeAmount      types.Number  `json:"feeAmt"`
	DeductCurrency currency.Code `json:"deductCcy"`
	DeductAmount   types.Number  `json:"deductAmt"`
	FillPrice      types.Number  `json:"fpx"`
	FillQuantity   types.Number  `json:"fqty"`
	UpdateTime     types.Time    `json:"uTime"`
}

// FuturesOrderDetails represents a futures v3 order detail
type FuturesOrderDetails struct {
	Symbol                     currency.Pair     `json:"symbol"`
	Side                       order.Side        `json:"side"`
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
	DeductCurrency             currency.Code     `json:"deductCcy"`
	ExecQuantity               types.Number      `json:"execQty"`
	FeeAmount                  types.Number      `json:"feeAmt"`
	FeeCurrency                currency.Code     `json:"feeCcy"`
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
	AutoDeleveraging       string        `json:"adl"`
	AvailQuantity          types.Number  `json:"availQty"`
	CreationTime           types.Time    `json:"cTime"`
	InitialMargin          types.Number  `json:"im"`
	Leverage               types.Number  `json:"lever"`
	LiquidationPrice       types.Number  `json:"liqPx"`
	MarkPrice              types.Number  `json:"markPx"`
	IsolatedPositionMargin string        `json:"mgn"`
	MarginMode             string        `json:"mgnMode"`
	PositionSide           string        `json:"posSide"`
	MarginRatio            types.Number  `json:"mgnRatio"`
	MaintenanceMargin      string        `json:"mm"`
	MaxWithdrawnAmount     types.Number  `json:"maxWAmt"`
	OpenAveragePrice       types.Number  `json:"openAvgPx"`
	ProfitAndLoss          types.Number  `json:"pnl"`
	Quantity               types.Number  `json:"qty"`
	Side                   string        `json:"side"`
	State                  string        `json:"state"`
	Symbol                 currency.Pair `json:"symbol"`
	UpdateTime             types.Time    `json:"uTime"`
	UnrealizedPNL          types.Number  `json:"upl"`
	UnrealizedPNLRatio     types.Number  `json:"uplRatio"`
	CloseAveragePrice      string        `json:"closeAvgPx"`
	ClosedQuantity         string        `json:"closedQty"`
	FFee                   string        `json:"fFee"`
	Fee                    string        `json:"fee"`
	ID                     string        `json:"id"`
	LastPrice              types.Number  `json:"lastPx"`
	IndexPrice             types.Number  `json:"indexPx"`
	AccountType            string        `json:"actType"`
	TakeProfitTriggerPrice types.Number  `json:"tpTrgPx"`
	StopLossTriggerPrice   types.Number  `json:"slTrgPx"`
}

// AdjustFuturesMarginResponse represents a response data after adjusting futures margin positions
type AdjustFuturesMarginResponse struct {
	Amount       types.Number  `json:"amt"`
	Leverage     types.Number  `json:"lever"`
	Symbol       currency.Pair `json:"symbol"`
	PositionSide string        `json:"posSide"`
	OrderType    string        `json:"type"`
}

// FuturesLeverage represents futures symbols leverage information
type FuturesLeverage struct {
	Leverage     types.Number  `json:"lever"`
	MarginMode   string        `json:"mgnMode"`
	PositionSide string        `json:"posSide"`
	Symbol       currency.Pair `json:"symbol"`
}

// FuturesOrderbook represents an orderbook data for v3 futures instruments
type FuturesOrderbook struct {
	Asks      orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids      orderbook.LevelsArrayPriceAmount `json:"bids"`
	Scale     types.Number                     `json:"s"`
	Timestamp types.Time                       `json:"ts"`
}

// WSFuturesOrderbook represents an orderbook data for v3 websocket futures instruments
type WSFuturesOrderbook struct {
	ID           int64                            `json:"id"`
	Symbol       currency.Pair                    `json:"s"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	CreationTime types.Time                       `json:"cT"`
	Timestamp    types.Time                       `json:"ts"`
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
	return json.Unmarshal(data, &[9]any{&v.LowestPrice, &v.HighestPrice, &v.OpeningPrice, &v.ClosingPrice, &v.QuoteAmount, &v.BaseAmount, &v.Trades, &v.StartTime, &v.EndTime})
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

// LiquidationPrice represents a liquidiation price detail for an instrument
type LiquidationPrice struct {
	Symbol                         string       `json:"symbol"`
	PositionSide                   string       `json:"posSide"`
	Side                           string       `json:"side"`
	Size                           types.Number `json:"sz"`
	PriceOfCommissionedTransaction types.Number `json:"bkPx"`
	UpdateTime                     types.Time   `json:"uTime"`
}

// FuturesTickerDetails represents a v3 futures instrument ticker detail
type FuturesTickerDetails struct {
	Symbol       currency.Pair `json:"s"`
	OpeningPrice types.Number  `json:"o"`
	LowPrice     types.Number  `json:"l"`
	HighPrice    types.Number  `json:"h"`
	ClosingPrice types.Number  `json:"c"`
	Quantity     types.Number  `json:"qty"`
	Amount       types.Number  `json:"amt"`
	TradeCount   int64         `json:"tC"`
	StartTime    types.Time    `json:"sT"`
	EndTime      types.Time    `json:"cT"`
	DailyPrice   types.Number  `json:"dC"`
	DN           string        `json:"dN"` // TODO: give the proper naming when the documentation is updated
	BestBidPrice types.Number  `json:"bPx"`
	BestBidSize  types.Number  `json:"bSz"`
	BestAskPrice types.Number  `json:"aPx"`
	BestAskSize  types.Number  `json:"aSz"`
	MarkPrice    types.Number  `json:"mPx"`
	Timestamp    types.Time    `json:"ts"`
}

// InstrumentIndexPrice represents a symbols index price
type InstrumentIndexPrice struct {
	Symbol     currency.Pair `json:"symbol"`
	Timestamp  types.Time    `json:"ts"`
	IndexPrice types.Number  `json:"iPx"`
}

// IndexPriceComponent represents an index price component detail
type IndexPriceComponent struct {
	Symbol currency.Pair `json:"s"`
	Price  types.Number  `json:"px"`
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
	return json.Unmarshal(data, &[6]any{&v.OpenPrice, &v.HighPrice, &v.LowestPrice, &v.ClosingPrice, &v.StartTime, &v.EndTime})
}

// FuturesMarkPrice represents a mark price instance
type FuturesMarkPrice struct {
	MarkPrice types.Number  `json:"mPx"`
	Symbol    currency.Pair `json:"symbol"`
	Timestamp types.Time    `json:"ts"`
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
	return json.Unmarshal(data, &[6]any{&v.OpeningPrice, &v.HighestPrice, &v.LowestPrice, &v.ClosingPrice, &v.StartTime, &v.EndTime})
}

// ProductDetail represents basic information of the all product
type ProductDetail struct {
	Alias                 string        `json:"alias"`
	BaseAsset             string        `json:"bAsset"`
	BaseCurrency          string        `json:"bCcy"`
	ContractType          string        `json:"ctType"`
	ContractValue         types.Number  `json:"ctVal"`
	InitialMarginRate     types.Number  `json:"iM"`
	Leverage              types.Number  `json:"lever"`
	LotSize               types.Number  `json:"lotSz"`
	MaintenanceMarginRate types.Number  `json:"mM"`
	MaximumRiskLimit      types.Number  `json:"mR"`
	MaxLeverage           string        `json:"maxLever"`
	MaxPrice              types.Number  `json:"maxPx"`
	MaxQuantity           types.Number  `json:"maxQty"`
	MinPrice              types.Number  `json:"minPx"`
	MinQuantity           types.Number  `json:"minQty"`
	MinSize               types.Number  `json:"minSz"`
	ListingDate           types.Time    `json:"oDate"`
	PriceScale            string        `json:"pxScale"`
	QuoteCurrency         currency.Code `json:"qCcy"`
	SettlementCurrency    currency.Code `json:"sCcy"`
	Status                string        `json:"status"`
	Symbol                currency.Pair `json:"symbol"`
	TickSize              types.Number  `json:"tSz"`
	TradableStartTime     types.Time    `json:"tradableStartTime"`
	VisibleStartTime      types.Time    `json:"visibleStartTime"`
	OrderPriceRange       types.Number  `json:"ordPxRange"`
	MarketMaxQty          types.Number  `json:"marketMaxQty"`
	LimitMaxQty           types.Number  `json:"limitMaxQty"`
}

// FuturesFundingRate represents symbols funding rate information
type FuturesFundingRate struct {
	Symbol                   currency.Pair `json:"s"`
	FundingRate              types.Number  `json:"fR"`
	FundingRateSettleTime    types.Time    `json:"fT"`
	NextPredictedFundingRate types.Number  `json:"nFR"`
	NextFundingTime          types.Time    `json:"nFT"`
	Timestamp                types.Time    `json:"ts"`
}

// OpenInterestData represents an open interest data
type OpenInterestData struct {
	CurrentOpenInterest types.Number  `json:"oInterest"`
	Symbol              currency.Pair `json:"s"`
}

// InsuranceFundInfo represents an insurance fund information of a currency
type InsuranceFundInfo struct {
	Amount     types.Number  `json:"amt"`
	Currency   currency.Code `json:"ccy"`
	UpdateTime types.Time    `json:"uTime"`
}

// RiskLimit represents a risk limit of futures instrument
type RiskLimit struct {
	Symbol                 currency.Pair `json:"symbol"`
	MarginMode             string        `json:"mgnMode"`
	Tier                   string        `json:"tier"`
	MaxLeverage            types.Number  `json:"maxLever"`
	MaintenanceMarginRatio types.Number  `json:"mMRatio"`
	MaxSize                types.Number  `json:"maxSize"`
	MinSize                types.Number  `json:"minSize"`
}

// WsFuturesCandlesctick represents a kline data for futures instrument
type WsFuturesCandlesctick struct {
	Symbol       currency.Pair
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
	return json.Unmarshal(data, &[11]any{&o.Symbol, &o.LowestPrice, &o.HighestPrice, &o.OpenPrice, &o.ClosePrice, &o.Amount, &o.Quantity, &o.Trades, &o.StartTime, &o.EndTime, &o.PushTime})
}

// FuturesTrades represents a futures trades detail
type FuturesTrades struct {
	ID           int64         `json:"id"`
	Timestamp    types.Time    `json:"ts"`
	Symbol       currency.Pair `json:"s"`
	Price        types.Number  `json:"px"`
	Quantity     types.Number  `json:"qty"`
	Amount       types.Number  `json:"amt"`
	Side         string        `json:"side"`
	CreationTime types.Time    `json:"cT"`
}

// WsFuturesMarkAndIndexPriceCandle represents a websocket k-line data for mark/index candlestick data
type WsFuturesMarkAndIndexPriceCandle struct {
	OpeningPrice  types.Number
	HighestPrice  types.Number
	LowestPrice   types.Number
	ClosingPrice  types.Number
	StartTime     types.Time
	EndTime       types.Time
	Symbol        currency.Pair
	PushTimestamp types.Time
}

// UnmarshalJSON deserializes byte data into WsFuturesMarkAndIndexPriceCandle instance
func (v *WsFuturesMarkAndIndexPriceCandle) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[8]any{&v.Symbol, &v.LowestPrice, &v.HighestPrice, &v.OpeningPrice, &v.ClosingPrice, &v.StartTime, &v.EndTime, &v.PushTimestamp})
}

// MarginModeSwitchResponse represents a response detail after switching margin mode for a symbol
type MarginModeSwitchResponse struct {
	MarginMode string        `json:"mgnMode"`
	Symbol     currency.Pair `json:"symbol"`
}

// ContractLimitPrice holds a contracts highest buy price and lowest sell price limits
type ContractLimitPrice struct {
	Symbol    currency.Pair `json:"symbol"`
	BuyLimit  float64       `json:"buyLmt"`
	SellLimit float64       `json:"sellLmt"`
}

// FuturesSubscriptionResp represents a subscription response item.
type FuturesSubscriptionResp struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
	Action  string          `json:"action"`
	Event   string          `json:"event"`
	Message string          `json:"message"`
}

// FuturesOrders represents a paginated list of Futures orders.
type FuturesOrders struct {
	CurrentPage int64          `json:"currentPage"`
	PageSize    int64          `json:"pageSize"`
	TotalNum    int64          `json:"totalNum"`
	TotalPage   int64          `json:"totalPage"`
	Items       []FuturesOrder `json:"items"`
}

// FuturesOrder represents a futures order detail.
type FuturesOrder struct {
	OrderID             string            `json:"id"`
	Symbol              currency.Pair     `json:"symbol"`
	OrderType           string            `json:"type"`
	Side                string            `json:"side"`
	Price               types.Number      `json:"price"`
	Size                float64           `json:"size"`
	Value               types.Number      `json:"value"`
	FilledValue         types.Number      `json:"filledValue"`
	FilledSize          float64           `json:"filledSize"`
	SelfTradePrevention string            `json:"stp"`
	Stop                string            `json:"stop"`
	StopPriceType       string            `json:"stopPriceType"`
	StopTriggered       bool              `json:"stopTriggered"`
	StopPrice           float64           `json:"stopPrice"`
	TimeInForce         order.TimeInForce `json:"timeInForce"`
	PostOnly            bool              `json:"postOnly"`
	Hidden              bool              `json:"hidden"`
	Iceberg             bool              `json:"iceberg"`
	VisibleSize         float64           `json:"visibleSize"`
	Leverage            types.Number      `json:"leverage"`
	ForceHold           bool              `json:"forceHold"`
	CloseOrder          bool              `json:"closeOrder"`
	ReduceOnly          bool              `json:"reduceOnly"`
	ClientOrderID       string            `json:"clientOid"`
	Remark              string            `json:"remark"`
	IsActive            bool              `json:"isActive"`
	CancelExist         bool              `json:"cancelExist"`
	CreatedAt           types.Time        `json:"createdAt"`
	SettleCurrency      currency.Code     `json:"settleCurrency"`
	Status              string            `json:"status"`
	UpdatedAt           types.Time        `json:"updatedAt"`
	OrderTime           types.Time        `json:"orderTime"`

	MarginType int64 `json:"marginType"` // Margin Mode, 0 (Isolated) or 1 (Cross)
	Trades     []struct {
		FeePay  float64 `json:"feePay"`
		TradeID string  `json:"tradeId"`
	} `json:"trades"`
}

// AuthenticationResponse represents an authentication response for futures websocket connection
type AuthenticationResponse struct {
	Data struct {
		Success   bool       `json:"success"`
		Message   string     `json:"message"`
		Timestamp types.Time `json:"ts"`
	} `json:"data"`
	Channel string `json:"channel"`
}
