package okx

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// CurrencyConvertType represents two types of currency convert 1: currency-contract and 2: contract-currency
type CurrencyConvertType uint

const (
	// CurrencyToContract 1: Convert currency to contract. It will be rounded up when the value of contract is decimal
	CurrencyToContract = CurrencyConvertType(1)
	// ContractToCurrency 2: Convert contract to currency.
	ContractToCurrency = CurrencyConvertType(2)

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
	OkxOrderLimit = "LIMIT"
	// OkxOrderMarket Market order
	OkxOrderMarket = "MARKET"
	// OkxOrderPostOnly POST_ONLY order type
	OkxOrderPostOnly = "POST_ONLY"
	// OkxOrderFOK fill or kill order type
	OkxOrderFOK = "FOK"
	// OkxOrderIOC IOC (immediate or cancel)
	OkxOrderIOC = "IOC"
	// OkxOrderOptimalLimitIOC OPTIMAL_LIMIT_IOC
	OkxOrderOptimalLimitIOC = "OPTIMAL_LIMIT_IOC"

	// Instrument Types ( Asset Types )

	okxInstTypeFutures  = "FUTURES"  // Okx Instrument Type "futures"
	okxInstTypeANY      = "ANY"      // Okx Instrument Type ""
	okxInstTypeSpot     = "SPOT"     // Okx Instrument Type "spot"
	okxInstTypeSwap     = "SWAP"     // Okx Instrument Type "swap"
	okxInstTypeOption   = "OPTION"   // Okx Instrument Type "option"
	okxInstTypeMargin   = "MARGIN"   // Okx Instrument Type "margin"
	okxInstTypeContract = "CONTRACT" // Okx Instrument Type "contract"
)

// Market Data Endoints

// TickerResponse represents the market data endpoint ticker detail
type TickerResponse struct {
	InstrumentType asset.Item `json:"instType"`
	InstrumentID   string     `json:"instId"`
	LastTradePrice float64    `json:"last,string"`
	LastTradeSize  float64    `json:"lastSz,string"`
	BestAskPrice   float64    `json:"askPx,string"`
	BestAskSize    float64    `json:"askSz,string"`
	BidPrice       float64    `json:"bidPx,string"`
	BidSize        float64    `json:"bidSz,string"`
	Open24H        float64    `json:"open24h,string"`
	High24H        float64    `json:"high24h,string"`
	Low24H         float64    `json:"low24h"`
	VolCcy24H      float64    `json:"volCcy24h,string"`
	Vol24H         float64    `json:"vol24h,string"`

	OpenPriceInUTC0          string    `json:"sodUtc0"`
	OpenPriceInUTC8          string    `json:"sodUtc8"`
	TickerDataGenerationTime time.Time `json:"ts"`
}

// IndexTicker represents Index ticker data.
type IndexTicker struct {
	InstID    string    `json:"instId"`
	IdxPx     float64   `json:"idxPx,string"`
	High24H   float64   `json:"high24h,string"`
	SodUtc0   float64   `json:"sodUtc0,string"`
	Open24H   float64   `json:"open24h,string"`
	Low24H    float64   `json:"low24h,string"`
	SodUtc8   float64   `json:"sodUtc8,string"`
	Timestamp time.Time `json:"ts"`
}

// OrderBookResponse holds the order asks and bids at a specific timestamp
type OrderBookResponse struct {
	Asks                [][4]string `json:"asks"`
	Bids                [][4]string `json:"bids"`
	GenerationTimeStamp time.Time   `json:"ts"`
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
	LiquidationOrders int
	NumberOfOrders    int
}

// OrderBid represents currencies bid detailed information.
type OrderBid struct {
	DepthPrice        float64
	BaseCurrencies    float64
	LiquidationOrders int
	NumberOfOrders    int
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
		GenerationTimestamp: a.GenerationTimeStamp,
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
		liquidation, er := strconv.Atoi(a.Asks[x][2])
		if er != nil {
			return nil, er
		}
		orders, er := strconv.Atoi(a.Asks[x][3])
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
		liquidation, er := strconv.Atoi(a.Bids[x][2])
		if er != nil {
			return nil, er
		}
		orders, er := strconv.Atoi(a.Bids[x][3])
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
	InstrumentID string    `json:"instId"`
	TradeID      string    `json:"tradeId"`
	Price        float64   `json:"px,string"`
	Quantity     float64   `json:"sz,string"`
	Side         string    `json:"side"`
	Timestamp    time.Time `json:"ts"`
}

// TradingVolumeIn24HR response model.
type TradingVolumeIn24HR struct {
	BlockVolumeInCNY   float64   `json:"blockVolCny"`
	BlockVolumeInUSD   float64   `json:"blockVolUsd"`
	TradingVolumeInUSD float64   `json:"volUsd,string"`
	TradingVolumeInCny float64   `json:"volCny,string"`
	Timestamp          time.Time `json:"ts"`
}

// OracleSmartContractResponse returns the crypto price of signing using Open Oracle smart contract.
type OracleSmartContractResponse struct {
	Messages   []string          `json:"messages"`
	Prices     map[string]string `json:"prices"`
	Signatures []string          `json:"signatures"`
	Timestamp  time.Time         `json:"timestamp"`
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
	Timestamp  time.Time            `json:"ts"`
}

// IndexComponentItem an item representing the index component item
type IndexComponentItem struct {
	Symbol          string `json:"symbol"`
	SymbolPairPrice string `json:"symbolPx"`
	Weights         string `json:"wgt"`
	ConverToPrice   string `json:"cnvPx"`
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
	MaxLeverage                     float64    `json:"lever,string"`
	TickSize                        float64    `json:"tickSz,string"`
	LotSize                         float64    `json:"lotSz,string"`
	MinimumOrderSize                float64    `json:"minSz,string"`
	ContractType                    string     `json:"ctType"`
	Alias                           string     `json:"alias"`
	State                           string     `json:"state"`
	MaxQuantityoOfSpotLimitOrder    float64    `json:"maxLmtSz,string"`
	MaxQuantityOfMarketLimitOrder   float64    `json:"maxMktSz,string"`
	MaxQuantityOfSpotTwapLimitOrder float64    `json:"maxTwapSz,string"`
	MaxSpotIcebergSize              float64    `json:"maxIcebergSz,string"`
	MaxTriggerSize                  float64    `json:"maxTriggerSz,string"`
	MaxStopSize                     float64    `json:"maxStopSz,string"`
}

// DeliveryHistoryDetail holds instrument id and delivery price information detail
type DeliveryHistoryDetail struct {
	Type          string  `json:"type"`
	InstrumentID  string  `json:"insId"`
	DeliveryPrice float64 `json:"px,string"`
}

// DeliveryHistory represents list of delivery history detail items and timestamp information
type DeliveryHistory struct {
	Timestamp time.Time               `json:"ts"`
	Details   []DeliveryHistoryDetail `json:"details"`
}

// OpenInterest Retrieve the total open interest for contracts on OKX.
type OpenInterest struct {
	InstrumentType       asset.Item `json:"instType"`
	InstrumentID         string     `json:"instId"`
	OpenInterest         float64    `json:"oi,string"`
	OpenInterestCurrency float64    `json:"oiCcy,string"`
	Timestamp            time.Time  `json:"ts"`
}

// FundingRateResponse response data for the Funding Rate for an instruction type
type FundingRateResponse struct {
	FundingRate     float64    `json:"fundingRate"`
	FundingTime     time.Time  `json:"fundingTime"`
	InstrumentID    string     `json:"instId"`
	InstrumentType  asset.Item `json:"instType"`
	NextFundingRate float64    `json:"nextFundingRate"`
	NextFundingTime time.Time  `json:"nextFundingTime"`
}

// LimitPriceResponse hold an information for
type LimitPriceResponse struct {
	InstrumentType asset.Item `json:"instType"`
	InstID         string     `json:"instId"`
	BuyLimit       float64    `json:"buyLmt,string"`
	SellLimit      float64    `json:"sellLmt,string"`
	Timestamp      time.Time  `json:"ts"`
}

// OptionMarketDataResponse holds response data for option market data
type OptionMarketDataResponse struct {
	InstrumentType asset.Item `json:"instType"`
	InstrumentID   string     `json:"instId"`
	Underlying     string     `json:"uly"`
	Delta          float64    `json:"delta,string"`
	Gamma          float64    `json:"gamma,string"`
	Theta          float64    `json:"theta,string"`
	Vega           float64    `json:"vega,string"`
	DeltaBS        float64    `json:"deltaBS,string"`
	GammaBS        float64    `json:"gammaBS,string"`
	ThetaBS        float64    `json:"thetaBS,string"`
	VegaBS         float64    `json:"vegaBS,string"`
	RealVol        string     `json:"realVol"`
	BidVolatility  string     `json:"bidVol"`
	AskVolatility  float64    `json:"askVol,string"`
	MarkVolitility float64    `json:"markVol,string"`
	Leverage       float64    `json:"lever,string"`
	ForwardPrice   string     `json:"fwdPx"`
	Timestamp      time.Time  `json:"ts"`
}

// DeliveryEstimatedPrice holds an estimated delivery or exercise price response.
type DeliveryEstimatedPrice struct {
	InstrumentType         asset.Item `json:"instType"`
	InstrumentID           string     `json:"instId"`
	EstimatedDeliveryPrice string     `json:"settlePx"`
	Timestamp              time.Time  `json:"ts"`
}

// DiscountRate represents the discount rate amount, currency, and other discount related informations.
type DiscountRate struct {
	Amount            string                 `json:"amt"`
	Currency          string                 `json:"ccy"`
	DiscountInfo      []DiscountRateInfoItem `json:"discountInfo"`
	DiscountRateLevel string                 `json:"discountLv"`
}

// DiscountRateInfoItem represents discount info list item for discount rate response
type DiscountRateInfoItem struct {
	DiscountRate string  `json:"discountRate"`
	MaxAmount    float64 `json:"maxAmt"`
	MinAmount    float64 `json:"minAmt"`
}

// ServerTime returning  the server time instance.
type ServerTime struct {
	Timestamp time.Time `json:"ts"`
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
	InstrumentType asset.Item                   `json:"instType"`
	TotalLoss      string                       `json:"totalLoss"`
	Underlying     string                       `json:"uly"`
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

// MarkPrice represents a mark price information for a single instrument id
type MarkPrice struct {
	InstrumentType asset.Item `json:"instType"`
	InstrumentID   string     `json:"instId"`
	MarkPrice      string     `json:"markPx"`
	Timestamp      time.Time  `json:"ts"`
}

// PositionTiers represents position tier detailed information.
type PositionTiers struct {
	BaseMaxLoan                   string  `json:"baseMaxLoan"`
	InitialMarginRequirement      string  `json:"imr"`
	InstrumentID                  string  `json:"instId"`
	MaximumLeverage               string  `json:"maxLever"`
	MaximumSize                   float64 `json:"maxSz,string"`
	MinSize                       float64 `json:"minSz,string"`
	MaintainanceMarginRequirement string  `json:"mmr"`
	OptionalMarginFactor          string  `json:"optMgnFactor"`
	QuoteMaxLoan                  string  `json:"quoteMaxLoan"`
	Tier                          string  `json:"tier"`
	Underlying                    string  `json:"uly"`
}

// InterestRateLoanQuotaBasic holds the basic Currency, loan,and interest rate informations.
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
	Amount    float64   `json:"amt,string"`
	Balance   float64   `json:"balance,string"`
	Currency  string    `json:"ccy"`
	Timestamp time.Time `json:"ts"`
	Type      string    `json:"type"`
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
	InstrumentID          string  `json:"instId"`
	TradeMode             string  `json:"tdMode"` // cash isolated
	ClientSupplierOrderID string  `json:"clOrdId"`
	Currency              string  `json:"ccy,omitempty"` // Only applicable to cross MARGIN orders in Single-currency margin.
	OrderTag              string  `json:"tag"`
	Side                  string  `json:"side"`
	PositionSide          string  `json:"posSide"`
	OrderType             string  `json:"ordType"`
	QuantityToBuyOrSell   float64 `json:"sz,string"`
	Price                 float64 `json:"px,string"`
	ReduceOnly            bool    `json:"reduceOnly,string,omitempty"`
	QuantityType          string  `json:"tgtCcy,omitempty"` // values base_ccy and quote_ccy

	// Added in the websocket requests
	BanAmend   bool      `json:"banAmend"` // Whether the SPOT Market Order size can be amended by the system.
	ExpiryTime time.Time `json:"expTime"`
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
	CancelOnFail            bool    `json:"cxlOnFail"`
	OrderID                 string  `json:"ordId"`
	ClientSuppliedOrderID   string  `json:"clOrdId"`
	ClientSuppliedRequestID string  `json:"reqId"`
	NewQuantity             float64 `json:"newSz,string"`
	NewPrice                float64 `json:"newPx,string"`
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
	InstrumentType             asset.Item `json:"instType"`
	InstrumentID               string     `json:"instId"`
	Currency                   string     `json:"ccy"`
	OrderID                    string     `json:"ordId"`
	ClientSupplierOrderID      string     `json:"clOrdId"`
	Tag                        string     `json:"tag"`
	ProfitAndLoss              string     `json:"pnl"`
	OrderType                  string     `json:"ordType"`
	Side                       order.Side `json:"side"`
	PositionSide               string     `json:"posSide"`
	TradeMode                  string     `json:"tdMode"`
	TradeID                    string     `json:"tradeId"`
	FillTime                   time.Time  `json:"fillTime"`
	Source                     string     `json:"source,omitempty"`
	State                      string     `json:"state"`
	TakeProfitTriggerPriceType string     `json:"tpTriggerPxType"`
	StopLossTriggerPriceType   string     `json:"slTriggerPxType"`
	StopLossOrdPx              string     `json:"slOrdPx"`
	RebateCurrency             string     `json:"rebateCcy"`
	QuantityType               string     `json:"tgtCcy"`   // base_ccy and quote_ccy
	Category                   string     `json:"category"` // normal, twap, adl, full_liquidation, partial_liquidation, delivery, ddh
	AccumulatedFillSize        float64    `json:"accFillSz,string"`
	FillPrice                  float64    `json:"fillPx,string"`
	FillSize                   float64    `json:"fillSz,string"`
	RebateAmount               float64    `json:"rebate"`
	FeeCurrency                string     `json:"feeCcy"`
	TransactionFee             float64    `json:"fee,string"`
	AveragePrice               float64    `json:"avgPx,string"`
	Leverage                   float64    `json:"lever,string"`
	Price                      float64    `json:"px,string"`
	Size                       float64    `json:"sz,string"`
	TakeProfitTriggerPrice     float64    `json:"tpTriggerPx,string"`
	TakeProfitOrderPrice       float64    `json:"tpOrdPx,string"`
	StopLossTriggerPrice       float64    `json:"slTriggerPx,string"`
	UpdateTime                 time.Time  `json:"uTime"`
	CreationTime               time.Time  `json:"cTime"`
}

// OrderListRequestParams represents order list requesting parameters.
type OrderListRequestParams struct {
	InstrumentType string    `json:"instType"` // SPOT , MARGIN, SWAP, FUTURES , option
	Underlying     string    `json:"uly"`
	InstrumentID   string    `json:"instId"`
	OrderType      string    `json:"orderType"`
	State          string    `json:"state"` // live, partially_filled
	After          time.Time `json:"after"`
	Before         time.Time `json:"before"`
	Limit          int64     `json:"limit"`
}

// OrderHistoryRequestParams holds parameters to request order data history of last 7 days.
type OrderHistoryRequestParams struct {
	OrderListRequestParams
	Category string `json:"category"` // twap, adl, full_liquidation, partial_liquidation, delivery, ddh
}

// PendingOrderItem represents a pending order Item in pending orders list.
type PendingOrderItem struct {
	AccumulatedFillSize        float64    `json:"accFillSz"`
	AveragePrice               float64    `json:"avgPx"`
	CreationTime               time.Time  `json:"cTime"`
	Category                   string     `json:"category"`
	Currency                   string     `json:"ccy"`
	ClientSupplierOrderID      string     `json:"clOrdId"`
	TransactionFee             string     `json:"fee"`
	FeeCurrency                string     `json:"feeCcy"`
	LastFilledPrice            string     `json:"fillPx"`
	LastFilledSize             float64    `json:"fillSz"`
	FillTime                   string     `json:"fillTime"`
	InstrumentID               string     `json:"instId"`
	InstrumentType             asset.Item `json:"instType"`
	Leverage                   float64    `json:"lever"`
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
	Source                     string     `json:"source"`
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
	InstrumentType        asset.Item `json:"instType"`
	InstrumentID          string     `json:"instId"`
	TradeID               string     `json:"tradeId"`
	OrderID               string     `json:"ordId"`
	ClientSuppliedOrderID string     `json:"clOrdId"`
	BillID                string     `json:"billId"`
	Tag                   string     `json:"tag"`
	FillPrice             float64    `json:"fillPx,string"`
	FillSize              float64    `json:"fillSz,string"`
	Side                  string     `json:"side"`
	PositionSide          string     `json:"posSide"`
	ExecType              string     `json:"execType"`
	FeeCurrency           string     `json:"feeCcy"`
	Fee                   string     `json:"fee"`
	Timestamp             time.Time  `json:"ts"`
}

// AlgoOrderParams holds algo order informations.
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

// AlgoOrderResponse holds algo order informations.
type AlgoOrderResponse struct {
	InstrumentType             asset.Item `json:"instType"`
	InstrumentID               string     `json:"instId"`
	OrderID                    string     `json:"ordId"`
	Currency                   string     `json:"ccy"`
	AlgoOrderID                string     `json:"algoId"`
	Quantity                   string     `json:"sz"`
	OrderType                  string     `json:"ordType"`
	Side                       string     `json:"side"`
	PositionSide               string     `json:"posSide"`
	TradeMode                  string     `json:"tdMode"`
	QuantityType               string     `json:"tgtCcy"`
	State                      string     `json:"state"`
	Lever                      string     `json:"lever"`
	TakeProfitTriggerPrice     string     `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string     `json:"tpTriggerPxType"`
	TakeProfitOrdPrice         string     `json:"tpOrdPx"`
	StopLossTriggerPriceType   string     `json:"slTriggerPxType"`
	StopLossTriggerPrice       string     `json:"slTriggerPx"`
	TriggerPrice               string     `json:"triggerPx"`
	TriggerPriceType           string     `json:"triggerPxType"`
	OrdPrice                   string     `json:"ordPx"`
	ActualSize                 string     `json:"actualSz"`
	ActualPrice                string     `json:"actualPx"`
	ActualSide                 string     `json:"actualSide"`
	PriceVar                   string     `json:"pxVar"`
	PriceSpread                string     `json:"pxSpread"`
	PriceLimit                 string     `json:"pxLimit"`
	SizeLimit                  string     `json:"szLimit"`
	TimeInterval               string     `json:"timeInterval"`
	TriggerTime                time.Time  `json:"triggerTime"`
	CallbackRatio              string     `json:"callbackRatio"`
	CallbackSpread             string     `json:"callbackSpread"`
	ActivePrice                string     `json:"activePx"`
	MoveTriggerPrice           string     `json:"moveTriggerPx"`
	CreationTime               time.Time  `json:"cTime"`
}

// CurrencyResponse represents a currency item detail response data.
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
	TotalBal  float64   `json:"totalBal,string"`
	Timestamp time.Time `json:"ts"`
}

// FundingTransferRequestInput represents funding account request input.
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
	SubAccountt    string  `json:"subAcct"`
	To             string  `json:"to"`
	ToInstrumentID string  `json:"toInstId"`
	TransrumentID  string  `json:"transId"`
	Type           int     `json:"type,string"`
}

// AssetBillDetail represents  the billing record
type AssetBillDetail struct {
	BillID         string    `json:"billId"`
	Currency       string    `json:"ccy"`
	ClientID       string    `json:"clientId"`
	BalanceChange  string    `json:"balChg"`
	AccountBalance string    `json:"bal"`
	Type           int       `json:"type,string"`
	Timestamp      time.Time `json:"ts"`
}

// LightningDepositItem for creating an invoice.
type LightningDepositItem struct {
	CreationTime time.Time `json:"cTime"`
	Invoice      string    `json:"invoice"`
}

// CurrencyDepositResponseItem represents the deposit address information item.
type CurrencyDepositResponseItem struct {
	Tag                       string `json:"tag"`
	Chain                     string `json:"chain"`
	ContractAddress           string `json:"ctAddr"`
	Currency                  string `json:"ccy"`
	ToBeneficiaryAccount      string `json:"to"`
	Address                   string `json:"addr"`
	Selected                  bool   `json:"selected"`
	Memo                      string `json:"memo"`
	DepositAddressAttachement string `json:"addrEx"`
	PaymentID                 string `json:"pmtId"`
}

// DepositHistoryResponseItem deposit history response item.
type DepositHistoryResponseItem struct {
	Amount           float64   `json:"amt,string"`
	TransactionID    string    `json:"txId"` // Hash record of the deposit
	Currency         string    `json:"ccy"`
	Chain            string    `json:"chain"`
	From             string    `json:"from"`
	ToDepositAddress string    `json:"to"`
	Timestamp        time.Time `json:"ts"`
	State            int       `json:"state,string"`
	DepositID        string    `json:"depId"`
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
	WithdrawalID string    `json:"wdId"`
	CreationTime time.Time `json:"cTime"`
}

// WithdrawalHistoryResponse represents the withdrawal response history.
type WithdrawalHistoryResponse struct {
	ChainName            string    `json:"chain"`
	WithdrawalFee        float64   `json:"fee,string"`
	Currency             string    `json:"ccy"`
	ClientID             string    `json:"clientId"`
	Amount               float64   `json:"amt,string"`
	TransactionID        string    `json:"txId"` // Hash record of the withdrawal. This parameter will not be returned for internal transfers.
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
	Amount    float64   `json:"amt,string"`
	Currency  string    `json:"ccy"`
	Rate      float64   `json:"rate,string"`
	Timestamp time.Time `json:"ts"`
}

// ConvertCurrency represents currency conversion detailed data.
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

// Account holds currency account balance and related information
type Account struct {
	AdjEq       string          `json:"adjEq"`
	Details     []AccountDetail `json:"details"`
	Imr         string          `json:"imr"` // Frozen equity for open positions and pending orders in USD level Applicable to Multi-currency margin and Portfolio margin
	IsoEq       string          `json:"isoEq"`
	MgnRatio    string          `json:"mgnRatio"`
	Mmr         string          `json:"mmr"` // Maintenance margin requirement in USD level Applicable to Multi-currency margin and Portfolio margin
	NotionalUsd string          `json:"notionalUsd"`
	OrdFroz     string          `json:"ordFroz"` // Margin frozen for pending orders in USD level Applicable to Multi-currency margin and Portfolio margin
	TotalEquity string          `json:"totalEq"` // Total Equity in USD level
	UpdateTime  time.Time       `json:"uTime"`   // UpdateTime
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
	AutoDeleverging               string     `json:"adl"`      // Auto-deleveraging (ADL) indicator Divided into 5 levels, from 1 to 5, the smaller the number, the weaker the adl intensity.
	AvailablePosition             string     `json:"availPos"` // Position that can be closed Only applicable to MARGIN, FUTURES/SWAP in the long-short mode, OPTION in Simple and isolated OPTION in margin Account.
	AveragePrice                  string     `json:"avgPx"`
	CreationTime                  time.Time  `json:"cTime"`
	Currency                      string     `json:"ccy"`
	DeltaBS                       string     `json:"deltaBS"` // delta：Black-Scholes Greeks in dollars,only applicable to OPTION
	DeltaPA                       string     `json:"deltaPA"` // delta：Greeks in coins,only applicable to OPTION
	GammaBS                       string     `json:"gammaBS"` // gamma：Black-Scholes Greeks in dollars,only applicable to OPTION
	GammaPA                       string     `json:"gammaPA"` // gamma：Greeks in coins,only applicable to OPTION
	InitionMarginRequirement      string     `json:"imr"`     // Initial margin requirement, only applicable to cross.
	InstrumentID                  string     `json:"instId"`
	InstrumentType                asset.Item `json:"instType"`
	Interest                      string     `json:"interest"`
	USDPrice                      string     `json:"usdPx"`
	LastTradePrice                string     `json:"last"`
	Leverage                      string     `json:"lever"`   // Leverage, not applicable to OPTION seller
	Liabilities                   string     `json:"liab"`    // Liabilities, only applicable to MARGIN.
	LiabilitiesCurrency           string     `json:"liabCcy"` // Liabilities currency, only applicable to MARGIN.
	LiquidationPrice              string     `json:"liqPx"`   // Estimated liquidation price Not applicable to OPTION
	MarkPx                        string     `json:"markPx"`
	Margin                        string     `json:"margin"`
	MgnMode                       string     `json:"mgnMode"`
	MgnRatio                      string     `json:"mgnRatio"`
	MaintainanceMarginRequirement string     `json:"mmr"`         // Maintenance margin requirement in USD level Applicable to Multi-currency margin and Portfolio margin
	NotionalUsd                   string     `json:"notionalUsd"` // Quality of Positions -- usd
	OptionValue                   string     `json:"optVal"`      // Option Value, only application to position.
	QuantityOfPosition            string     `json:"pos"`         // Quantity of positions,In the mode of autonomous transfer from position to position, after the deposit is transferred, a position with pos of 0 will be generated
	PositionCurrency              string     `json:"posCcy"`
	PositionID                    string     `json:"posId"`
	PositionSide                  string     `json:"posSide"`
	ThetaBS                       string     `json:"thetaBS"` // theta：Black-Scholes Greeks in dollars,only applicable to OPTION
	ThetaPA                       string     `json:"thetaPA"` // theta：Greeks in coins,only applicable to OPTION
	TradeID                       string     `json:"tradeId"`
	UpdatedTime                   time.Time  `json:"uTime"`                     // Latest time position was adjusted,
	Upl                           float64    `json:"upl,string,omitempty"`      // Unrealized profit and loss
	UPLRatio                      float64    `json:"uplRatio,string,omitempty"` // Unrealized profit and loss ratio
	VegaBS                        string     `json:"vegaBS"`                    // vega：Black-Scholes Greeks in dollars,only applicable to OPTION
	VegaPA                        string     `json:"vegaPA"`                    // vega：Greeks in coins,only applicable to OPTION

	// PushTime added feature in the websocket push data.

	PushTime time.Time `json:"pTime"` // The time when the account position data is pushed.
}

// AccountPositionHistory hold account position history.
type AccountPositionHistory struct {
	CreationTime       time.Time  `json:"cTime"`
	Currency           string     `json:"ccy"`
	CloseAveragePrice  float64    `json:"closeAvgPx,string,omitempty"`
	CloseTotalPosition float64    `json:"closeTotalPos,string,omitempty"`
	InstrumentID       string     `json:"instId"`
	InstrumentType     asset.Item `json:"instType"`
	Leverage           string     `json:"lever"`
	ManagementMode     string     `json:"mgnMode"`
	OpenAveragePrice   string     `json:"openAvgPx"`
	OpenMaxPosition    string     `json:"openMaxPos"`
	ProfitAndLoss      float64    `json:"pnl,string,omitempty"`
	ProfitAndLossRatio float64    `json:"pnlRatio,string,omitempty"`
	PositionID         string     `json:"posId"`
	PositionSide       string     `json:"posSide"`
	TriggerPrice       string     `json:"triggerPx"`
	Type               string     `json:"type"`
	UpdateTime         time.Time  `json:"uTime"`
	Underlying         string     `json:"uly"`
}

// AccountBalanceData represents currency account balance.
type AccountBalanceData struct {
	Currency       string `json:"ccy"`
	DiscountEquity string `json:"disEq"` // discount equity of the currency in USD level.
	Equity         string `json:"eq"`    // Equity of the currency
}

// PositionData holds account position data.
type PositionData struct {
	BaseBal          string     `json:"baseBal"`
	Currency         string     `json:"ccy"`
	InstrumentID     string     `json:"instId"`
	InstrumentType   asset.Item `json:"instType"`
	MamagementMode   string     `json:"mgnMode"`
	NotionalCurrency string     `json:"notionalCcy"`
	NotionalUsd      string     `json:"notionalUsd"`
	Position         string     `json:"pos"`
	PositionedCcy    string     `json:"posCcy"`
	PositionedID     string     `json:"posId"`
	PositionedSide   string     `json:"posSide"`
	QuoteBalance     string     `json:"quoteBal"`
}

// AccountAndPositionRisk holds information.
type AccountAndPositionRisk struct {
	AdjEq               string               `json:"adjEq"`
	AccountBalanceDatas []AccountBalanceData `json:"balData"`
	PosData             []PositionData       `json:"posData"`
	Timestamp           time.Time            `json:"ts"`
}

// BillsDetailQueryParameter represents bills detail query parameter
type BillsDetailQueryParameter struct {
	InstrumentType string // Instrument type "SPOT" "MARGIN" "SWAP" "FUTURES" "OPTION"
	Currency       string
	MarginMode     string // Margin mode "isolated" "cross"
	ContractType   string // Contract type "linear" & "inverse" Only applicable to FUTURES/SWAP
	BillType       uint   // Bill type 1: Transfer 2: Trade 3: Delivery 4: Auto token conversion 5: Liquidation 6: Margin transfer 7: Interest deduction 8: Funding fee 9: ADL 10: Clawback 11: System token conversion 12: Strategy transfer 13: ddh
	BillSubType    int    // allowed bill substype values are [ 1,2,3,4,5,6,9,11,12,14,160,161,162,110,111,118,119,100,101,102,103,104,105,106,110,125,126,127,128,131,132,170,171,172,112,113,117,173,174,200,201,202,203 ], link: https://www.okx.com/docs-v5/en/#rest-api-account-get-bills-details-last-7-days
	After          string
	Before         string
	BeginTime      time.Time
	EndTime        time.Time
	Limit          int64
}

// BillsDetailResponse represents account bills informaiton.
type BillsDetailResponse struct {
	Balance                    string     `json:"bal"`
	BalanceChange              string     `json:"balChg"`
	BillID                     string     `json:"billId"`
	Currency                   string     `json:"ccy"`
	ExecType                   string     `json:"execType"` // Order flow type, T：taker M：maker
	Fee                        string     `json:"fee"`      // Fee Negative number represents the user transaction fee charged by the platform. Positive number represents rebate.
	From                       string     `json:"from"`     // The remitting account 6: FUNDING 18: Trading account When bill type is not transfer, the field returns "".
	InstrumentID               string     `json:"instId"`
	InstrumentType             asset.Item `json:"instType"`
	ManegementMode             string     `json:"mgnMode"`
	Notes                      string     `json:"notes"` // notes When bill type is not transfer, the field returns "".
	OrderID                    string     `json:"ordId"`
	ProfitAndLoss              string     `json:"pnl"`
	PositionLevelBalance       string     `json:"posBal"`
	PositionLevelBalanceChange string     `json:"posBalChg"`
	SubType                    string     `json:"subType"`
	Size                       string     `json:"sz"`
	To                         string     `json:"to"`
	Timestamp                  time.Time  `json:"ts"`
	Type                       string     `json:"type"`
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

// SetLeverageInput represents set leverate request input
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
	InstrumentID      string  `json:"instId"`
	PositionSide      string  `json:"posSide"`
	Type              string  `json:"type"`
	Amount            float64 `json:"amt,string"`
	Currency          string  `json:"ccy"`
	AutoLoadTransffer bool    `json:"auto"`
	LoadTransffer     bool    `json:"loanTrans"`
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
	Category         string     `json:"category"`
	DeliveryFeeRate  string     `json:"delivery"`
	Exercise         string     `json:"exercise"`
	InstrumentType   asset.Item `json:"instType"`
	FeeRateLevel     string     `json:"level"`
	FeeRateMaker     string     `json:"maker"`
	FeeRateMakerUSDT string     `json:"makerU"`
	FeeRateTaker     string     `json:"taker"`
	FeeRateTakerUSDT string     `json:"takerU"`
	Timestamp        time.Time  `json:"ts"`
}

// InterestAccruedData represents interest rate accrued response
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
	IsTheAccountAtRisk bool          `json:"atRisk"`
	AtRiskIdx          []interface{} `json:"atRiskIdx"` // derivatives risk unit list
	AtRiskMgn          []interface{} `json:"atRiskMgn"` // margin risk unit list
	Timestamp          time.Time     `json:"ts"`
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
	Currency   string    `json:"ccy"`
	TradedLoan string    `json:"tradedLoan"`
	Timestamp  time.Time `json:"ts"`
	Type       string    `json:"type"`
	UsedLoan   string    `json:"usedLoan"`
}

// BorrowInterestAndLimitResponse represents borrow interest and limit rate for different loan type.
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
	ExtreamMarketMove            string                `json:"mr6"`
	TransactionCostAndSlippage   string                `json:"mr7"`
	PositionData                 []PositionBuilderData `json:"posData"` // List of positions
	RiskUnit                     string                `json:"riskUnit"`
	Timestamp                    time.Time             `json:"ts"`
}

// PositionBuilderData represent a position item.
type PositionBuilderData struct {
	Delta              string     `json:"delta"`
	Gamma              string     `json:"gamma"`
	InstrumentID       string     `json:"instId"`
	InstrumentType     asset.Item `json:"instType"`
	NotionalUsd        string     `json:"notionalUsd"` // Quantity of positions usd
	QuantityOfPosition string     `json:"pos"`         // Quantity of positions
	Theta              string     `json:"theta"`       // Sensitivity of option price to remaining maturity
	Vega               string     `json:"vega"`        // Sensitivity of option price to implied volatility
}

// GreeksItem represents greeks response
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
	RfqID               []string `json:"rfqId"`
	ClientSuppliedRFQID []string `json:"clRfqId"`
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
	Timestamp time.Time `json:"ts"`
}

// ExecuteQuoteParams represents Execute quote request params
type ExecuteQuoteParams struct {
	RfqID   string `json:"rfqId"`
	QuoteID string `json:"quoteId"`
}

// ExecuteQuoteResponse represents execute quote response.
type ExecuteQuoteResponse struct {
	BlockTradedID         string     `json:"blockTdId"`
	RfqID                 string     `json:"rfqId"`
	ClientSuppliedRfqID   string     `json:"clRfqId"`
	QuoteID               string     `json:"quoteId"`
	ClientSuppliedQuoteID string     `json:"clQuoteId"`
	TraderCode            string     `json:"tTraderCode"`
	MakerTraderCode       string     `json:"mTraderCode"`
	CreationTime          time.Time  `json:"cTime"`
	Legs                  []OrderLeg `json:"legs"`
}

// OrderLeg represents legs information for both websocket and REST available Quote informations.
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

// CancelQuoteRequestParams represents cancel quote request params
type CancelQuoteRequestParams struct {
	QuoteID               string `json:"quoteId"`
	ClientSuppliedQuoteID string `json:"clQuoteId"`
}

// CancelQuotesRequestParams represents cancel multiple quotes request params
type CancelQuotesRequestParams struct {
	QuoteIDs               []string `json:"quoteId"`
	ClientSuppliedQuoteIDs []string `json:"clQuoteId"`
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
	BeginingID          string
	EndID               string
	Limit               int64
}

// RFQResponse RFQ response detail.
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
	BlockTradeID string        `json:"blockTdId"`
	CreationTime time.Time     `json:"cTime"`
	Legs         []RFQTradeLeg `json:"legs"`
}

// SubaccountInfo represents subaccount information detail.
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

// SubaccountBalanceDetail represents subaccount balance detail
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

// SubaccountBalanceResponse represents subaccount balance response
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

// SubaccountBillItem represents subaccount balance bill item
type SubaccountBillItem struct {
	BillID                 string    `json:"billId"`
	Type                   string    `json:"type"`
	AccountCurrencyBalance string    `json:"ccy"`
	Amount                 string    `json:"amt"`
	SubAccount             string    `json:"subAcct"`
	Timestamp              time.Time `json:"ts"`
}

// TransferIDInfo represents master account transfer between subaccount.
type TransferIDInfo struct {
	TransferID string `json:"transId"`
}

// PermissingOfTransfer represents subaccount transfer information and it's permission.
type PermissingOfTransfer struct {
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
	BaseSize  float64 `json:"baseSz,string"`  // nvest amount for base currency Either "instId" or "ccy" is required

	// Contract Grid Order
	BasePosition bool    `json:"basePos"` // Wether or not open a position when strategy actives Default is false Neutral contract grid should omit the parameter
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
	ActualLever               string     `json:"actualLever"`
	AlgoID                    string     `json:"algoId"`
	AlgoOrderType             string     `json:"algoOrdType"`
	ArbitrageNumber           string     `json:"arbitrageNum"`
	BasePosition              bool       `json:"basePos"`
	BaseSize                  string     `json:"baseSz"`
	CancelType                string     `json:"cancelType"`
	Direction                 string     `json:"direction"`
	FloatProfit               string     `json:"floatProfit"`
	GridQuantity              string     `json:"gridNum"`
	GridProfit                string     `json:"gridProfit"`
	InstrumentID              string     `json:"instId"`
	InstrumentType            asset.Item `json:"instType"`
	Investment                string     `json:"investment"`
	Leverage                  string     `json:"lever"`
	EstimatedLiquidationPrice string     `json:"liqPx"`
	MaximumPrice              string     `json:"maxPx"`
	MinimumPrice              string     `json:"minPx"`
	ProfitAndLossRatio        string     `json:"pnlRatio"`
	QuoteSize                 string     `json:"quoteSz"`
	RunType                   string     `json:"runType"`
	StopLossTriggerPx         string     `json:"slTriggerPx"`
	State                     string     `json:"state"`
	StopResult                string     `json:"stopResult,omitempty"`
	StopType                  string     `json:"stopType"`
	Size                      string     `json:"sz"`
	Tag                       string     `json:"tag"`
	TotalProfitAndLoss        string     `json:"totalPnl"`
	TakeProfitTriggerPrice    string     `json:"tpTriggerPx"`
	CreationTime              time.Time  `json:"cTime"`
	UpdateTime                time.Time  `json:"uTime"`
	Underlying                string     `json:"uly"`

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
	ActualLeverage      string     `json:"actualLever"`
	AlgoID              string     `json:"algoId"`
	AlgoOrderType       string     `json:"algoOrdType"`
	AnnualizedRate      string     `json:"annualizedRate"`
	ArbitrageNum        string     `json:"arbitrageNum"`
	BasePosition        bool       `json:"basePos"`
	BaseSize            string     `json:"baseSz"`
	CancelType          string     `json:"cancelType"`
	CurBaseSz           string     `json:"curBaseSz"`
	CurQuoteSz          string     `json:"curQuoteSz"`
	Direction           string     `json:"direction"`
	EquityOfStrength    string     `json:"eq"`
	FloatProfit         string     `json:"floatProfit"`
	GridQuantity        string     `json:"gridNum"`
	GridProfit          string     `json:"gridProfit"`
	InstrumentID        string     `json:"instId"`
	InstrumentType      asset.Item `json:"instType"`
	Investment          string     `json:"investment"`
	Leverage            string     `json:"lever"`
	LiquidationPx       string     `json:"liqPx"`
	MaximumPrice        string     `json:"maxPx"`
	MinimumPrice        string     `json:"minPx"`
	PerMaxProfitRate    string     `json:"perMaxProfitRate"`
	PerMinProfitRate    string     `json:"perMinProfitRate"`
	ProfitAndLossRatio  string     `json:"pnlRatio"`
	Profit              string     `json:"profit"`
	QuoteSize           string     `json:"quoteSz"`
	RunType             string     `json:"runType"`
	Runpx               string     `json:"runpx"`
	SingleAmount        string     `json:"singleAmt"`
	StopLossTriggerPx   string     `json:"slTriggerPx"`
	State               string     `json:"state"`
	StopResult          string     `json:"stopResult"`
	StopType            string     `json:"stopType"`
	Size                string     `json:"sz"`
	Tag                 string     `json:"tag"`
	TotalAnnualizedRate string     `json:"totalAnnualizedRate"`
	TotalProfitAndLoss  string     `json:"totalPnl"`
	TakeProfitTriggerPx string     `json:"tpTriggerPx"`
	TradeNum            string     `json:"tradeNum"`
	UpdateTime          time.Time  `json:"uTime"`
	CreationTime        time.Time  `json:"cTime"`
}

// AlgoOrderPosition represents algo order position detailed data.
type AlgoOrderPosition struct {
	AutoDecreasingLine            string     `json:"adl"`
	AlgoID                        string     `json:"algoId"`
	AveragePrice                  string     `json:"avgPx"`
	Currency                      string     `json:"ccy"`
	InitialMarginRequirement      string     `json:"imr"`
	InstrumentID                  string     `json:"instId"`
	InstrumentType                asset.Item `json:"instType"`
	LastTradedPrice               string     `json:"last"`
	Leverage                      string     `json:"lever"`
	LiquidationPrice              string     `json:"liqPx"`
	MarkPrice                     string     `json:"markPx"`
	MarginMode                    string     `json:"mgnMode"`
	MarginRatio                   string     `json:"mgnRatio"`
	MaintainanceMarginRequirement string     `json:"mmr"`
	NotionalUSD                   string     `json:"notionalUsd"`
	QuantityPosition              string     `json:"pos"`
	PositionSide                  string     `json:"posSide"`
	UnrealizedProfitAndLoss       string     `json:"upl"`
	UnrealizedProfitAndLossRatio  string     `json:"uplRatio"`
	UpdateTime                    time.Time  `json:"uTime"`
	CreationTime                  time.Time  `json:"cTime"`
}

// AlgoOrderWithdrawalProfit algo withdrawal order profit info.
type AlgoOrderWithdrawalProfit struct {
	AlgoID         string `json:"algoId"`
	WithdrawProfit string `json:"profit"`
}

// SystemStatusResponse represents the system status and other details.
type SystemStatusResponse struct {
	Title               string    `json:"title"`
	State               string    `json:"state"`
	Begin               time.Time `json:"begin"` // Begin time of system maintenance,
	End                 time.Time `json:"end"`   // Time of resuming trading totally.
	Href                string    `json:"href"`  // Hyperlink for system maintenance details
	ServiceType         string    `json:"serviceType"`
	System              string    `json:"system"`
	ScheduleDescription string    `json:"scheDesc"`

	// PushTime timestamp information when the data is pushed
	PushTime time.Time `json:"ts"`
}

// BlockTicker holds block trading information.
type BlockTicker struct {
	InstrumentType           asset.Item `json:"instType"`
	InstrumentID             string     `json:"instId"`
	TradingVolumeInCCY24Hour float64    `json:"volCcy24h,string"`
	TradingVolumeInUSD24Hour float64    `json:"vol24h,string"`
	Timestamp                time.Time  `json:"ts"`
}

// BlockTrade represents a block trade.
type BlockTrade struct {
	InstrumentID string     `json:"instId"`
	TradeID      string     `json:"tradeId"`
	Price        float64    `json:"px,string"`
	Size         float64    `json:"sz,string"`
	Side         order.Side `json:"side"`
	Timestamp    time.Time  `json:"ts"`
}

// UnitConvertResponse unit convert response.
type UnitConvertResponse struct {
	InstrumentID string              `json:"instId"`
	Price        float64             `json:"px,string"`
	Size         float64             `json:"sz,string"`
	ConvertType  CurrencyConvertType `json:"type"`
	Unit         string              `json:"unit"`
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

// WSSubscriptionInformations websocket subscription and unsubscription operation inputs.
type WSSubscriptionInformations struct {
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
	ID        string        `json:"id,omitempty"`
	Operation string        `json:"op,omitempty"`
	Data      []interface{} `json:"data,omitempty"`
}

// copyToSubscriptionResponse returns a *SubscriptionOperationResponse instance.
func (w *wsIncomingData) copyToSubscriptionResponse() *SubscriptionOperationResponse {
	return &SubscriptionOperationResponse{
		Event:    w.Event,
		Argument: &w.Argument,
		Code:     w.Code,
		Msg:      w.Msg,
	}
}

// copyToPlaceOrderResponse returns WSPlaceOrderResponse struct instance
func (w *wsIncomingData) copyToPlaceOrderResponse() (*WSOrderResponse, error) {
	if len(w.Data) == 0 {
		return nil, errEmptyPlaceOrderResponse
	}
	var placeOrds []OrderData
	value, err := json.Marshal(w.Data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(value, &placeOrds)
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
	InstrumentID string    `json:"instId"`
	TradeID      string    `json:"tradeId"`
	Price        float64   `json:"px,string"`
	Size         float64   `json:"sz,string"`
	Side         string    `json:"side"`
	Timestamp    time.Time `json:"ts"`
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

// WsCancelOrderInput websocker cancel order request
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

// WsAmendOrderResponse holds websocket response Amendment request
type WsAmendOrderResponse struct {
	ID        string      `json:"id"`
	Operation string      `json:"op"`
	Data      []OrderData `json:"data"`
	Code      string      `json:"code"`
	Msg       string      `json:"msg"`
}

// SubscriptionOperationInput represents the account channel input datas
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

// WsPositionResponse represents pushd position data through the websocket channel.
type WsPositionResponse struct {
	Argument  SubscriptionInfo  `json:"arg"`
	Arguments []AccountPosition `json:"data"`
}

// PositionDataDetail position data information for the websocket push data
type PositionDataDetail struct {
	PositionID       string    `json:"posId"`
	TradeID          string    `json:"tradeId"`
	InstrumentID     string    `json:"instId"`
	InstrumentType   string    `json:"instType"`
	MarginMode       string    `json:"mgnMode"`
	PositionSide     string    `json:"posSide"`
	Position         string    `json:"pos"`
	Currency         string    `json:"ccy"`
	PositionCurrency string    `json:"posCcy"`
	AveragePrice     string    `json:"avgPx"`
	UpdateTime       time.Time `json:"uTIme"`
}

// BalanceData represents currency and it's Cash balance with the update time.
type BalanceData struct {
	Currency    string    `json:"ccy"`
	CashBalance string    `json:"cashBal"`
	UpdateTime  time.Time `json:"uTime"`
}

// BalanceAndPositionData represents balance and position data with the push time.
type BalanceAndPositionData struct {
	PushTime      time.Time            `json:"pTime"`
	EventType     string               `json:"eventType"`
	BalanceData   []BalanceData        `json:"balData"`
	PositionDatas []PositionDataDetail `json:"posData"`
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
	InstrumentType             string    `json:"instType"`
	InstrumentID               string    `json:"instId"`
	OrderID                    string    `json:"ordId"`
	Currency                   string    `json:"ccy"`
	AlgoID                     string    `json:"algoId"`
	Price                      string    `json:"px"`
	Size                       string    `json:"sz"`
	TradeMode                  string    `json:"tdMode"`
	TargetCurrency             string    `json:"tgtCcy"`
	NotionalUsd                string    `json:"notionalUsd"`
	OrderType                  string    `json:"ordType"`
	Side                       string    `json:"side"`
	PositionSide               string    `json:"posSide"`
	State                      string    `json:"state"`
	Leverage                   string    `json:"lever"`
	TakeProfitTriggerPrice     string    `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string    `json:"tpTriggerPxType"`
	TakeProfitOrdPrice         string    `json:"tpOrdPx"`
	StopLossTriggerPrice       string    `json:"slTriggerPx"`
	StopLossTriggerPriceType   string    `json:"slTriggerPxType"`
	TriggerPrice               string    `json:"triggerPx"`
	TriggerPriceType           string    `json:"triggerPxType"`
	OrderPrice                 float64   `json:"ordPx,string"`
	ActualSize                 string    `json:"actualSz"`
	ActualPrice                string    `json:"actualPx"`
	Tag                        string    `json:"tag"`
	ActualSide                 string    `json:"actualSide"`
	TriggerTime                time.Time `json:"triggerTime"`
	CreationTime               time.Time `json:"cTime"`
}

// WsAdvancedAlgoOrder advanced algo order response.
type WsAdvancedAlgoOrder struct {
	Argument SubscriptionInfo            `json:"arg"`
	Data     []WsAdvancedAlgoOrderDetail `json:"data"`
}

// WsAdvancedAlgoOrderDetail advanced algo order response pushed through the websocket conn
type WsAdvancedAlgoOrderDetail struct {
	ActualPrice            string    `json:"actualPx"`
	ActualSide             string    `json:"actualSide"`
	ActualSize             string    `json:"actualSz"`
	AlgoID                 string    `json:"algoId"`
	Currency               string    `json:"ccy"`
	Count                  string    `json:"count"`
	InstrumentID           string    `json:"instId"`
	InstrumentType         string    `json:"instType"`
	Leverage               string    `json:"lever"`
	NotionalUsd            string    `json:"notionalUsd"`
	OrderPrice             string    `json:"ordPx"`
	OrdType                string    `json:"ordType"`
	PositionSide           string    `json:"posSide"`
	PriceLimit             string    `json:"pxLimit"`
	PriceSpread            string    `json:"pxSpread"`
	PriceVariation         string    `json:"pxVar"`
	Side                   string    `json:"side"`
	StopLossOrderPrice     string    `json:"slOrdPx"`
	StopLossTriggerPrice   string    `json:"slTriggerPx"`
	State                  string    `json:"state"`
	Size                   string    `json:"sz"`
	SizeLimit              string    `json:"szLimit"`
	TradeMode              string    `json:"tdMode"`
	TimeInterval           string    `json:"timeInterval"`
	TakeProfitOrderPrice   string    `json:"tpOrdPx"`
	TakeProfitTriggerPrice string    `json:"tpTriggerPx"`
	Tag                    string    `json:"tag"`
	TriggerPrice           string    `json:"triggerPx"`
	CallbackRatio          string    `json:"callbackRatio"`
	CallbackSpread         string    `json:"callbackSpread"`
	ActivePrice            string    `json:"activePx"`
	MoveTriggerPrice       string    `json:"moveTriggerPx"`
	CreationTime           time.Time `json:"cTime"`
	PushTime               time.Time `json:"pTime"`
	TriggerTime            time.Time `json:"triggerTime"`
}

// WsGreeks greeks push data with the subcription info through websocket channel
type WsGreeks struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []WsGreekData    `json:"data"`
}

// WsGreekData greeks push data through websocket channel
type WsGreekData struct {
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
	ValidUntil            time.Time  `json:"validUntil"`
	UpdatedTime           time.Time  `json:"uTime"`
	CreationTime          time.Time  `json:"cTime"`
	Legs                  []OrderLeg `json:"legs"`
	QuoteID               string     `json:"quoteId"`
	RfqID                 string     `json:"rfqId"`
	TraderCode            string     `json:"traderCode"`
	QuoteSide             string     `json:"quoteSide"`
	State                 string     `json:"state"`
	ClientSuppliedQuoteID string     `json:"clQuoteId"`
}

// WsStructureBlocTrade represents websocket push data for "struc-block-trades" subscription
type WsStructureBlocTrade struct {
	Argument SubscriptionInfo      `json:"arg"`
	Data     []WsBlocTradeResponse `json:"data"`
}

// WsBlocTradeResponse represents a structure bloc order information
type WsBlocTradeResponse struct {
	CreationTime          time.Time  `json:"cTime"`
	RfqID                 string     `json:"rfqId"`
	ClientSuppliedRfqID   string     `json:"clRfqId"`
	QuoteID               string     `json:"quoteId"`
	ClientSuppliedQuoteID string     `json:"clQuoteId"`
	BlockTradeID          string     `json:"blockTdId"`
	TakerTraderCode       string     `json:"tTraderCode"`
	MakerTraderCode       string     `json:"mTraderCode"`
	Legs                  []OrderLeg `json:"legs"`
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
	StopType               string    `json:"stopType"`
	TotalAnnualizedRate    string    `json:"totalAnnualizedRate"`
	TotalProfitAndLoss     string    `json:"totalPnl"`
	TakeProfitTriggerPrice string    `json:"tpTriggerPx"`
	TradeNum               string    `json:"tradeNum"`
	TriggerTime            time.Time `json:"triggerTime"`
	CreationTime           time.Time `json:"cTime"`
	PushTime               time.Time `json:"pTime"`
	UpdateTime             time.Time `json:"uTime"`
}

// WsContractGridAlgoOrder represents websocket push data for "grid-orders-contract" subscription
type WsContractGridAlgoOrder struct {
	Argument SubscriptionInfo        `json:"arg"`
	Data     []ContractGridAlgoOrder `json:"data"`
}

// ContractGridAlgoOrder represents contrat grid algo order
type ContractGridAlgoOrder struct {
	ActualLever            string    `json:"actualLever"`
	AlgoID                 string    `json:"algoId"`
	AlgoOrderType          string    `json:"algoOrdType"`
	AnnualizedRate         string    `json:"annualizedRate"`
	ArbitrageNumber        string    `json:"arbitrageNum"`
	BasePosition           bool      `json:"basePos"`
	CancelType             string    `json:"cancelType"`
	Direction              string    `json:"direction"`
	Eq                     string    `json:"eq"`
	FloatProfit            string    `json:"floatProfit"`
	GridQuantitty          string    `json:"gridNum"`
	GridProfit             string    `json:"gridProfit"`
	InstrumentID           string    `json:"instId"`
	InstrumentType         string    `json:"instType"`
	Investment             string    `json:"investment"`
	Leverage               string    `json:"lever"`
	LiqPrice               string    `json:"liqPx"`
	MaxPrice               string    `json:"maxPx"`
	MinPrice               string    `json:"minPx"`
	CreationTime           time.Time `json:"cTime"`
	PushTime               time.Time `json:"pTime"`
	PerMaxProfitRate       string    `json:"perMaxProfitRate"`
	PerMinProfitRate       string    `json:"perMinProfitRate"`
	ProfitAndLossRatio     string    `json:"pnlRatio"`
	RunPrice               string    `json:"runPx"`
	RunType                string    `json:"runType"`
	SingleAmount           string    `json:"singleAmt"`
	SlTriggerPx            string    `json:"slTriggerPx"`
	State                  string    `json:"state"`
	StopType               string    `json:"stopType"`
	Size                   string    `json:"sz"`
	Tag                    string    `json:"tag"`
	TotalAnnualizedRate    string    `json:"totalAnnualizedRate"`
	TotalProfitAndLoss     string    `json:"totalPnl"`
	TakeProfitTriggerPrice string    `json:"tpTriggerPx"`
	TradeNumber            string    `json:"tradeNum"`
	TriggerTime            string    `json:"triggerTime"`
	UpdateTime             string    `json:"uTime"`
	Underlying             string    `json:"uly"`
}

// WsGridPosition represents websocket push data for "grid-positions" subscription
type WsGridPosition struct {
	Argument SubscriptionInfo   `json:"arg"`
	Data     []GridPositionData `json:"data"`
}

// GridPositionData represents a position data
type GridPositionData struct {
	AutoDeleverging               string    `json:"adl"`
	AlgoID                        string    `json:"algoId"`
	AveragePrice                  string    `json:"avgPx"`
	Currency                      string    `json:"ccy"`
	InitialMarginRequirement      string    `json:"imr"`
	InstrumentID                  string    `json:"instId"`
	InstrumentType                string    `json:"instType"`
	Last                          string    `json:"last"`
	Leverage                      string    `json:"lever"`
	LiquidationPrice              string    `json:"liqPx"`
	MarkPrice                     string    `json:"markPx"`
	MarginMode                    string    `json:"mgnMode"`
	MarginRatio                   string    `json:"mgnRatio"`
	MaintainanceMarginRequirement string    `json:"mmr"`
	NotionalUsd                   string    `json:"notionalUsd"`
	QuantityOfPositions           string    `json:"pos"`
	PositionSide                  string    `json:"posSide"`
	UnrealizedProfitAndLoss       string    `json:"upl"`
	UnrealizedProfitAndLossRatio  string    `json:"uplRatio"`
	PushTime                      time.Time `json:"pTime"`
	UpdateTime                    time.Time `json:"uTime"`
	CreationTime                  time.Time `json:"cTime"`
}

// WsGridSubOrderData to retrieve grid sub orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing order.
type WsGridSubOrderData struct {
	Argument SubscriptionInfo   `json:"arg"`
	Data     []GridSubOrderData `json:"data"`
}

// GridSubOrderData represents a single sub order detailed info
type GridSubOrderData struct {
	AccumulatedFillSize string    `json:"accFillSz"`
	AlgoID              string    `json:"algoId"`
	AlgoOrderType       string    `json:"algoOrdType"`
	AveragePrice        string    `json:"avgPx"`
	CreationTime        string    `json:"cTime"`
	ContractValue       string    `json:"ctVal"`
	Fee                 string    `json:"fee"`
	FeeCurrency         string    `json:"feeCcy"`
	GroupID             string    `json:"groupId"`
	InstrumentID        string    `json:"instId"`
	InstrumentType      string    `json:"instType"`
	Leverage            string    `json:"lever"`
	OrderID             string    `json:"ordId"`
	OrderType           string    `json:"ordType"`
	PushTime            time.Time `json:"pTime"`
	ProfitAdLoss        string    `json:"pnl"`
	PositionSide        string    `json:"posSide"`
	Price               string    `json:"px"`
	Side                string    `json:"side"`
	State               string    `json:"state"`
	Size                string    `json:"sz"`
	Tag                 string    `json:"tag"`
	TradeMode           string    `json:"tdMode"`
	UpdateTime          time.Time `json:"uTime"`
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
	Asks      [][4]string `json:"asks"`
	Bids      [][4]string `json:"bids"`
	Timestamp time.Time   `json:"ts"`
	Checksum  int32       `json:"checksum,omitempty"`
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

// WsPublicTradesResponse represents websocket push data of structured bloc trades as a result of subscription to "public-struc-block-trades"
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
	FilFromSize  float64   `json:"fillFromSz,string"`
	FillToSize   float64   `json:"fillToSz,string"`
	FromCurrency string    `json:"fromCcy"`
	Status       string    `json:"status"`
	ToCurrency   string    `json:"toCcy"`
	UpdateTime   time.Time `json:"uTime"`
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
	DebtCurrency  string    `json:"debtCcy"`
	FillFromSize  float64   `json:"fillFromSz,string"`
	FillRepaySize float64   `json:"fillRepaySz,string"`
	FillToSize    float64   `json:"fillToSz,string"`
	RepayCurrency string    `json:"repayCcy"`
	Status        string    `json:"status"`
	UpdateTime    time.Time `json:"uTime"`
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
	SubAccountName   string `json:"subAcct"`         // Sub-account name
	APIKey           string `json:"apiKey"`          // Sub-accountAPI public key
	Label            string `json:"label,omitempty"` // Sub-account APIKey label
	APIKeyPermission string `json:"perm,omitempty"`  // Sub-account APIKey permissions
	IP               string `json:"ip,omitempty"`    // Sub-account APIKey linked IP addresses, separate with commas if more than
}

// SubAccountAPIKeyResponse represents sub-account api key reset response
type SubAccountAPIKeyResponse struct {
	SubAccountName   string    `json:"subAcct"`
	APIKey           string    `json:"apiKey"`
	Label            string    `json:"label"`
	APIKeyPermission string    `json:"perm"`
	IP               string    `json:"ip"`
	Timestamp        time.Time `json:"ts"`
}

// MarginBalanceParam represents compute margin balance request param
type MarginBalanceParam struct {
	AlgoID     string  `json:"algoId"`
	Type       string  `json:"type"`
	Amount     float64 `json:"amt,string"`               // Adjust margin balance amount Either amt or percent is required.
	Percentage float64 `json:"percent,string,omitempty"` // Adjust margin balance percentage, used In Adjusting margin balance
}

// ComputeMarginBalance represents compute marign amount request response
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
	Apy          float64           `json:"apy"`
	EarlyRedeem  bool              `json:"earlyRedeem"`
	InvestData   []OfferInvestData `json:"investData"`
	EarningData  []struct {
		Currency    string `json:"ccy"`
		EarningType string `json:"earningType"`
	} `json:"earningData"`
}

// OfferInvestData represents currencies invest data information for an offer
type OfferInvestData struct {
	Currency      string  `json:"ccy"`
	Balance       float64 `json:"bal"`
	MinimumAmount float64 `json:"minAmt"`
	MaximumAmount float64 `json:"maxAmt"`
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
	PurchasedTime time.Time `json:"purchasedTime"`
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
	PurchasedTime time.Time `json:"purchasedTime"`
	RedeemedTime  time.Time `json:"redeemedTime"`
	EarningCcy    []string  `json:"earningCcy,omitempty"`
}

// wsRequestDataChannelsMultiplexer a single multiplexer instance to multiplex websocket messages multiplexer channels
type wsRequestDataChannelsMultiplexer struct {
	// To Synchronize incoming messages coming through the websocket channel
	WsResponseChannelsMap map[string]*wsRequestInfo
	Register              chan *wsRequestInfo
	Unregister            chan string
	Message               chan *wsIncomingData
}
