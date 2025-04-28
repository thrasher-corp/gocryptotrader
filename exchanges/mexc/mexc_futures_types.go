package mexc

import (
	"encoding/json"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// WsFuturesData holds push data information for futures data streams
type WsFuturesData struct {
	Symbol    string          `json:"symbol,omitempty"`
	Data      json.RawMessage `json:"data"`
	Channel   string          `json:"channel"`
	Timestamp types.Time      `json:"ts"`

	Message string `json:"msg"`
	Code    int64  `json:"code"`
	ID      int64  `json:"id"`
}

// WsFuturesReq holds a futures request payload.
type WsFuturesReq struct {
	Method string              `json:"method"`
	GZip   bool                `json:"gzip"`
	Param  *FWebsocketReqParam `json:"param,omitempty"`
}

// FWebsocketReqParam holds the param detail or futures websocket subscription request
type FWebsocketReqParam struct {
	Symbol   string `json:"symbol,omitempty"`
	Compress bool   `json:"compress,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// FuturesTickerItem holds futures ticker data item
type FuturesTickerItem struct {
	FairPrice    float64 `json:"fairPrice"`
	LastPrice    float64 `json:"lastPrice"`
	RiseFallRate float64 `json:"riseFallRate"`
	Symbol       string  `json:"symbol"`
	Volume24     float64 `json:"volume24"`
}

// FuturesPriceTickerDetail represents a futures price ticker detail
type FuturesPriceTickerDetail struct {
	Ask1          float64    `json:"ask1"`
	Bid1          float64    `json:"bid1"`
	ContractID    int64      `json:"contractId"`
	FairPrice     float64    `json:"fairPrice"`
	FundingRate   float64    `json:"fundingRate"`
	High24Price   float64    `json:"high24Price"`
	IndexPrice    float64    `json:"indexPrice"`
	LastPrice     float64    `json:"lastPrice"`
	Lower24Price  float64    `json:"lower24Price"`
	MaxBidPrice   float64    `json:"maxBidPrice"`
	MinAskPrice   float64    `json:"minAskPrice"`
	RiseFallRate  float64    `json:"riseFallRate"`
	RiseFallValue float64    `json:"riseFallValue"`
	Symbol        string     `json:"symbol"`
	Timestamp     types.Time `json:"timestamp"`
	HoldVol       float64    `json:"holdVol"`
	Volume24      float64    `json:"volume24"`
}

// FuturesTransactionFills holds latest futures deal push data
type FuturesTransactionFills struct {
	IsAutoTransact       int        `json:"M"`
	OpenPosition         int        `json:"O"` // open position, 1: open position,2:close position,3:position no change,volume is the additional position when O is 1
	TransactionDirection int        `json:"T"`
	Price                float64    `json:"p"`
	TransationTime       types.Time `json:"t"`
	Volume               float64    `json:"v"`
}

// FuturesWsDepth holds futures instruments orderbook depth
type FuturesWsDepth struct {
	Asks    [][]float64 `json:"asks"`
	Bids    [][]float64 `json:"bids"`
	Version int64       `json:"version"`
}

// FuturesWebsocketKline holds candlestick data for futures instruments returned through websocket stream
type FuturesWebsocketKline struct {
	Amount                 float64    `json:"a"`
	ClosePrice             float64    `json:"c"`
	HighestPrice           float64    `json:"h"`
	Interval               string     `json:"interval"`
	LowestPrice            float64    `json:"l"`
	OpeningPrice           float64    `json:"o"`
	TotalTransactionVolume float64    `json:"q"`
	Symbol                 string     `json:"symbol"`
	TradeTime              types.Time `json:"t"`
}

// FuturesWsFundingRate holds futures websocket funding rate detail
type FuturesWsFundingRate struct {
	Rate   float64 `json:"rate"`
	Symbol string  `json:"symbol"`
}

// PriceAndSymbol holds a symbol and corresponding price information
type PriceAndSymbol struct {
	Price  float64 `json:"price"`
	Symbol string  `json:"symbol"`
}

// WsFuturesPersonalOrder holds users futures order detail
type WsFuturesPersonalOrder struct {
	Category     int64      `json:"category"`
	CreateTime   types.Time `json:"createTime"`
	DealAvgPrice float64    `json:"dealAvgPrice"`
	DealVol      float64    `json:"dealVol"`
	ErrorCode    int64      `json:"errorCode"`
	ExternalOid  string     `json:"externalOid"`
	FeeCurrency  string     `json:"feeCurrency"`
	Leverage     float64    `json:"leverage"`
	MakerFee     float64    `json:"makerFee"`
	OpenType     int64      `json:"openType"`
	OrderID      string     `json:"orderId"`
	OrderMargin  int64      `json:"orderMargin"`
	OrderType    int64      `json:"orderType"`
	PositionID   int64      `json:"positionId"`
	Price        float64    `json:"price"`
	Profit       float64    `json:"profit"`
	RemainVol    float64    `json:"remainVol"`
	Side         int64      `json:"side"`
	State        int64      `json:"state"`
	Symbol       string     `json:"symbol"`
	TakerFee     float64    `json:"takerFee"`
	UpdateTime   types.Time `json:"updateTime"`
	UsedMargin   float64    `json:"usedMargin"`
	Version      int        `json:"version"`
	Volume       float64    `json:"vol"`
}

// FuturesPersonalAsset represents a futures asset instance.
type FuturesPersonalAsset struct {
	AvailableBalance float64 `json:"availableBalance"`
	Bonus            int64   `json:"bonus"`
	Currency         string  `json:"currency"`
	FrozenBalance    float64 `json:"frozenBalance"`
	PositionMargin   float64 `json:"positionMargin"`
}

// FuturesWsPersonalPosition holds user's futures account personal position detail
type FuturesWsPersonalPosition struct {
	AutoAddIm             bool    `json:"autoAddIm"`
	CloseAvgPrice         float64 `json:"closeAvgPrice"`
	CloseVolume           float64 `json:"closeVol"`
	FrozenVolume          float64 `json:"frozenVol"`
	HoldAvgPrice          float64 `json:"holdAvgPrice"`
	HoldFee               float64 `json:"holdFee"`
	HoldVolume            float64 `json:"holdVol"`
	InitialMargin         float64 `json:"im"`
	Leverage              float64 `json:"leverage"`
	LiquidatePrice        float64 `json:"liquidatePrice"`
	OriginalInitialMargin float64 `json:"oim"`
	OpenAvgPrice          float64 `json:"openAvgPrice"`
	OpenType              int64   `json:"openType"`
	PositionID            int64   `json:"positionId"`
	PositionType          int64   `json:"positionType"`
	Realised              float64 `json:"realised"`
	State                 int64   `json:"state"`
	Symbol                string  `json:"symbol"`
}

// FuturesWebsocketRiskLimit holds a futures asset risk limit information
type FuturesWebsocketRiskLimit struct {
	Symbol                string  `json:"symbol"`
	PositionType          int     `json:"positionType"`
	RiskSource            int64   `json:"riskSource"`
	RiskLevel             int64   `json:"level"`
	MaxVolume             float64 `json:"maxVol"`
	MaxLeverage           float64 `json:"maxLeverage"`
	MaintenanceMarginRate float64 `json:"mmr"`
	InitialMarginRate     float64 `json:"imr"`
}

// FuturesADLLevel holds a futures adl
type FuturesADLLevel struct {
	AdlLevel   int `json:"adlLevel"`
	PositionID int `json:"positionId"`
}

// FuturesPositionMode holds futures account position mode information
type FuturesPositionMode struct {
	PositionMode int `json:"positionMode"`
}
