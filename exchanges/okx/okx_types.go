package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Market Data Endoints

// OkxMarkerDataResponse
type OkxMarketDataResponse struct {
	Code int                  `json:"code,string"`
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
	BestAskSize              float64 `json:"askSz,string"`
	BidBidPrice              float64 `json:"bidPx,string,"`
	BidBidSize               float64 `json:"bidSz,string,"`
	Open24H                  float64 `json:"open24h,string"`
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
	ContactValue                    string    `json:"ctVal"`
	ContractMultiplier              string    `json:"ctMult"`
	ContractValueCurrency           string    `json:"ctValCcy"`
	OptionType                      string    `json:"optType"`
	StrikePrice                     string    `json:"stk"`
	ListTime                        time.Time `json:"listTime"`
	ExpTime                         time.Time `json:"expTime"`
	MaxLeverage                     string    `json:"lever"`
	TickSize                        float64   `json:"tickSz,string"`
	LotSize                         float64   `json:"lotSz,string"`
	MinimumOrderSize                float64   `json:"minSz,string"`
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

// DeliveryHistory
type DeliveryHistory struct {
	Timestamp time.Time                `json:"ts"`
	Details   []*DeliveryHistoryDetail `json:"details"`
}

// DeliveryHistoryResponse represents the direct response of delivery history coming from the server.
type DeliveryHistoryResponse struct {
	Code string             `json:"code"`
	Msg  string             `json:"msg"`
	Data []*DeliveryHistory `json:"data"`
}

// OpenInterestResponse Retrieve the total open interest for contracts on OKX.
type OpenInterestResponse struct {
	InstrumentType       string    `json:"instType"`
	InstrumentID         string    `json:"instId"`
	OpenInterest         float64   `json:"oi,string"`
	OpenInterestCurrency float64   `json:"oiCcy,string"`
	Timestamp            time.Time `json:"ts"`
}

// FundingRateResponse response data for the Funding Rate for an instruction type
type FundingRateResponse struct {
	FundingRate     float64   `json:"fundingRate,string"`
	FundingTime     time.Time `json:"fundingTime"`
	InstID          string    `json:"instId"`
	InstType        string    `json:"instType"`
	NextFundingRate float64   `json:"nextFundingRate,string"`
	NextFundingTime time.Time `json:"nextFundingTime"`
}

// LimitPriceResponse hold an information for
type LimitPriceResponse struct {
	InstType  string    `json:"instType"`
	InstID    string    `json:"instId"`
	BuyLimit  float64   `json:"buyLmt,string"`
	SellLimit float64   `json:"sellLmt,string"`
	Timestamp time.Time `json:"ts"`
}

// OptionMarketDataResponse
type OptionMarketDataResponse struct {
	InstrumentType string    `json:"instType"`
	InstrumentID   string    `json:"instId"`
	Underlying     string    `json:"uly"`
	Delta          float64   `json:"delta,string"`
	Gamma          float64   `json:"gamma,string"`
	Theta          float64   `json:"theta,string"`
	Vega           float64   `json:"vega,string"`
	DeltaBS        float64   `json:"deltaBS,string"`
	GammaBS        float64   `json:"gammaBS,string"`
	ThetaBS        float64   `json:"thetaBS,string"`
	VegaBS         float64   `json:"vegaBS,string"`
	RealVol        string    `json:"realVol"`
	BidVolatility  string    `json:"bidVol"`
	AskVolatility  float64   `json:"askVol,string"`
	MarkVolitility float64   `json:"markVol,string"`
	Leverage       float64   `json:"lever,string"`
	ForwardPrice   string    `json:"fwdPx"`
	Timestamp      time.Time `json:"ts"`
}

// DeliveryEstimatedPriceResponse holds an estimated delivery or exercise price response.
type DeliveryEstimatedPrice struct {
	InstrumentType         string    `json:"instType"`
	InstrumentID           string    `json:"instId"`
	EstimatedDeliveryPrice string    `json:"settlePx"`
	Timestamp              time.Time `json:"ts"`
}

// DeliveryEstimatedPriceResponse
type DeliveryEstimatedPriceResponse struct {
	Code string                    `json:"code"`
	Msg  string                    `json:"msg"`
	Data []*DeliveryEstimatedPrice `json:"data"`
}

// DiscountRateResponse
type DiscountRateResponse struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data []*DiscountRate `json:"data"`
}

// DiscountRate represents the discount rate amount, currency, and other discount related informations.
type DiscountRate struct {
	Amount       string `json:"amt"`
	Currency     string `json:"ccy"`
	DiscountInfo []struct {
		DiscountRate string `json:"discountRate"`
		MaxAmount    string `json:"maxAmt"`
		MinAmount    string `json:"minAmt"`
	} `json:"discountInfo"`
	DiscountRateLevel string `json:"discountLv"`
}

// ServerTime returning  the server time instance.
type ServerTime struct {
	Timestamp time.Time `json:"ts"`
}

// LiquidationOrderRequestParams
type LiquidationOrderRequestParams struct {
	InstrumentType string
	MarginMode     string // values are either isolated or crossed
	InstrumentID   string
	Currency       currency.Code
	Underlying     string
	Alias          string
	State          string
	Before         time.Time
	After          time.Time
	Limit          int64
}

// LiquidationOrder
type LiquidationOrder struct {
	Details        []*LiquidationOrderDetailItem `json:"details"`
	InstrumentID   string                        `json:"instId"`
	InstrumentType string                        `json:"instType"`
	TotalLoss      string                        `json:"totalLoss"`
	Underlying     string                        `json:"uly"`
}

// LiquidationOrderResponse
type LiquidationOrderResponse struct {
	Code string              `json:"code"`
	Msg  string              `json:"msg"`
	Data []*LiquidationOrder `json:"data"`
}

// LiquidationOrderDetailItem represents the detail information of liquidation order
type LiquidationOrderDetailItem struct {
	BankruptcyLoss        string    `json:"bkLoss"`
	BankruptcyPx          string    `json:"bkPx"`
	Currency              string    `json:"ccy"`
	PosSide               string    `json:"posSide"`
	Side                  string    `json:"side"`
	QuantityOfLiquidation float64   `json:"sz,string"`
	Timestamp             time.Time `json:"ts"`
}

// MarkPrice endpoint response data; this holds list of information for mark price.
type MarkPriceResponse struct {
	Code string       `json:"code"`
	Msg  string       `json:"msg"`
	Data []*MarkPrice `json:"data"`
}

// MarkPrice represents a mark price information for a single instrument id
type MarkPrice struct {
	InstrumentType string    `json:"instType"`
	InstrumentID   string    `json:"instId"`
	MarkPrice      string    `json:"markPx"`
	Timestamp      time.Time `json:"ts"`
}

// PositionTiersResponse response
type PositionTiersResponse struct {
	Code string           `json:"code"`
	Msg  string           `json:"msg"`
	Data []*PositionTiers `json:"data"`
}

// PositionTiers ...
type PositionTiers struct {
	BaseMaxLoan                   string  `json:"baseMaxLoan"`
	InitialMarginRequirement      float64 `json:"imr,string"`
	InstID                        string  `json:"instId"`
	MaxLever                      string  `json:"maxLever"`
	MaxSize                       float64 `json:"maxSz,string"`
	MinSize                       float64 `json:"minSz,string"`
	MaintainanceMarginRequirement float64 `json:"mmr,string"`
	OptionalMarginFactor          string  `json:"optMgnFactor"`
	QuoteMaxLoan                  float64 `json:"quoteMaxLoan,string"`
	Tier                          string  `json:"tier"`
	Underlying                    string  `json:"uly"`
}
