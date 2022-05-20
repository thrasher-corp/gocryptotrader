package okx

import "time"

// Market Data Endoints

// OkxMarkerDataResponse
type OkxMarketDataResponse struct {
	Code string               `json:"code"`
	Msg  string               `json:"msg"`
	Data []MarketDataResponse `json:"data"`
}

// MarketData represents the Market data endpoint.
type MarketDataResponse struct {
	InstrumentType           string  `json:"instType"`
	InstrumentID             string  `json:"instId"`
	LastTradePrice           float64 `json:"last,string"`
	LastTradeSize            float64 `json:"lastSz,string"`
	BestAskPrice             float64 `json:"askPx,string"`
	BestAskSize              int     `json:"askSz,string"`
	BidBidPrice              float64 `json:"bidPx,string,"`
	BidBidSize               int     `json:"bidSz,string,"`
	Open24H                  string  `json:"open24h"`
	High24H                  float64 `json:"high24h,string"`
	Low24H                   float64 `json:"low24h,string"`
	VolCcy24H                float64 `json:"volCcy24h,string"`
	Vol24H                   float64 `json:"vol24h,string"`
	OpenPriceInUTC0          float64 `json:"sodUtc0,string"`
	OpenPriceInUTC8          float64 `json:"sodUtc8,string"`
	TickerDataGenerationTime uint64  `json:"ts,string"`
}

type OKXIndexTickerResponse struct {
	InstID  string  `json:"instId"`
	IdxPx   float64 `json:"idxPx,string"`
	High24H float64 `json:"high24h,string"`
	SodUtc0 float64 `json:"sodUtc0,string"`
	Open24H float64 `json:"open24h,string"`
	Low24H  float64 `json:"low24h,string"`
	SodUtc8 float64 `json:"sodUtc8,string"`
	Ts      uint64  `json:"ts,string"`
}

// OrderBookResponse  returns the order asks and bids at a specific timestamp
type OrderBookResponse struct {
	Asks                [][4]string `json:"asks"`
	Bids                [][4]string `json:"bids"`
	GenerationTimeStamp time.Time   `json:"ts,string"`
}

// CandleStick  holds kline data
type CandleStick struct {
	OpenTime         time.Time
	OpenPrice        float64
	HighestPrice     float64
	LowestPrice      float64
	ClosePrice       float64
	Volume           float64
	QuoteAssetVolume float64
}

// TradeRsponse represents the recent transaction instance.
type TradeResponse struct {
	InstrumentID string    `json:"instId"`
	TradeId      int       `json:"tradeId,string"`
	Price        float64   `json:"px,string"`
	Quantity     float64   `json:"sz,string"`
	Side         string    `json:"side"`
	TimeStamp    time.Time `json:"ts"`
}

// TradingVolumdIn24HR response model.
type TradingVolumdIn24HR struct {
	TradingVolumnInUSD         string    `json:"volUsd"`
	TradingVolumeInThePlatform string    `json:"volCny"`
	Timestamp                  time.Time `json:"ts"`
}

// OracleSmartContractResponse returns the crypto price of signing using Open Oracle smart contract.
type OracleSmartContractResponse struct {
	Messages   []string          `json:"messages"`
	Prices     map[string]string `json:"prices"`
	Signatures []string          `json:"signatures"`
	Timestamp  time.Time         `json:"timestamp"`
}

// UsdCnvExchangeRate the exchange rate for converting from USD to CNV
type UsdCnyExchangeRate struct {
	USD_CNY float64 `json:"usdCny,string"`
}

// IndexComponent represents index component data on the market
type IndexComponent struct {
	Components []*IndexComponentItem `json:"components"`
	Last       float64               `json:"last,string"`
	Index      string                `json:"index"`
	Timestamp  time.Time             `json:"ts"`
}

// IndexParameter  an item representing the index component item
type IndexComponentItem struct {
	Symbol          string  `json:"symbol"`
	SymbolPairPrice float64 `json:"symbolPx,string"`
	Weights         float64 `json:"wgt,string"`
	ConverToPrice   float64 `json:"cnvPx,string"`
	ExchangeName    string  `json:"exch"`
}

// InstrumentsFetchParams ...
type InstrumentsFetchParams struct {
	InstrumentType string // Mandatory
	Underlying     string // Optional
	InstrumentID   string // Optional
}

// Instrument  representing an instrument with open contract.
type Instrument struct {
	InstType                        string    `json:"instType"`
	InstID                          string    `json:"instId"`
	Underlying                      string    `json:"uly"`
	Category                        string    `json:"category"`
	BaseCurrency                    string    `json:"baseCcy"`
	QuoteCurrency                   string    `json:"quoteCcy"`
	SettlementCurrency              string    `json:"settleCcy"`
	ContactValue                    int       `json:"ctVal,string"`
	ContractMultiplier              int       `json:"ctMult,string"`
	ContractValueCurrency           string    `json:"ctValCcy"`
	OptionType                      string    `json:"optType"`
	StrikePrice                     string    `json:"stk"`
	ListTime                        time.Time `json:"listTime"`
	ExpTime                         time.Time `json:"expTime"`
	MaxLeverage                     int64     `json:"lever,string"`
	TickSize                        float64   `json:"tickSz,string"`
	LotSize                         int64     `json:"lotSz,string"`
	MinimumOrderSize                int64     `json:"minSz,string"`
	ContractType                    string    `json:"ctType"`
	Alias                           string    `json:"alias"`
	State                           string    `json:"state"`
	MaxQuantityoOfSpotLimitOrder    float64   `json:"maxLmtSz,string"`
	MaxQuantityOfMarketLimitOrder   float64   `json:"maxMktSz,string"`
	MaxQuantityOfSpotTwapLimitOrder float64   `json:"maxTwapSz,string"`
	MaxSpotIcebergSize              float64   `json:"maxIcebergSz,string"`
	MaxTriggerSize                  float64   `json:"maxTriggerSz,string"`
	MaxStopSize                     float64   `json:"maxStopSz,string"`
}

// {   "ts":"1597026383085",
// "details":[
// 	{
// 		"type":"delivery",
// 		"instId":"BTC-USD-190927",
// 		"px":"0.016"
// 	}
// ]
// },

// DeliveryHistoryDetail ...
type DeliveryHistoryDetail struct {
	Type          string  `json:"type"`
	InstrumentID  string  `json:"instId"`
	DeliveryPrice float64 `json:"px,string"`
}

// DeliveryHistoryResponse
type DeliveryHistoryResponse struct {
	Timestamp time.Time                `json:"ts"`
	Details   []*DeliveryHistoryDetail `json:"details"`
}

// OpenInterestResponse Retrieve the total open interest for contracts on OKX.
type OpenInterestResponse struct {
	InstrumentType       string    `json:"instType"`
	InstrumentID         string    `json:"instId"`
	OpenInterest         float64   `json:"oi"`
	OpenInterestCurrency float64   `json:"oiCcy"`
	Timestamp            time.Time `json:"ts"`
}
