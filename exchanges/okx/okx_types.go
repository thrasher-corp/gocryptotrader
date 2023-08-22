package okx

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Trade Modes

	// TradeModeCross trade mode, cross
	TradeModeCross = "cross"
	// TradeModeIsolated trade mode, isolated
	TradeModeIsolated = "isolated"
	// TradeModeCash trade mode, cash
	TradeModeCash = "cash"

	// Algo Order Types

	// AlgoOrdTypeGrid algo order type,grid
	AlgoOrdTypeGrid = "grid"
	// AlgoOrdTypeContractGrid algo order type, contract_grid
	AlgoOrdTypeContractGrid = "contract_grid"

	// Position Side for placing order
	positionSideLong  = "long"
	positionSideShort = "short"
	positionSideNet   = "net"
)

const (
	// OkxOrderLimit Limit order
	OkxOrderLimit = "limit"
	// OkxOrderMarket Market order
	OkxOrderMarket = "market"
	// OkxOrderPostOnly POST_ONLY order type
	OkxOrderPostOnly = "post_only"
	// OkxOrderFOK fill or kill order type
	OkxOrderFOK = "fok"
	// OkxOrderIOC IOC (immediate or cancel)
	OkxOrderIOC = "ioc"
	// OkxOrderOptimalLimitIOC OPTIMAL_LIMIT_IOC
	OkxOrderOptimalLimitIOC = "optimal_limit_ioc"

	// Instrument Types ( Asset Types )

	okxInstTypeFutures  = "FUTURES"  // Okx Instrument Type "futures"
	okxInstTypeANY      = "ANY"      // Okx Instrument Type ""
	okxInstTypeSpot     = "SPOT"     // Okx Instrument Type "spot"
	okxInstTypeSwap     = "SWAP"     // Okx Instrument Type "swap"
	okxInstTypeOption   = "OPTION"   // Okx Instrument Type "option"
	okxInstTypeMargin   = "MARGIN"   // Okx Instrument Type "margin"
	okxInstTypeContract = "CONTRACT" // Okx Instrument Type "contract"

	operationSubscribe   = "subscribe"
	operationUnsubscribe = "unsubscribe"
	operationLogin       = "login"
)

// testNetKey this key is designed for using the testnet endpoints
// setting context.WithValue(ctx, testNetKey("testnet"), useTestNet)
// will ensure the appropriate headers are sent to OKx to use the testnet
type testNetKey string

var testNetVal = testNetKey("testnet")

// Market Data Endpoints

// TickerResponse represents the market data endpoint ticker detail
type TickerResponse struct {
	InstrumentType string            `json:"instType"`
	InstrumentID   string            `json:"instId"`
	LastTradePrice okxNumericalValue `json:"last"`
	LastTradeSize  okxNumericalValue `json:"lastSz"`
	BestAskPrice   okxNumericalValue `json:"askPx"`
	BestAskSize    okxNumericalValue `json:"askSz"`
	BestBidPrice   okxNumericalValue `json:"bidPx"`
	BestBidSize    okxNumericalValue `json:"bidSz"`
	Open24H        okxNumericalValue `json:"open24h"`
	High24H        okxNumericalValue `json:"high24h"`
	Low24H         okxNumericalValue `json:"low24h"`
	VolCcy24H      okxNumericalValue `json:"volCcy24h"`
	Vol24H         okxNumericalValue `json:"vol24h"`

	OpenPriceInUTC0          string           `json:"sodUtc0"`
	OpenPriceInUTC8          string           `json:"sodUtc8"`
	TickerDataGenerationTime okxUnixMilliTime `json:"ts"`
}

// IndexTicker represents Index ticker data.
type IndexTicker struct {
	InstID    string           `json:"instId"`
	IdxPx     float64          `json:"idxPx,string"`
	High24H   float64          `json:"high24h,string"`
	SodUtc0   float64          `json:"sodUtc0,string"`
	Open24H   float64          `json:"open24h,string"`
	Low24H    float64          `json:"low24h,string"`
	SodUtc8   float64          `json:"sodUtc8,string"`
	Timestamp okxUnixMilliTime `json:"ts"`
}

// OrderBookResponse holds the order asks and bids at a specific timestamp
type OrderBookResponse struct {
	Asks                [][4]string      `json:"asks"`
	Bids                [][4]string      `json:"bids"`
	GenerationTimeStamp okxUnixMilliTime `json:"ts"`
}

// OrderBookResponseDetail holds the order asks and bids in a struct field with the corresponding order generation timestamp.
type OrderBookResponseDetail struct {
	Asks                []OrderAsk
	Bids                []OrderBid
	GenerationTimestamp time.Time
}

// OrderAsk represents currencies bid detailed information.
type OrderAsk struct {
	DepthPrice        float64
	NumberOfContracts float64
	LiquidationOrders int64
	NumberOfOrders    int64
}

// OrderBid represents currencies bid detailed information.
type OrderBid struct {
	DepthPrice        float64
	BaseCurrencies    float64
	LiquidationOrders int64
	NumberOfOrders    int64
}

// GetOrderBookResponseDetail returns the OrderBookResponseDetail instance from OrderBookResponse object.
func (a *OrderBookResponse) GetOrderBookResponseDetail() (*OrderBookResponseDetail, error) {
	asks, er := a.GetAsks()
	if er != nil {
		return nil, er
	}
	bids, er := a.GetBids()
	if er != nil {
		return nil, er
	}
	return &OrderBookResponseDetail{
		Asks:                asks,
		Bids:                bids,
		GenerationTimestamp: a.GenerationTimeStamp.Time(),
	}, nil
}

// GetAsks returns list of asks from an order book response instance.
func (a *OrderBookResponse) GetAsks() ([]OrderAsk, error) {
	asks := make([]OrderAsk, len(a.Asks))
	for x := range a.Asks {
		depthPrice, er := strconv.ParseFloat(a.Asks[x][0], 64)
		if er != nil {
			return nil, er
		}
		contracts, er := strconv.ParseFloat(a.Asks[x][1], 64)
		if er != nil {
			return nil, er
		}
		liquidation, er := strconv.ParseInt(a.Asks[x][2], 10, 64)
		if er != nil {
			return nil, er
		}
		orders, er := strconv.ParseInt(a.Asks[x][3], 10, 64)
		if er != nil {
			return nil, er
		}
		asks[x] = OrderAsk{
			DepthPrice:        depthPrice,
			NumberOfContracts: contracts,
			LiquidationOrders: liquidation,
			NumberOfOrders:    orders,
		}
	}
	return asks, nil
}

// GetBids returns list of order bids instance from list of slice.
func (a *OrderBookResponse) GetBids() ([]OrderBid, error) {
	bids := make([]OrderBid, len(a.Bids))
	for x := range a.Bids {
		depthPrice, er := strconv.ParseFloat(a.Bids[x][0], 64)
		if er != nil {
			return nil, er
		}
		currencies, er := strconv.ParseFloat(a.Bids[x][1], 64)
		if er != nil {
			return nil, er
		}
		liquidation, er := strconv.ParseInt(a.Bids[x][2], 10, 64)
		if er != nil {
			return nil, er
		}
		orders, er := strconv.ParseInt(a.Bids[x][3], 10, 64)
		if er != nil {
			return nil, er
		}
		bids[x] = OrderBid{
			DepthPrice:        depthPrice,
			BaseCurrencies:    currencies,
			LiquidationOrders: liquidation,
			NumberOfOrders:    orders,
		}
	}
	return bids, nil
}

// CandleStick  holds candlestick price data
type CandleStick struct {
	OpenTime         time.Time
	OpenPrice        float64
	HighestPrice     float64
	LowestPrice      float64
	ClosePrice       float64
	Volume           float64
	QuoteAssetVolume float64
}

// TradeResponse represents the recent transaction instance.
type TradeResponse struct {
	InstrumentID string           `json:"instId"`
	TradeID      string           `json:"tradeId"`
	Price        float64          `json:"px,string"`
	Quantity     float64          `json:"sz,string"`
	Side         string           `json:"side"`
	Timestamp    okxUnixMilliTime `json:"ts"`
}

// TradingVolumeIn24HR response model.
type TradingVolumeIn24HR struct {
	BlockVolumeInCNY   okxNumericalValue `json:"blockVolCny"`
	BlockVolumeInUSD   okxNumericalValue `json:"blockVolUsd"`
	TradingVolumeInUSD float64           `json:"volUsd,string"`
	TradingVolumeInCny float64           `json:"volCny,string"`
	Timestamp          okxUnixMilliTime  `json:"ts"`
}

// OracleSmartContractResponse returns the crypto price of signing using Open Oracle smart contract.
type OracleSmartContractResponse struct {
	Messages   []string          `json:"messages"`
	Prices     map[string]string `json:"prices"`
	Signatures []string          `json:"signatures"`
	Timestamp  okxUnixMilliTime  `json:"timestamp"`
}

// UsdCnyExchangeRate the exchange rate for converting from USD to CNV
type UsdCnyExchangeRate struct {
	UsdCny float64 `json:"usdCny,string"`
}

// IndexComponent represents index component data on the market
type IndexComponent struct {
	Components []IndexComponentItem `json:"components"`
	Last       float64              `json:"last,string"`
	Index      string               `json:"index"`
	Timestamp  okxUnixMilliTime     `json:"ts"`
}

// IndexComponentItem an item representing the index component item
type IndexComponentItem struct {
	Symbol          string `json:"symbol"`
	SymbolPairPrice string `json:"symbolPx"`
	Weights         string `json:"wgt"`
	ConvertToPrice  string `json:"cnvPx"`
	ExchangeName    string `json:"exch"`
}

// InstrumentsFetchParams request params for requesting list of instruments
type InstrumentsFetchParams struct {
	InstrumentType string // Mandatory
	Underlying     string // Optional
	InstrumentID   string // Optional
}

// Instrument  representing an instrument with open contract.
type Instrument struct {
	InstrumentType                  asset.Item `json:"instType"`
	InstrumentID                    string     `json:"instId"`
	InstrumentFamily                string     `json:"instFamily"`
	Underlying                      string     `json:"uly"`
	Category                        string     `json:"category"`
	BaseCurrency                    string     `json:"baseCcy"`
	QuoteCurrency                   string     `json:"quoteCcy"`
	SettlementCurrency              string     `json:"settleCcy"`
	ContractValue                   string     `json:"ctVal"`
	ContractMultiplier              string     `json:"ctMult"`
	ContractValueCurrency           string     `json:"ctValCcy"`
	OptionType                      string     `json:"optType"`
	StrikePrice                     string     `json:"stk"`
	ListTime                        time.Time  `json:"listTime"`
	ExpTime                         time.Time  `json:"expTime"`
	MaxLeverage                     float64    `json:"lever"`
	TickSize                        float64    `json:"tickSz"`
	LotSize                         float64    `json:"lotSz"`
	MinimumOrderSize                float64    `json:"minSz"`
	ContractType                    string     `json:"ctType"`
	Alias                           string     `json:"alias"`
	State                           string     `json:"state"`
	MaxQuantityOfSpotLimitOrder     float64    `json:"maxLmtSz"`
	MaxQuantityOfMarketLimitOrder   float64    `json:"maxMktSz"`
	MaxQuantityOfSpotTwapLimitOrder float64    `json:"maxTwapSz"`
	MaxSpotIcebergSize              float64    `json:"maxIcebergSz"`
	MaxTriggerSize                  float64    `json:"maxTriggerSz"`
	MaxStopSize                     float64    `json:"maxStopSz"`
}

// DeliveryHistoryDetail holds instrument id and delivery price information detail
type DeliveryHistoryDetail struct {
	Type          string  `json:"type"`
	InstrumentID  string  `json:"insId"`
	DeliveryPrice float64 `json:"px,string"`
}

// DeliveryHistory represents list of delivery history detail items and timestamp information
type DeliveryHistory struct {
	Timestamp okxUnixMilliTime        `json:"ts"`
	Details   []DeliveryHistoryDetail `json:"details"`
}

// OpenInterest Retrieve the total open interest for contracts on OKX.
type OpenInterest struct {
	InstrumentType       asset.Item       `json:"instType"`
	InstrumentID         string           `json:"instId"`
	OpenInterest         float64          `json:"oi,string"`
	OpenInterestCurrency float64          `json:"oiCcy,string"`
	Timestamp            okxUnixMilliTime `json:"ts"`
}

// FundingRateResponse response data for the Funding Rate for an instruction type
type FundingRateResponse struct {
	FundingRate     convert.StringToFloat64 `json:"fundingRate"`
	RealisedRate    convert.StringToFloat64 `json:"realizedRate"`
	FundingTime     okxUnixMilliTime        `json:"fundingTime"`
	InstrumentID    string                  `json:"instId"`
	InstrumentType  string                  `json:"instType"`
	NextFundingRate convert.StringToFloat64 `json:"nextFundingRate"`
	NextFundingTime okxUnixMilliTime        `json:"nextFundingTime"`
}

// LimitPriceResponse hold an information for
type LimitPriceResponse struct {
	InstrumentType string           `json:"instType"`
	InstID         string           `json:"instId"`
	BuyLimit       float64          `json:"buyLmt,string"`
	SellLimit      float64          `json:"sellLmt,string"`
	Timestamp      okxUnixMilliTime `json:"ts"`
}

// OptionMarketDataResponse holds response data for option market data
type OptionMarketDataResponse struct {
	InstrumentType string           `json:"instType"`
	InstrumentID   string           `json:"instId"`
	Underlying     string           `json:"uly"`
	Delta          float64          `json:"delta,string"`
	Gamma          float64          `json:"gamma,string"`
	Theta          float64          `json:"theta,string"`
	Vega           float64          `json:"vega,string"`
	DeltaBS        float64          `json:"deltaBS,string"`
	GammaBS        float64          `json:"gammaBS,string"`
	ThetaBS        float64          `json:"thetaBS,string"`
	VegaBS         float64          `json:"vegaBS,string"`
	RealVol        string           `json:"realVol"`
	BidVolatility  string           `json:"bidVol"`
	AskVolatility  float64          `json:"askVol,string"`
	MarkVolatility float64          `json:"markVol,string"`
	Leverage       float64          `json:"lever,string"`
	ForwardPrice   string           `json:"fwdPx"`
	Timestamp      okxUnixMilliTime `json:"ts"`
}

// DeliveryEstimatedPrice holds an estimated delivery or exercise price response.
type DeliveryEstimatedPrice struct {
	InstrumentType         string           `json:"instType"`
	InstrumentID           string           `json:"instId"`
	EstimatedDeliveryPrice string           `json:"settlePx"`
	Timestamp              okxUnixMilliTime `json:"ts"`
}

// DiscountRate represents the discount rate amount, currency, and other discount related information.
type DiscountRate struct {
	Amount            string                 `json:"amt"`
	Currency          string                 `json:"ccy"`
	DiscountInfo      []DiscountRateInfoItem `json:"discountInfo"`
	DiscountRateLevel string                 `json:"discountLv"`
}

// DiscountRateInfoItem represents discount info list item for discount rate response
type DiscountRateInfoItem struct {
	DiscountRate string            `json:"discountRate"`
	MaxAmount    okxNumericalValue `json:"maxAmt"`
	MinAmount    okxNumericalValue `json:"minAmt"`
}

// ServerTime returning  the server time instance.
type ServerTime struct {
	Timestamp okxUnixMilliTime `json:"ts"`
}

// LiquidationOrderRequestParams holds information to request liquidation orders
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

// LiquidationOrder represents liquidation order item detailed information
type LiquidationOrder struct {
	Details        []LiquidationOrderDetailItem `json:"details"`
	InstrumentID   string                       `json:"instId"`
	InstrumentType string                       `json:"instType"`
	TotalLoss      string                       `json:"totalLoss"`
	Underlying     string                       `json:"uly"`
}

// LiquidationOrderDetailItem represents the detail information of liquidation order
type LiquidationOrderDetailItem struct {
	BankruptcyLoss        string           `json:"bkLoss"`
	BankruptcyPx          string           `json:"bkPx"`
	Currency              string           `json:"ccy"`
	PosSide               string           `json:"posSide"`
	Side                  string           `json:"side"`
	QuantityOfLiquidation float64          `json:"sz,string"`
	Timestamp             okxUnixMilliTime `json:"ts"`
}

// MarkPrice represents a mark price information for a single instrument id
type MarkPrice struct {
	InstrumentType string           `json:"instType"`
	InstrumentID   string           `json:"instId"`
	MarkPrice      string           `json:"markPx"`
	Timestamp      okxUnixMilliTime `json:"ts"`
}

// PositionTiers represents position tier detailed information.
type PositionTiers struct {
	BaseMaxLoan                  string  `json:"baseMaxLoan"`
	InitialMarginRequirement     string  `json:"imr"`
	InstrumentID                 string  `json:"instId"`
	MaximumLeverage              string  `json:"maxLever"`
	MaximumSize                  float64 `json:"maxSz,string"`
	MinSize                      float64 `json:"minSz,string"`
	MaintenanceMarginRequirement string  `json:"mmr"`
	OptionalMarginFactor         string  `json:"optMgnFactor"`
	QuoteMaxLoan                 string  `json:"quoteMaxLoan"`
	Tier                         string  `json:"tier"`
	Underlying                   string  `json:"uly"`
}

// InterestRateLoanQuotaBasic holds the basic Currency, loan,and interest rate information.
type InterestRateLoanQuotaBasic struct {
	Currency     string  `json:"ccy"`
	LoanQuota    string  `json:"quota"`
	InterestRate float64 `json:"rate,string"`
}

// InterestRateLoanQuotaItem holds the basic Currency, loan,interest rate, and other level and VIP related information.
type InterestRateLoanQuotaItem struct {
	InterestRateLoanQuotaBasic
	InterestRateDiscount float64 `json:"0.7,string"`
	LoanQuotaCoefficient float64 `json:"loanQuotaCoef,string"`
	Level                string  `json:"level"`
}

// VIPInterestRateAndLoanQuotaInformation holds interest rate and loan quoata information for VIP users.
type VIPInterestRateAndLoanQuotaInformation struct {
	InterestRateLoanQuotaBasic
	LevelList []struct {
		Level     string  `json:"level"`
		LoanQuota float64 `json:"loanQuota,string"`
	} `json:"levelList"`
}

// InsuranceFundInformationRequestParams insurance fund balance information.
type InsuranceFundInformationRequestParams struct {
	InstrumentType string    `json:"instType"`
	Type           string    `json:"type"` //  Type values allowed are `liquidation_balance_deposit, bankruptcy_loss, and platform_revenue`
	Underlying     string    `json:"uly"`
	Currency       string    `json:"ccy"`
	Before         time.Time `json:"before"`
	After          time.Time `json:"after"`
	Limit          int64     `json:"limit"`
}

// InsuranceFundInformation holds insurance fund information data.
type InsuranceFundInformation struct {
	Details []InsuranceFundInformationDetail `json:"details"`
	Total   float64                          `json:"total,string"`
}

// InsuranceFundInformationDetail represents an Insurance fund information item for a
// single currency and type
type InsuranceFundInformationDetail struct {
	Amount    float64          `json:"amt,string"`
	Balance   float64          `json:"balance,string"`
	Currency  string           `json:"ccy"`
	Timestamp okxUnixMilliTime `json:"ts"`
	Type      string           `json:"type"`
}

// SupportedCoinsData holds information about currencies supported by the trading data endpoints.
type SupportedCoinsData struct {
	Contract       []string `json:"contract"`
	TradingOptions []string `json:"option"`
	Spot           []string `json:"spot"`
}

// TakerVolume represents taker volume information with creation timestamp
type TakerVolume struct {
	Timestamp  time.Time `json:"ts"`
	SellVolume float64
	BuyVolume  float64
}

// MarginLendRatioItem represents margin lend ration information and creation timestamp
type MarginLendRatioItem struct {
	Timestamp       time.Time `json:"ts"`
	MarginLendRatio float64   `json:"ratio"`
}

// LongShortRatio represents the ratio of users with net long vs net short positions for futures and perpetual swaps.
type LongShortRatio struct {
	Timestamp       time.Time `json:"ts"`
	MarginLendRatio float64   `json:"ratio"`
}

// OpenInterestVolume represents open interest and trading volume item for currencies of futures and perpetual swaps.
type OpenInterestVolume struct {
	Timestamp    time.Time `json:"ts"`
	OpenInterest float64   `json:"oi"`
	Volume       float64   `json:"vol"`
}

// OpenInterestVolumeRatio represents open interest and trading volume ratio for currencies of futures and perpetual swaps.
type OpenInterestVolumeRatio struct {
	Timestamp         time.Time `json:"ts"`
	OpenInterestRatio float64   `json:"oiRatio"`
	VolumeRatio       float64   `json:"volRatio"`
}

// ExpiryOpenInterestAndVolume represents  open interest and trading volume of calls and puts for each upcoming expiration.
type ExpiryOpenInterestAndVolume struct {
	Timestamp        time.Time
	ExpiryTime       time.Time
	CallOpenInterest float64
	PutOpenInterest  float64
	CallVolume       float64
	PutVolume        float64
}

// StrikeOpenInterestAndVolume represents open interest and volume for both buyers and sellers of calls and puts.
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
	AssetType             asset.Item `json:"-"`
	InstrumentID          string     `json:"instId"`
	TradeMode             string     `json:"tdMode,omitempty"` // cash isolated
	ClientSupplierOrderID string     `json:"clOrdId,omitempty"`
	Currency              string     `json:"ccy,omitempty"` // Only applicable to cross MARGIN orders in Single-currency margin.
	OrderTag              string     `json:"tag,omitempty"`
	Side                  string     `json:"side,omitempty"`
	PositionSide          string     `json:"posSide,omitempty"`
	OrderType             string     `json:"ordType,omitempty"`
	Amount                float64    `json:"sz,string,omitempty"`
	Price                 float64    `json:"px,string,omitempty"`
	ReduceOnly            bool       `json:"reduceOnly,string,omitempty"`
	QuantityType          string     `json:"tgtCcy,omitempty"` // values base_ccy and quote_ccy
	// Added in the websocket requests
	BanAmend   bool             `json:"banAmend,omitempty"` // Whether the SPOT Market Order size can be amended by the system.
	ExpiryTime okxUnixMilliTime `json:"expTime,omitempty"`
}

// OrderData response message for place, cancel, and amend an order requests.
type OrderData struct {
	OrderID               string `json:"ordId,omitempty"`
	RequestID             string `json:"reqId,omitempty"`
	ClientSupplierOrderID string `json:"clOrdId,omitempty"`
	Tag                   string `json:"tag,omitempty"`
	SCode                 string `json:"sCode,omitempty"`
	SMessage              string `json:"sMsg,omitempty"`
}

// CancelOrderRequestParam represents order parameters to cancel an order.
type CancelOrderRequestParam struct {
	InstrumentID          string `json:"instId"`
	OrderID               string `json:"ordId"`
	ClientSupplierOrderID string `json:"clOrdId,omitempty"`
}

// AmendOrderRequestParams represents amend order requesting parameters.
type AmendOrderRequestParams struct {
	InstrumentID            string  `json:"instId"`
	CancelOnFail            bool    `json:"cxlOnFail,omitempty"`
	OrderID                 string  `json:"ordId,omitempty"`
	ClientSuppliedOrderID   string  `json:"clOrdId,omitempty"`
	ClientSuppliedRequestID string  `json:"reqId,omitempty"`
	NewQuantity             float64 `json:"newSz,string,omitempty"`
	NewPrice                float64 `json:"newPx,string,omitempty"`
}

// ClosePositionsRequestParams input parameters for close position endpoints
type ClosePositionsRequestParams struct {
	InstrumentID          string `json:"instId"` // REQUIRED
	PositionSide          string `json:"posSide"`
	MarginMode            string `json:"mgnMode"` // cross or isolated
	Currency              string `json:"ccy"`
	AutomaticallyCanceled bool   `json:"autoCxl"`
	ClientSuppliedID      string `json:"clOrdId,omitempty"`
	Tag                   string `json:"tag,omitempty"`
}

// ClosePositionResponse response data for close position.
type ClosePositionResponse struct {
	InstrumentID string `json:"instId"`
	PositionSide string `json:"posSide"`
}

// OrderDetailRequestParam payload data to request order detail
type OrderDetailRequestParam struct {
	InstrumentID          string `json:"instId"`
	OrderID               string `json:"ordId"`
	ClientSupplierOrderID string `json:"clOrdId"`
}

// OrderDetail returns a order detail information
type OrderDetail struct {
	InstrumentType             string            `json:"instType"`
	InstrumentID               string            `json:"instId"`
	Currency                   string            `json:"ccy"`
	OrderID                    string            `json:"ordId"`
	ClientSupplierOrderID      string            `json:"clOrdId"`
	Tag                        string            `json:"tag"`
	ProfitAndLoss              string            `json:"pnl"`
	OrderType                  string            `json:"ordType"`
	Side                       order.Side        `json:"side"`
	PositionSide               string            `json:"posSide"`
	TradeMode                  string            `json:"tdMode"`
	TradeID                    string            `json:"tradeId"`
	FillTime                   time.Time         `json:"fillTime"`
	Source                     string            `json:"source"`
	State                      string            `json:"state"`
	TakeProfitTriggerPriceType string            `json:"tpTriggerPxType"`
	StopLossTriggerPriceType   string            `json:"slTriggerPxType"`
	StopLossOrdPx              string            `json:"slOrdPx"`
	RebateCurrency             string            `json:"rebateCcy"`
	QuantityType               string            `json:"tgtCcy"`   // base_ccy and quote_ccy
	Category                   string            `json:"category"` // normal, twap, adl, full_liquidation, partial_liquidation, delivery, ddh
	AccumulatedFillSize        okxNumericalValue `json:"accFillSz"`
	FillPrice                  okxNumericalValue `json:"fillPx"`
	FillSize                   okxNumericalValue `json:"fillSz"`
	RebateAmount               okxNumericalValue `json:"rebate"`
	FeeCurrency                string            `json:"feeCcy"`
	TransactionFee             okxNumericalValue `json:"fee"`
	AveragePrice               okxNumericalValue `json:"avgPx"`
	Leverage                   okxNumericalValue `json:"lever"`
	Price                      okxNumericalValue `json:"px"`
	Size                       okxNumericalValue `json:"sz"`
	TakeProfitTriggerPrice     okxNumericalValue `json:"tpTriggerPx"`
	TakeProfitOrderPrice       okxNumericalValue `json:"tpOrdPx"`
	StopLossTriggerPrice       okxNumericalValue `json:"slTriggerPx"`
	UpdateTime                 time.Time         `json:"uTime"`
	CreationTime               time.Time         `json:"cTime"`
}

// OrderListRequestParams represents order list requesting parameters.
type OrderListRequestParams struct {
	InstrumentType string    `json:"instType"` // SPOT , MARGIN, SWAP, FUTURES , OPTIONS
	Underlying     string    `json:"uly"`
	InstrumentID   string    `json:"instId"`
	OrderType      string    `json:"orderType"`
	State          string    `json:"state"`            // live, partially_filled
	Before         string    `json:"before,omitempty"` // used for order IDs
	After          string    `json:"after,omitempty"`  // used for order IDs
	Start          time.Time `json:"begin"`
	End            time.Time `json:"end"`
	Limit          int64     `json:"limit,omitempty"`
}

// OrderHistoryRequestParams holds parameters to request order data history of last 7 days.
type OrderHistoryRequestParams struct {
	OrderListRequestParams
	Category string `json:"category"` // twap, adl, full_liquidation, partial_liquidation, delivery, ddh
}

// PendingOrderItem represents a pending order Item in pending orders list.
type PendingOrderItem struct {
	AccumulatedFillSize        okxNumericalValue `json:"accFillSz"`
	AveragePrice               okxNumericalValue `json:"avgPx"`
	CreationTime               time.Time         `json:"cTime"`
	Category                   string            `json:"category"`
	Currency                   string            `json:"ccy"`
	ClientSupplierOrderID      string            `json:"clOrdId"`
	TransactionFee             string            `json:"fee"`
	FeeCurrency                string            `json:"feeCcy"`
	LastFilledPrice            string            `json:"fillPx"`
	LastFilledSize             okxNumericalValue `json:"fillSz"`
	FillTime                   string            `json:"fillTime"`
	InstrumentID               string            `json:"instId"`
	InstrumentType             string            `json:"instType"`
	Leverage                   okxNumericalValue `json:"lever"`
	OrderID                    string            `json:"ordId"`
	OrderType                  string            `json:"ordType"`
	ProfitAndLose              string            `json:"pnl"`
	PositionSide               string            `json:"posSide"`
	RebateAmount               string            `json:"rebate"`
	RebateCurrency             string            `json:"rebateCcy"`
	Side                       order.Side        `json:"side"`
	StopLossOrdPrice           string            `json:"slOrdPx"`
	StopLossTriggerPrice       string            `json:"slTriggerPx"`
	StopLossTriggerPriceType   string            `json:"slTriggerPxType"`
	State                      string            `json:"state"`
	Price                      float64           `json:"px,string"`
	Size                       float64           `json:"sz,string"`
	Tag                        string            `json:"tag"`
	QuantityType               string            `json:"tgtCcy"`
	TradeMode                  string            `json:"tdMode"`
	Source                     string            `json:"source"`
	TakeProfitOrdPrice         string            `json:"tpOrdPx"`
	TakeProfitTriggerPrice     string            `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string            `json:"tpTriggerPxType"`
	TradeID                    string            `json:"tradeId"`
	UpdateTime                 time.Time         `json:"uTime"`
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
	Limit          int64     `json:"limit"`
}

// TransactionDetail holds ecently-filled transaction detail data.
type TransactionDetail struct {
	InstrumentType        string           `json:"instType"`
	InstrumentID          string           `json:"instId"`
	TradeID               string           `json:"tradeId"`
	OrderID               string           `json:"ordId"`
	ClientSuppliedOrderID string           `json:"clOrdId"`
	BillID                string           `json:"billId"`
	Tag                   string           `json:"tag"`
	FillPrice             float64          `json:"fillPx,string"`
	FillSize              float64          `json:"fillSz,string"`
	Side                  string           `json:"side"`
	PositionSide          string           `json:"posSide"`
	ExecType              string           `json:"execType"`
	FeeCurrency           string           `json:"feeCcy"`
	Fee                   string           `json:"fee"`
	Timestamp             okxUnixMilliTime `json:"ts"`
}

// AlgoOrderParams holds algo order information.
type AlgoOrderParams struct {
	InstrumentID string     `json:"instId"` // Required
	TradeMode    string     `json:"tdMode"` // Required
	Currency     string     `json:"ccy,omitempty"`
	Side         order.Side `json:"side"` // Required
	PositionSide string     `json:"posSide,omitempty"`
	OrderType    string     `json:"ordType"`   // Required
	Size         float64    `json:"sz,string"` // Required
	ReduceOnly   bool       `json:"reduceOnly,omitempty"`
	OrderTag     string     `json:"tag,omitempty"`
	QuantityType string     `json:"tgtCcy,omitempty"`

	// Place Stop Order params
	TakeProfitTriggerPrice     float64 `json:"tpTriggerPx,string,omitempty"`
	TakeProfitOrderPrice       float64 `json:"tpOrdPx,string,omitempty"`
	StopLossTriggerPrice       float64 `json:"slTriggerPx,string,omitempty"`
	StopLossOrderPrice         float64 `json:"slOrdPx,string,omitempty"`
	StopLossTriggerPriceType   string  `json:"slTriggerPxType,omitempty"`
	TakeProfitTriggerPriceType string  `json:"tpTriggerPxType,omitempty"`

	// Trigger Price  Or TrailingStopOrderRequestParam
	CallbackRatio          float64 `json:"callbackRatio,omitempty,string"`
	ActivePrice            float64 `json:"activePx,string,omitempty"`
	CallbackSpreadVariance string  `json:"callbackSpread,omitempty"`

	// trigger algo orders params.
	// notice: Trigger orders are not available in the net mode of futures and perpetual swaps
	TriggerPrice     float64 `json:"triggerPx,string,omitempty"`
	OrderPrice       float64 `json:"orderPx,string,omitempty"` // if the price i -1, then the order will be executed on the market.
	TriggerPriceType string  `json:"triggerPxType,omitempty"`  // last, index, and mark

	PriceVariance string  `json:"pxVar,omitempty"`          // Optional
	PriceSpread   string  `json:"pxSpread,omitempty"`       // Optional
	SizeLimit     float64 `json:"szLimit,string,omitempty"` // Required
	PriceLimit    float64 `json:"pxLimit,string,omitempty"` // Required

	// TWAPOrder
	TimeInterval kline.Interval `json:"interval,omitempty"` // Required
}

// StopOrderParams holds stop order request payload.
type StopOrderParams struct {
	AlgoOrderParams
	TakeProfitTriggerPrice     string `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string `json:"tpTriggerPxType"`
	TakeProfitOrderType        string `json:"tpOrdPx"`
	StopLossTriggerPrice       string `json:"slTriggerPx"`
	StopLossTriggerPriceType   string `json:"slTriggerPxType"`
	StopLossOrderPrice         string `json:"slOrdPx"`
}

// AlgoOrder algo order requests response.
type AlgoOrder struct {
	AlgoID     string `json:"algoId"`
	StatusCode string `json:"sCode"`
	StatusMsg  string `json:"sMsg"`
}

// AlgoOrderCancelParams algo order request parameter
type AlgoOrderCancelParams struct {
	AlgoOrderID  string `json:"algoId"`
	InstrumentID string `json:"instId"`
}

// AlgoOrderResponse holds algo order information.
type AlgoOrderResponse struct {
	InstrumentType             string           `json:"instType"`
	InstrumentID               string           `json:"instId"`
	OrderID                    string           `json:"ordId"`
	Currency                   string           `json:"ccy"`
	AlgoOrderID                string           `json:"algoId"`
	Quantity                   string           `json:"sz"`
	OrderType                  string           `json:"ordType"`
	Side                       string           `json:"side"`
	PositionSide               string           `json:"posSide"`
	TradeMode                  string           `json:"tdMode"`
	QuantityType               string           `json:"tgtCcy"`
	State                      string           `json:"state"`
	Lever                      string           `json:"lever"`
	TakeProfitTriggerPrice     string           `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string           `json:"tpTriggerPxType"`
	TakeProfitOrdPrice         string           `json:"tpOrdPx"`
	StopLossTriggerPriceType   string           `json:"slTriggerPxType"`
	StopLossTriggerPrice       string           `json:"slTriggerPx"`
	TriggerPrice               string           `json:"triggerPx"`
	TriggerPriceType           string           `json:"triggerPxType"`
	OrdPrice                   string           `json:"ordPx"`
	ActualSize                 string           `json:"actualSz"`
	ActualPrice                string           `json:"actualPx"`
	ActualSide                 string           `json:"actualSide"`
	PriceVar                   string           `json:"pxVar"`
	PriceSpread                string           `json:"pxSpread"`
	PriceLimit                 string           `json:"pxLimit"`
	SizeLimit                  string           `json:"szLimit"`
	TimeInterval               string           `json:"timeInterval"`
	TriggerTime                okxUnixMilliTime `json:"triggerTime"`
	CallbackRatio              string           `json:"callbackRatio"`
	CallbackSpread             string           `json:"callbackSpread"`
	ActivePrice                string           `json:"activePx"`
	MoveTriggerPrice           string           `json:"moveTriggerPx"`
	CreationTime               okxUnixMilliTime `json:"cTime"`
}

// CurrencyResponse represents a currency item detail response data.
type CurrencyResponse struct {
	CanDeposit          bool    `json:"canDep"`        // Availability to deposit from chain. false: not available true: available
	CanInternalTransfer bool    `json:"canInternal"`   // Availability to internal transfer.
	CanWithdraw         bool    `json:"canWd"`         // Availability to withdraw to chain.
	Currency            string  `json:"ccy"`           //
	Chain               string  `json:"chain"`         //
	LogoLink            string  `json:"logoLink"`      // Logo link of currency
	MainNet             bool    `json:"mainNet"`       // If current chain is main net then return true, otherwise return false
	MaxFee              float64 `json:"maxFee,string"` // Minimum withdrawal fee
	MaxWithdrawal       float64 `json:"maxWd,string"`  // Minimum amount of currency withdrawal in a single transaction
	MinFee              float64 `json:"minFee,string"` // Minimum withdrawal fee
	MinWithdrawal       string  `json:"minWd"`         // Minimum amount of currency withdrawal in a single transaction
	Name                string  `json:"name"`          // Chinese name of currency
	UsedWithdrawalQuota string  `json:"usedWdQuota"`   // Amount of currency withdrawal used in the past 24 hours, unit in BTC
	WithdrawalQuota     string  `json:"wdQuota"`       // Minimum amount of currency withdrawal in a single transaction
	WithdrawalTickSize  string  `json:"wdTickSz"`      // Withdrawal precision, indicating the number of digits after the decimal point
}

// AssetBalance represents account owner asset balance
type AssetBalance struct {
	AvailBal      float64 `json:"availBal,string"`
	Balance       float64 `json:"bal,string"`
	Currency      string  `json:"ccy"`
	FrozenBalance float64 `json:"frozenBal,string"`
}

// AccountAssetValuation represents view account asset valuation data
type AccountAssetValuation struct {
	Details struct {
		Classic float64 `json:"classic,string"`
		Earn    float64 `json:"earn,string"`
		Funding float64 `json:"funding,string"`
		Trading float64 `json:"trading,string"`
	} `json:"details"`
	TotalBalance float64          `json:"totalBal,string"`
	Timestamp    okxUnixMilliTime `json:"ts"`
}

// FundingTransferRequestInput represents funding account request input.
type FundingTransferRequestInput struct {
	Currency     string  `json:"ccy"`
	Type         int     `json:"type,string"`
	Amount       float64 `json:"amt,string"`
	From         string  `json:"from"` // "6": Funding account, "18": Trading account
	To           string  `json:"to"`
	SubAccount   string  `json:"subAcct"`
	LoanTransfer bool    `json:"loanTrans,string"`
	ClientID     string  `json:"clientId"` // Client-supplied ID A combination of case-sensitive alphanumerics, all numbers, or all letters of up to 32 characters.
}

// FundingTransferResponse represents funding transfer and trading account transfer response.
type FundingTransferResponse struct {
	TransferID string  `json:"transId"`
	Currency   string  `json:"ccy"`
	ClientID   string  `json:"clientId"`
	From       int64   `json:"from,string"`
	Amount     float64 `json:"amt,string"`
	To         int64   `json:"to,string"`
}

// TransferFundRateResponse represents funcing transfer rate response
type TransferFundRateResponse struct {
	Amount         float64 `json:"amt,string"`
	Currency       string  `json:"ccy"`
	ClientID       string  `json:"clientId"`
	From           string  `json:"from"`
	InstrumentID   string  `json:"instId"`
	State          string  `json:"state"`
	SubAccount     string  `json:"subAcct"`
	To             string  `json:"to"`
	ToInstrumentID string  `json:"toInstId"`
	TransferID     string  `json:"transId"`
	Type           int     `json:"type,string"`
}

// AssetBillDetail represents  the billing record
type AssetBillDetail struct {
	BillID         string           `json:"billId"`
	Currency       string           `json:"ccy"`
	ClientID       string           `json:"clientId"`
	BalanceChange  string           `json:"balChg"`
	AccountBalance string           `json:"bal"`
	Type           int              `json:"type,string"`
	Timestamp      okxUnixMilliTime `json:"ts"`
}

// LightningDepositItem for creating an invoice.
type LightningDepositItem struct {
	CreationTime okxUnixMilliTime `json:"cTime"`
	Invoice      string           `json:"invoice"`
}

// CurrencyDepositResponseItem represents the deposit address information item.
type CurrencyDepositResponseItem struct {
	Tag                      string            `json:"tag"`
	Chain                    string            `json:"chain"`
	ContractAddress          string            `json:"ctAddr"`
	Currency                 string            `json:"ccy"`
	ToBeneficiaryAccount     string            `json:"to"`
	Address                  string            `json:"addr"`
	Selected                 bool              `json:"selected"`
	Memo                     string            `json:"memo"`
	DepositAddressAttachment map[string]string `json:"addrEx"`
	PaymentID                string            `json:"pmtId"`
}

// DepositHistoryResponseItem deposit history response item.
type DepositHistoryResponseItem struct {
	Amount           float64          `json:"amt,string"`
	TransactionID    string           `json:"txId"` // Hash record of the deposit
	Currency         string           `json:"ccy"`
	Chain            string           `json:"chain"`
	From             string           `json:"from"`
	ToDepositAddress string           `json:"to"`
	Timestamp        okxUnixMilliTime `json:"ts"`
	State            int              `json:"state,string"`
	DepositID        string           `json:"depId"`
}

// WithdrawalInput represents request parameters for cryptocurrency withdrawal
type WithdrawalInput struct {
	Amount                float64 `json:"amt,string"`
	TransactionFee        float64 `json:"fee,string"`
	WithdrawalDestination string  `json:"dest"`
	Currency              string  `json:"ccy"`
	ChainName             string  `json:"chain"`
	ToAddress             string  `json:"toAddr"`
	ClientSuppliedID      string  `json:"clientId"`
}

// WithdrawalResponse cryptocurrency withdrawal response
type WithdrawalResponse struct {
	Amount       float64 `json:"amt,string"`
	WithdrawalID string  `json:"wdId"`
	Currency     string  `json:"ccy"`
	ClientID     string  `json:"clientId"`
	Chain        string  `json:"chain"`
}

// LightningWithdrawalRequestInput to request Lightning Withdrawal requests.
type LightningWithdrawalRequestInput struct {
	Currency string `json:"ccy"`     // REQUIRED Token symbol. Currently only BTC is supported.
	Invoice  string `json:"invoice"` // REQUIRED Invoice text
	Memo     string `json:"memo"`    // Lightning withdrawal memo
}

// LightningWithdrawalResponse response item for holding lightning withdrawal requests.
type LightningWithdrawalResponse struct {
	WithdrawalID string           `json:"wdId"`
	CreationTime okxUnixMilliTime `json:"cTime"`
}

// WithdrawalHistoryResponse represents the withdrawal response history.
type WithdrawalHistoryResponse struct {
	ChainName            string           `json:"chain"`
	WithdrawalFee        float64          `json:"fee,string"`
	Currency             string           `json:"ccy"`
	ClientID             string           `json:"clientId"`
	Amount               float64          `json:"amt,string"`
	TransactionID        string           `json:"txId"` // Hash record of the withdrawal. This parameter will not be returned for internal transfers.
	FromRemittingAddress string           `json:"from"`
	ToReceivingAddress   string           `json:"to"`
	StateOfWithdrawal    string           `json:"state"`
	Timestamp            okxUnixMilliTime `json:"ts"`
	WithdrawalID         string           `json:"wdId"`
	PaymentID            string           `json:"pmtId,omitempty"`
	Memo                 string           `json:"memo"`
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

// SavingsPurchaseRedemptionInput input json to SavingPurchase Post merthod.
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

// LendingRate represents lending rate response
type LendingRate struct {
	Currency string  `json:"ccy"`
	Rate     float64 `json:"rate,string"`
}

// LendingHistory holds lending history responses
type LendingHistory struct {
	Currency  string           `json:"ccy"`
	Amount    float64          `json:"amt,string"`
	Earnings  float64          `json:"earnings,string,omitempty"`
	Rate      float64          `json:"rate,string"`
	Timestamp okxUnixMilliTime `json:"ts"`
}

// PublicBorrowInfo holds a currency's borrow info.
type PublicBorrowInfo struct {
	Currency         string            `json:"ccy"`
	AverageAmount    okxNumericalValue `json:"avgAmt"`
	AverageAmountUSD okxNumericalValue `json:"avgAmtUsd"`
	AverageRate      okxNumericalValue `json:"avgRate"`
	PreviousRate     okxNumericalValue `json:"preRate"`
	EstimatedRate    okxNumericalValue `json:"estRate"`
}

// PublicBorrowHistory holds a currencies borrow history.
type PublicBorrowHistory struct {
	Amount    float64          `json:"amt,string"`
	Currency  string           `json:"ccy"`
	Rate      float64          `json:"rate,string"`
	Timestamp okxUnixMilliTime `json:"ts"`
}

// ConvertCurrency represents currency conversion detailed data.
type ConvertCurrency struct {
	Currency string            `json:"currency"`
	Min      okxNumericalValue `json:"min"`
	Max      okxNumericalValue `json:"max"`
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

// EstimateQuoteRequestInput represents estimate quote request parameters
type EstimateQuoteRequestInput struct {
	BaseCurrency         string  `json:"baseCcy,omitempty"`
	QuoteCurrency        string  `json:"quoteCcy,omitempty"`
	Side                 string  `json:"side,omitempty"`
	RFQAmount            float64 `json:"rfqSz,omitempty"`
	RFQSzCurrency        string  `json:"rfqSzCcy,omitempty"`
	ClientRequestOrderID string  `json:"clQReqId,string,omitempty"`
	Tag                  string  `json:"tag,omitempty"`
}

// EstimateQuoteResponse represents estimate quote response data.
type EstimateQuoteResponse struct {
	BaseCurrency            string           `json:"baseCcy"`
	BaseSize                string           `json:"baseSz"`
	ClientSupplierRequestID string           `json:"clQReqId"`
	ConvertPrice            string           `json:"cnvtPx"`
	OrigRfqSize             string           `json:"origRfqSz"`
	QuoteCurrency           string           `json:"quoteCcy"`
	QuoteID                 string           `json:"quoteId"`
	QuoteSize               string           `json:"quoteSz"`
	QuoteTime               okxUnixMilliTime `json:"quoteTime"`
	RFQSize                 string           `json:"rfqSz"`
	RFQSizeCurrency         string           `json:"rfqSzCcy"`
	Side                    string           `json:"side"`
	TTLMs                   string           `json:"ttlMs"` // Validity period of quotation in milliseconds
}

// ConvertTradeInput represents convert trade request input
type ConvertTradeInput struct {
	BaseCurrency          string  `json:"baseCcy"`
	QuoteCurrency         string  `json:"quoteCcy"`
	Side                  string  `json:"side"`
	Size                  float64 `json:"sz,string"`
	SizeCurrency          string  `json:"szCcy"`
	QuoteID               string  `json:"quoteId"`
	ClientSupplierOrderID string  `json:"clTReqId"`
	Tag                   string  `json:"tag"`
}

// ConvertTradeResponse represents convert trade response
type ConvertTradeResponse struct {
	BaseCurrency          string           `json:"baseCcy"`
	ClientSupplierOrderID string           `json:"clTReqId"`
	FillBaseSize          float64          `json:"fillBaseSz,string"`
	FillPrice             string           `json:"fillPx"`
	FillQuoteSize         float64          `json:"fillQuoteSz,string"`
	InstrumentID          string           `json:"instId"`
	QuoteCurrency         string           `json:"quoteCcy"`
	QuoteID               string           `json:"quoteId"`
	Side                  string           `json:"side"`
	State                 string           `json:"state"`
	TradeID               string           `json:"tradeId"`
	Timestamp             okxUnixMilliTime `json:"ts"`
}

// ConvertHistory holds convert trade history response
type ConvertHistory struct {
	InstrumentID  string           `json:"instId"`
	Side          string           `json:"side"`
	FillPrice     float64          `json:"fillPx,string"`
	BaseCurrency  string           `json:"baseCcy"`
	QuoteCurrency string           `json:"quoteCcy"`
	FillBaseSize  float64          `json:"fillBaseSz,string"`
	State         string           `json:"state"`
	TradeID       string           `json:"tradeId"`
	FillQuoteSize float64          `json:"fillQuoteSz,string"`
	Timestamp     okxUnixMilliTime `json:"ts"`
}

// Account holds currency account balance and related information
type Account struct {
	AdjEq       okxNumericalValue `json:"adjEq"`
	Details     []AccountDetail   `json:"details"`
	Imr         okxNumericalValue `json:"imr"` // Frozen equity for open positions and pending orders in USD level Applicable to Multi-currency margin and Portfolio margin
	IsoEq       okxNumericalValue `json:"isoEq"`
	MgnRatio    okxNumericalValue `json:"mgnRatio"`
	Mmr         okxNumericalValue `json:"mmr"` // Maintenance margin requirement in USD level Applicable to Multi-currency margin and Portfolio margin
	NotionalUsd okxNumericalValue `json:"notionalUsd"`
	OrdFroz     okxNumericalValue `json:"ordFroz"` // Margin frozen for pending orders in USD level Applicable to Multi-currency margin and Portfolio margin
	TotalEquity okxNumericalValue `json:"totalEq"` // Total Equity in USD level
	UpdateTime  okxUnixMilliTime  `json:"uTime"`   // UpdateTime
}

// AccountDetail account detail information.
type AccountDetail struct {
	AvailableBalance              okxNumericalValue `json:"availBal"`
	AvailableEquity               okxNumericalValue `json:"availEq"`
	CashBalance                   okxNumericalValue `json:"cashBal"` // Cash Balance
	Currency                      string            `json:"ccy"`
	CrossLiab                     okxNumericalValue `json:"crossLiab"`
	DiscountEquity                okxNumericalValue `json:"disEq"`
	EquityOfCurrency              okxNumericalValue `json:"eq"`
	EquityUsd                     okxNumericalValue `json:"eqUsd"`
	FrozenBalance                 okxNumericalValue `json:"frozenBal"`
	Interest                      okxNumericalValue `json:"interest"`
	IsoEquity                     okxNumericalValue `json:"isoEq"`
	IsolatedLiabilities           okxNumericalValue `json:"isoLiab"`
	IsoUpl                        okxNumericalValue `json:"isoUpl"` // Isolated unrealized profit and loss of the currency applicable to Single-currency margin and Multi-currency margin and Portfolio margin
	LiabilitiesOfCurrency         okxNumericalValue `json:"liab"`
	MaxLoan                       okxNumericalValue `json:"maxLoan"`
	MarginRatio                   okxNumericalValue `json:"mgnRatio"`      // Equity of the currency
	NotionalLever                 okxNumericalValue `json:"notionalLever"` // Leverage of the currency applicable to Single-currency margin
	OpenOrdersMarginFrozen        okxNumericalValue `json:"ordFrozen"`
	Twap                          okxNumericalValue `json:"twap"`
	UpdateTime                    okxUnixMilliTime  `json:"uTime"`
	UnrealizedProfit              okxNumericalValue `json:"upl"`
	UnrealizedCurrencyLiabilities okxNumericalValue `json:"uplLiab"`
	StrategyEquity                okxNumericalValue `json:"stgyEq"`  // strategy equity
	TotalEquity                   okxNumericalValue `json:"totalEq"` // Total equity in USD level
}

// AccountPosition account position.
type AccountPosition struct {
	AutoDeleveraging             string           `json:"adl"`      // Auto-deleveraging (ADL) indicator Divided into 5 levels, from 1 to 5, the smaller the number, the weaker the adl intensity.
	AvailablePosition            string           `json:"availPos"` // Position that can be closed Only applicable to MARGIN, FUTURES/SWAP in the long-short mode, OPTION in Simple and isolated OPTION in margin Account.
	AveragePrice                 string           `json:"avgPx"`
	CreationTime                 okxUnixMilliTime `json:"cTime"`
	Currency                     string           `json:"ccy"`
	DeltaBS                      string           `json:"deltaBS"` // delta：Black-Scholes Greeks in dollars,only applicable to OPTION
	DeltaPA                      string           `json:"deltaPA"` // delta：Greeks in coins,only applicable to OPTION
	GammaBS                      string           `json:"gammaBS"` // gamma：Black-Scholes Greeks in dollars,only applicable to OPTION
	GammaPA                      string           `json:"gammaPA"` // gamma：Greeks in coins,only applicable to OPTION
	InitialMarginRequirement     string           `json:"imr"`     // Initial margin requirement, only applicable to cross.
	InstrumentID                 string           `json:"instId"`
	InstrumentType               string           `json:"instType"`
	Interest                     string           `json:"interest"`
	USDPrice                     string           `json:"usdPx"`
	LastTradePrice               string           `json:"last"`
	Leverage                     string           `json:"lever"`   // Leverage, not applicable to OPTION seller
	Liabilities                  string           `json:"liab"`    // Liabilities, only applicable to MARGIN.
	LiabilitiesCurrency          string           `json:"liabCcy"` // Liabilities currency, only applicable to MARGIN.
	LiquidationPrice             string           `json:"liqPx"`   // Estimated liquidation price Not applicable to OPTION
	MarkPx                       string           `json:"markPx"`
	Margin                       string           `json:"margin"`
	MgnMode                      string           `json:"mgnMode"`
	MgnRatio                     string           `json:"mgnRatio"`
	MaintenanceMarginRequirement string           `json:"mmr"`         // Maintenance margin requirement in USD level Applicable to Multi-currency margin and Portfolio margin
	NotionalUsd                  string           `json:"notionalUsd"` // Quality of Positions -- usd
	OptionValue                  string           `json:"optVal"`      // Option Value, only application to position.
	QuantityOfPosition           string           `json:"pos"`         // Quantity of positions,In the mode of autonomous transfer from position to position, after the deposit is transferred, a position with pos of 0 will be generated
	PositionCurrency             string           `json:"posCcy"`
	PositionID                   string           `json:"posId"`
	PositionSide                 string           `json:"posSide"`
	ThetaBS                      string           `json:"thetaBS"` // theta：Black-Scholes Greeks in dollars,only applicable to OPTION
	ThetaPA                      string           `json:"thetaPA"` // theta：Greeks in coins,only applicable to OPTION
	TradeID                      string           `json:"tradeId"`
	UpdatedTime                  okxUnixMilliTime `json:"uTime"`                     // Latest time position was adjusted,
	Upl                          float64          `json:"upl,string,omitempty"`      // Unrealized profit and loss
	UPLRatio                     float64          `json:"uplRatio,string,omitempty"` // Unrealized profit and loss ratio
	VegaBS                       string           `json:"vegaBS"`                    // vega：Black-Scholes Greeks in dollars,only applicable to OPTION
	VegaPA                       string           `json:"vegaPA"`                    // vega：Greeks in coins,only applicable to OPTION

	// PushTime added feature in the websocket push data.

	PushTime okxUnixMilliTime `json:"pTime"` // The time when the account position data is pushed.
}

// AccountPositionHistory hold account position history.
type AccountPositionHistory struct {
	CreationTime       okxUnixMilliTime `json:"cTime"`
	Currency           string           `json:"ccy"`
	CloseAveragePrice  float64          `json:"closeAvgPx,string,omitempty"`
	CloseTotalPosition float64          `json:"closeTotalPos,string,omitempty"`
	InstrumentID       string           `json:"instId"`
	InstrumentType     string           `json:"instType"`
	Leverage           string           `json:"lever"`
	ManagementMode     string           `json:"mgnMode"`
	OpenAveragePrice   string           `json:"openAvgPx"`
	OpenMaxPosition    string           `json:"openMaxPos"`
	ProfitAndLoss      float64          `json:"pnl,string,omitempty"`
	ProfitAndLossRatio float64          `json:"pnlRatio,string,omitempty"`
	PositionID         string           `json:"posId"`
	PositionSide       string           `json:"posSide"`
	TriggerPrice       string           `json:"triggerPx"`
	Type               string           `json:"type"`
	UpdateTime         okxUnixMilliTime `json:"uTime"`
	Underlying         string           `json:"uly"`
}

// AccountBalanceData represents currency account balance.
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
	ManagementMode   string `json:"mgnMode"`
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
	AdjEq              string               `json:"adjEq"`
	AccountBalanceData []AccountBalanceData `json:"balData"`
	PosData            []PositionData       `json:"posData"`
	Timestamp          okxUnixMilliTime     `json:"ts"`
}

// BillsDetailQueryParameter represents bills detail query parameter
type BillsDetailQueryParameter struct {
	InstrumentType string // Instrument type "SPOT" "MARGIN" "SWAP" "FUTURES" "OPTION"
	Currency       string
	MarginMode     string // Margin mode "isolated" "cross"
	ContractType   string // Contract type "linear" & "inverse" Only applicable to FUTURES/SWAP
	BillType       uint   // Bill type 1: Transfer 2: Trade 3: Delivery 4: Auto token conversion 5: Liquidation 6: Margin transfer 7: Interest deduction 8: Funding fee 9: ADL 10: Clawback 11: System token conversion 12: Strategy transfer 13: ddh
	BillSubType    int    // allowed bill subtype values are [ 1,2,3,4,5,6,9,11,12,14,160,161,162,110,111,118,119,100,101,102,103,104,105,106,110,125,126,127,128,131,132,170,171,172,112,113,117,173,174,200,201,202,203 ], link: https://www.okx.com/docs-v5/en/#rest-api-account-get-bills-details-last-7-days
	After          string
	Before         string
	BeginTime      time.Time
	EndTime        time.Time
	Limit          int64
}

// BillsDetailResponse represents account bills information.
type BillsDetailResponse struct {
	Balance                    okxNumericalValue       `json:"bal"`
	BalanceChange              string                  `json:"balChg"`
	BillID                     string                  `json:"billId"`
	Currency                   string                  `json:"ccy"`
	ExecType                   string                  `json:"execType"` // Order flow type, T：taker M：maker
	Fee                        convert.StringToFloat64 `json:"fee"`      // Fee Negative number represents the user transaction fee charged by the platform. Positive number represents rebate.
	From                       string                  `json:"from"`     // The remitting account 6: FUNDING 18: Trading account When bill type is not transfer, the field returns "".
	InstrumentID               string                  `json:"instId"`
	InstrumentType             asset.Item              `json:"instType"`
	MarginMode                 string                  `json:"mgnMode"`
	Notes                      string                  `json:"notes"` // notes When bill type is not transfer, the field returns "".
	OrderID                    string                  `json:"ordId"`
	ProfitAndLoss              convert.StringToFloat64 `json:"pnl"`
	PositionLevelBalance       convert.StringToFloat64 `json:"posBal"`
	PositionLevelBalanceChange convert.StringToFloat64 `json:"posBalChg"`
	SubType                    string                  `json:"subType"`
	Size                       convert.StringToFloat64 `json:"sz"`
	To                         string                  `json:"to"`
	Timestamp                  okxUnixMilliTime        `json:"ts"`
	Type                       string                  `json:"type"`
}

// AccountConfigurationResponse represents account configuration response.
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

// PositionMode represents position mode response
type PositionMode struct {
	PositionMode string `json:"posMode"` // "long_short_mode": long/short, only applicable to FUTURES/SWAP "net_mode": net
}

// SetLeverageInput represents set leverage request input
type SetLeverageInput struct {
	Leverage     int    `json:"lever,string"`     // set leverage for isolated
	MarginMode   string `json:"mgnMode"`          // Margin Mode "cross" and "isolated"
	InstrumentID string `json:"instId,omitempty"` // Optional:
	Currency     string `json:"ccy,omitempty"`    // Optional:
	PositionSide string `json:"posSide,omitempty"`
}

// SetLeverageResponse represents set leverage response
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
	InstrumentID string `json:"instId"`
	AvailBuy     string `json:"availBuy"`
	AvailSell    string `json:"availSell"`
}

// IncreaseDecreaseMarginInput represents increase or decrease the margin of the isolated position.
type IncreaseDecreaseMarginInput struct {
	InstrumentID     string  `json:"instId"`
	PositionSide     string  `json:"posSide"`
	Type             string  `json:"type"`
	Amount           float64 `json:"amt,string"`
	Currency         string  `json:"ccy"`
	AutoLoadTransfer bool    `json:"auto"`
	LoadTransfer     bool    `json:"loanTrans"`
}

// IncreaseDecreaseMargin represents increase or decrease the margin of the isolated position response
type IncreaseDecreaseMargin struct {
	Amt          string `json:"amt"`
	Ccy          string `json:"ccy"`
	InstrumentID string `json:"instId"`
	Leverage     string `json:"leverage"`
	PosSide      string `json:"posSide"`
	Type         string `json:"type"`
}

// LeverageResponse instrument id leverage response.
type LeverageResponse struct {
	InstrumentID string `json:"instId"`
	MarginMode   string `json:"mgnMode"`
	PositionSide string `json:"posSide"`
	Leverage     uint   `json:"lever,string"`
}

// MaximumLoanInstrument represents maximum loan of an instrument id.
type MaximumLoanInstrument struct {
	InstrumentID string `json:"instId"`
	MgnMode      string `json:"mgnMode"`
	MgnCcy       string `json:"mgnCcy"`
	MaxLoan      string `json:"maxLoan"`
	Ccy          string `json:"ccy"`
	Side         string `json:"side"`
}

// TradeFeeRate holds trade fee rate information for a given instrument type.
type TradeFeeRate struct {
	Category         string                  `json:"category"`
	DeliveryFeeRate  string                  `json:"delivery"`
	Exercise         string                  `json:"exercise"`
	InstrumentType   asset.Item              `json:"instType"`
	FeeRateLevel     string                  `json:"level"`
	FeeRateMaker     convert.StringToFloat64 `json:"maker"`
	FeeRateMakerUSDT convert.StringToFloat64 `json:"makerU"`
	FeeRateMakerUSDC convert.StringToFloat64 `json:"makerUSDC"`
	FeeRateTaker     convert.StringToFloat64 `json:"taker"`
	FeeRateTakerUSDT convert.StringToFloat64 `json:"takerU"`
	FeeRateTakerUSDC convert.StringToFloat64 `json:"takerUSDC"`
	Timestamp        okxUnixMilliTime        `json:"ts"`
}

// InterestAccruedData represents interest rate accrued response
type InterestAccruedData struct {
	Currency     string           `json:"ccy"`
	InstrumentID string           `json:"instId"`
	Interest     string           `json:"interest"`
	InterestRate string           `json:"interestRate"` // Interest rate in an hour.
	Liability    string           `json:"liab"`
	MarginMode   string           `json:"mgnMode"` //  	Margin mode "cross" "isolated"
	Timestamp    okxUnixMilliTime `json:"ts"`
	LoanType     string           `json:"type"`
}

// InterestRateResponse represents interest rate response.
type InterestRateResponse struct {
	InterestRate float64 `json:"interestRate,string"`
	Currency     string  `json:"ccy"`
}

// GreeksType represents for greeks type response
type GreeksType struct {
	GreeksType string `json:"greeksType"` // Display type of Greeks. PA: Greeks in coins BS: Black-Scholes Greeks in dollars
}

// IsolatedMode represents Isolated margin trading settings
type IsolatedMode struct {
	IsoMode        string `json:"isoMode"` // "automatic":Auto transfers "autonomy":Manual transfers
	InstrumentType string `json:"type"`    // Instrument type "MARGIN" "CONTRACTS"
}

// MaximumWithdrawal represents maximum withdrawal amount query response.
type MaximumWithdrawal struct {
	Currency            string `json:"ccy"`
	MaximumWithdrawal   string `json:"maxWd"`   // Max withdrawal (not allowing borrowed crypto transfer out under Multi-currency margin)
	MaximumWithdrawalEx string `json:"maxWdEx"` // Max withdrawal (allowing borrowed crypto transfer out under Multi-currency margin)
}

// AccountRiskState represents account risk state.
type AccountRiskState struct {
	IsTheAccountAtRisk bool             `json:"atRisk"`
	AtRiskIdx          []interface{}    `json:"atRiskIdx"` // derivatives risk unit list
	AtRiskMgn          []interface{}    `json:"atRiskMgn"` // margin risk unit list
	Timestamp          okxUnixMilliTime `json:"ts"`
}

// LoanBorrowAndReplayInput represents currency VIP borrow or repay request params.
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

// BorrowRepayHistory represents borrow and repay history item data
type BorrowRepayHistory struct {
	Currency   string           `json:"ccy"`
	TradedLoan string           `json:"tradedLoan"`
	Timestamp  okxUnixMilliTime `json:"ts"`
	Type       string           `json:"type"`
	UsedLoan   string           `json:"usedLoan"`
}

// BorrowInterestAndLimitResponse represents borrow interest and limit rate for different loan type.
type BorrowInterestAndLimitResponse struct {
	Debt             string           `json:"debt"`
	Interest         string           `json:"interest"`
	NextDiscountTime okxUnixMilliTime `json:"nextDiscountTime"`
	NextInterestTime okxUnixMilliTime `json:"nextInterestTime"`
	Records          []struct {
		AvailLoan  string `json:"availLoan"`
		Currency   string `json:"ccy"`
		Interest   string `json:"interest"`
		LoanQuota  string `json:"loanQuota"`
		PosLoan    string `json:"posLoan"` // Frozen amount for current account Only applicable to VIP loans
		Rate       string `json:"rate"`
		SurplusLmt string `json:"surplusLmt"`
		UsedLmt    string `json:"usedLmt"`
		UsedLoan   string `json:"usedLoan"`
	} `json:"records"`
}

// PositionItem represents current position of the user.
type PositionItem struct {
	Position     string `json:"pos"`
	InstrumentID string `json:"instId"`
}

// PositionBuilderInput represents request parameter for position builder item.
type PositionBuilderInput struct {
	InstrumentType         string         `json:"instType,omitempty"`
	InstrumentID           string         `json:"instId,omitempty"`
	ImportExistingPosition bool           `json:"inclRealPos,omitempty"` // "true"：Import existing positions and hedge with simulated ones "false"：Only use simulated positions The default is true
	ListOfPositions        []PositionItem `json:"simPos,omitempty"`
	PositionsCount         uint           `json:"pos,omitempty"`
}

// PositionBuilderResponse represents a position builder endpoint response.
type PositionBuilderResponse struct {
	InitialMarginRequirement     string                `json:"imr"` // Initial margin requirement of riskUnit dimension
	MaintenanceMarginRequirement string                `json:"mmr"` // Maintenance margin requirement of riskUnit dimension
	SpotAndVolumeMovement        string                `json:"mr1"`
	ThetaDecay                   string                `json:"mr2"`
	VegaTermStructure            string                `json:"mr3"`
	BasicRisk                    string                `json:"mr4"`
	InterestRateRisk             string                `json:"mr5"`
	ExtremeMarketMove            string                `json:"mr6"`
	TransactionCostAndSlippage   string                `json:"mr7"`
	PositionData                 []PositionBuilderData `json:"posData"` // List of positions
	RiskUnit                     string                `json:"riskUnit"`
	Timestamp                    okxUnixMilliTime      `json:"ts"`
}

// PositionBuilderData represent a position item.
type PositionBuilderData struct {
	Delta              string `json:"delta"`
	Gamma              string `json:"gamma"`
	InstrumentID       string `json:"instId"`
	InstrumentType     string `json:"instType"`
	NotionalUsd        string `json:"notionalUsd"` // Quantity of positions usd
	QuantityOfPosition string `json:"pos"`         // Quantity of positions
	Theta              string `json:"theta"`       // Sensitivity of option price to remaining maturity
	Vega               string `json:"vega"`        // Sensitivity of option price to implied volatility
}

// GreeksItem represents greeks response
type GreeksItem struct {
	ThetaBS   string           `json:"thetaBS"`
	ThetaPA   string           `json:"thetaPA"`
	DeltaBS   string           `json:"deltaBS"`
	DeltaPA   string           `json:"deltaPA"`
	GammaBS   string           `json:"gammaBS"`
	GammaPA   string           `json:"gammaPA"`
	VegaBS    string           `json:"vegaBS"`
	VegaPA    string           `json:"vegaPA"`
	Currency  string           `json:"ccy"`
	Timestamp okxUnixMilliTime `json:"ts"`
}

// CounterpartiesResponse represents
type CounterpartiesResponse struct {
	TraderName string `json:"traderName"`
	TraderCode string `json:"traderCode"`
	Type       string `json:"type"`
}

// RFQOrderLeg represents Rfq Order responses leg.
type RFQOrderLeg struct {
	Size         string `json:"sz"`
	Side         string `json:"side"`
	InstrumentID string `json:"instId"`
	TgtCurrency  string `json:"tgtCcy,omitempty"`
}

// CreateRFQInput RFQ create method input.
type CreateRFQInput struct {
	Anonymous           bool          `json:"anonymous"`
	CounterParties      []string      `json:"counterparties"`
	ClientSuppliedRFQID string        `json:"clRfqId"`
	Legs                []RFQOrderLeg `json:"legs"`
}

// CancelRFQRequestParam represents cancel RFQ order request params
type CancelRFQRequestParam struct {
	RfqID               string `json:"rfqId"`
	ClientSuppliedRFQID string `json:"clRfqId"`
}

// CancelRFQRequestsParam represents cancel multiple RFQ orders request params
type CancelRFQRequestsParam struct {
	RfqID               []string `json:"rfqIds"`
	ClientSuppliedRFQID []string `json:"clRfqIds"`
}

// CancelRFQResponse represents cancel RFQ orders response
type CancelRFQResponse struct {
	RfqID               string `json:"rfqId"`
	ClientSuppliedRfqID string `json:"clRfqId"`
	StatusCode          string `json:"sCode"`
	StatusMsg           string `json:"sMsg"`
}

// TimestampResponse holds timestamp response only.
type TimestampResponse struct {
	Timestamp okxUnixMilliTime `json:"ts"`
}

// ExecuteQuoteParams represents Execute quote request params
type ExecuteQuoteParams struct {
	RfqID   string `json:"rfqId"`
	QuoteID string `json:"quoteId"`
}

// ExecuteQuoteResponse represents execute quote response.
type ExecuteQuoteResponse struct {
	BlockTradedID         string           `json:"blockTdId"`
	RfqID                 string           `json:"rfqId"`
	ClientSuppliedRfqID   string           `json:"clRfqId"`
	QuoteID               string           `json:"quoteId"`
	ClientSuppliedQuoteID string           `json:"clQuoteId"`
	TraderCode            string           `json:"tTraderCode"`
	MakerTraderCode       string           `json:"mTraderCode"`
	CreationTime          okxUnixMilliTime `json:"cTime"`
	Legs                  []OrderLeg       `json:"legs"`
}

// OrderLeg represents legs information for both websocket and REST available Quote information.
type OrderLeg struct {
	Price          string `json:"px"`
	Size           string `json:"sz"`
	InstrumentID   string `json:"instId"`
	Side           string `json:"side"`
	TargetCurrency string `json:"tgtCcy"`

	// available in REST only
	Fee         float64 `json:"fee,string"`
	FeeCurrency string  `json:"feeCcy"`
	TradeID     string  `json:"tradeId"`
}

// CreateQuoteParams holds information related to create quote.
type CreateQuoteParams struct {
	RfqID                 string     `json:"rfqId"`
	ClientSuppliedQuoteID string     `json:"clQuoteId"`
	QuoteSide             order.Side `json:"quoteSide"`
	Legs                  []QuoteLeg `json:"legs"`
}

// QuoteLeg the legs of the Quote.
type QuoteLeg struct {
	Price          float64    `json:"px,string"`
	SizeOfQuoteLeg float64    `json:"sz,string"`
	InstrumentID   string     `json:"instId"`
	Side           order.Side `json:"side"`

	// TargetCurrency represents target currency
	TargetCurrency string `json:"tgtCcy,omitempty"`
}

// QuoteResponse holds create quote response variables.
type QuoteResponse struct {
	CreationTime          okxUnixMilliTime `json:"cTime"`
	UpdateTime            okxUnixMilliTime `json:"uTime"`
	ValidUntil            okxUnixMilliTime `json:"validUntil"`
	QuoteID               string           `json:"quoteId"`
	ClientSuppliedQuoteID string           `json:"clQuoteId"`
	RfqID                 string           `json:"rfqId"`
	QuoteSide             string           `json:"quoteSide"`
	ClientSuppliedRfqID   string           `json:"clRfqId"`
	TraderCode            string           `json:"traderCode"`
	State                 string           `json:"state"`
	Legs                  []QuoteLeg       `json:"legs"`
}

// CancelQuoteRequestParams represents cancel quote request params
type CancelQuoteRequestParams struct {
	QuoteID               string `json:"quoteId"`
	ClientSuppliedQuoteID string `json:"clQuoteId"`
}

// CancelQuotesRequestParams represents cancel multiple quotes request params
type CancelQuotesRequestParams struct {
	QuoteIDs               []string `json:"quoteIds,omitempty"`
	ClientSuppliedQuoteIDs []string `json:"clQuoteIds,omitempty"`
}

// CancelQuoteResponse represents cancel quote response
type CancelQuoteResponse struct {
	QuoteID               string `json:"quoteId"`
	ClientSuppliedQuoteID string `json:"clQuoteId"`
	SCode                 string `json:"sCode"`
	SMsg                  string `json:"sMsg"`
}

// RfqRequestParams represents get RFQ orders param
type RfqRequestParams struct {
	RfqID               string
	ClientSuppliedRfqID string
	State               string
	BeginningID         string
	EndID               string
	Limit               int64
}

// RFQResponse RFQ response detail.
type RFQResponse struct {
	CreateTime          okxUnixMilliTime `json:"cTime"`
	UpdateTime          okxUnixMilliTime `json:"uTime"`
	ValidUntil          okxUnixMilliTime `json:"validUntil"`
	TraderCode          string           `json:"traderCode"`
	RFQID               string           `json:"rfqId"`
	ClientSuppliedRFQID string           `json:"clRfqId"`
	State               string           `json:"state"`
	Counterparties      []string         `json:"counterparties"`
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
	Limit                 int64
}

// RFQTradesRequestParams represents RFQ trades request param
type RFQTradesRequestParams struct {
	RfqID                 string
	ClientSuppliedRfqID   string
	QuoteID               string
	BlockTradeID          string
	ClientSuppliedQuoteID string
	State                 string
	BeginID               string
	EndID                 string
	Limit                 int64
}

// RfqTradeResponse RFQ trade response
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

// RFQTradeLeg RFQ trade response leg.
type RFQTradeLeg struct {
	InstrumentID string  `json:"instId"`
	Side         string  `json:"side"`
	Size         string  `json:"sz"`
	Price        float64 `json:"px,string"`
	TradeID      string  `json:"tradeId"`

	Fee         float64 `json:"fee,string,omitempty"`
	FeeCurrency string  `json:"feeCcy,omitempty"`
}

// PublicTradesResponse represents data will be pushed whenever there is a block trade.
type PublicTradesResponse struct {
	BlockTradeID string           `json:"blockTdId"`
	CreationTime okxUnixMilliTime `json:"cTime"`
	Legs         []RFQTradeLeg    `json:"legs"`
}

// SubaccountInfo represents subaccount information detail.
type SubaccountInfo struct {
	Enable          bool             `json:"enable"`
	SubAccountName  string           `json:"subAcct"`
	SubaccountType  string           `json:"type"` // sub-account note
	SubaccountLabel string           `json:"label"`
	MobileNumber    string           `json:"mobile"`      // Mobile number that linked with the sub-account.
	GoogleAuth      bool             `json:"gAuth"`       // If the sub-account switches on the Google Authenticator for login authentication.
	CanTransferOut  bool             `json:"canTransOut"` // If can transfer out, false: can not transfer out, true: can transfer.
	Timestamp       okxUnixMilliTime `json:"ts"`
}

// SubaccountBalanceDetail represents subaccount balance detail
type SubaccountBalanceDetail struct {
	AvailableBalance               string           `json:"availBal"`
	AvailableEquity                string           `json:"availEq"`
	CashBalance                    string           `json:"cashBal"`
	Currency                       string           `json:"ccy"`
	CrossLiability                 string           `json:"crossLiab"`
	DiscountEquity                 string           `json:"disEq"`
	Equity                         string           `json:"eq"`
	EquityUsd                      string           `json:"eqUsd"`
	FrozenBalance                  string           `json:"frozenBal"`
	Interest                       string           `json:"interest"`
	IsoEquity                      string           `json:"isoEq"`
	IsolatedLiabilities            string           `json:"isoLiab"`
	LiabilitiesOfCurrency          string           `json:"liab"`
	MaxLoan                        string           `json:"maxLoan"`
	MarginRatio                    string           `json:"mgnRatio"`
	NotionalLeverage               string           `json:"notionalLever"`
	OrdFrozen                      string           `json:"ordFrozen"`
	Twap                           string           `json:"twap"`
	UpdateTime                     okxUnixMilliTime `json:"uTime"`
	UnrealizedProfitAndLoss        string           `json:"upl"`
	UnrealizedProfitAndLiabilities string           `json:"uplLiab"`
}

// SubaccountBalanceResponse represents subaccount balance response
type SubaccountBalanceResponse struct {
	AdjustedEffectiveEquity      string                    `json:"adjEq"`
	Details                      []SubaccountBalanceDetail `json:"details"`
	Imr                          string                    `json:"imr"`
	IsolatedMarginEquity         string                    `json:"isoEq"`
	MarginRatio                  string                    `json:"mgnRatio"`
	MaintenanceMarginRequirement string                    `json:"mmr"`
	NotionalUsd                  string                    `json:"notionalUsd"`
	OrdFroz                      string                    `json:"ordFroz"`
	TotalEq                      string                    `json:"totalEq"`
	UpdateTime                   okxUnixMilliTime          `json:"uTime"`
}

// FundingBalance holds function balance.
type FundingBalance struct {
	AvailableBalance string `json:"availBal"`
	Balance          string `json:"bal"`
	Currency         string `json:"ccy"`
	FrozenBalance    string `json:"frozenBal"`
}

// SubaccountBillItem represents subaccount balance bill item
type SubaccountBillItem struct {
	BillID                 string           `json:"billId"`
	Type                   string           `json:"type"`
	AccountCurrencyBalance string           `json:"ccy"`
	Amount                 string           `json:"amt"`
	SubAccount             string           `json:"subAcct"`
	Timestamp              okxUnixMilliTime `json:"ts"`
}

// SubAccountAssetTransferParams represents subaccount asset transfer request parameters.
type SubAccountAssetTransferParams struct {
	Currency         string  `json:"ccy"`            // {REQUIRED}
	Amount           float64 `json:"amt,string"`     // {REQUIRED}
	From             int64   `json:"from,string"`    // {REQUIRED} 6:Funding Account 18:Trading account
	To               int64   `json:"to,string"`      // {REQUIRED} 6:Funding Account 18:Trading account
	FromSubAccount   string  `json:"fromSubAccount"` // {REQUIRED} subaccount name.
	ToSubAccount     string  `json:"toSubAccount"`   // {REQUIRED} destination sub-account
	LoanTransfer     bool    `json:"loanTrans,omitempty"`
	OmitPositionRisk bool    `json:"omitPosRisk,omitempty"`
}

// TransferIDInfo represents master account transfer between subaccount.
type TransferIDInfo struct {
	TransferID string `json:"transId"`
}

// PermissionOfTransfer represents subaccount transfer information and it's permission.
type PermissionOfTransfer struct {
	SubAcct     string `json:"subAcct"`
	CanTransOut bool   `json:"canTransOut"`
}

// SubaccountName represents single subaccount name
type SubaccountName struct {
	SubaccountName string `json:"subAcct"`
}

// GridAlgoOrder represents grid algo order.
type GridAlgoOrder struct {
	InstrumentID string  `json:"instId"`
	AlgoOrdType  string  `json:"algoOrdType"`
	MaxPrice     float64 `json:"maxPx,string"`
	MinPrice     float64 `json:"minPx,string"`
	GridQuantity float64 `json:"gridNum,string"`
	GridType     string  `json:"runType"` // "1": Arithmetic, "2": Geometric Default is Arithmetic

	// Spot Grid Order
	QuoteSize float64 `json:"quoteSz,string"` // Invest amount for quote currency Either "instId" or "ccy" is required
	BaseSize  float64 `json:"baseSz,string"`  // Invest amount for base currency Either "instId" or "ccy" is required

	// Contract Grid Order
	BasePosition bool    `json:"basePos"` // Whether or not open a position when strategy actives Default is false Neutral contract grid should omit the parameter
	Size         float64 `json:"sz,string"`
	Direction    string  `json:"direction"`
	Lever        string  `json:"lever"`
}

// GridAlgoOrderIDResponse represents grid algo order
type GridAlgoOrderIDResponse struct {
	AlgoOrderID string `json:"algoId"`
	SCode       string `json:"sCode"`
	SMsg        string `json:"sMsg"`
}

// GridAlgoOrderAmend represents amend algo order response
type GridAlgoOrderAmend struct {
	AlgoID                 string `json:"algoId"`
	InstrumentID           string `json:"instId"`
	StopLossTriggerPrice   string `json:"slTriggerPx"`
	TakeProfitTriggerPrice string `json:"tpTriggerPx"`
}

// StopGridAlgoOrderRequest represents stop grid algo order request parameter
type StopGridAlgoOrderRequest struct {
	AlgoID        string `json:"algoId"`
	InstrumentID  string `json:"instId"`
	StopType      uint   `json:"stopType,string"` // Spot grid "1": Sell base currency "2": Keep base currency | Contract grid "1": Market Close All positions "2": Keep positions
	AlgoOrderType string `json:"algoOrdType"`
}

// GridAlgoOrderResponse a complete information of grid algo order item response.
type GridAlgoOrderResponse struct {
	ActualLever               string           `json:"actualLever"`
	AlgoID                    string           `json:"algoId"`
	AlgoOrderType             string           `json:"algoOrdType"`
	ArbitrageNumber           string           `json:"arbitrageNum"`
	BasePosition              bool             `json:"basePos"`
	BaseSize                  string           `json:"baseSz"`
	CancelType                string           `json:"cancelType"`
	Direction                 string           `json:"direction"`
	FloatProfit               string           `json:"floatProfit"`
	GridQuantity              string           `json:"gridNum"`
	GridProfit                string           `json:"gridProfit"`
	InstrumentID              string           `json:"instId"`
	InstrumentType            string           `json:"instType"`
	Investment                string           `json:"investment"`
	Leverage                  string           `json:"lever"`
	EstimatedLiquidationPrice string           `json:"liqPx"`
	MaximumPrice              string           `json:"maxPx"`
	MinimumPrice              string           `json:"minPx"`
	ProfitAndLossRatio        string           `json:"pnlRatio"`
	QuoteSize                 string           `json:"quoteSz"`
	RunType                   string           `json:"runType"`
	StopLossTriggerPx         string           `json:"slTriggerPx"`
	State                     string           `json:"state"`
	StopResult                string           `json:"stopResult,omitempty"`
	StopType                  string           `json:"stopType"`
	Size                      string           `json:"sz"`
	Tag                       string           `json:"tag"`
	TotalProfitAndLoss        string           `json:"totalPnl"`
	TakeProfitTriggerPrice    string           `json:"tpTriggerPx"`
	CreationTime              okxUnixMilliTime `json:"cTime"`
	UpdateTime                okxUnixMilliTime `json:"uTime"`
	Underlying                string           `json:"uly"`

	// Added in Detail

	EquityOfStrength    string `json:"eq,omitempty"`
	PerMaxProfitRate    string `json:"perMaxProfitRate,omitempty"`
	PerMinProfitRate    string `json:"perMinProfitRate,omitempty"`
	Profit              string `json:"profit,omitempty"`
	Runpx               string `json:"runpx,omitempty"`
	SingleAmt           string `json:"singleAmt,omitempty"`
	TotalAnnualizedRate string `json:"totalAnnualizedRate,omitempty"`
	TradeNumber         string `json:"tradeNum,omitempty"`

	// Suborders Detail

	AnnualizedRate string `json:"annualizedRate,omitempty"`
	CurBaseSz      string `json:"curBaseSz,omitempty"`
	CurQuoteSz     string `json:"curQuoteSz,omitempty"`
}

// GridAlgoSuborder represents a grid algo suborder item.
type GridAlgoSuborder struct {
	ActualLeverage      string           `json:"actualLever"`
	AlgoID              string           `json:"algoId"`
	AlgoOrderType       string           `json:"algoOrdType"`
	AnnualizedRate      string           `json:"annualizedRate"`
	ArbitrageNum        string           `json:"arbitrageNum"`
	BasePosition        bool             `json:"basePos"`
	BaseSize            string           `json:"baseSz"`
	CancelType          string           `json:"cancelType"`
	CurBaseSz           string           `json:"curBaseSz"`
	CurQuoteSz          string           `json:"curQuoteSz"`
	Direction           string           `json:"direction"`
	EquityOfStrength    string           `json:"eq"`
	FloatProfit         string           `json:"floatProfit"`
	GridQuantity        string           `json:"gridNum"`
	GridProfit          string           `json:"gridProfit"`
	InstrumentID        string           `json:"instId"`
	InstrumentType      string           `json:"instType"`
	Investment          string           `json:"investment"`
	Leverage            string           `json:"lever"`
	LiquidationPx       string           `json:"liqPx"`
	MaximumPrice        string           `json:"maxPx"`
	MinimumPrice        string           `json:"minPx"`
	PerMaxProfitRate    string           `json:"perMaxProfitRate"`
	PerMinProfitRate    string           `json:"perMinProfitRate"`
	ProfitAndLossRatio  string           `json:"pnlRatio"`
	Profit              string           `json:"profit"`
	QuoteSize           string           `json:"quoteSz"`
	RunType             string           `json:"runType"`
	Runpx               string           `json:"runpx"`
	SingleAmount        string           `json:"singleAmt"`
	StopLossTriggerPx   string           `json:"slTriggerPx"`
	State               string           `json:"state"`
	StopResult          string           `json:"stopResult"`
	StopType            string           `json:"stopType"`
	Size                string           `json:"sz"`
	Tag                 string           `json:"tag"`
	TotalAnnualizedRate string           `json:"totalAnnualizedRate"`
	TotalProfitAndLoss  string           `json:"totalPnl"`
	TakeProfitTriggerPx string           `json:"tpTriggerPx"`
	TradeNum            string           `json:"tradeNum"`
	UpdateTime          okxUnixMilliTime `json:"uTime"`
	CreationTime        okxUnixMilliTime `json:"cTime"`
}

// AlgoOrderPosition represents algo order position detailed data.
type AlgoOrderPosition struct {
	AutoDecreasingLine           string           `json:"adl"`
	AlgoID                       string           `json:"algoId"`
	AveragePrice                 string           `json:"avgPx"`
	Currency                     string           `json:"ccy"`
	InitialMarginRequirement     string           `json:"imr"`
	InstrumentID                 string           `json:"instId"`
	InstrumentType               string           `json:"instType"`
	LastTradedPrice              string           `json:"last"`
	Leverage                     string           `json:"lever"`
	LiquidationPrice             string           `json:"liqPx"`
	MarkPrice                    string           `json:"markPx"`
	MarginMode                   string           `json:"mgnMode"`
	MarginRatio                  string           `json:"mgnRatio"`
	MaintenanceMarginRequirement string           `json:"mmr"`
	NotionalUSD                  string           `json:"notionalUsd"`
	QuantityPosition             string           `json:"pos"`
	PositionSide                 string           `json:"posSide"`
	UnrealizedProfitAndLoss      string           `json:"upl"`
	UnrealizedProfitAndLossRatio string           `json:"uplRatio"`
	UpdateTime                   okxUnixMilliTime `json:"uTime"`
	CreationTime                 okxUnixMilliTime `json:"cTime"`
}

// AlgoOrderWithdrawalProfit algo withdrawal order profit info.
type AlgoOrderWithdrawalProfit struct {
	AlgoID         string `json:"algoId"`
	WithdrawProfit string `json:"profit"`
}

// SystemStatusResponse represents the system status and other details.
type SystemStatusResponse struct {
	Title               string           `json:"title"`
	State               string           `json:"state"`
	Begin               okxUnixMilliTime `json:"begin"` // Begin time of system maintenance,
	End                 okxUnixMilliTime `json:"end"`   // Time of resuming trading totally.
	Href                string           `json:"href"`  // Hyperlink for system maintenance details
	ServiceType         string           `json:"serviceType"`
	System              string           `json:"system"`
	ScheduleDescription string           `json:"scheDesc"`

	// PushTime timestamp information when the data is pushed
	PushTime okxUnixMilliTime `json:"ts"`
}

// BlockTicker holds block trading information.
type BlockTicker struct {
	InstrumentType           string           `json:"instType"`
	InstrumentID             string           `json:"instId"`
	TradingVolumeInCCY24Hour float64          `json:"volCcy24h,string"`
	TradingVolumeInUSD24Hour float64          `json:"vol24h,string"`
	Timestamp                okxUnixMilliTime `json:"ts"`
}

// BlockTrade represents a block trade.
type BlockTrade struct {
	InstrumentID string           `json:"instId"`
	TradeID      string           `json:"tradeId"`
	Price        float64          `json:"px,string"`
	Size         float64          `json:"sz,string"`
	Side         order.Side       `json:"side"`
	Timestamp    okxUnixMilliTime `json:"ts"`
}

// UnitConvertResponse unit convert response.
type UnitConvertResponse struct {
	InstrumentID string  `json:"instId"`
	Price        float64 `json:"px,string"`
	Size         float64 `json:"sz,string"`
	ConvertType  uint64  `json:"type"`
	Unit         string  `json:"unit"`
}

// Websocket Models

// WebsocketEventRequest contains event data for a websocket channel
type WebsocketEventRequest struct {
	Operation string               `json:"op"`   // 1--subscribe 2--unsubscribe 3--login
	Arguments []WebsocketLoginData `json:"args"` // args: the value is the channel name, which can be one or more channels
}

// WebsocketLoginData represents the websocket login data input json data.
type WebsocketLoginData struct {
	APIKey     string    `json:"apiKey"`
	Passphrase string    `json:"passphrase"`
	Timestamp  time.Time `json:"timestamp"`
	Sign       string    `json:"sign"`
}

// WSLoginResponse represents a websocket login response.
type WSLoginResponse struct {
	Event string `json:"event"`
	Code  string `json:"code"`
	Msg   string `json:"msg"`
}

// SubscriptionInfo holds the channel and instrument IDs.
type SubscriptionInfo struct {
	Channel        string `json:"channel"`
	InstrumentID   string `json:"instId,omitempty"`
	InstrumentType string `json:"instType,omitempty"`
	Underlying     string `json:"uly,omitempty"`
	UID            string `json:"uid,omitempty"` // user identifier

	// For Algo Orders
	AlgoID   string `json:"algoId,omitempty"`
	Currency string `json:"ccy,omitempty"` // Currency:
}

// WSSubscriptionInformation websocket subscription and unsubscription operation inputs.
type WSSubscriptionInformation struct {
	Operation string           `json:"op"`
	Arguments SubscriptionInfo `json:"arg"`
}

// WSSubscriptionInformationList websocket subscription and unsubscription operation inputs.
type WSSubscriptionInformationList struct {
	Operation string             `json:"op"`
	Arguments []SubscriptionInfo `json:"args"`
}

// WSSubscriptionResponse represents websocket subscription information.
type WSSubscriptionResponse struct {
	Event    string           `json:"event"`
	Argument SubscriptionInfo `json:"arg,,omitempty"`
	Code     int              `json:"code,string,omitempty"`
	Msg      string           `json:"msg,omitempty"`
}

// WSInstrumentsResponse represents instrument subscription response.
type WSInstrumentsResponse struct {
	Arguments []SubscriptionInfo `json:"args"`
	Data      []Instrument       `json:"data"`
}

// WSMarketDataResponse represents market data response and it's arguments.
type WSMarketDataResponse struct {
	Arguments []SubscriptionInfo `json:"args"`
	Data      []TickerResponse   `json:"data"`
}

// WSPlaceOrderData holds websocket order information.
type WSPlaceOrderData struct {
	ClientSuppliedOrderID string  `json:"clOrdId,omitempty"`
	Currency              string  `json:"ccy,omitempty"`
	Tag                   string  `json:"tag,omitempty"`
	PositionSide          string  `json:"posSide,omitempty"`
	ExpiryTime            int64   `json:"expTime,string,omitempty"`
	BanAmend              bool    `json:"banAmend,omitempty"`
	Side                  string  `json:"side"`
	InstrumentID          string  `json:"instId"`
	TradeMode             string  `json:"tdMode"`
	OrderType             string  `json:"ordType"`
	Size                  float64 `json:"sz"`
	Price                 float64 `json:"px,string,omitempty"`
	ReduceOnly            bool    `json:"reduceOnly,string,omitempty"`
	TargetCurrency        string  `json:"tgtCurrency,omitempty"`
}

// WSPlaceOrder holds the websocket place order input data.
type WSPlaceOrder struct {
	ID        string             `json:"id"`
	Operation string             `json:"op"`
	Arguments []WSPlaceOrderData `json:"args"`
}

// WSOrderResponse place order response thought the websocket connection.
type WSOrderResponse struct {
	ID        string      `json:"id"`
	Operation string      `json:"op"`
	Data      []OrderData `json:"data"`
	Code      string      `json:"code,omitempty"`
	Msg       string      `json:"msg,omitempty"`
}

// WebsocketDataResponse represents all pushed websocket data coming thought the websocket connection
type WebsocketDataResponse struct {
	Argument SubscriptionInfo `json:"arg"`
	Action   string           `json:"action"`
	Data     []interface{}    `json:"data"`
}

type wsRequestInfo struct {
	ID             string
	Chan           chan *wsIncomingData
	Event          string
	Channel        string
	InstrumentType string
	InstrumentID   string
}

type wsIncomingData struct {
	Event    string           `json:"event,omitempty"`
	Argument SubscriptionInfo `json:"arg,omitempty"`
	Code     string           `json:"code,omitempty"`
	Msg      string           `json:"msg,omitempty"`

	// For Websocket Trading Endpoints websocket responses
	ID        string          `json:"id,omitempty"`
	Operation string          `json:"op,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// copyToPlaceOrderResponse returns WSPlaceOrderResponse struct instance
func (w *wsIncomingData) copyToPlaceOrderResponse() (*WSOrderResponse, error) {
	if len(w.Data) == 0 {
		return nil, errEmptyPlaceOrderResponse
	}
	var placeOrds []OrderData
	err := json.Unmarshal(w.Data, &placeOrds)
	if err != nil {
		return nil, err
	}
	return &WSOrderResponse{
		Operation: w.Operation,
		ID:        w.ID,
		Data:      placeOrds,
	}, nil
}

// WSInstrumentResponse represents websocket instruments push message.
type WSInstrumentResponse struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []Instrument     `json:"data"`
}

// WSTickerResponse represents websocket ticker response.
type WSTickerResponse struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []TickerResponse `json:"data"`
}

// WSCandlestickData represents candlestick data coming through the web socket channels
type WSCandlestickData struct {
	Timestamp                     time.Time `json:"ts"`
	OpenPrice                     float64   `json:"o"`
	HighestPrice                  float64   `json:"p"`
	LowestPrice                   float64   `json:"l"`
	ClosePrice                    float64   `json:"c"`
	TradingVolume                 float64   `json:"vol"`
	TradingVolumeWithCurrencyUnit float64   `json:"volCcy"`
}

// WSCandlestickResponse represents candlestick response of with list of candlestick and
type WSCandlestickResponse struct {
	Argument SubscriptionInfo    `json:"arg"`
	Data     []WSCandlestickData `json:"data"`
}

// WSOpenInterestResponse represents an open interest instance.
type WSOpenInterestResponse struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []OpenInterest   `json:"data"`
}

// WSTradeData websocket trade data response.
type WSTradeData struct {
	InstrumentID string           `json:"instId"`
	TradeID      string           `json:"tradeId"`
	Price        float64          `json:"px,string"`
	Size         float64          `json:"sz,string"`
	Side         string           `json:"side"`
	Timestamp    okxUnixMilliTime `json:"ts"`
}

// WSPlaceOrderInput place order input variables as a json.
type WSPlaceOrderInput struct {
	Side           order.Side `json:"side"`
	InstrumentID   string     `json:"instId"`
	TradeMode      string     `json:"tdMode"`
	OrderType      string     `json:"ordType"`
	Size           float64    `json:"sz,string"`
	Currency       string     `json:"ccy"`
	ClientOrderID  string     `json:"clOrdId,omitempty"`
	Tag            string     `json:"tag,omitempty"`
	PositionSide   string     `json:"posSide,omitempty"`
	Price          float64    `json:"px,string,omitempty"`
	ReduceOnly     bool       `json:"reduceOnly,omitempty"`
	TargetCurrency string     `json:"tgtCcy"`
}

// WsPlaceOrderInput for all websocket request inputs.
type WsPlaceOrderInput struct {
	ID        string                   `json:"id"`
	Operation string                   `json:"op"`
	Arguments []PlaceOrderRequestParam `json:"args"`
}

// WsCancelOrderInput websocket cancel order request
type WsCancelOrderInput struct {
	ID        string                    `json:"id"`
	Operation string                    `json:"op"`
	Arguments []CancelOrderRequestParam `json:"args"`
}

// WsAmendOrderInput websocket handler amend Order response
type WsAmendOrderInput struct {
	ID        string                    `json:"id"`
	Operation string                    `json:"op"`
	Arguments []AmendOrderRequestParams `json:"args"`
}

// WsOrderActionResponse holds websocket response Amendment request
type WsOrderActionResponse struct {
	ID        string      `json:"id"`
	Operation string      `json:"op"`
	Data      []OrderData `json:"data"`
	Code      string      `json:"code"`
	Msg       string      `json:"msg"`
}

func (a *WsOrderActionResponse) populateFromIncomingData(incoming *wsIncomingData) error {
	if incoming == nil {
		return errNilArgument
	}
	a.ID = incoming.ID
	a.Code = incoming.Code
	a.Operation = incoming.Operation
	a.Msg = incoming.Msg
	return nil
}

// SubscriptionOperationInput represents the account channel input data
type SubscriptionOperationInput struct {
	Operation string             `json:"op"`
	Arguments []SubscriptionInfo `json:"args"`
}

// SubscriptionOperationResponse holds account subscription response thought the websocket channel.
type SubscriptionOperationResponse struct {
	Event    string            `json:"event"`
	Argument *SubscriptionInfo `json:"arg,omitempty"`
	Code     string            `json:"code,omitempty"`
	Msg      string            `json:"msg,omitempty"`
}

// WsAccountChannelPushData holds the websocket push data following the subscription.
type WsAccountChannelPushData struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []Account        `json:"data,omitempty"`
}

// WsPositionResponse represents pushed position data through the websocket channel.
type WsPositionResponse struct {
	Argument  SubscriptionInfo  `json:"arg"`
	Arguments []AccountPosition `json:"data"`
}

// PositionDataDetail position data information for the websocket push data
type PositionDataDetail struct {
	PositionID       string           `json:"posId"`
	TradeID          string           `json:"tradeId"`
	InstrumentID     string           `json:"instId"`
	InstrumentType   string           `json:"instType"`
	MarginMode       string           `json:"mgnMode"`
	PositionSide     string           `json:"posSide"`
	Position         string           `json:"pos"`
	Currency         string           `json:"ccy"`
	PositionCurrency string           `json:"posCcy"`
	AveragePrice     string           `json:"avgPx"`
	UpdateTime       okxUnixMilliTime `json:"uTIme"`
}

// BalanceData represents currency and it's Cash balance with the update time.
type BalanceData struct {
	Currency    string           `json:"ccy"`
	CashBalance string           `json:"cashBal"`
	UpdateTime  okxUnixMilliTime `json:"uTime"`
}

// BalanceAndPositionData represents balance and position data with the push time.
type BalanceAndPositionData struct {
	PushTime     okxUnixMilliTime     `json:"pTime"`
	EventType    string               `json:"eventType"`
	BalanceData  []BalanceData        `json:"balData"`
	PositionData []PositionDataDetail `json:"posData"`
}

// WsBalanceAndPosition websocket push data for lis of BalanceAndPosition information.
type WsBalanceAndPosition struct {
	Argument SubscriptionInfo         `json:"arg"`
	Data     []BalanceAndPositionData `json:"data"`
}

// WsOrder represents a websocket order.
type WsOrder struct {
	PendingOrderItem
	AmendResult     string  `json:"amendResult"`
	Code            string  `json:"code"`
	ExecType        string  `json:"execType"`
	FillFee         string  `json:"fillFee"`
	FillFeeCurrency string  `json:"fillFeeCcy"`
	FillNationalUsd float64 `json:"fillNationalUsd,string"`
	Msg             string  `json:"msg"`
	NationalUSD     string  `json:"nationalUsd"`
	ReduceOnly      bool    `json:"reduceOnly"`
	RequestID       string  `json:"reqId"`
}

// WsOrderResponse holds order list push data through the websocket connection
type WsOrderResponse struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []WsOrder        `json:"data"`
}

// WsAlgoOrder algo order detailed data.
type WsAlgoOrder struct {
	Argument SubscriptionInfo    `json:"arg"`
	Data     []WsAlgoOrderDetail `json:"data"`
}

// WsAlgoOrderDetail algo order response pushed through the websocket conn
type WsAlgoOrderDetail struct {
	InstrumentType             string           `json:"instType"`
	InstrumentID               string           `json:"instId"`
	OrderID                    string           `json:"ordId"`
	Currency                   string           `json:"ccy"`
	AlgoID                     string           `json:"algoId"`
	Price                      string           `json:"px"`
	Size                       string           `json:"sz"`
	TradeMode                  string           `json:"tdMode"`
	TargetCurrency             string           `json:"tgtCcy"`
	NotionalUsd                string           `json:"notionalUsd"`
	OrderType                  string           `json:"ordType"`
	Side                       string           `json:"side"`
	PositionSide               string           `json:"posSide"`
	State                      string           `json:"state"`
	Leverage                   string           `json:"lever"`
	TakeProfitTriggerPrice     string           `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string           `json:"tpTriggerPxType"`
	TakeProfitOrdPrice         string           `json:"tpOrdPx"`
	StopLossTriggerPrice       string           `json:"slTriggerPx"`
	StopLossTriggerPriceType   string           `json:"slTriggerPxType"`
	TriggerPrice               string           `json:"triggerPx"`
	TriggerPriceType           string           `json:"triggerPxType"`
	OrderPrice                 float64          `json:"ordPx,string"`
	ActualSize                 string           `json:"actualSz"`
	ActualPrice                string           `json:"actualPx"`
	Tag                        string           `json:"tag"`
	ActualSide                 string           `json:"actualSide"`
	TriggerTime                okxUnixMilliTime `json:"triggerTime"`
	CreationTime               okxUnixMilliTime `json:"cTime"`
}

// WsAdvancedAlgoOrder advanced algo order response.
type WsAdvancedAlgoOrder struct {
	Argument SubscriptionInfo            `json:"arg"`
	Data     []WsAdvancedAlgoOrderDetail `json:"data"`
}

// WsAdvancedAlgoOrderDetail advanced algo order response pushed through the websocket conn
type WsAdvancedAlgoOrderDetail struct {
	ActualPrice            string           `json:"actualPx"`
	ActualSide             string           `json:"actualSide"`
	ActualSize             string           `json:"actualSz"`
	AlgoID                 string           `json:"algoId"`
	Currency               string           `json:"ccy"`
	Count                  string           `json:"count"`
	InstrumentID           string           `json:"instId"`
	InstrumentType         string           `json:"instType"`
	Leverage               string           `json:"lever"`
	NotionalUsd            string           `json:"notionalUsd"`
	OrderPrice             string           `json:"ordPx"`
	OrdType                string           `json:"ordType"`
	PositionSide           string           `json:"posSide"`
	PriceLimit             string           `json:"pxLimit"`
	PriceSpread            string           `json:"pxSpread"`
	PriceVariation         string           `json:"pxVar"`
	Side                   string           `json:"side"`
	StopLossOrderPrice     string           `json:"slOrdPx"`
	StopLossTriggerPrice   string           `json:"slTriggerPx"`
	State                  string           `json:"state"`
	Size                   string           `json:"sz"`
	SizeLimit              string           `json:"szLimit"`
	TradeMode              string           `json:"tdMode"`
	TimeInterval           string           `json:"timeInterval"`
	TakeProfitOrderPrice   string           `json:"tpOrdPx"`
	TakeProfitTriggerPrice string           `json:"tpTriggerPx"`
	Tag                    string           `json:"tag"`
	TriggerPrice           string           `json:"triggerPx"`
	CallbackRatio          string           `json:"callbackRatio"`
	CallbackSpread         string           `json:"callbackSpread"`
	ActivePrice            string           `json:"activePx"`
	MoveTriggerPrice       string           `json:"moveTriggerPx"`
	CreationTime           okxUnixMilliTime `json:"cTime"`
	PushTime               okxUnixMilliTime `json:"pTime"`
	TriggerTime            okxUnixMilliTime `json:"triggerTime"`
}

// WsGreeks greeks push data with the subscription info through websocket channel
type WsGreeks struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []WsGreekData    `json:"data"`
}

// WsGreekData greeks push data through websocket channel
type WsGreekData struct {
	ThetaBS   string           `json:"thetaBS"`
	ThetaPA   string           `json:"thetaPA"`
	DeltaBS   string           `json:"deltaBS"`
	DeltaPA   string           `json:"deltaPA"`
	GammaBS   string           `json:"gammaBS"`
	GammaPA   string           `json:"gammaPA"`
	VegaBS    string           `json:"vegaBS"`
	VegaPA    string           `json:"vegaPA"`
	Currency  string           `json:"ccy"`
	Timestamp okxUnixMilliTime `json:"ts"`
}

// WsRFQ represents websocket push data for "rfqs" subscription
type WsRFQ struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []WsRfqData      `json:"data"`
}

// WsRfqData represents rfq order response data streamed through the websocket channel
type WsRfqData struct {
	CreationTime        time.Time     `json:"cTime"`
	UpdateTime          time.Time     `json:"uTime"`
	TraderCode          string        `json:"traderCode"`
	RfqID               string        `json:"rfqId"`
	ClientSuppliedRfqID string        `json:"clRfqId"`
	State               string        `json:"state"`
	ValidUntil          string        `json:"validUntil"`
	Counterparties      []string      `json:"counterparties"`
	Legs                []RFQOrderLeg `json:"legs"`
}

// WsQuote represents websocket push data for "quotes" subscription
type WsQuote struct {
	Arguments SubscriptionInfo `json:"arg"`
	Data      []WsQuoteData    `json:"data"`
}

// WsQuoteData represents a single quote order information
type WsQuoteData struct {
	ValidUntil            okxUnixMilliTime `json:"validUntil"`
	UpdatedTime           okxUnixMilliTime `json:"uTime"`
	CreationTime          okxUnixMilliTime `json:"cTime"`
	Legs                  []OrderLeg       `json:"legs"`
	QuoteID               string           `json:"quoteId"`
	RfqID                 string           `json:"rfqId"`
	TraderCode            string           `json:"traderCode"`
	QuoteSide             string           `json:"quoteSide"`
	State                 string           `json:"state"`
	ClientSuppliedQuoteID string           `json:"clQuoteId"`
}

// WsStructureBlocTrade represents websocket push data for "struc-block-trades" subscription
type WsStructureBlocTrade struct {
	Argument SubscriptionInfo       `json:"arg"`
	Data     []WsBlockTradeResponse `json:"data"`
}

// WsBlockTradeResponse represents a structure block order information
type WsBlockTradeResponse struct {
	CreationTime          okxUnixMilliTime `json:"cTime"`
	RfqID                 string           `json:"rfqId"`
	ClientSuppliedRfqID   string           `json:"clRfqId"`
	QuoteID               string           `json:"quoteId"`
	ClientSuppliedQuoteID string           `json:"clQuoteId"`
	BlockTradeID          string           `json:"blockTdId"`
	TakerTraderCode       string           `json:"tTraderCode"`
	MakerTraderCode       string           `json:"mTraderCode"`
	Legs                  []OrderLeg       `json:"legs"`
}

// WsSpotGridAlgoOrder represents websocket push data for "struc-block-trades" subscription
type WsSpotGridAlgoOrder struct {
	Argument SubscriptionInfo   `json:"arg"`
	Data     []SpotGridAlgoData `json:"data"`
}

// SpotGridAlgoData represents spot grid algo orders.
type SpotGridAlgoData struct {
	AlgoID          string `json:"algoId"`
	AlgoOrderType   string `json:"algoOrdType"`
	AnnualizedRate  string `json:"annualizedRate"`
	ArbitrageNumber string `json:"arbitrageNum"`
	BaseSize        string `json:"baseSz"`
	// Algo order stop reason 0: None 1: Manual stop 2: Take profit
	// 3: Stop loss 4: Risk control 5: delivery
	CancelType           string `json:"cancelType"`
	CurBaseSize          string `json:"curBaseSz"`
	CurQuoteSize         string `json:"curQuoteSz"`
	FloatProfit          string `json:"floatProfit"`
	GridNumber           string `json:"gridNum"`
	GridProfit           string `json:"gridProfit"`
	InstrumentID         string `json:"instId"`
	InstrumentType       string `json:"instType"`
	Investment           string `json:"investment"`
	MaximumPrice         string `json:"maxPx"`
	MinimumPrice         string `json:"minPx"`
	PerMaximumProfitRate string `json:"perMaxProfitRate"`
	PerMinimumProfitRate string `json:"perMinProfitRate"`
	ProfitAndLossRatio   string `json:"pnlRatio"`
	QuoteSize            string `json:"quoteSz"`
	RunPrice             string `json:"runPx"`
	RunType              string `json:"runType"`
	SingleAmount         string `json:"singleAmt"`
	StopLossTriggerPrice string `json:"slTriggerPx"`
	State                string `json:"state"`
	// Stop result of spot grid
	// 0: default, 1: Successful selling of currency at market price,
	// -1: Failed to sell currency at market price
	StopResult string `json:"stopResult"`
	// Stop type Spot grid 1: Sell base currency 2: Keep base currency
	// Contract grid 1: Market Close All positions 2: Keep positions
	StopType               string           `json:"stopType"`
	TotalAnnualizedRate    string           `json:"totalAnnualizedRate"`
	TotalProfitAndLoss     string           `json:"totalPnl"`
	TakeProfitTriggerPrice string           `json:"tpTriggerPx"`
	TradeNum               string           `json:"tradeNum"`
	TriggerTime            okxUnixMilliTime `json:"triggerTime"`
	CreationTime           okxUnixMilliTime `json:"cTime"`
	PushTime               okxUnixMilliTime `json:"pTime"`
	UpdateTime             okxUnixMilliTime `json:"uTime"`
}

// WsContractGridAlgoOrder represents websocket push data for "grid-orders-contract" subscription
type WsContractGridAlgoOrder struct {
	Argument SubscriptionInfo        `json:"arg"`
	Data     []ContractGridAlgoOrder `json:"data"`
}

// ContractGridAlgoOrder represents contract grid algo order
type ContractGridAlgoOrder struct {
	ActualLever            string           `json:"actualLever"`
	AlgoID                 string           `json:"algoId"`
	AlgoOrderType          string           `json:"algoOrdType"`
	AnnualizedRate         string           `json:"annualizedRate"`
	ArbitrageNumber        string           `json:"arbitrageNum"`
	BasePosition           bool             `json:"basePos"`
	CancelType             string           `json:"cancelType"`
	Direction              string           `json:"direction"`
	Eq                     string           `json:"eq"`
	FloatProfit            string           `json:"floatProfit"`
	GridQuantity           string           `json:"gridNum"`
	GridProfit             string           `json:"gridProfit"`
	InstrumentID           string           `json:"instId"`
	InstrumentType         string           `json:"instType"`
	Investment             string           `json:"investment"`
	Leverage               string           `json:"lever"`
	LiqPrice               string           `json:"liqPx"`
	MaxPrice               string           `json:"maxPx"`
	MinPrice               string           `json:"minPx"`
	CreationTime           okxUnixMilliTime `json:"cTime"`
	PushTime               okxUnixMilliTime `json:"pTime"`
	PerMaxProfitRate       string           `json:"perMaxProfitRate"`
	PerMinProfitRate       string           `json:"perMinProfitRate"`
	ProfitAndLossRatio     string           `json:"pnlRatio"`
	RunPrice               string           `json:"runPx"`
	RunType                string           `json:"runType"`
	SingleAmount           string           `json:"singleAmt"`
	SlTriggerPx            string           `json:"slTriggerPx"`
	State                  string           `json:"state"`
	StopType               string           `json:"stopType"`
	Size                   string           `json:"sz"`
	Tag                    string           `json:"tag"`
	TotalAnnualizedRate    string           `json:"totalAnnualizedRate"`
	TotalProfitAndLoss     string           `json:"totalPnl"`
	TakeProfitTriggerPrice string           `json:"tpTriggerPx"`
	TradeNumber            string           `json:"tradeNum"`
	TriggerTime            string           `json:"triggerTime"`
	UpdateTime             string           `json:"uTime"`
	Underlying             string           `json:"uly"`
}

// WsGridPosition represents websocket push data for "grid-positions" subscription
type WsGridPosition struct {
	Argument SubscriptionInfo   `json:"arg"`
	Data     []GridPositionData `json:"data"`
}

// GridPositionData represents a position data
type GridPositionData struct {
	AutoDeleveraging             string           `json:"adl"`
	AlgoID                       string           `json:"algoId"`
	AveragePrice                 string           `json:"avgPx"`
	Currency                     string           `json:"ccy"`
	InitialMarginRequirement     string           `json:"imr"`
	InstrumentID                 string           `json:"instId"`
	InstrumentType               string           `json:"instType"`
	Last                         string           `json:"last"`
	Leverage                     string           `json:"lever"`
	LiquidationPrice             string           `json:"liqPx"`
	MarkPrice                    string           `json:"markPx"`
	MarginMode                   string           `json:"mgnMode"`
	MarginRatio                  string           `json:"mgnRatio"`
	MaintenanceMarginRequirement string           `json:"mmr"`
	NotionalUsd                  string           `json:"notionalUsd"`
	QuantityOfPositions          string           `json:"pos"`
	PositionSide                 string           `json:"posSide"`
	UnrealizedProfitAndLoss      string           `json:"upl"`
	UnrealizedProfitAndLossRatio string           `json:"uplRatio"`
	PushTime                     okxUnixMilliTime `json:"pTime"`
	UpdateTime                   okxUnixMilliTime `json:"uTime"`
	CreationTime                 okxUnixMilliTime `json:"cTime"`
}

// WsGridSubOrderData to retrieve grid sub orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing order.
type WsGridSubOrderData struct {
	Argument SubscriptionInfo   `json:"arg"`
	Data     []GridSubOrderData `json:"data"`
}

// GridSubOrderData represents a single sub order detailed info
type GridSubOrderData struct {
	AccumulatedFillSize string           `json:"accFillSz"`
	AlgoID              string           `json:"algoId"`
	AlgoOrderType       string           `json:"algoOrdType"`
	AveragePrice        string           `json:"avgPx"`
	CreationTime        string           `json:"cTime"`
	ContractValue       string           `json:"ctVal"`
	Fee                 string           `json:"fee"`
	FeeCurrency         string           `json:"feeCcy"`
	GroupID             string           `json:"groupId"`
	InstrumentID        string           `json:"instId"`
	InstrumentType      string           `json:"instType"`
	Leverage            string           `json:"lever"`
	OrderID             string           `json:"ordId"`
	OrderType           string           `json:"ordType"`
	PushTime            okxUnixMilliTime `json:"pTime"`
	ProfitAdLoss        string           `json:"pnl"`
	PositionSide        string           `json:"posSide"`
	Price               string           `json:"px"`
	Side                string           `json:"side"`
	State               string           `json:"state"`
	Size                string           `json:"sz"`
	Tag                 string           `json:"tag"`
	TradeMode           string           `json:"tdMode"`
	UpdateTime          okxUnixMilliTime `json:"uTime"`
}

// WsTradeOrder represents a trade push data response as a result subscription to "trades" channel
type WsTradeOrder struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []TradeResponse  `json:"data"`
}

// WsMarkPrice represents an estimated mark price push data as a result of subscription to "mark-price" channel
type WsMarkPrice struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []MarkPrice      `json:"data"`
}

// WsDeliveryEstimatedPrice represents an estimated delivery/exercise price push data as a result of subscription to "estimated-price" channel
type WsDeliveryEstimatedPrice struct {
	Argument SubscriptionInfo         `json:"arg"`
	Data     []DeliveryEstimatedPrice `json:"data"`
}

// CandlestickMarkPrice represents candlestick mark price push data as a result of  subscription to "mark-price-candle*" channel.
type CandlestickMarkPrice struct {
	Timestamp    time.Time `json:"ts"`
	OpenPrice    float64   `json:"o"`
	HighestPrice float64   `json:"h"`
	LowestPrice  float64   `json:"l"`
	ClosePrice   float64   `json:"s"`
}

// WsOrderBook order book represents order book push data which is returned as a result of subscription to "books*" channel
type WsOrderBook struct {
	Argument SubscriptionInfo  `json:"arg"`
	Action   string            `json:"action"`
	Data     []WsOrderBookData `json:"data"`
}

// WsOrderBookData represents a book order push data.
type WsOrderBookData struct {
	Asks      [][4]string      `json:"asks"`
	Bids      [][4]string      `json:"bids"`
	Timestamp okxUnixMilliTime `json:"ts"`
	Checksum  int32            `json:"checksum,omitempty"`
}

// WsOptionSummary represents option summary
type WsOptionSummary struct {
	Argument SubscriptionInfo           `json:"arg"`
	Data     []OptionMarketDataResponse `json:"data"`
}

// WsFundingRate represents websocket push data funding rate response.
type WsFundingRate struct {
	Argument SubscriptionInfo      `json:"arg"`
	Data     []FundingRateResponse `json:"data"`
}

// WsIndexTicker represents websocket push data index ticker response
type WsIndexTicker struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []IndexTicker    `json:"data"`
}

// WsSystemStatusResponse represents websocket push data system status push data
type WsSystemStatusResponse struct {
	Argument SubscriptionInfo       `json:"arg"`
	Data     []SystemStatusResponse `json:"data"`
}

// WsPublicTradesResponse represents websocket push data of structured block trades as a result of subscription to "public-struc-block-trades"
type WsPublicTradesResponse struct {
	Argument SubscriptionInfo       `json:"arg"`
	Data     []PublicTradesResponse `json:"data"`
}

// WsBlockTicker represents websocket push data as a result of subscription to channel "block-tickers".
type WsBlockTicker struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []BlockTicker    `json:"data"`
}

// PMLimitationResponse represents portfolio margin mode limitation for specific underlying
type PMLimitationResponse struct {
	MaximumSize  float64 `json:"maxSz,string"`
	PositionType string  `json:"postType"`
	Underlying   string  `json:"uly"`
}

// EasyConvertDetail represents easy convert currencies list and their detail.
type EasyConvertDetail struct {
	FromData   []EasyConvertFromData `json:"fromData"`
	ToCurrency []string              `json:"toCcy"`
}

// EasyConvertFromData represents convert currency from detail
type EasyConvertFromData struct {
	FromAmount   float64 `json:"fromAmt,string"`
	FromCurrency string  `json:"fromCcy"`
}

// PlaceEasyConvertParam represents easy convert request params
type PlaceEasyConvertParam struct {
	FromCurrency []string `json:"fromCcy"`
	ToCurrency   string   `json:"toCcy"`
}

// EasyConvertItem represents easy convert place order response.
type EasyConvertItem struct {
	FilFromSize  float64          `json:"fillFromSz,string"`
	FillToSize   float64          `json:"fillToSz,string"`
	FromCurrency string           `json:"fromCcy"`
	Status       string           `json:"status"`
	ToCurrency   string           `json:"toCcy"`
	UpdateTime   okxUnixMilliTime `json:"uTime"`
}

// OneClickRepayCurrencyItem represents debt currency data and repay currencies.
type OneClickRepayCurrencyItem struct {
	DebtData  []CurrencyDebtAmount  `json:"debtData"`
	DebtType  string                `json:"debtType"`
	RepayData []CurrencyRepayAmount `json:"repayData"`
}

// CurrencyDebtAmount represents debt currency data
type CurrencyDebtAmount struct {
	DebtAmount   float64 `json:"debtAmt,string"`
	DebtCurrency string  `json:"debtCcy"`
}

// CurrencyRepayAmount represents rebat currency amount.
type CurrencyRepayAmount struct {
	RepayAmount   float64 `json:"repayAmt,string"`
	RepayCurrency string  `json:"repayCcy"`
}

// TradeOneClickRepayParam represents click one repay param
type TradeOneClickRepayParam struct {
	DebtCurrency  []string `json:"debtCcy"`
	RepayCurrency string   `json:"repayCcy"`
}

// CurrencyOneClickRepay represents one click repay currency
type CurrencyOneClickRepay struct {
	DebtCurrency  string            `json:"debtCcy"`
	FillFromSize  okxNumericalValue `json:"fillFromSz"`
	FillRepaySize okxNumericalValue `json:"fillRepaySz"`
	FillToSize    float64           `json:"fillToSz,string"`
	RepayCurrency string            `json:"repayCcy"`
	Status        string            `json:"status"`
	UpdateTime    time.Time         `json:"uTime"`
}

// SetQuoteProductParam represents set quote product request param
type SetQuoteProductParam struct {
	InstrumentType string                   `json:"instType"`
	Data           []MakerInstrumentSetting `json:"data"`
}

// MakerInstrumentSetting represents set quote product setting info
type MakerInstrumentSetting struct {
	Underlying     string  `json:"uly"`
	InstrumentID   string  `json:"instId"`
	MaxBlockSize   float64 `json:"maxBlockSz,string"`
	MakerPriceBand float64 `json:"makerPxBand,string"`
}

// SetQuoteProductsResult represents set quote products result
type SetQuoteProductsResult struct {
	Result bool `json:"result"`
}

// SubAccountAPIKeyParam represents Reset the APIKey of a sub-account request param
type SubAccountAPIKeyParam struct {
	SubAccountName   string   `json:"subAcct"`         // Sub-account name
	APIKey           string   `json:"apiKey"`          // Sub-accountAPI public key
	Label            string   `json:"label,omitempty"` // Sub-account APIKey label
	APIKeyPermission string   `json:"perm,omitempty"`  // Sub-account APIKey permissions
	IP               string   `json:"ip,omitempty"`    // Sub-account APIKey linked IP addresses, separate with commas if more than
	Permissions      []string `json:"-"`
}

// SubAccountAPIKeyResponse represents sub-account api key reset response
type SubAccountAPIKeyResponse struct {
	SubAccountName   string           `json:"subAcct"`
	APIKey           string           `json:"apiKey"`
	Label            string           `json:"label"`
	APIKeyPermission string           `json:"perm"`
	IP               string           `json:"ip"`
	Timestamp        okxUnixMilliTime `json:"ts"`
}

// MarginBalanceParam represents compute margin balance request param
type MarginBalanceParam struct {
	AlgoID     string  `json:"algoId"`
	Type       string  `json:"type"`
	Amount     float64 `json:"amt,string"`               // Adjust margin balance amount Either amt or percent is required.
	Percentage float64 `json:"percent,string,omitempty"` // Adjust margin balance percentage, used In Adjusting margin balance
}

// ComputeMarginBalance represents compute margin amount request response
type ComputeMarginBalance struct {
	Leverage      float64 `json:"lever,string"`
	MaximumAmount float64 `json:"maxAmt,string"`
}

// AdjustMarginBalanceResponse represents algo id for response for margin balance adjust request.
type AdjustMarginBalanceResponse struct {
	AlgoID string `json:"algoId"`
}

// GridAIParameterResponse represents gri AI parameter response.
type GridAIParameterResponse struct {
	AlgoOrderType        string  `json:"algoOrdType"`
	AnnualizedRate       string  `json:"annualizedRate"`
	Currency             string  `json:"ccy"`
	Direction            string  `json:"direction"`
	Duration             string  `json:"duration"`
	GridNum              string  `json:"gridNum"`
	InstrumentID         string  `json:"instId"`
	Leverage             float64 `json:"lever,string"`
	MaximumPrice         float64 `json:"maxPx,string"`
	MinimumInvestment    float64 `json:"minInvestment,string"`
	MinimumPrice         float64 `json:"minPx,string"`
	PerMaximumProfitRate float64 `json:"perMaxProfitRate,string"`
	PerMinimumProfitRate float64 `json:"perMinProfitRate,string"`
	RunType              string  `json:"runType"`
}

// Offer represents an investment offer information for different 'staking' and 'defi' protocols
type Offer struct {
	Currency     string            `json:"ccy"`
	ProductID    string            `json:"productId"`
	Protocol     string            `json:"protocol"`
	ProtocolType string            `json:"protocolType"`
	EarningCcy   []string          `json:"earningCcy"`
	Term         string            `json:"term"`
	Apy          okxNumericalValue `json:"apy"`
	EarlyRedeem  bool              `json:"earlyRedeem"`
	InvestData   []OfferInvestData `json:"investData"`
	EarningData  []struct {
		Currency    string `json:"ccy"`
		EarningType string `json:"earningType"`
	} `json:"earningData"`
}

// OfferInvestData represents currencies invest data information for an offer
type OfferInvestData struct {
	Currency      string            `json:"ccy"`
	Balance       okxNumericalValue `json:"bal"`
	MinimumAmount okxNumericalValue `json:"minAmt"`
	MaximumAmount okxNumericalValue `json:"maxAmt"`
}

// PurchaseRequestParam represents purchase request param specific product
type PurchaseRequestParam struct {
	ProductID  string                   `json:"productId"`
	Term       int                      `json:"term,string,omitempty"`
	InvestData []PurchaseInvestDataItem `json:"investData"`
}

// PurchaseInvestDataItem represents purchase invest data information having the currency and amount information
type PurchaseInvestDataItem struct {
	Currency string  `json:"ccy"`
	Amount   float64 `json:"amt,string"`
}

// OrderIDResponse represents purchase order ID
type OrderIDResponse struct {
	OrderID string `json:"orderId"`
}

// RedeemRequestParam represents redeem request input param
type RedeemRequestParam struct {
	OrderID          string `json:"ordId"`
	ProtocolType     string `json:"protocolType"`
	AllowEarlyRedeem bool   `json:"allowEarlyRedeem"`
}

// CancelFundingParam cancel purchase or redemption request
type CancelFundingParam struct {
	OrderID      string `json:"ordId"`
	ProtocolType string `json:"protocolType"`
}

// ActiveFundingOrder represents active purchase orders
type ActiveFundingOrder struct {
	OrderID      string `json:"ordId"`
	State        string `json:"state"`
	Currency     string `json:"ccy"`
	Protocol     string `json:"protocol"`
	ProtocolType string `json:"protocolType"`
	Term         string `json:"term"`
	Apy          string `json:"apy"`
	InvestData   []struct {
		Currency string  `json:"ccy"`
		Amount   float64 `json:"amt,string"`
	} `json:"investData"`
	EarningData []struct {
		Ccy         string  `json:"ccy"`
		EarningType string  `json:"earningType"`
		Earnings    float64 `json:"earnings,string"`
	} `json:"earningData"`
	PurchasedTime okxUnixMilliTime `json:"purchasedTime"`
}

// FundingOrder represents orders of earning, purchase, and redeem
type FundingOrder struct {
	OrderID      string  `json:"ordId"`
	State        string  `json:"state"`
	Currency     string  `json:"ccy"`
	Protocol     string  `json:"protocol"`
	ProtocolType string  `json:"protocolType"`
	Term         string  `json:"term"`
	Apy          float64 `json:"apy,string"`
	InvestData   []struct {
		Currency string  `json:"ccy"`
		Amount   float64 `json:"amt,string"`
	} `json:"investData"`
	EarningData []struct {
		Currency         string  `json:"ccy"`
		EarningType      string  `json:"earningType"`
		RealizedEarnings float64 `json:"realizedEarnings,string"`
	} `json:"earningData"`
	PurchasedTime okxUnixMilliTime `json:"purchasedTime"`
	RedeemedTime  okxUnixMilliTime `json:"redeemedTime"`
	EarningCcy    []string         `json:"earningCcy,omitempty"`
}

// wsRequestDataChannelsMultiplexer a single multiplexer instance to multiplex websocket messages multiplexer channels
type wsRequestDataChannelsMultiplexer struct {
	// To Synchronize incoming messages coming through the websocket channel
	WsResponseChannelsMap map[string]*wsRequestInfo
	Register              chan *wsRequestInfo
	Unregister            chan string
	Message               chan *wsIncomingData
	shutdown              chan bool
}

// wsSubscriptionParameters represents toggling boolean values for subscription parameters.
type wsSubscriptionParameters struct {
	InstrumentType bool
	InstrumentID   bool
	Underlying     bool
	Currency       bool
}
