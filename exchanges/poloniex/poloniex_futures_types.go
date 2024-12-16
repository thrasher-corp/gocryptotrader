package poloniex

import (
	"encoding/json"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Contracts represents a list of open contract.
type Contracts struct {
	Code string         `json:"code"`
	Data []ContractItem `json:"data"`
}

// ContractItem represents a single open contract instance.
type ContractItem struct {
	Symbol                  string     `json:"symbol"`
	TakerFixFee             float64    `json:"takerFixFee"`
	NextFundingRateTime     int64      `json:"nextFundingRateTime"`
	MakerFixFee             float64    `json:"makerFixFee"`
	ContractType            string     `json:"type"`
	PredictedFundingFeeRate float64    `json:"predictedFundingFeeRate"`
	TurnoverOf24H           float64    `json:"turnoverOf24h"`
	InitialMargin           float64    `json:"initialMargin"`
	IsDeleverage            bool       `json:"isDeleverage"`
	CreatedAt               types.Time `json:"createdAt"`
	FundingBaseSymbol       string     `json:"fundingBaseSymbol"`
	LowPriceOf24H           float64    `json:"lowPriceOf24h"`
	LastTradePrice          float64    `json:"lastTradePrice"`
	IndexPriceTickSize      float64    `json:"indexPriceTickSize"`
	FairMethod              string     `json:"fairMethod"`
	TakerFeeRate            float64    `json:"takerFeeRate"`
	Order                   int64      `json:"order"`
	UpdatedAt               types.Time `json:"updatedAt"`
	DisplaySettleCurrency   string     `json:"displaySettleCurrency"`
	IndexPrice              float64    `json:"indexPrice"`
	Multiplier              float64    `json:"multiplier"`
	MinOrderQty             float64    `json:"minOrderQty"`
	MaxLeverage             float64    `json:"maxLeverage"`
	FundingQuoteSymbol      string     `json:"fundingQuoteSymbol"`
	QuoteCurrency           string     `json:"quoteCurrency"`
	MaxOrderQty             float64    `json:"maxOrderQty"`
	MaxPrice                float64    `json:"maxPrice"`
	MaintainMargin          float64    `json:"maintainMargin"`
	Status                  string     `json:"status"`
	DisplayNameMap          struct {
		ContractNameKoKR string `json:"contractName_ko-KR"`
		ContractNameZhCN string `json:"contractName_zh-CN"`
		ContractNameEnUS string `json:"contractName_en-US"`
	} `json:"displayNameMap"`
	OpenInterest      string  `json:"openInterest"`
	HighPriceOf24H    float64 `json:"highPriceOf24h"`
	FundingFeeRate    float64 `json:"fundingFeeRate"`
	VolumeOf24H       float64 `json:"volumeOf24h"`
	RiskStep          float64 `json:"riskStep"`
	IsQuanto          bool    `json:"isQuanto"`
	MaxRiskLimit      float64 `json:"maxRiskLimit"`
	RootSymbol        string  `json:"rootSymbol"`
	BaseCurrency      string  `json:"baseCurrency"`
	FirstOpenDate     int64   `json:"firstOpenDate"`
	TickSize          float64 `json:"tickSize"`
	MarkMethod        string  `json:"markMethod"`
	IndexSymbol       string  `json:"indexSymbol"`
	MarkPrice         float64 `json:"markPrice"`
	MinRiskLimit      float64 `json:"minRiskLimit"`
	SettlementFixFee  float64 `json:"settlementFixFee"`
	SettlementSymbol  string  `json:"settlementSymbol"`
	PriceChgPctOf24H  float64 `json:"priceChgPctOf24h"`
	FundingRateSymbol string  `json:"fundingRateSymbol"`
	MakerFeeRate      float64 `json:"makerFeeRate"`
	IsInverse         bool    `json:"isInverse"`
	LotSize           float64 `json:"lotSize"`
	SettleCurrency    string  `json:"settleCurrency"`
	SettlementFeeRate float64 `json:"settlementFeeRate"`
	DefaultLeverage   int     `json:"defaultLeverage"`
	ExpireDate        any     `json:"expireDate"`
	Scale             string  `json:"scale"`
	SettleDate        any     `json:"settleDate"`
}

// TickerInfo represents a ticker information for a single symbol.
type TickerInfo struct {
	Sequence     int64        `json:"sequence"`
	Symbol       string       `json:"symbol"`
	Side         string       `json:"side"`
	Size         float64      `json:"size"`
	Price        types.Number `json:"price"`
	BestBidSize  float64      `json:"bestBidSize"`
	BestBidPrice types.Number `json:"bestBidPrice"`
	BestAskSize  float64      `json:"bestAskSize"`
	BestAskPrice types.Number `json:"bestAskPrice"`
	TradeID      string       `json:"tradeId"`
	Timestamp    types.Time   `json:"ts"`
}

// TickerDetail represents a ticker detail information.
type TickerDetail struct {
	Code string     `json:"code"`
	Data TickerInfo `json:"data"`
}

// TickersDetail represents a list of tickers detail.
type TickersDetail struct {
	Code string       `json:"code"`
	Data []TickerInfo `json:"data"`
}

// Orderbook item detail for a single symbol
type Orderbook struct {
	Code string `json:"code"`
	Data struct {
		Symbol   string           `json:"symbol"`
		Sequence int64            `json:"sequence"`
		Asks     [][]types.Number `json:"asks"`
		Bids     [][]types.Number `json:"bids"`

		// Added for level2 data.
		Timestamp types.Time `json:"ts"`
	} `json:"data"`
}

// GetOBBase creates an orderbook.Base instance from *Orderbook instance.
func (a *Orderbook) GetOBBase() (*orderbook.Base, error) {
	cp, err := currency.NewPairFromString(a.Data.Symbol)
	if err != nil {
		return nil, err
	}
	base := &orderbook.Base{
		Pair:         cp,
		Asset:        asset.Futures,
		LastUpdateID: a.Data.Sequence,
	}
	base.Asks = make(orderbook.Tranches, len(a.Data.Asks))
	for i := range a.Data.Asks {
		base.Asks[i].Price = a.Data.Asks[i][0].Float64()
		base.Asks[i].Amount = a.Data.Asks[i][1].Float64()
	}
	base.Bids = make(orderbook.Tranches, len(a.Data.Bids))
	for i := range a.Data.Bids {
		base.Bids[i].Price = a.Data.Bids[i][0].Float64()
		base.Bids[i].Amount = a.Data.Bids[i][1].Float64()
	}
	return base, nil
}

// OrderbookChange represents an orderbook change data
type OrderbookChange struct {
	Symbol   string `json:"symbol"`
	Sequence int64  `json:"sequence"`
	Change   string `json:"change"`
}

// OrderbookChanges represents a list of orderbook data change
type OrderbookChanges struct {
	Code string            `json:"code"`
	Data []OrderbookChange `json:"data"`
}

// Level3PullingMessage represents a level 3 orderbook data pulled.
type Level3PullingMessage struct {
	Symbol   string `json:"symbol"`
	Sequence int    `json:"sequence"`
	Change   string `json:"change"`
}

// Level3PullingMessageResponse represents response for orderbook level 3 pulled missing data.
type Level3PullingMessageResponse struct {
	Code string                 `json:"code"`
	Data []Level3PullingMessage `json:"data"`
}

// TransactionHistory represents a trades for a symbol.
type TransactionHistory struct {
	Code string `json:"code"`
	Data []struct {
		Sequence     int64        `json:"sequence"`
		Side         string       `json:"side"`
		Size         types.Number `json:"size"`
		Price        types.Number `json:"price"`
		TakerOrderID string       `json:"takerOrderId"`
		MakerOrderID string       `json:"makerOrderId"`
		TradeID      string       `json:"tradeId"`
		Timestamp    types.Time   `json:"ts"`
	} `json:"data"`
}

// IndexInfo represents an interest rate detail.
type IndexInfo struct {
	DataList []struct {
		Symbol      string  `json:"symbol"`
		Granularity int     `json:"granularity"`
		TimePoint   int64   `json:"timePoint"`
		Value       float64 `json:"value"`

		DecomposionList []struct {
			Exchange string  `json:"exchange"`
			Price    float64 `json:"price"`
			Weight   float64 `json:"weight"`
		} `json:"decomposionList"`
	} `json:"dataList"`
	HasMore bool `json:"hasMore"`
}

// MarkPriceDetail represents the current mark price.
type MarkPriceDetail struct {
	Symbol      string     `json:"symbol"`
	Granularity int64      `json:"granularity"`
	TimePoint   types.Time `json:"timePoint"`
	MarkPrice   float64    `json:"value"`
	IndexPrice  float64    `json:"indexPrice"`
}

// FundingRate represents a funding rate response.
type FundingRate struct {
	Symbol         string     `json:"symbol"`
	Granularity    int64      `json:"granularity"`
	TimePoint      types.Time `json:"timePoint"`
	Value          float64    `json:"value"`
	PredictedValue float64    `json:"predictedValue"`
}

// ServerTimeResponse represents a server time response.
type ServerTimeResponse struct {
	Code string     `json:"code"`
	Msg  string     `json:"msg"`
	Data types.Time `json:"data"`
}

// ServiceStatus represents system service status response.
type ServiceStatus struct {
	Code string `json:"code"`
	Data struct {
		Status  string `json:"status"` // possible values: open, close, cancelonly
		Message string `json:"msg"`    // possible values: remark for operation
	} `json:"data"`
}

// KlineChartResponse represents K chart.
type KlineChartResponse struct {
	Code string      `json:"code"`
	Data [][]float64 `json:"data"`
}

// ExtractKlineChart converts the []float64 data into klineChartData instance.
func (a *KlineChartResponse) ExtractKlineChart() []KlineChartData {
	chart := make([]KlineChartData, len(a.Data))
	for i := range a.Data {
		chart[i] = KlineChartData{
			Timestamp:     time.UnixMilli(int64(a.Data[i][0])),
			EntryPrice:    a.Data[i][1],
			HighestPrice:  a.Data[i][2],
			LowestPrice:   a.Data[i][3],
			ClosePrice:    a.Data[i][4],
			TradingVolume: a.Data[i][5],
		}
	}
	return chart
}

// KlineChartData represents K chart.
type KlineChartData struct {
	Timestamp     time.Time
	EntryPrice    float64
	HighestPrice  float64
	LowestPrice   float64
	ClosePrice    float64
	TradingVolume float64
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
		Currency              string       `json:"ccy"`
		Equity                types.Number `json:"eq"`
		IsoEquity             types.Number `json:"isoEq"`
		Available             types.Number `json:"avail"`
		TrdHold               types.Number `json:"trdHold"`
		UnrealisedPNL         types.Number `json:"upl"`
		IsoAvailable          types.Number `json:"isoAvail"`
		IsoHold               string       `json:"isoHold"`
		IsoUpl                string       `json:"isoUpl"`
		InitialMargin         types.Number `json:"im"`
		MaintenanceMargin     types.Number `json:"mm"`
		MaintenanceMarginRate types.Number `json:"mmr"`
		InitialMarginRate     types.Number `json:"imr"`
		CreationTime          types.Time   `json:"cTime"`
		UpdateTime            types.Time   `json:"uTime"`
	} `json:"details"`
}

// BillDetail represents a bill type detail information
type BillDetail struct {
	ID           string       `json:"id"`
	AccountType  string       `json:"actType"`
	BillType     string       `json:"type"`
	Currency     string       `json:"ccy"`
	CreationTime types.Time   `json:"cTime"`
	Size         types.Number `json:"sz"`
	Symbol       string       `json:"symbol"`
	MarginMode   string       `json:"mgnMode"`
	PositionSide string       `json:"posSide"`
}

// FuturesV2Params represents a futures order parameters
type FuturesV2Params struct {
	Symbol                  string  `json:"symbol"`
	Side                    string  `json:"side"`
	MarginMode              string  `json:"mgnMode"`
	PositionSide            string  `json:"posSide"`
	OrderType               string  `json:"type,omitempty"`
	ClientOrderID           string  `json:"clOrdId,omitempty,string"`
	Price                   float64 `json:"px,omitempty,string"`
	Size                    float64 `json:"sz,omitempty"`
	ReduceOnly              bool    `json:"reduceOnly,omitempty"`
	TimeInForce             string  `json:"timeInForce,omitempty"`
	SelfTradePreventionMode string  `json:"stpMode,omitempty"`
}

// FuturesV3OrderIDResponse represents a futures order creation response
type FuturesV3OrderIDResponse struct {
	ClOrdID string `json:"clOrdId"`
	OrdID   string `json:"ordId"`

	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// CancelOrderParams represents a single order cancellation parameters
type CancelOrderParams struct {
	Symbol        string `json:"symbol"`
	OrderID       string `json:"ordId,omitempty"`
	ClientOrderID string `json:"clOrdId,omitempty"`
}

// CancelOrdersParams represents multiple order cancellation parameters
type CancelOrdersParams struct {
	Symbol         string   `json:"symbol"`
	OrderIDs       []string `json:"ordIds,omitempty"`
	ClientOrderIDs []string `json:"clOrdIds,omitempty"`
}

// FuturesV3Order represents a futures v3 order detail
type FuturesV3Order struct {
	OrderID                    string       `json:"ordId"`
	AveragePrice               types.Number `json:"avgPx"`
	CreationTime               types.Time   `json:"cTime"`
	ClientOrderID              string       `json:"clOrdId"`
	DeductAmount               types.Number `json:"deductAmt"`
	ExecutedAmount             types.Number `json:"execAmt"`
	DeductCurrency             string       `json:"deductCcy"`
	ExecQuantity               types.Number `json:"execQty"`
	FeeAmount                  types.Number `json:"feeAmt"`
	FeeCurrency                string       `json:"feeCcy"`
	PositionSide               string       `json:"posSide"`
	Leverage                   string       `json:"lever"`
	MarginMode                 string       `json:"mgnMode"`
	Price                      types.Number `json:"px"`
	ReduceOnly                 bool         `json:"reduceOnly"`
	StopLossPrice              types.Number `json:"slPx"`
	Side                       string       `json:"side"`
	StopLossTriggerPrice       string       `json:"slTrgPx"`
	StopLossTriggerPriceType   string       `json:"slTrgPxType"`
	Source                     string       `json:"source"`
	State                      string       `json:"state"`
	SelfTradePreventionMode    string       `json:"stpMode"`
	Symbol                     string       `json:"symbol"`
	Size                       types.Number `json:"sz"`
	TimeInForce                string       `json:"timeInForce"`
	TakeProfitPrice            types.Number `json:"tpPx"`
	TakeProfitTriggerPrice     types.Number `json:"tpTrgPx"`
	TakeProfitTriggerPriceType string       `json:"tpTrgPxType"`
	Type                       string       `json:"type"`
	UpdateTime                 types.Time   `json:"uTime"`
	FeeRate                    types.Number `json:"feeRate"`
	ID                         string       `json:"id"`
	OrderType                  string       `json:"ordType"`
	Quantity                   types.Number `json:"qty"`
	Role                       string       `json:"role"`
	TradeID                    string       `json:"trdId"`
	CancelReason               string       `json:"cancelReason"`
}

// V3FuturesPosition represents a v3 futures position detail
type V3FuturesPosition struct {
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

	CloseAvgPx string `json:"closeAvgPx"`
	ClosedQty  string `json:"closedQty"`
	FFee       string `json:"fFee"`
	Fee        string `json:"fee"`
	ID         string `json:"id"`
}

// AdjustV3FuturesMarginResponse represents a response data after adjusting futures margin positions
type AdjustV3FuturesMarginResponse struct {
	Amount       types.Number `json:"amt"`
	Leverage     types.Number `json:"lever"`
	Symbol       string       `json:"symbol"`
	PositionSide string       `json:"posSide"`
	OrderType    string       `json:"type"`
}

// V3FuturesLeverage represents futures symbols leverage information
type V3FuturesLeverage struct {
	Leverage     types.Number `json:"lever"`
	MarginMode   string       `json:"mgnMode"`
	PositionSide string       `json:"posSide"`
	Symbol       string       `json:"symbol"`
}

// FuturesV3Orderbook represents an orderbook data for v3 futures instruments
type FuturesV3Orderbook struct {
	Asks      [][]types.Number `json:"asks"`
	Bids      [][]types.Number `json:"bids"`
	Depth     types.Number     `json:"s"`
	Timestamp types.Time       `json:"ts"`
}

// V3FuturesCandle represents a kline data for v3 futures instrument
type V3FuturesCandle struct {
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
func (v *V3FuturesCandle) UnmarshalJSON(data []byte) error {
	target := [11]any{&v.LowestPrice, &v.HighestPrice, &v.OpeningPrice, &v.ClosingPrice, &v.QuoteAmount, &v.BaseAmount, &v.Trades, &v.StartTime, &v.EndTime}
	return json.Unmarshal(data, &target)
}

// V3FuturesExecutionInfo represents a V3 futures instruments execution information
type V3FuturesExecutionInfo struct {
	ID           int64        `json:"id"`
	Price        types.Number `json:"px"`
	Quantity     types.Number `json:"qty"`
	Amount       types.Number `json:"amt"`
	Side         string       `json:"side"`
	CreationTime types.Time   `json:"cT"`
}

// LiquidiationPriceInfo represents a liquidiation price detail for an instrument
type LiquidiationPriceInfo struct {
	Symbol                         string       `json:"symbol"`
	PositionSide                   string       `json:"posSide"`
	Side                           string       `json:"side"`
	Size                           types.Number `json:"sz"`
	PriceOfCommissionedTransaction types.Number `json:"bkPx"`
	UpdateTime                     types.Time   `json:"uTime"`
}

// V3FuturesTickerDetail represents a v3 futures instrument ticker detail
type V3FuturesTickerDetail struct {
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
}

// InstrumentIndexPrice represents a symbols index price
type InstrumentIndexPrice struct {
	Symbol     string       `json:"symbol"`
	IndexPrice types.Number `json:"iPx"`
}
