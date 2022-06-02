package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	InitialMarginRequirement      string  `json:"imr"`
	InstID                        string  `json:"instId"`
	MaxLever                      string  `json:"maxLever"`
	MaxSize                       float64 `json:"maxSz,string"`
	MinSize                       float64 `json:"minSz,string"`
	MaintainanceMarginRequirement string  `json:"mmr"`
	OptionalMarginFactor          string  `json:"optMgnFactor"`
	QuoteMaxLoan                  string  `json:"quoteMaxLoan"`
	Tier                          string  `json:"tier"`
	Underlying                    string  `json:"uly"`
}

// InterestRateLoanQuotaItem holds the basic Currency, loan,and interest rate informations.
type InterestRateLoanQuotaBasic struct {
	Currency     string  `json:"ccy"`
	LoanQuota    string  `json:"quota"`
	InterestRate float64 `json:"rate,string"`
}

// InterestRateLoanQuotaItem holds the basic Currency, loan,interest rate, and other level and VIP related informations.
type InterestRateLoanQuotaItem struct {
	InterestRateLoanQuotaBasic
	InterestRateDiscount float64 `json:"0.7,string"`
	LoanQuotaCoefficient float64 `json:"loanQuotaCoef,string"`
	Level                string  `json:"level"`
}

// InterestRateLoanQuotaResponse holds a response information for InterestRateLoadQuotaItem informations.
type InterestRateLoanQuotaResponse struct {
	Msg  string                                    `json:"msg"`
	Code string                                    `json:"code"`
	Data []map[string][]*InterestRateLoanQuotaItem `json:"data"`
}

// VIPInterestRateAndLoanQuotaInformation holds interest rate and loan quoata information for VIP users.
type VIPInterestRateAndLoanQuotaInformation struct {
	InterestRateLoanQuotaBasic
	LevelList []struct {
		Level     string  `json:"level"`
		LoanQuota float64 `json:"loanQuota,string"`
	} `json:"levelList"`
}

// VIPInterestRateAndLoanQuotaInformationResponse holds the response information for VIPInterestRateAndLoanQuotaInformation messages.
type VIPInterestRateAndLoanQuotaInformationResponse struct {
	Code string                                    `json:"code"`
	Msg  string                                    `json:"msg"`
	Data []*VIPInterestRateAndLoanQuotaInformation `json:"data"`
}

// InsuranceFundInformationRequestParams insurance fund balance information.
type InsuranceFundInformationRequestParams struct {
	InstrumentType string    `json:"instType"`
	Type           string    `json:"type"` //  Type values allowed are `liquidation_balance_deposit, bankruptcy_loss, and platform_revenue`
	Underlying     string    `json:"uly"`
	Currency       string    `json:"ccy"`
	Before         time.Time `json:"before"`
	After          time.Time `json:"after"`
	Limit          uint      `json:"limit"`
}

// InsuranceFundInformationResponse holds the insurance fund information response data coming from the server.
type InsuranceFundInformationResponse struct {
	Code string                      `json:"code"`
	Msg  string                      `json:"msg"`
	Data []*InsuranceFundInformation `json:"data"`
}

// InsuranceFundInformation holds insurance fund information data.
type InsuranceFundInformation struct {
	Details []*InsuranceFundInformationDetail `json:"details"`
	Total   float64                           `json:"total,string"`
}

// InsuranceFundInformationDetail represents an Insurance fund information item for a
// single currency and type
type InsuranceFundInformationDetail struct {
	Amount    float64   `json:"amt,string"`
	Balance   float64   `json:"balance,string"`
	Currency  string    `json:"ccy"`
	Timestamp time.Time `json:"ts"`
	Type      string    `json:"type"`
}

// SupportedCoinsResponse Retrieve the currencies supported by the trading data endpoints.
type SupportedCoinsResponse struct {
	Code string              `json:"code"`
	Msg  string              `json:"msg"`
	Data *SupportedCoinsData `json:"data"`
}

// SupportedCoinsData holds information about currencies supported by the trading data endpoints.
type SupportedCoinsData struct {
	Contract                      []string `json:"contract"`
	TradingOptions                []string `json:"option"`
	CurrenciesSupportedBySpotSpot []string `json:"spot"`
}

// TakerVolumeResponse list of taker volume
type TakerVolumeResponse struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data [][3]string `json:"data"`
}

// TakerVolume
type TakerVolume struct {
	Timestamp  time.Time `json:"ts"`
	SellVolume float64
	BuyVolume  float64
}

// MarginLendRatioItem
type MarginLendRatioItem struct {
	Timestamp       time.Time `json:"ts"`
	MarginLendRatio float64   `json:"ratio"`
}

// MarginLendRatioResponse
type MarginLendRatioResponse struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data [][2]string `json:"data"`
}

// // LongShortRatio
type LongShortRatio struct {
	Timestamp       time.Time `json:"ts"`
	MarginLendRatio float64   `json:"ratio"`
}

// LongShortRatioResponse
type LongShortRatioResponse struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data [][2]string `json:"data"`
}

// OpenIntereseVolume
type OpenInterestVolume struct {
	Timestamp    time.Time `json:"ts"`
	OpenInterest float64   `json:"oi"`
	Volume       float64   `json:"vol"`
}

// OpenInterestVolumeResponse
type OpenInterestVolumeResponse struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data [][3]string `json:"data"`
}

// OpenInterestVolumeRatio
type OpenInterestVolumeRatio struct {
	Timestamp         time.Time `json:"ts"`
	OpenInterestRatio float64   `json:"oiRatio"`
	VolumeRatio       float64   `json:"volRatio"`
}

// ExpiryOpenInterestAndVolume
type ExpiryOpenInterestAndVolume struct {
	Timestamp        time.Time
	ExpiryTime       time.Time
	CallOpenInterest float64
	PutOpenInterest  float64
	CallVolume       float64
	PutVolume        float64
}

// StrikeOpenInterestAndVolume ..
type StrikeOpenInterestAndVolume struct {
	Timestamp        time.Time
	Strike           int64
	CallOpenInterest float64
	PutOpenInterest  float64
	CallVolume       float64
	PutVolume        float64
}

// CurrencyTakerFlow holds the taker volume information for a single currency.
type CurrencyTakerFlow struct {
	Timestamp       time.Time
	CallBuyVolume   float64
	CallSellVolume  float64
	PutBuyVolume    float64
	PutSellVolume   float64
	CallBlockVolume float64
	PutBlockVolume  float64
}

// PlaceOrderRequestParam requesting parameter for placing an order.
type PlaceOrderRequestParam struct {
	InstrumentID          string  `json:"instId"`
	TradeMode             string  `json:"tdMode"` // cash isolated
	ClientSupplierOrderID string  `json:"clOrdId"`
	Currency              string  `json:"ccy,omitempty"` // Only applicable to cross MARGIN orders in Single-currency margin.
	OrderTag              string  `json:"tag"`
	Side                  string  `json:"side"`
	PositionSide          string  `json:"posSide"`
	OrderType             string  `json:"ordType"`
	QuantityToBuyOrSell   float64 `json:"sz,string"`
	OrderPrice            float64 `json:"px,string"`
	ReduceOnly            bool    `json:"reduceOnly,string,omitempty"`
	QuantityType          string  `json:"tgtCcy,omitempty"` // values base_ccy and quote_ccy

}

// PlaceOrderResponse respnse message for placing an order.
type PlaceOrderResponse struct {
	OrderID               string `json:"ordId"`
	ClientSupplierOrderID string `json:"clOrdId"`
	Tag                   string `json:"tag"`
	StatusCode            int    `json:"sCode,string"`
	StatusMessage         string `json:"sMsg"`
}

// CancelOrderRequestParams
type CancelOrderRequestParam struct {
	InstrumentID          string `json:"instId"`
	OrderID               string `json:"ordId"`
	ClientSupplierOrderID string `json:"clOrdId,omitempty"`
}

// CancelOrderResponse
type CancelOrderResponse struct {
	OrderID       string `json:"ordId"`
	ClientOrderID string `json:"clOrdId"`
	StatusCode    int    `json:"sCode,string"`
	Msg           string `json:"sMsg"`
}

// // OrderResponse
// type CancelOrderResponse struct {
// 	OrderID       string `json:"orderId"`
// 	ClientOrderID string `json:"clOrdId"`
// 	StatusCode    string `json:"sCode"`
// 	StatusMessage string `json:"sMsg"`
// }

// AmendOrderRequestParams
type AmendOrderRequestParams struct {
	InstrumentID            string  `json:"instId"`
	CancelOnFail            bool    `json:"cxlOnFail"`
	OrderID                 string  `json:"ordId"`
	ClientSuppliedOrderID   string  `json:"clOrdId"`
	ClientSuppliedRequestID string  `json:"reqId"`
	NewQuantity             float64 `json:"newSz,string"`
	NewPrice                float64 `json:"newPx,string"`
}

// AmendOrderResponse
type AmendOrderResponse struct {
	OrderID                 string  `json:"ordId"`
	ClientSuppliedOrderID   string  `json:"clOrdId"`
	ClientSuppliedRequestID string  `json:"reqId"`
	StatusCode              float64 `json:"sCode,string"`
	StatusMsg               string  `json:"sMsg"`
}

// ClosePositionsRequestParams input parameters for close position endpoints
type ClosePositionsRequestParams struct {
	InstrumentID          string `json:"instId"` // REQUIRED
	PositionSide          string `json:"posSide"`
	MarginMode            string `json:"mgnMode"` // cross or isolated
	Currency              string `json:"ccy"`
	AutomaticallyCanceled bool   `json:"autoCxl"`
}

// ClosePositionResponse response data for close position.
type ClosePositionResponse struct {
	InstrumentID string `json:"instId"`
	POsitionSide string `json:"posSide"`
}

// OrderDetailRequestParam payload data to request order detail
type OrderDetailRequestParam struct {
	InstrumentID          string `json:"instId"`
	OrderID               string `json:"ordId"`
	ClientSupplierOrderID string `json:"clOrdId"`
}

// OrderDetailResponse returns a order detail information
type OrderDetail struct {
	InstrumentType             string     `json:"instType"`
	InstrumentID               string     `json:"instId"`
	Currency                   string     `json:"ccy"`
	OrderID                    string     `json:"ordId"`
	ClientSupplierOrderID      string     `json:"clOrdId"`
	Tag                        string     `json:"tag"`
	Price                      float64    `json:"px,string"`
	Size                       float64    `json:"sz,string"`
	ProfitAndLoss              string     `json:"pnl"`
	OrderType                  string     `json:"ordType"`
	Side                       order.Side `json:"side"`
	PositionSide               string     `json:"posSide"`
	TradeMode                  string     `json:"tdMode"`
	AccumulatedFillSize        string     `json:"accFillSz"`
	FillPrice                  string     `json:"fillPx"`
	TradeID                    string     `json:"tradeId"`
	FillSize                   string     `json:"fillSz"`
	FillTime                   string     `json:"fillTime"`
	State                      string     `json:"state"`
	AvgPrice                   string     `json:"avgPx"`
	Leverage                   float64    `json:"lever,string"`
	TakeProfitTriggerPrice     string     `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string     `json:"tpTriggerPxType"`
	TakeProfitOrdPrice         string     `json:"tpOrdPx"`
	StopLossTriggerPrice       string     `json:"slTriggerPx"`
	StopLossTriggerPriceType   string     `json:"slTriggerPxType"`
	StopLossOrdPx              string     `json:"slOrdPx"`
	FeeCurrency                string     `json:"feeCcy"`
	TransactionFee             string     `json:"fee"`
	RebateCurrency             string     `json:"rebateCcy"`
	RebateAmount               string     `json:"rebate"`
	QuantityType               string     `json:"tgtCcy"`   // base_ccy and quote_ccy
	Category                   string     `json:"category"` // normal, twap, adl, full_liquidation, partial_liquidation, delivery, ddh
	UpdateTime                 time.Time  `json:"uTime"`
	CreationTime               time.Time  `json:"cTime"`
}

// OrderListRequestParams
type OrderListRequestParams struct {
	InstrumentType string    `json:"instType"` // SPOT , MARGIN, SWAP, FUTURES , option
	Underlying     string    `json:"uly"`
	InstrumentID   string    `json:"instId"`
	OrderType      string    `json:"orderType"`
	State          string    `json:"state"` // live, partially_filled
	After          time.Time `json:"after"`
	Before         time.Time `json:"before"`
	Limit          int       `json:"limit"`
}

// OrderHistoryRequestParams holds parameters to request order data history of last 7 days.
type OrderHistoryRequestParams struct {
	OrderListRequestParams
	Category string `json:"category"` // twap, adl, full_liquidation, partial_liquidation, delivery, ddh
}

// PendingOrderItem represents a pending order Item in pending orders list.
type PendingOrderItem struct {
	AccumulatedFillSize        string     `json:"accFillSz"`
	AvgPx                      string     `json:"avgPx"`
	CreationTime               time.Time  `json:"cTime"`
	Category                   string     `json:"category"`
	Currency                   string     `json:"ccy"`
	ClientSupplierOrderID      string     `json:"clOrdId"`
	TransactionFee             string     `json:"fee"`
	FeeCcy                     string     `json:"feeCcy"`
	LastFilledPrice            string     `json:"fillPx"`
	FillSize                   string     `json:"fillSz"`
	FillTime                   string     `json:"fillTime"`
	InstrumentID               string     `json:"instId"`
	InstrumentType             string     `json:"instType"`
	Leverage                   float64    `json:"lever,string"`
	OrderID                    string     `json:"ordId"`
	OrderType                  string     `json:"ordType"`
	ProfitAndLose              string     `json:"pnl"`
	PositionSide               string     `json:"posSide"`
	RebateAmount               string     `json:"rebate"`
	RebateCurrency             string     `json:"rebateCcy"`
	Side                       order.Side `json:"side"`
	StopLossOrdPrice           string     `json:"slOrdPx"`
	StopLossTriggerPrice       string     `json:"slTriggerPx"`
	StopLossTriggerPriceType   string     `json:"slTriggerPxType"`
	State                      string     `json:"state"`
	Price                      float64    `json:"px,string"`
	Size                       float64    `json:"sz,string"`
	Tag                        string     `json:"tag"`
	QuantityType               string     `json:"tgtCcy"`
	TradeMode                  string     `json:"tdMode"`
	Source                     string     `json:"source"` //
	TakeProfitOrdPrice         string     `json:"tpOrdPx"`
	TakeProfitTriggerPrice     string     `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string     `json:"tpTriggerPxType"`
	TradeID                    string     `json:"tradeId"`
	UpdateTime                 time.Time  `json:"uTime"`
}

// TransactionDetailRequestParams retrieve recently-filled transaction details in the last 3 day.
type TransactionDetailRequestParams struct {
	InstrumentType string    `json:"instType"` // SPOT , MARGIN, SWAP, FUTURES , option
	Underlying     string    `json:"uly"`
	InstrumentID   string    `json:"instId"`
	OrderID        string    `json:"ordId"`
	OrderType      string    `json:"orderType"`
	After          string    `json:"after"`  // after billid
	Before         string    `json:"before"` // before billid
	Begin          time.Time `json:"begin"`
	End            time.Time `json:"end"`
	Limit          int       `json:"limit"`
}

// TransactionDetail holds ecently-filled transaction detail data.
type TransactionDetail struct {
	InstrumentType        string    `json:"instType"`
	InstrumentID          string    `json:"instId"`
	TradeID               string    `json:"tradeId"`
	OrderID               string    `json:"ordId"`
	ClientSuppliedOrderID string    `json:"clOrdId"`
	BillID                string    `json:"billId"`
	Tag                   string    `json:"tag"`
	FillPrice             float64   `json:"fillPx,string"`
	FillSize              float64   `json:"fillSz,string"`
	Side                  string    `json:"side"`
	PositionSide          string    `json:"posSide"`
	ExecType              string    `json:"execType"`
	FeeCurrency           string    `json:"feeCcy"`
	Fee                   string    `json:"fee"`
	Timestamp             time.Time `json:"ts"`
}

// AlgoOrderParams holds algo order informations.
type AlgoOrderParams struct {
	InstrumentID string     `json:"instId"` // Required
	TradeMode    string     `json:"tdMode"` // Required
	Currency     string     `json:"ccy"`
	Side         order.Side `json:"side"` // Required
	PositionSide string     `json:"posSide"`
	OrderType    string     `json:"ordType"`   // Required
	Size         float64    `json:"sz,string"` // Required
	OrderTag     string     `json:"tag"`
	ReduceOnly   bool       `json:"reduceOnly"`
	QuantityType string     `json:"tgtCcy"`
}

// StopOrder holds stop order request payload.
type StopOrderParams struct {
	TakeProfitTriggerPrice     string `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string `json:"tpTriggerPxType"`
	TakeProfitOrderType        string `json:"tpOrdPx"`
	StopLossTriggerPrice       string `json:"slTriggerPx"`
	StopLossTriggerPriceType   string `json:"slTriggerPxType"`
	StopLossOrderPrice         string `json:"slOrdPx"`
}

// AlgoOrderResponse algo requests response
type AlgoOrder struct {
	AlgoID     string `json:"algoId"`
	StatusCode string `json:"sCode"`
	StatusMsg  string `json:"sMsg"`
}

// TriggerAlogOrderParams trigger algo orders params.
// notice: Trigger orders are not available in the net mode of futures and perpetual swaps
type TriggerAlogOrderParams struct {
	TriggerPrice     float64 `json:"triggerPx,string"`
	TriggerPriceType string  `json:"triggerPxType"`  // last, index, and mark
	OrderPrice       int     `json:"orderPx,string"` // if the price i -1, then the order will be executed on the market.
}

// TrailingStopOrderRequestParam
type TrailingStopOrderRequestParam struct {
	CallbackRatio          float64 `json:"callbackRatio,string"` // Optional
	CallbackSpreadVariance string  `json:"callbackSpread"`       // Optional
	ActivePrice            string  `json:"activePx"`
}

// IceburgOrder ...
type IceburgOrder struct {
	PriceRatio    string  `json:"pxVar"`          // Optional
	PriceVariance string  `json:"pxSpread"`       // Optional
	AverageAmount float64 `json:"szLimit,string"` // Required
	PriceLimit    float64 `json:"pxLimit,string"` // Required
}

// TWAPOrder
type TWAPOrderRequestParams struct {
	PriceRatio    string         `json:"pxVar"`          // optional with pxSpread
	PriceVariance string         `json:"pxSpread"`       // optional
	AverageAmount float64        `json:"szLimit,string"` // Required
	PriceLimit    float64        `json:"pxLimit"`        // Required
	Timeinterval  kline.Interval `json:"interval"`       // Required
}

// AlgoOrderCancelParams algo order request parameter
type AlgoOrderCancelParams struct {
	AlgoOrderID  string `json:"algoId"`
	InstrumentID string `json:"instId"`
}

// AlgoOrderResponse holds algo order informations.
type AlgoOrderResponse struct {
	InstrumentType             string    `json:"instType"`
	InstrumentID               string    `json:"instId"`
	OrderID                    string    `json:"ordId"`
	Currency                   string    `json:"ccy"`
	AlgoOrderID                string    `json:"algoId"`
	Quantity                   string    `json:"sz"`
	OrderType                  string    `json:"ordType"`
	Side                       string    `json:"side"`
	PositionSide               string    `json:"posSide"`
	TradeMode                  string    `json:"tdMode"`
	QuantityType               string    `json:"tgtCcy"`
	State                      string    `json:"state"`
	Lever                      string    `json:"lever"`
	TakeProfitTriggerPrice     string    `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string    `json:"tpTriggerPxType"`
	TakeProfitOrdPrice         string    `json:"tpOrdPx"`
	StopLossTriggerPriceType   string    `json:"slTriggerPxType"`
	StopLossTriggerPrice       string    `json:"slTriggerPx"`
	TriggerPrice               string    `json:"triggerPx"`
	TriggerPriceType           string    `json:"triggerPxType"`
	OrdPrice                   string    `json:"ordPx"`
	ActualSize                 string    `json:"actualSz"`
	ActualPrice                string    `json:"actualPx"`
	ActualSide                 string    `json:"actualSide"`
	PriceVar                   string    `json:"pxVar"`
	PriceSpread                string    `json:"pxSpread"`
	PriceLimit                 string    `json:"pxLimit"`
	SizeLimit                  string    `json:"szLimit"`
	TimeInterval               string    `json:"timeInterval"`
	TriggerTime                time.Time `json:"triggerTime"`
	CallbackRatio              string    `json:"callbackRatio"`
	CallbackSpread             string    `json:"callbackSpread"`
	ActivePrice                string    `json:"activePx"`
	MoveTriggerPrice           string    `json:"moveTriggerPx"`
	CreationTime               time.Time `json:"cTime"`
}

// CurrencyResponse
type CurrencyResponse struct {
	CanDeposit           bool    `json:"canDep"`        // Availability to deposit from chain. false: not available true: available
	CanInternalTransffer bool    `json:"canInternal"`   // Availability to internal transfer.
	CanWithdraw          bool    `json:"canWd"`         // Availability to withdraw to chain.
	Currency             string  `json:"ccy"`           //
	Chain                string  `json:"chain"`         //
	LogoLink             string  `json:"logoLink"`      // Logo link of currency
	MainNet              bool    `json:"mainNet"`       // If current chain is main net then return true, otherwise return false
	MaxFee               float64 `json:"maxFee,string"` // Minimum withdrawal fee
	MaxWithdrawal        float64 `json:"maxWd,string"`  // Minimum amount of currency withdrawal in a single transaction
	MinFee               float64 `json:"minFee,string"` // Minimum withdrawal fee
	MinWithdrawal        string  `json:"minWd"`         // Minimum amount of currency withdrawal in a single transaction
	Name                 string  `json:"name"`          // Chinese name of currency
	UsedWithdrawalQuota  string  `json:"usedWdQuota"`   // Amount of currency withdrawal used in the past 24 hours, unit in BTC
	WithdrawalQuota      string  `json:"wdQuota"`       // Minimum amount of currency withdrawal in a single transaction
	WithdrawalTickSize   string  `json:"wdTickSz"`      // Withdrawal precision, indicating the number of digits after the decimal point
}

// AssetBalance  asset balance
type AssetBalance struct {
	AvailBal      float64 `json:"availBal,string"`
	Balance       string  `json:"bal"`
	Currency      string  `json:"ccy"`
	FrozenBalance float64 `json:"frozenBal,string"`
}

// AccountAssetValuation
type AccountAssetValuation struct {
	Details struct {
		Classic float64 `json:"classic,string"`
		Earn    float64 `json:"earn,string"`
		Funding float64 `json:"funding,string"`
		Trading float64 `json:"trading,string"`
	} `json:"details"`
	TotalBal  float64   `json:"totalBal,string"`
	Timestamp time.Time `json:"ts"`
}

// FundingTransferRequestInput
type FundingTransferRequestInput struct {
	Currency      string  `json:"ccy"`
	Type          int     `json:"type,string"`
	Amount        float64 `json:"amt,string"`
	From          string  `json:"from"` // "6": Funding account, "18": Trading account
	To            string  `json:"to"`
	SubAccount    string  `json:"subAcct"`
	LoanTransffer bool    `json:"loanTrans,string"`
	ClientID      string  `json:"clientId"` // Client-supplied ID A combination of case-sensitive alphanumerics, all numbers, or all letters of up to 32 characters.
}

// FundingTransferResponse
type FundingTransferResponse struct {
	TransferID string  `json:"transId"`
	Currency   string  `json:"ccy"`
	ClientID   string  `json:"clientId"`
	From       int64   `json:"from,string"`
	Amount     float64 `json:"amt,string"`
	To         int64   `json:"to,string"`
}

// TransferFundRateResponse
type TransferFundRateResponse struct {
	Amount         float64 `json:"amt,string"`
	Currency       string  `json:"ccy"`
	ClientID       string  `json:"clientId"`
	From           string  `json:"from"`
	InstrumentID   string  `json:"instId"`
	State          string  `json:"state"`
	SubAccountt    string  `json:"subAcct"`
	To             string  `json:"to"`
	ToInstrumentID string  `json:"toInstId"`
	TransrumentID  string  `json:"transId"`
	Type           int     `json:"type,string"`
}

// AssetBillDetail response
type AssetBillDetail struct {
	BillID         string    `json:"billId"`
	Currency       string    `json:"ccy"`
	ClientID       string    `json:"clientId"`
	BalanceChange  string    `json:"balChg"`
	AccountBalance string    `json:"bal"`
	Type           int       `json:"type,string"`
	Timestamp      time.Time `json:"ts"`
}

// LightningDeposit for creating an invoice
type LightningDepositItem struct {
	CreationTime time.Time `json:"cTime"`
	Invoice      string    `json:"invoice"`
}

// CurrencyDepositResponseItem represents the deposit address information item.
type CurrencyDepositResponseItem struct {
	Chain                string `json:"chain"`
	ContractAddress      string `json:"ctAddr"`
	Currency             string `json:"ccy"`
	ToBeneficiaryAccount string `json:"to"`
	Address              string `json:"addr"`
	Selected             bool   `json:"selected"`
}

// DepositHistoryResponseItem deposit history response item.
type DepositHistoryResponseItem struct {
	Amount           string    `json:"amt"`
	TransactionID    string    `json:"txId"` // Hash record of the deposit
	Currency         string    `json:"ccy"`
	Chain            string    `json:"chain"`
	From             string    `json:"from"`
	ToDepositAddress string    `json:"to"`
	Timestamp        time.Time `json:"ts"`
	State            int       `json:"state,string"`
	DepositID        string    `json:"depId"`
}

// WithdrawalInput ...
type WithdrawalInput struct {
	Amount                float64 `json:"amt,string"`
	TransactionFee        float64 `json:"fee,string"`
	WithdrawalDestination string  `json:"dest"`
	Currency              string  `json:"ccy"`
	ChainName             string  `json:"chain"`
	ToAddress             string  `json:"toAddr"`
	ClientSuppliedID      string  `json:"clientId"`
}

// WithdrawalResponse
type WithdrawalResponse struct {
	Amount       float64 `json:"amt,string"`
	WithdrawalID string  `json:"wdId"`
	Currency     string  `json:"ccy"`
	ClientID     string  `json:"clientId"`
	Chain        string  `json:"chain"`
}
