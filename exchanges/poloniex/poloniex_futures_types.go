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

// MarginMode represents margin mode type for futures orders
type MarginMode margin.Type

// MarshalText implements encoding.TextMarshaler and serializes MarginMode to a string
func (m MarginMode) MarshalText() ([]byte, error) {
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

// FuturesAccountBalance represents a futures account balance detail
type FuturesAccountBalance struct {
	State                   string                           `json:"state"`
	Equity                  types.Number                     `json:"eq"`
	IsolatedEquity          types.Number                     `json:"isoEq"`
	InitialMargin           types.Number                     `json:"im"`
	MaintenanceMargin       types.Number                     `json:"mm"`
	MaintenanceMarginRate   types.Number                     `json:"mmr"`
	UnrealizedProfitAndLoss types.Number                     `json:"upl"`
	AvailableMargin         types.Number                     `json:"availMgn"`
	CreationTime            types.Time                       `json:"cTime"`
	UpdateTime              types.Time                       `json:"uTime"`
	Details                 []*FuturesCurrencyAccountBalance `json:"details"`
}

// FuturesCurrencyAccountBalance holds a futures account currency balance detail
type FuturesCurrencyAccountBalance struct {
	Currency              currency.Code `json:"ccy"`
	Equity                types.Number  `json:"eq"`
	IsolatedEquity        types.Number  `json:"isoEq"`
	Available             types.Number  `json:"avail"`
	TradeHold             types.Number  `json:"trdHold"`
	UnrealisedPNL         types.Number  `json:"upl"`
	IsolatedAvailable     types.Number  `json:"isoAvail"`
	IsolatedHold          types.Number  `json:"isoHold"`
	IsolatedUPL           types.Number  `json:"isoUpl"`
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
	Symbol                  currency.Pair `json:"symbol"`
	Side                    string        `json:"side"`
	MarginMode              MarginMode    `json:"mgnMode"`
	PositionSide            order.Side    `json:"posSide"`
	OrderType               OrderType     `json:"type,omitempty"`
	ClientOrderID           string        `json:"clOrdId,omitempty"`
	Price                   float64       `json:"px,omitempty,string"`
	Size                    float64       `json:"sz,omitempty,string"`
	ReduceOnly              bool          `json:"reduceOnly,omitempty"`
	TimeInForce             TimeInForce   `json:"timeInForce,omitempty"`
	SelfTradePreventionMode string        `json:"stpMode,omitempty"`
}

// FuturesOrderIDResponse represents a futures order creation response
type FuturesOrderIDResponse struct {
	ClientOrderID string `json:"clOrdId"`
	OrderID       string `json:"ordId"`
	Code          int64  `json:"code"`
	Message       string `json:"msg"`
}

func (s *FuturesOrderIDResponse) Error() error {
	if s.Code != 0 && s.Code != 200 {
		return fmt.Errorf("error code: %d; message: %s", s.Code, s.Message)
	}
	return nil
}

// CancelOrderRequest represents a single order cancellation parameters
type CancelOrderRequest struct {
	Symbol        currency.Pair `json:"symbol"`
	OrderID       string        `json:"ordId,omitempty"`
	ClientOrderID string        `json:"clOrdId,omitempty"`
}

// CancelFuturesOrdersRequest represents cancel futures order request parameters
type CancelFuturesOrdersRequest struct {
	Symbol         currency.Pair `json:"symbol"`
	OrderIDs       []string      `json:"ordIds,omitempty"`
	ClientOrderIDs []string      `json:"clOrdIds,omitempty"`
}

// FuturesTradeFill represents a trade executions
type FuturesTradeFill struct {
	ID             string        `json:"id"`
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
	UpdateTime     types.Time    `json:"uTime"`
	CreationTime   types.Time    `json:"cTime"`
	FeeRate        types.Number  `json:"feeRate"`
	MarginMode     string        `json:"mgnMode"`
	PositionSide   string        `json:"posSide"`
	OrderType      string        `json:"ordType"`
	Price          types.Number  `json:"px"`
	BaseAmount     types.Number  `json:"qty"`
	Type           string        `json:"type"`
	AccountType    string        `json:"actType"`
	QuoteCurrency  currency.Code `json:"qCcy"`
	Value          types.Number  `json:"value"`
}

// WSTradeFill represents a websocket streamed trade fill data
type WSTradeFill struct {
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
	Type           string        `json:"type"`
	FillPrice      types.Number  `json:"fpx"`
	FillQuantity   types.Number  `json:"fqty"`
	UpdateTime     types.Time    `json:"uTime"`
	Timestamp      types.Time    `json:"ts"`
}

// FuturesOrderDetails represents a futures v3 order detail
type FuturesOrderDetails struct {
	Symbol                     currency.Pair     `json:"symbol"`
	Side                       order.Side        `json:"side"`
	MarginMode                 string            `json:"mgnMode"`
	PositionSide               string            `json:"posSide"`
	AccountType                string            `json:"actType"`
	QuoteCurrency              currency.Code     `json:"qCcy"`
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
	ExecutedQuantity           types.Number      `json:"execQty"`
	FeeAmount                  types.Number      `json:"feeAmt"`
	FeeCurrency                currency.Code     `json:"feeCcy"`
	Leverage                   types.Number      `json:"lever"`
	ReduceOnly                 types.Boolean     `json:"reduceOnly"`
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
	CancelReason               string            `json:"cancelReason"`
}

// FuturesWebsocketOrderDetails represents a futures websocket order detail
type FuturesWebsocketOrderDetails struct {
	Symbol           currency.Pair     `json:"symbol"`
	Side             order.Side        `json:"side"`
	OrderType        string            `json:"type"`
	MarginMode       string            `json:"mgnMode"`
	TimeInForce      order.TimeInForce `json:"timeInForce"`
	ClientOrderID    string            `json:"clOrdId"`
	Size             types.Number      `json:"sz"`
	Price            types.Number      `json:"px"`
	ReduceOnly       bool              `json:"reduceOnly"`
	PositionSide     string            `json:"posSide"`
	OrderID          string            `json:"ordId"`
	State            string            `json:"state"`
	CancelReason     string            `json:"cancelReason"`
	Source           string            `json:"source"`
	AveragePrice     types.Number      `json:"avgPx"`
	ExecutedQuantity types.Number      `json:"execQty"`
	ExecutedAmount   types.Number      `json:"execAmt"`
	FeeCurrency      currency.Code     `json:"feeCcy"`
	FeeAmount        types.Number      `json:"feeAmt"`
	DeductCurrency   currency.Code     `json:"deductCcy"`
	DeductAmount     types.Number      `json:"deductAmt"`
	FillSize         types.Number      `json:"fillSz"`
	CreationTime     types.Time        `json:"cTime"`
	UpdateTime       types.Time        `json:"uTime"`
	Timestamp        types.Time        `json:"ts"`
}

// FuturesPosition represents a v3 futures position detail
type FuturesPosition struct {
	ID                string        `json:"id"`
	Symbol            currency.Pair `json:"symbol"`
	Side              string        `json:"side"`
	MarginMode        string        `json:"mgnMode"`
	PositionSide      string        `json:"posSide"`
	OpenAveragePrice  types.Number  `json:"openAvgPx"`
	CloseAveragePrice string        `json:"closeAvgPx"`
	ClosedQuantity    string        `json:"closedQty"`
	AvailableQuantity types.Number  `json:"availQty"`
	BaseAmount        types.Number  `json:"qty"`
	ProfitAndLoss     types.Number  `json:"pnl"`
	Fee               string        `json:"fee"`
	FundingFee        string        `json:"fFee"`
	State             string        `json:"state"`
	CreationTime      types.Time    `json:"cTime"`
	UpdateTime        types.Time    `json:"uTime"`
}

// OpenFuturesPosition represents a v3 futures open position detail
type OpenFuturesPosition struct {
	Symbol                 currency.Pair `json:"symbol"`
	Side                   string        `json:"side"`
	MarginMode             string        `json:"mgnMode"`
	PositionSide           string        `json:"posSide"`
	OpenAveragePrice       string        `json:"openAvgPx"`
	BaseAmount             string        `json:"qty"`
	AvailableQuantity      types.Number  `json:"availQty"`
	Leverage               types.Number  `json:"lever"`
	AutoDeleveraging       string        `json:"adl"`
	LiquidationPrice       types.Number  `json:"liqPx"`
	InitialMargin          types.Number  `json:"im"`
	MaintenanceMargin      types.Number  `json:"mm"`
	IsolatedPositionMargin string        `json:"mgn"`
	MaxWithdrawalAmount    types.Number  `json:"maxWAmt"`
	UnrealizedPNL          types.Number  `json:"upl"`
	UnrealizedPNLRatio     types.Number  `json:"uplRatio"`
	ProfitAndLoss          types.Number  `json:"pnl"`
	MarkPrice              types.Number  `json:"markPx"`
	LastPrice              types.Number  `json:"lastPx"`
	IndexPrice             types.Number  `json:"indexPx"`
	MarginRatio            types.Number  `json:"mgnRatio"`
	State                  string        `json:"state"`
	AccountType            string        `json:"actType"`
	TakeProfitTriggerPrice types.Number  `json:"tpTrgPx"`
	StopLossTriggerPrice   types.Number  `json:"slTrgPx"`
	CreateTime             types.Time    `json:"cTime"`
	UpdateTime             types.Time    `json:"uTime"`
}

// WsFuturesPosition represents a futures websocket position
type WsFuturesPosition struct {
	Symbol                     currency.Pair `json:"symbol"`
	PositionSide               string        `json:"posSide"`
	Side                       string        `json:"side"`
	MarginMode                 string        `json:"mgnMode"`
	OpenAveragePrice           types.Number  `json:"openAvgPx"`
	BaseAmount                 types.Number  `json:"qty"`
	OldQuantity                types.Number  `json:"oldQty"`
	AvailableQuantity          types.Number  `json:"availQty"`
	Leverage                   uint16        `json:"lever"`
	Fee                        types.Number  `json:"fee"`
	AutoDeleveraging           string        `json:"adl"`
	LiquidationPrice           types.Number  `json:"liqPx"`
	IsolatedPositionMargin     types.Number  `json:"mgn"`
	InitialMargin              types.Number  `json:"im"`
	MaintenanceMargin          types.Number  `json:"mm"`
	UnrealizedPNL              types.Number  `json:"upl"`
	UnrealizedPNLRatio         types.Number  `json:"uplRatio"`
	LatestClosingProfitAndLoss types.Number  `json:"fpnl"`
	MarkPrice                  types.Number  `json:"markPx"`
	MarginRatio                types.Number  `json:"mgnRatio"`
	State                      string        `json:"state"`
	CreateTime                 types.Time    `json:"cTime"`
	UpdateTime                 types.Time    `json:"uTime"`
	Timestamp                  types.Time    `json:"ts"`
	ProfitAndLoss              types.Number  `json:"pnl"`
	FundingFee                 types.Number  `json:"ffee"`
}

// AdjustFuturesMarginResponse represents a response data after adjusting futures margin positions
type AdjustFuturesMarginResponse struct {
	Amount       types.Number  `json:"amt"`
	Leverage     uint8         `json:"lever,string"`
	Symbol       currency.Pair `json:"symbol"`
	PositionSide string        `json:"posSide"`
	OrderType    string        `json:"type"`
}

// FuturesLeverage represents futures symbols leverage information
type FuturesLeverage struct {
	Leverage     uint8         `json:"lever,string"`
	MarginMode   string        `json:"mgnMode"`
	PositionSide string        `json:"posSide"`
	Symbol       currency.Pair `json:"symbol"`
}

// FuturesOpenInterest represents futures open interest valud for symbol
type FuturesOpenInterest struct {
	Symbol       currency.Pair `json:"s"`
	OpenInterest types.Number  `json:"oInterest"`
	Timestamp    types.Time    `json:"ts"`
}

// FuturesLiquidiationOrder represents a futures liquidiation price detail
type FuturesLiquidiationOrder struct {
	Symbol          currency.Pair `json:"s"`
	Side            string        `json:"side"`
	PositionSide    string        `json:"posSide"`
	Size            types.Number  `json:"sz"`
	BankruptcyPrice types.Number  `json:"bkPx"`
	UpdateTime      types.Time    `json:"uTime"`
	Timestamp       types.Time    `json:"ts"`
}

// FuturesLimitPrice represents a futures limit price info
type FuturesLimitPrice struct {
	Timestamp types.Time    `json:"ts"`
	Symbol    currency.Pair `json:"s"`
	BuyLimit  types.Number  `json:"buyLmt"`
	SellLimit types.Number  `json:"sellLmt"`
}

// UserPositionRiskLimit represents a user position risk limit detail
type UserPositionRiskLimit struct {
	Symbol                 currency.Pair `json:"symbol"`
	MarginMode             string        `json:"mgnMode"`
	PositionSide           string        `json:"posSide"`
	Tier                   string        `json:"tier"`
	MaxLeverage            uint16        `json:"maxLever,string"`
	MaintenanceMarginRatio types.Number  `json:"mMRatio"`
	MaxSize                types.Number  `json:"maxSize"`
	MinSize                types.Number  `json:"minSize"`
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
	ID            int64                            `json:"id"`
	LastVersionID int64                            `json:"lid"`
	Symbol        currency.Pair                    `json:"s"`
	Asks          orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids          orderbook.LevelsArrayPriceAmount `json:"bids"`
	CreationTime  types.Time                       `json:"cT"`
	Timestamp     types.Time                       `json:"ts"`
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
	BaseAmount   types.Number `json:"qty"`
	QuoteAmount  types.Number `json:"amt"`
	Side         string       `json:"side"`
	CreationTime types.Time   `json:"cT"`
}

// LiquidationPrice represents a liquidiation price detail for an instrument
type LiquidationPrice struct {
	Symbol                         currency.Pair `json:"symbol"`
	PositionSide                   string        `json:"posSide"`
	Side                           string        `json:"side"`
	Size                           types.Number  `json:"sz"`
	PriceOfCommissionedTransaction types.Number  `json:"bkPx"`
	UpdateTime                     types.Time    `json:"uTime"`
}

// FuturesTickerDetails represents a v3 futures instrument ticker detail
type FuturesTickerDetails struct {
	Symbol       currency.Pair `json:"s"`
	OpeningPrice types.Number  `json:"o"`
	LowPrice     types.Number  `json:"l"`
	HighPrice    types.Number  `json:"h"`
	ClosingPrice types.Number  `json:"c"`
	BaseAmount   types.Number  `json:"qty"`
	QuoteAmount  types.Number  `json:"amt"`
	TradeCount   int64         `json:"tC"`
	StartTime    types.Time    `json:"sT"`
	EndTime      types.Time    `json:"cT"`
	DailyPrice   types.Number  `json:"dC"`
	DisplayName  string        `json:"dN"`
	BestBidPrice types.Number  `json:"bPx"`
	BestBidSize  types.Number  `json:"bSz"`
	BestAskPrice types.Number  `json:"aPx"`
	BestAskSize  types.Number  `json:"aSz"`
	MarkPrice    types.Number  `json:"mPx"`
	IndexPrice   types.Number  `json:"iPx"`
	Timestamp    types.Time    `json:"ts"`
}

// InstrumentIndexPrice represents a symbols index price
type InstrumentIndexPrice struct {
	Symbol    currency.Pair `json:"s"`
	Timestamp types.Time    `json:"ts"`
}

// WSInstrumentIndexPrice represents a symbols index price of websocket response
type WSInstrumentIndexPrice struct {
	Symbol     currency.Pair `json:"s"`
	Timestamp  types.Time    `json:"ts"`
	IndexPrice types.Number  `json:"iPx"`
}

// IndexPriceComponent represents an index price component detail
type IndexPriceComponent struct {
	Symbol currency.Pair        `json:"s"`
	Price  types.Number         `json:"px"`
	Cs     []SymbolPriceDetails `json:"cs"`
}

// SymbolPriceDetails represents symbol's price from different exchanges and its weight factor.
type SymbolPriceDetails struct {
	Exchange              string       `json:"e"`
	WeightFactor          types.Number `json:"w"`
	TradingPairPrice      types.Number `json:"sPx"`
	TradingPairIndexPrice types.Number `json:"cPx"`
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
	return json.Unmarshal(data, &[6]any{&v.LowestPrice, &v.HighPrice, &v.OpenPrice, &v.ClosingPrice, &v.StartTime, &v.EndTime})
}

// FuturesMarkPrice represents a mark price instance
type FuturesMarkPrice struct {
	MarkPrice types.Number  `json:"mPx"`
	Symbol    currency.Pair `json:"s"`
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
	return json.Unmarshal(data, &[6]any{&v.LowestPrice, &v.HighestPrice, &v.OpeningPrice, &v.ClosingPrice, &v.StartTime, &v.EndTime})
}

// WSProductDetail represents websocket response of basic information of all products
type WSProductDetail struct {
	Symbol                string        `json:"s"`
	VisibleStartTime      types.Time    `json:"visibleST"`
	TradableStartTime     types.Time    `json:"tradableST"`
	PriceScale            string        `json:"pxScale"`
	LotSize               float64       `json:"lotSz"`
	MinSize               float64       `json:"minSz"`
	ContractFeeValue      types.Number  `json:"ctVal"`
	Status                string        `json:"status"`
	MaxPrice              types.Number  `json:"maxPx"`
	MinPrice              types.Number  `json:"minPx"`
	MaxQuantity           types.Number  `json:"maxQty"`
	MinQuantity           types.Number  `json:"minQty"`
	MaxLeverage           uint16        `json:"maxLever,string"`
	Leverage              string        `json:"lever"`
	OrderPriceRange       string        `json:"ordPxRange"`
	ContractType          string        `json:"ctType"`
	Alias                 string        `json:"alias"`
	MarketMaxQuantity     types.Number  `json:"marketMaxQty"`
	LimitMaxQuantity      types.Number  `json:"limitMaxQty"`
	Timestamp             types.Time    `json:"ts"`
	BaseCurrency          currency.Code `json:"bCcy"`
	UnderlyingAsset       string        `json:"bAsset"`
	QuoteCurrency         currency.Code `json:"qCcy"`
	SettlementCurrency    currency.Code `json:"sCcy"`
	TickSize              types.Number  `json:"tSz"`
	ListingDate           types.Time    `json:"oDate"`
	InitialMarginRate     types.Number  `json:"iM"`
	MaximumRiskLimit      types.Number  `json:"mR"`
	MaintenanceMarginRate types.Number  `json:"mM"`
}

// ProductDetail represents basic information of the all product
type ProductDetail struct {
	Alias                 string        `json:"alias"`
	BaseAsset             string        `json:"bAsset"`
	BaseCurrency          string        `json:"bCcy"`
	ContractType          string        `json:"ctType"`
	ContractValue         types.Number  `json:"ctVal"`
	InitialMarginRate     types.Number  `json:"iM"`
	Leverage              uint16        `json:"lever,string"`
	MaxLeverage           uint16        `json:"maxLever,string"`
	SizePrecision         uint16        `json:"lotSz"`
	MaintenanceMarginRate types.Number  `json:"mM"`
	MaximumRiskLimit      types.Number  `json:"mR"`
	MaxPrice              types.Number  `json:"maxPx"`
	MaxQuantity           types.Number  `json:"maxQty"`
	MinPrice              types.Number  `json:"minPx"`
	MinQuantity           types.Number  `json:"minQty"`
	MinSize               uint16        `json:"minSz"`
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
}

// WSFuturesFundingRate represents symbols funding rate information pushed through websocket
type WSFuturesFundingRate struct {
	Symbol                   currency.Pair `json:"s"`
	FundingRate              types.Number  `json:"fR"`
	FundingRateSettleTime    types.Time    `json:"fT"`
	NextPredictedFundingRate types.Number  `json:"nFR"`
	NextFundingTime          types.Time    `json:"nFT"`
	PushTime                 types.Time    `json:"ts"`
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
	Tier                   uint8         `json:"tier,string"`
	MaxLeverage            uint16        `json:"maxLever,string"`
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
	BaseAmount   types.Number
	QuoteAmount  types.Number
	Trades       uint64
	StartTime    types.Time
	EndTime      types.Time
	PushTime     types.Time
}

// UnmarshalJSON deserializes byte data into futures candlesticks into *WsFuturesCandlesctick
func (o *WsFuturesCandlesctick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[11]any{&o.Symbol, &o.LowestPrice, &o.HighestPrice, &o.OpenPrice, &o.ClosePrice, &o.QuoteAmount, &o.BaseAmount, &o.Trades, &o.StartTime, &o.EndTime, &o.PushTime})
}

// FuturesTrades represents a futures trades detail
type FuturesTrades struct {
	ID           int64         `json:"id"`
	Timestamp    types.Time    `json:"ts"`
	Symbol       currency.Pair `json:"s"`
	Price        types.Number  `json:"px"`
	BaseAmount   types.Number  `json:"qty"`
	QuoteAmount  types.Number  `json:"amt"`
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

	MarginType int64           `json:"marginType"` // Margin Mode, 0 (Isolated) or 1 (Cross)
	Trades     []TradeIDAndFee `json:"trades"`
}

// TradeIDAndFee holds a trade ID and fee information
type TradeIDAndFee struct {
	FeePay  float64 `json:"feePay"`
	TradeID string  `json:"tradeId"`
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
