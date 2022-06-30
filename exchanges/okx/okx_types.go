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

// LightningRequestInput to request Lightning Withdrawal requests.
type LightningWithdrawalRequestInput struct {
	Currency string `json:"ccy"`     // REQUIRED Token symbol. Currently only BTC is supported.
	Invoice  string `json:"invoice"` // REQUIRED Invoice text
	Memo     string `json:"memo"`    // Lightning withdrawal memo
}

// LightningWithdrawalResponse response item for holding lightning withdrawal requests.
type LightningWithdrawalResponse struct {
	WithdrawalID string    `json:"wdId"`
	CreationTime time.Time `json:"cTime"`
}

// WithdrawalHistoryResponse represents the withdrawal response history.
type WithdrawalHistoryResponse struct {
	ChainName            string    `json:"chain"`
	WithdrawalFee        string    `json:"fee"`
	Currency             string    `json:"ccy"`
	ClientID             string    `json:"clientId"`
	TakeAmount           string    `json:"amt"`
	TxID                 string    `json:"txId"` // Hash record of the withdrawal. This parameter will not be returned for internal transfers.
	FromRemittingAddress string    `json:"from"`
	ToReceivingAddress   string    `json:"to"`
	StateOfWithdrawal    string    `json:"state"`
	Timestamp            time.Time `json:"ts"`
	WithdrawalID         string    `json:"wdId"`
	PaymentID            string    `json:"pmtId,omitempty"`
	Memo                 string    `json:"memo"`
}

// SmallAssetConvertResponse represents a response of converting a small asset to OKB.
type SmallAssetConvertResponse struct {
	Details []struct {
		Amount        string `json:"amt"`    // Quantity of currency assets before conversion
		Currency      string `json:"ccy"`    //
		ConvertAmount string `json:"cnvAmt"` // Quantity of OKB after conversion
		ConversionFee string `json:"fee"`    // Fee for conversion, unit in OKB
	} `json:"details"`
	TotalConvertAmount string `json:"totalCnvAmt"` // Total quantity of OKB after conversion
}

// SavingBalanceResponse returns a saving response.
type SavingBalanceResponse struct {
	Earnings      float64 `json:"earnings,string"`
	RedemptAmount float64 `json:"redemptAmt,string"`
	Rate          float64 `json:"rate,string"`
	Currency      string  `json:"ccy"`
	Amount        float64 `json:"amt,string"`
	LoanAmount    float64 `json:"loanAmt,string"`
	PendingAmount float64 `json:"pendingAmt,string"`
}

// SavingsPurchaseInput input json to SavingPurchase Post merthod.
type SavingsPurchaseRedemptionInput struct {
	Currency   string  `json:"ccy"`         // REQUIRED:
	Amount     float64 `json:"amt,string"`  // REQUIRED: purchase or redemption amount
	ActionType string  `json:"side"`        // REQUIRED: action type \"purchase\" or \"redemption\"
	Rate       float64 `json:"rate,string"` // REQUIRED:
}

// SavingsPurchaseRedemptionResponse response json to SavingPurchase or SavingRedemption Post method.
type SavingsPurchaseRedemptionResponse struct {
	Currency   string  `json:"ccy"`
	Amount     float64 `json:"amt,string"`
	ActionType string  `json:"side"`
	Rate       float64 `json:"rate,string"`
}

// LendingRate
type LendingRate struct {
	Currency string  `json:"ccy"`
	Rate     float64 `json:"rate,string"`
}

/**{
	"ccy": "BTC",
	"amt": "0.01",
	"earnings": "0.001",
	"rate": "0.01",
	"ts": "1597026383085"
}, **/

// LendingHistory holds lending hostory responses
type LendingHistory struct {
	Currency  string    `json:"ccy"`
	Amount    float64   `json:"amt,string"`
	Earnings  float64   `json:"earnings,string,omitempty"`
	Rate      float64   `json:"rate,string"`
	Timestamp time.Time `json:"ts"`
}

// PublicBorrowInfo holds borrow info.
type PublicBorrowInfo struct {
	Ccy       string  `json:"ccy"`
	AvgAmt    float64 `json:"avgAmt,string"`
	AvgAmtUsd float64 `json:"avgAmtUsd,string"`
	AvgRate   float64 `json:"avgRate,string"`
	PreRate   float64 `json:"preRate,string"`
	EstRate   float64 `json:"estRate,string"`
}

// ConvertCurrency struct representing a convert currency item response.
type ConvertCurrency struct {
	Currency string  `json:"currency"`
	Min      float64 `json:"min,string"`
	Max      float64 `json:"max,string"`
}

// ConvertCurrencyPair holds information related to conversion between two pairs.
type ConvertCurrencyPair struct {
	InstrumentID     string  `json:"instId"`
	BaseCurrency     string  `json:"baseCcy"`
	BaseCurrencyMax  float64 `json:"baseCcyMax,string,omitempty"`
	BaseCurrencyMin  float64 `json:"baseCcyMin,string,omitempty"`
	QuoteCurrency    string  `json:"quoteCcy,omitempty"`
	QuoteCurrencyMax float64 `json:"quoteCcyMax,string,omitempty"`
	QuoteCurrencyMin float64 `json:"quoteCcyMin,string,omitempty"`
}

// EstimateQuoteRequestInput
type EstimateQuoteRequestInput struct {
	BaseCurrency         string     `json:"baseCcy"`
	QuoteCurrency        string     `json:"quoteCcy"`
	Side                 order.Side `json:"side"`
	RFQAmount            float64    `json:"rfqSz"`
	RFQSzCurrency        string     `json:"rfqSzCcy"`
	ClientRequestOrderID string     `json:"clQReqId,string"`
	Tag                  string     `json:"tag"`
}

// EstimateQuoteResponse
type EstimateQuoteResponse struct {
	BaseCurrency            string    `json:"baseCcy"`
	BaseSize                string    `json:"baseSz"`
	ClientSupplierRequestID string    `json:"clQReqId"`
	ConvertPrice            string    `json:"cnvtPx"`
	OrigRfqSize             string    `json:"origRfqSz"`
	QuoteCurrency           string    `json:"quoteCcy"`
	QuoteID                 string    `json:"quoteId"`
	QuoteSize               string    `json:"quoteSz"`
	QuoteTime               time.Time `json:"quoteTime"`
	RFQSize                 string    `json:"rfqSz"`
	RFQSizeCurrency         string    `json:"rfqSzCcy"`
	Side                    string    `json:"side"`
	TTLMs                   string    `json:"ttlMs"` // Validity period of quotation in milliseconds
}

// ConvertTradeInput
type ConvertTradeInput struct {
	BaseCurrency          string     `json:"baseCcy"`
	QuoteCurrency         string     `json:"quoteCcy"`
	Side                  order.Side `json:"side"`
	Size                  float64    `json:"sz,string"`
	SizeCurrency          string     `json:"szCcy"`
	QuoteID               string     `json:"quoteId"`
	ClientSupplierOrderID string     `json:"clTReqId"`
	Tag                   string     `json:"tag"`
}

// ConvertTradeResponse response
type ConvertTradeResponse struct {
	BaseCurrency          string    `json:"baseCcy"`
	ClientSupplierOrderID string    `json:"clTReqId"`
	FillBaseSize          float64   `json:"fillBaseSz,string"`
	FillPrice             string    `json:"fillPx"`
	FillQuoteSize         float64   `json:"fillQuoteSz,string"`
	InstrumentID          string    `json:"instId"`
	QuoteCurrency         string    `json:"quoteCcy"`
	QuoteID               string    `json:"quoteId"`
	Side                  string    `json:"side"`
	State                 string    `json:"state"`
	TradeID               string    `json:"tradeId"`
	Timestamp             time.Time `json:"ts"`
}

// ConvertHistory holds convert trade history response
type ConvertHistory struct {
	InstrumentID  string    `json:"instId"`
	Side          string    `json:"side"`
	FillPrice     float64   `json:"fillPx,string"`
	BaseCurrency  string    `json:"baseCcy"`
	QuoteCurrency string    `json:"quoteCcy"`
	FillBaseSize  float64   `json:"fillBaseSz,string"`
	State         string    `json:"state"`
	TradeID       string    `json:"tradeId"`
	FillQuoteSize float64   `json:"fillQuoteSz,string"`
	Timestamp     time.Time `json:"ts"`
}

type Account struct {
	AdjEq       string           `json:"adjEq"`
	Details     []*AccountDetail `json:"details"`
	Imr         string           `json:"imr"` // Frozen equity for open positions and pending orders in USD level Applicable to Multi-currency margin and Portfolio margin
	IsoEq       string           `json:"isoEq"`
	MgnRatio    string           `json:"mgnRatio"`
	Mmr         string           `json:"mmr"` // Maintenance margin requirement in USD level Applicable to Multi-currency margin and Portfolio margin
	NotionalUsd string           `json:"notionalUsd"`
	OrdFroz     string           `json:"ordFroz"` // Margin frozen for pending orders in USD level Applicable to Multi-currency margin and Portfolio margin
	TotalEquity string           `json:"totalEq"` // Total Equity in USD level
	UpdateTime  time.Time        `json:"uTime"`   // UpdateTime
}

// AccountDetail account detail information.
type AccountDetail struct {
	AvailableBalance              string    `json:"availBal"`
	AvailableEquity               string    `json:"availEq"`
	CashBalance                   string    `json:"cashBal"` // Cash Balance
	Currency                      string    `json:"ccy"`
	CrossLiab                     string    `json:"crossLiab"`
	DiscountEquity                string    `json:"disEq"`
	EquityOfCurrency              string    `json:"eq"`
	EquityUsd                     string    `json:"eqUsd"`
	FrozenBalance                 string    `json:"frozenBal"`
	Interest                      string    `json:"interest"`
	IsoEquity                     string    `json:"isoEq"`
	IsolatedLiabilities           string    `json:"isoLiab"`
	IsoUpl                        string    `json:"isoUpl"` // Isolated unrealized profit and loss of the currency applicable to Single-currency margin and Multi-currency margin and Portfolio margin
	LiabilitiesOfCurrency         string    `json:"liab"`
	MaxLoan                       string    `json:"maxLoan"`
	MarginRatio                   string    `json:"mgnRatio"`      // Equity of the currency
	NotionalLever                 string    `json:"notionalLever"` // Leverage of the currency applicable to Single-currency margin
	OpenOrdersMarginFrozen        string    `json:"ordFrozen"`
	Twap                          string    `json:"twap"`
	UpdateTime                    time.Time `json:"uTime"`
	UnrealizedProfit              string    `json:"upl"`
	UnrealizedCurrencyLiabilities string    `json:"uplLiab"`
	StrategyEquity                string    `json:"stgyEq"`  // strategy equity
	TotalEquity                   string    `json:"totalEq"` // Total equity in USD level
}

// AccountPosition account position.
type AccountPosition struct {
	AutoDeleverging               string    `json:"adl"`      // Auto-deleveraging (ADL) indicator Divided into 5 levels, from 1 to 5, the smaller the number, the weaker the adl intensity.
	AvailablePosition             string    `json:"availPos"` // Position that can be closed Only applicable to MARGIN, FUTURES/SWAP in the long-short mode, OPTION in Simple and isolated OPTION in margin Account.
	AveragePrice                  string    `json:"avgPx"`
	CreationTime                  time.Time `json:"cTime"`
	Currency                      string    `json:"ccy"`
	DeltaBS                       string    `json:"deltaBS"` // delta：Black-Scholes Greeks in dollars,only applicable to OPTION
	DeltaPA                       string    `json:"deltaPA"` // delta：Greeks in coins,only applicable to OPTION
	GammaBS                       string    `json:"gammaBS"` // gamma：Black-Scholes Greeks in dollars,only applicable to OPTION
	GammaPA                       string    `json:"gammaPA"` // gamma：Greeks in coins,only applicable to OPTION
	InitionMarginRequirement      string    `json:"imr"`     // Initial margin requirement, only applicable to cross.
	InstrumentID                  string    `json:"instId"`
	InstrumentType                string    `json:"instType"`
	Interest                      string    `json:"interest"`
	USDPrice                      string    `json:"usdPx"`
	LastTradePrice                string    `json:"last"`
	Leverage                      string    `json:"lever"`   //Leverage, not applicable to OPTION seller
	Liabilities                   string    `json:"liab"`    // Liabilities, only applicable to MARGIN.
	LiabilitiesCurrency           string    `json:"liabCcy"` // Liabilities currency, only applicable to MARGIN.
	LiquidationPrice              string    `json:"liqPx"`   // Estimated liquidation price Not applicable to OPTION
	MarkPx                        string    `json:"markPx"`
	Margin                        string    `json:"margin"`
	MgnMode                       string    `json:"mgnMode"`
	MgnRatio                      string    `json:"mgnRatio"`
	MaintainanceMarginRequirement string    `json:"mmr"`         // Maintenance margin requirement in USD level Applicable to Multi-currency margin and Portfolio margin
	NotionalUsd                   string    `json:"notionalUsd"` // Quality of Positions -- usd
	OptionValue                   string    `json:"optVal"`      // Option Value, only application to position.
	QuantityOfPosition            string    `json:"pos"`         // Quantity of positions,In the mode of autonomous transfer from position to position, after the deposit is transferred, a position with pos of 0 will be generated
	PositionCurrency              string    `json:"posCcy"`
	PositionID                    string    `json:"posId"`
	PositionSide                  string    `json:"posSide"`
	ThetaBS                       string    `json:"thetaBS"` // theta：Black-Scholes Greeks in dollars,only applicable to OPTION
	ThetaPA                       string    `json:"thetaPA"` // theta：Greeks in coins,only applicable to OPTION
	TradeID                       string    `json:"tradeId"`
	UpdatedTime                   time.Time `json:"uTime"`                     // Latest time position was adjusted,
	Upl                           float64   `json:"upl,string,omitempty"`      // Unrealized profit and loss
	UPLRatio                      float64   `json:"uplRatio,string,omitempty"` // Unrealized profit and loss ratio
	VegaBS                        string    `json:"vegaBS"`                    // vega：Black-Scholes Greeks in dollars,only applicable to OPTION
	VegaPA                        string    `json:"vegaPA"`                    // vega：Greeks in coins,only applicable to OPTION
	// PTime                    time.Time `json:"pTime"`
}

// AccountPositionHistory hold account position history.
type AccountPositionHistory struct {
	CreationTime       time.Time `json:"cTime"`
	Currency           string    `json:"ccy"`
	CloseAveragePrice  float64   `json:"closeAvgPx,string,omitempty"`
	CloseTotalPosition float64   `json:"closeTotalPos,string,omitempty"`
	InstrumentID       string    `json:"instId"`
	InstrumentType     string    `json:"instType"`
	Leverage           string    `json:"lever"`
	ManagementMode     string    `json:"mgnMode"`
	OpenAveragePrice   string    `json:"openAvgPx"`
	OpenMaxPosition    string    `json:"openMaxPos"`
	ProfitAndLoss      float64   `json:"pnl,string,omitempty"`
	ProfitAndLossRatio float64   `json:"pnlRatio,string,omitempty"`
	PositionID         string    `json:"posId"`
	PositionSide       string    `json:"posSide"`
	TriggerPrice       string    `json:"triggerPx"`
	Type               string    `json:"type"`
	UpdateTime         time.Time `json:"uTime"`
	Underlying         string    `json:"uly"`
}

// AccountBalanceData
type AccountBalanceData struct {
	Currency       string `json:"ccy"`
	DiscountEquity string `json:"disEq"` // discount equity of the currency in USD level.
	Equity         string `json:"eq"`    // Equity of the currency
}

// PositionData holds account position data.
type PositionData struct {
	BaseBal          string `json:"baseBal"`
	Currency         string `json:"ccy"`
	InstrumentID     string `json:"instId"`
	InstrumentType   string `json:"instType"`
	MamagementMode   string `json:"mgnMode"`
	NotionalCurrency string `json:"notionalCcy"`
	NotionalUsd      string `json:"notionalUsd"`
	Position         string `json:"pos"`
	PositionedCcy    string `json:"posCcy"`
	PositionedID     string `json:"posId"`
	PositionedSide   string `json:"posSide"`
	QuoteBalance     string `json:"quoteBal"`
}

// AccountAndPositionRisk holds information.
type AccountAndPositionRisk struct {
	AdjEq               string                `json:"adjEq"`
	AccountBalanceDatas []*AccountBalanceData `json:"balData"`
	PosData             []*PositionData       `json:"posData"`
	Timestamp           time.Time             `json:"ts"`
}

//
type BillsDetailQueryParameter struct {
	InstrumentType string // Instrument type "SPOT" "MARGIN" "SWAP" "FUTURES" "OPTION"
	Currency       string
	MarginMode     string // Margin mode "isolated" "cross"
	ContractType   string // Contract type "linear" & "inverse" Only applicable to FUTURES/SWAP
	BillType       uint   // Bill type 1: Transfer 2: Trade 3: Delivery 4: Auto token conversion 5: Liquidation 6: Margin transfer 7: Interest deduction 8: Funding fee 9: ADL 10: Clawback 11: System token conversion 12: Strategy transfer 13: ddh
	BillSubType    uint   // allowed bill substype values are [ 1,2,3,4,5,6,9,11,12,14,160,161,162,110,111,118,119,100,101,102,103,104,105,106,110,125,126,127,128,131,132,170,171,172,112,113,117,173,174,200,201,202,203 ], link: https://www.okx.com/docs-v5/en/#rest-api-account-get-bills-details-last-7-days
	After          string
	Before         string
	BeginTime      time.Time
	EndTime        time.Time
	Limit          uint
}

// BillsDetailResponse represents account bills informaiton.
type BillsDetailResponse struct {
	Balance                    string    `json:"bal"`
	BalanceChange              string    `json:"balChg"`
	BillID                     string    `json:"billId"`
	Currency                   string    `json:"ccy"`
	ExecType                   string    `json:"execType"` // Order flow type, T：taker M：maker
	Fee                        string    `json:"fee"`      // Fee Negative number represents the user transaction fee charged by the platform. Positive number represents rebate.
	From                       string    `json:"from"`     // The remitting account 6: FUNDING 18: Trading account When bill type is not transfer, the field returns "".
	InstrumentID               string    `json:"instId"`
	InstrumentType             string    `json:"instType"`
	ManegementMode             string    `json:"mgnMode"`
	Notes                      string    `json:"notes"` // notes When bill type is not transfer, the field returns "".
	OrderID                    string    `json:"ordId"`
	ProfitAndLoss              string    `json:"pnl"`
	PositionLevelBalance       string    `json:"posBal"`
	PositionLevelBalanceChange string    `json:"posBalChg"`
	SubType                    string    `json:"subType"`
	Size                       string    `json:"sz"`
	To                         string    `json:"to"`
	Timestamp                  time.Time `json:"ts"`
	Type                       string    `json:"type"`
}

// AccountConfigurationResponse
type AccountConfigurationResponse struct {
	AccountLevel         uint   `json:"acctLv,string"` // 1: Simple 2: Single-currency margin 3: Multi-currency margin 4：Portfolio margin
	AutoLoan             bool   `json:"autoLoan"`      // Whether to borrow coins automatically true: borrow coins automatically false: not borrow coins automatically
	ContractIsolatedMode string `json:"ctIsoMode"`     // Contract isolated margin trading settings automatic：Auto transfers autonomy：Manual transfers
	GreeksType           string `json:"greeksType"`    // Current display type of Greeks PA: Greeks in coins BS: Black-Scholes Greeks in dollars
	Level                string `json:"level"`         // The user level of the current real trading volume on the platform, e.g lv1
	LevelTemporary       string `json:"levelTmp"`
	MarginIsolatedMode   string `json:"mgnIsoMode"` // Margin isolated margin trading settings automatic：Auto transfers autonomy：Manual transfers
	PositionMode         string `json:"posMode"`
	AccountID            string `json:"uid"`
}

// PositionMode
type PositionMode struct {
	PositionMode string `json:"posMode"` // "long_short_mode": long/short, only applicable to FUTURES/SWAP "net_mode": net
}

// SetLeverageInput
type SetLeverageInput struct {
	Leverage     string `json:"lever"`   // set leverage for isolated
	MarginMode   string `json:"mgnMode"` // Margin Mode "cross" and "isolated"
	InstrumentID string `json:"instId"`  // Optional:
	Currency     string `json:"ccy"`     // Optional:
	PositionSide string `json:"posSide"`
}

// SetLeverageResponse
type SetLeverageResponse struct {
	Leverage     string `json:"lever"`
	MarginMode   string `json:"mgnMode"` // Margin Mode "cross" and "isolated"
	InstrumentID string `json:"instId"`
	PositionSide string `json:"posSide"` // "long", "short", and "net"
}

// MaximumBuyAndSell get maximum buy , sell amount or open amount
type MaximumBuyAndSell struct {
	Currency     string `json:"ccy"`
	InstrumentID string `json:"instId"`
	MaximumBuy   string `json:"maxBuy"`
	MaximumSell  string `json:"maxSell"`
}

// MaximumTradableAmount represents get maximum tradable amount response
type MaximumTradableAmount struct {
	InstID    string `json:"instId"`
	AvailBuy  string `json:"availBuy"`
	AvailSell string `json:"availSell"`
}

// IncreaseDecreaseMarginInput
type IncreaseDecreaseMarginInput struct {
	InstrumentID      string  `json:"instId"`
	PositionSide      string  `json:"posSide"`
	Type              string  `json:"type"`
	Amount            float64 `json:"amt,string"`
	Currency          string  `json:"ccy"`
	AutoLoadTransffer bool    `json:"auto"`
	LoadTransffer     bool    `json:"loanTrans"`
}

// IncreateDecreaseMargin
type IncreaseDecreaseMargin struct {
	Amt      string `json:"amt"`
	Ccy      string `json:"ccy"`
	InstID   string `json:"instId"`
	Leverage string `json:"leverage"`
	PosSide  string `json:"posSide"`
	Type     string `json:"type"`
}

// LeverageResponse
type LeverageResponse struct {
	InstrumentID string `json:"instId"`
	MarginMode   string `json:"mgnMode"`
	PositionSide string `json:"posSide"`
	Leverage     uint   `json:"lever,string"`
}

// MaximumLoanInstrument
type MaximumLoanInstrument struct {
	InstID  string `json:"instId"`
	MgnMode string `json:"mgnMode"`
	MgnCcy  string `json:"mgnCcy"`
	MaxLoan string `json:"maxLoan"`
	Ccy     string `json:"ccy"`
	Side    string `json:"side"`
}

// TradeFeeRate holds trade fee rate information for a given instrument type.
type TradeFeeRate struct {
	Category         string    `json:"category"`
	DeliveryFeeRate  string    `json:"delivery"`
	Exercise         string    `json:"exercise"`
	InstrumentType   string    `json:"instType"`
	FeeRateLevel     string    `json:"level"`
	FeeRateMaker     string    `json:"maker"`
	FeeRateMakerUSDT string    `json:"makerU"`
	FeeRateTaker     string    `json:"taker"`
	FeeRateTakerUSDT string    `json:"takerU"`
	Timestamp        time.Time `json:"ts"`
}

// InterestAccruedData
type InterestAccruedData struct {
	Currency     string    `json:"ccy"`
	InstrumentID string    `json:"instId"`
	Interest     string    `json:"interest"`
	InterestRate string    `json:"interestRate"` // intereset rate in an hour.
	Liability    string    `json:"liab"`
	MarginMode   string    `json:"mgnMode"` //  	Margin mode "cross" "isolated"
	Timestamp    time.Time `json:"ts"`
	LoanType     string    `json:"type"`
}

// InterestRateResponse represents interest rate response.
type InterestRateResponse struct {
	InterstRate float64 `json:"interestRate,string"`
	Currency    string  `json:"ccy"`
}

// GreeksType
type GreeksType struct {
	GreeksType string `json:"greeksType"` // Display type of Greeks. PA: Greeks in coins BS: Black-Scholes Greeks in dollars
}

// IsolatedMode represents Isolated margin trading settings
type IsolatedMode struct {
	IsoMode        string `json:"isoMode"`        // "automatic":Auto transfers "autonomy":Manual transfers
	InstrumentType string `json:"type,omitempty"` // Instrument type "MARGIN" "CONTRACTS"
}

// MaximumWithdrawal
type MaximumWithdrawal struct {
	Currency            string `json:"ccy"`
	MaximumWithdrawal   string `json:"maxWd"`   // Max withdrawal (not allowing borrowed crypto transfer out under Multi-currency margin)
	MaximumWithdrawalEx string `json:"maxWdEx"` // Max withdrawal (allowing borrowed crypto transfer out under Multi-currency margin)
}

// AccountRiskState
type AccountRiskState struct {
	IsTheAccountAtRisk bool          `json:"atRisk"`
	AtRiskIdx          []interface{} `json:"atRiskIdx"` // derivatives risk unit list
	AtRiskMgn          []interface{} `json:"atRiskMgn"` // margin risk unit list
	Timestamp          time.Time     `json:"ts"`
}

// LoanBorrowAndReplayInput
type LoanBorrowAndReplayInput struct {
	Currency string  `json:"ccy"`
	Side     string  `json:"side,omitempty"`
	Amount   float64 `json:"amt,string,omitempty"`
}

// LoanBorrowAndReplay loans borrow and repay
type LoanBorrowAndReplay struct {
	Amount        string `json:"amt"`
	AvailableLoan string `json:"availLoan"`
	Currency      string `json:"ccy"`
	LoanQuota     string `json:"loanQuota"`
	PosLoan       string `json:"posLoan"`
	Side          string `json:"side"`
	UsedLoan      string `json:"usedLoan"`
}

// BorrowRepayHistory represents
type BorrowRepayHistory struct {
	Currency   string    `json:"ccy"`
	TradedLoan string    `json:"tradedLoan"`
	Timestamp  time.Time `json:"ts"`
	Type       string    `json:"type"`
	UsedLoan   string    `json:"usedLoan"`
}

// BorrowInterestAndLimitResponse
type BorrowInterestAndLimitResponse struct {
	Debt             string    `json:"debt"`
	Interest         string    `json:"interest"`
	NextDiscountTime time.Time `json:"nextDiscountTime"`
	NextInterestTime time.Time `json:"nextInterestTime"`
	Records          []struct {
		AvailLoan  string `json:"availLoan"`
		Currency   string `json:"ccy"`
		Interest   string `json:"interest"`
		LoanQuota  string `json:"loanQuota"`
		PosLoan    string `json:"posLoan"` // Frozon amount for current account Only applicable to VIP loans
		Rate       string `json:"rate"`
		SurplusLmt string `json:"surplusLmt"`
		UsedLmt    string `json:"usedLmt"`
		UsedLoan   string `json:"usedLoan"`
	} `json:"records"`
}

// PositionItem
type PositionItem struct {
	Position     string `json:"pos"`
	InstrumentID string `json:"instId"`
}

// PositionBuilderInput
type PositionBuilderInput struct {
	InstrumentType         string         `json:"instType,omitempty"`
	InstrumentID           string         `json:"instId,omitempty"`
	ImportExistingPosition bool           `json:"inclRealPos,omitempty"` // "true"：Import existing positions and hedge with simulated ones "false"：Only use simulated positions The default is true
	ListOfPositions        []PositionItem `json:"simPos,omitempty"`
	PositionsCount         uint           `json:"pos,omitempty"`
}

// PositionBuilderResponse represents a position builder endpoint response.
type PositionBuilderResponse struct {
	InitialMarginRequirement     string `json:"imr"` // Initial margin requirement of riskUnit dimension
	MaintenanceMarginRequirement string `json:"mmr"` // Maintenance margin requirement of riskUnit dimension
	Mr1                          string `json:"mr1"`
	Mr2                          string `json:"mr2"`
	Mr3                          string `json:"mr3"`
	Mr4                          string `json:"mr4"`
	Mr5                          string `json:"mr5"`
	Mr6                          string `json:"mr6"`
	Mr7                          string `json:"mr7"`
	PosData                      []struct {
		Delta              string `json:"delta"`
		Gamma              string `json:"gamma"`
		InstID             string `json:"instId"`
		InstType           string `json:"instType"`
		NotionalUsd        string `json:"notionalUsd"` // Quantity of positions usd
		QuantityOfPosition string `json:"pos"`         // Quantity of positions
		Theta              string `json:"theta"`       //Sensitivity of option price to remaining maturity
		Vega               string `json:"vega"`        // Sensitivity of option price to implied volatility
	} `json:"posData"` // List of positions
	RiskUnit  string    `json:"riskUnit"`
	Timestamp time.Time `json:"ts"`
}

// GreeksItem
type GreeksItem struct {
	ThetaBS   string    `json:"thetaBS"`
	ThetaPA   string    `json:"thetaPA"`
	DeltaBS   string    `json:"deltaBS"`
	DeltaPA   string    `json:"deltaPA"`
	GammaBS   string    `json:"gammaBS"`
	GammaPA   string    `json:"gammaPA"`
	VegaBS    string    `json:"vegaBS"`
	VegaPA    string    `json:"vegaPA"`
	Currency  string    `json:"ccy"`
	Timestamp time.Time `json:"ts"`
}

// CounterpartiesResponse
type CounterpartiesResponse struct {
	TraderName string `json:"traderName"`
	TraderCode string `json:"traderCode"`
	Type       string `json:"type"`
}

// CreateRFQInput RFQ create method input.
type CreateRFQInput struct {
	Anonymous           bool     `json:"anonymous"`
	CounterParties      []string `json:"counterparties"`
	ClientSuppliedRFQID string   `json:"clRfqId"`
	Legs                []struct {
		Size         string `json:"sz"`
		Side         string `json:"side"`
		InstrumentID string `json:"instId"`
		TgtCcy       string `json:"tgtCcy,omitempty"`
	} `json:"legs"`
}

// CancelRFQRequestParam
type CancelRFQRequestParam struct {
	RfqID               string `json:"rfqId"`
	ClientSuppliedRFQID string `json:"clRfqId"`
}

// CancelRFQRequestsParam
type CancelRFQRequestsParam struct {
	RfqID               []string `json:"rfqId"`
	ClientSuppliedRFQID []string `json:"clRfqId"`
}

// CancelRFQResponse
type CancelRFQResponse struct {
	RfqID   string `json:"rfqId"`
	ClRfqID string `json:"clRfqId"`
	SCode   string `json:"sCode"`
	SMsg    string `json:"sMsg"`
}

// TimestampResponse holds timestamp response only.
type TimestampResponse struct {
	Timestamp time.Time `json:"ts"`
}

// ExecuteQuoteParams
type ExecuteQuoteParams struct {
	RfqID   string `json:"rfqId"`
	QuoteID string `json:"quoteId"`
}

// ExecuteQuoteResponse
type ExecuteQuoteResponse struct {
	BlockTradedID         string    `json:"blockTdId"`
	RfqID                 string    `json:"rfqId"`
	ClientSuppliedRfqID   string    `json:"clRfqId"`
	QuoteID               string    `json:"quoteId"`
	ClientSuppliedQuoteID string    `json:"clQuoteId"`
	TraderCode            string    `json:"tTraderCode"`
	MakerTraderCode       string    `json:"mTraderCode"`
	CreationTime          time.Time `json:"cTime"`
	Legs                  []struct {
		Price        string `json:"px"`
		Size         string `json:"sz"`
		InstrumentID string `json:"instId"`
		Side         string `json:"side"`
		Fee          string `json:"fee"`
		FeeCurrency  string `json:"feeCcy"`
		TradeID      string `json:"tradeId"`
	} `json:"legs"`
}

// CreateQuoteParams holds information related to create quote.
type CreateQuoteParams struct {
	RfqID                 string     `json:"rfqId"`
	ClientSuppliedQuoteID string     `json:"clQuoteId"`
	QuoteSide             order.Side `json:"quoteSide"`
	Legs                  []struct {
		Price          float64    `json:"px,string"`
		SizeOfQuoteLeg float64    `json:"sz,string"`
		InstrumentID   string     `json:"instId"`
		Side           order.Side `json:"side"`
	} `json:"legs"`
}

// QuoteLeg the legs of the Quote.
type QuoteLeg struct {
	Price        string `json:"px"`
	Size         string `json:"sz"`
	InstrumentID string `json:"instId"`
	Side         string `json:"side"`
	TgtCcy       string `json:"tgtCcy"`
}

// QuoteResponse holds create quote response variables.
type QuoteResponse struct {
	CreationTime          time.Time  `json:"cTime"`
	UpdateTime            time.Time  `json:"uTime"`
	ValidUntil            time.Time  `json:"validUntil"`
	QuoteID               string     `json:"quoteId"`
	ClientSuppliedQuoteID string     `json:"clQuoteId"`
	RfqID                 string     `json:"rfqId"`
	QuoteSide             string     `json:"quoteSide"`
	ClientSuppliedRfqID   string     `json:"clRfqId"`
	TraderCode            string     `json:"traderCode"`
	State                 string     `json:"state"`
	Legs                  []QuoteLeg `json:"legs"`
}

// CancelQuoteRequestParams
type CancelQuoteRequestParams struct {
	QuoteID               string `json:"quoteId"`
	ClientSuppliedQuoteID string `json:"clQuoteId"`
}

// CancelQuotesRequestParams
type CancelQuotesRequestParams struct {
	QuoteIDs               []string `json:"quoteId"`
	ClientSuppliedQuoteIDs []string `json:"clQuoteId"`
}

// CancelQuoteResponse
type CancelQuoteResponse struct {
	QuoteID               string `json:"quoteId"`
	ClientSuppliedQuoteID string `json:"clQuoteId"`
	SCode                 string `json:"sCode"`
	SMsg                  string `json:"sMsg"`
}

// RfqRequestParams
type RfqRequestParams struct {
	RfqID               string
	ClientSuppliedRfqID string
	State               string
	BeginingID          string
	EndID               string
	Limit               uint
}

// RFQResponse
type RFQResponse struct {
	CreateTime          time.Time `json:"cTime"`
	UpdateTime          time.Time `json:"uTime"`
	ValidUntil          time.Time `json:"validUntil"`
	TraderCode          string    `json:"traderCode"`
	RFQID               string    `json:"rfqId"`
	ClientSuppliedRFQID string    `json:"clRfqId"`
	State               string    `json:"state"`
	Counterparties      []string  `json:"counterparties"`
	Legs                []struct {
		InstrumentID string `json:"instId"`
		Size         string `json:"sz"`
		Side         string `json:"side"`
		TgtCcy       string `json:"tgtCcy"`
	} `json:"legs"`
}

// QuoteRequestParams request params.
type QuoteRequestParams struct {
	RfqID                 string
	ClientSuppliedRfqID   string
	QuoteID               string
	ClientSuppliedQuoteID string
	State                 string
	BeginID               string
	EndID                 string
	Limit                 int
}

// RFQTradesRequestParams
type RFQTradesRequestParams struct {
	RfqID                 string
	ClientSuppliedRfqID   string
	QuoteID               string
	BlockTradeID          string
	ClientSuppliedQuoteID string
	State                 string
	BeginID               string
	EndID                 string
	Limit                 uint
}

// RfqTradeResponse
type RfqTradeResponse struct {
	RfqID                 string        `json:"rfqId"`
	ClientSuppliedRfqID   string        `json:"clRfqId"`
	QuoteID               string        `json:"quoteId"`
	ClientSuppliedQuoteID string        `json:"clQuoteId"`
	BlockTradeID          string        `json:"blockTdId"`
	Legs                  []RFQTradeLeg `json:"legs"`
	CreationTime          time.Time     `json:"cTime"`
	TakerTraderCode       string        `json:"tTraderCode"`
	MakerTraderCode       string        `json:"mTraderCode"`
}

// RFQTradeLeg
type RFQTradeLeg struct {
	InstrumentID string  `json:"instId"`
	Side         string  `json:"side"`
	Size         string  `json:"sz"`
	Price        float64 `json:"px,string"`
	TradeID      string  `json:"tradeId"`
	Fee          float64 `json:"fee,string,omitempty"`
	FeeCurrency  string  `json:"feeCcy,omitempty"`
}

// PublicTradesResponse
type PublicTradesResponse struct {
	BlockTradeID string        `json:"blockTdId"`
	Legs         []RFQTradeLeg `json:"legs"`
	CreationTime time.Time     `json:"cTime"`
}

// SubaccountInfo
type SubaccountInfo struct {
	Enable          bool      `json:"enable"`
	SubAccountName  string    `json:"subAcct"`
	SubaccountType  string    `json:"type"` // sub-account note
	SubaccountLabel string    `json:"label"`
	MobileNumber    string    `json:"mobile"`      // Mobile number that linked with the sub-account.
	GoogleAuth      bool      `json:"gAuth"`       // If the sub-account switches on the Google Authenticator for login authentication.
	CanTransferOut  bool      `json:"canTransOut"` // If can tranfer out, false: can not tranfer out, true: can transfer.
	Timestamp       time.Time `json:"ts"`
}

// SubaccountBalanceDetail
type SubaccountBalanceDetail struct {
	AvailableBalance               string    `json:"availBal"`
	AvailableEquity                string    `json:"availEq"`
	CashBalance                    string    `json:"cashBal"`
	Currency                       string    `json:"ccy"`
	CrossLiability                 string    `json:"crossLiab"`
	DiscountEquity                 string    `json:"disEq"`
	Equity                         string    `json:"eq"`
	EquityUsd                      string    `json:"eqUsd"`
	FrozenBalance                  string    `json:"frozenBal"`
	Interest                       string    `json:"interest"`
	IsoEquity                      string    `json:"isoEq"`
	IsolatedLiabilities            string    `json:"isoLiab"`
	LiabilitiesOfCurrency          string    `json:"liab"`
	MaxLoan                        string    `json:"maxLoan"`
	MarginRatio                    string    `json:"mgnRatio"`
	NotionalLeverage               string    `json:"notionalLever"`
	OrdFrozen                      string    `json:"ordFrozen"`
	Twap                           string    `json:"twap"`
	UpdateTime                     time.Time `json:"uTime"`
	UnrealizedProfitAndLoss        string    `json:"upl"`
	UnrealizedProfitAndLiabilities string    `json:"uplLiab"`
}

// SubaccountBalanceResponse
type SubaccountBalanceResponse struct {
	AdjustedEffectiveEquity       string                    `json:"adjEq"`
	Details                       []SubaccountBalanceDetail `json:"details"`
	Imr                           string                    `json:"imr"`
	IsolatedMarginEquity          string                    `json:"isoEq"`
	MarginRatio                   string                    `json:"mgnRatio"`
	MaintainanceMarginRequirement string                    `json:"mmr"`
	NotionalUsd                   string                    `json:"notionalUsd"`
	OrdFroz                       string                    `json:"ordFroz"`
	TotalEq                       string                    `json:"totalEq"`
	UpdateTime                    time.Time                 `json:"uTime"`
}

// FundingBalance holds function balance.
type FundingBalance struct {
	AvailableBalance string `json:"availBal"`
	Balance          string `json:"bal"`
	Currency         string `json:"ccy"`
	FrozenBalance    string `json:"frozenBal"`
}

// SubaccountBillItem
type SubaccountBillItem struct {
	BillID                 string    `json:"billId"`
	Type                   string    `json:"type"`
	AccountCurrencyBalance string    `json:"ccy"`
	Amount                 string    `json:"amt"`
	SubAccount             string    `json:"subAcct"`
	Timestamp              time.Time `json:"ts"`
}

// TransferIDInfo
type TransferIDInfo struct {
	TransferID string `json:"transId"`
}

// PermissingOfTransfer
type PermissingOfTransfer struct {
	SubAcct     string `json:"subAcct"`
	CanTransOut bool   `json:"canTransOut"`
}

// SubaccountName
type SubaccountName struct {
	SubaccountName string `json:"subAcct"`
}

// GridAlgoOrder
type GridAlgoOrder struct {
	InstrumentID string `json:"instId"`
	AlgoOrdType  string `json:"algoOrdType"`
	MaxPrice     string `json:"maxPx"`
	MinPrice     string `json:"minPx"`
	GridNumber   string `json:"gridNum"`
	GridType     string `json:"runType"`
	Size         string `json:"sz"`
	Direction    string `json:"direction"`
	Lever        string `json:"lever"`
}
