package poloniex

import (
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
	NextFundingRateTime     types.Time `json:"nextFundingRateTime"`
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
	OpenInterest      string     `json:"openInterest"`
	HighPriceOf24H    float64    `json:"highPriceOf24h"`
	FundingFeeRate    float64    `json:"fundingFeeRate"`
	VolumeOf24H       float64    `json:"volumeOf24h"`
	RiskStep          float64    `json:"riskStep"`
	IsQuanto          bool       `json:"isQuanto"`
	MaxRiskLimit      float64    `json:"maxRiskLimit"`
	RootSymbol        string     `json:"rootSymbol"`
	BaseCurrency      string     `json:"baseCurrency"`
	FirstOpenDate     types.Time `json:"firstOpenDate"`
	TickSize          float64    `json:"tickSize"`
	MarkMethod        string     `json:"markMethod"`
	IndexSymbol       string     `json:"indexSymbol"`
	MarkPrice         float64    `json:"markPrice"`
	MinRiskLimit      float64    `json:"minRiskLimit"`
	SettlementFixFee  float64    `json:"settlementFixFee"`
	SettlementSymbol  string     `json:"settlementSymbol"`
	PriceChgPctOf24H  float64    `json:"priceChgPctOf24h"`
	FundingRateSymbol string     `json:"fundingRateSymbol"`
	MakerFeeRate      float64    `json:"makerFeeRate"`
	IsInverse         bool       `json:"isInverse"`
	LotSize           float64    `json:"lotSize"`
	SettleCurrency    string     `json:"settleCurrency"`
	SettlementFeeRate float64    `json:"settlementFeeRate"`
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
