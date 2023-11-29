package okx

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
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

type candlestickState int8

// candlestickState represents index candlestick states.
const (
	StateUncompleted candlestickState = iota
	StateCompleted
)

// Market Data Endpoints

// TickerResponse represents the market data endpoint ticker detail
type TickerResponse struct {
	InstrumentType string                  `json:"instType"`
	InstrumentID   string                  `json:"instId"`
	LastTradePrice convert.StringToFloat64 `json:"last"`
	LastTradeSize  convert.StringToFloat64 `json:"lastSz"`
	BestAskPrice   convert.StringToFloat64 `json:"askPx"`
	BestAskSize    convert.StringToFloat64 `json:"askSz"`
	BestBidPrice   convert.StringToFloat64 `json:"bidPx"`
	BestBidSize    convert.StringToFloat64 `json:"bidSz"`
	Open24H        convert.StringToFloat64 `json:"open24h"`
	High24H        convert.StringToFloat64 `json:"high24h"`
	Low24H         convert.StringToFloat64 `json:"low24h"`
	VolCcy24H      convert.StringToFloat64 `json:"volCcy24h"`
	Vol24H         convert.StringToFloat64 `json:"vol24h"`

	OpenPriceInUTC0          string               `json:"sodUtc0"`
	OpenPriceInUTC8          string               `json:"sodUtc8"`
	TickerDataGenerationTime convert.ExchangeTime `json:"ts"`
}

// IndexTicker represents Index ticker data.
type IndexTicker struct {
	InstID    string                  `json:"instId"`
	IdxPx     convert.StringToFloat64 `json:"idxPx"`
	High24H   convert.StringToFloat64 `json:"high24h"`
	SodUtc0   convert.StringToFloat64 `json:"sodUtc0"`
	Open24H   convert.StringToFloat64 `json:"open24h"`
	Low24H    convert.StringToFloat64 `json:"low24h"`
	SodUtc8   convert.StringToFloat64 `json:"sodUtc8"`
	Timestamp convert.ExchangeTime    `json:"ts"`
}

// OrderBookResponse holds the order asks and bids at a specific timestamp
type OrderBookResponse struct {
	Asks                [][4]string          `json:"asks"`
	Bids                [][4]string          `json:"bids"`
	GenerationTimeStamp convert.ExchangeTime `json:"ts"`
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

// IndexCandlestickSlices represents index candlestick history represented by a slice of string
type IndexCandlestickSlices [][6]string

// ExtractIndexCandlestick extracts IndexCandlestick instance from slice of string.
func (a IndexCandlestickSlices) ExtractIndexCandlestick() ([]CandlestickHistoryItem, error) {
	if len(a) == 0 {
		return nil, errors.New("nil slice")
	}
	candles := make([]CandlestickHistoryItem, len(a))
	for i := range a {
		timestamp, err := strconv.ParseInt(a[i][0], 10, 64)
		if err != nil {
			return nil, err
		}
		candles[i] = CandlestickHistoryItem{
			Timestamp: time.UnixMilli(timestamp),
		}
		candles[i].OpenPrice, err = strconv.ParseFloat(a[i][1], 64)
		if err != nil {
			return nil, err
		}
		candles[i].HighestPrice, err = strconv.ParseFloat(a[i][2], 64)
		if err != nil {
			return nil, err
		}
		candles[i].LowestPrice, err = strconv.ParseFloat(a[i][3], 64)
		if err != nil {
			return nil, err
		}
		candles[i].ClosePrice, err = strconv.ParseFloat(a[i][4], 64)
		if err != nil {
			return nil, err
		}
		if a[i][5] == "1" {
			candles[i].Confirm = StateCompleted
		} else {
			candles[i].Confirm = StateUncompleted
		}
	}
	return candles, nil
}

// EconomicCalendar represents macro-economic calendar data
type EconomicCalendar struct {
	Actual        string               `json:"actual"`
	CalendarID    string               `json:"calendarId"`
	Category      string               `json:"category"`
	Currency      string               `json:"ccy"`
	Date          convert.ExchangeTime `json:"date"`
	DateSpan      string               `json:"dateSpan"`
	Event         string               `json:"event"`
	Forecast      string               `json:"forecast"`
	Importance    string               `json:"importance"`
	PrevInitial   string               `json:"prevInitial"`
	Previous      string               `json:"previous"`
	ReferenceDate convert.ExchangeTime `json:"refDate"`
	Region        string               `json:"region"`
	UpdateTime    convert.ExchangeTime `json:"uTime"`
	Unit          string               `json:"unit"`
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

// CandlestickHistoryItem retrieve the candlestick charts of the index/mark price from recent years.
type CandlestickHistoryItem struct {
	Timestamp    time.Time
	OpenPrice    float64
	HighestPrice float64
	LowestPrice  float64
	ClosePrice   float64
	Confirm      candlestickState
}

// TradeResponse represents the recent transaction instance.
type TradeResponse struct {
	InstrumentID string                  `json:"instId"`
	TradeID      string                  `json:"tradeId"`
	Price        convert.StringToFloat64 `json:"px"`
	Quantity     convert.StringToFloat64 `json:"sz"`
	Side         order.Side              `json:"side"`
	Timestamp    convert.ExchangeTime    `json:"ts"`
	Count        string                  `json:"count"`
}

// InstrumentFamilyTrade represents transaction information of instrument.
// nstrument family, e.g. BTC-USD Applicable to OPTION
type InstrumentFamilyTrade struct {
	Vol24H    convert.StringToFloat64 `json:"vol24h"`
	TradeInfo []struct {
		InstrumentID string                  `json:"instId"`
		TradeID      string                  `json:"tradeId"`
		Side         string                  `json:"side"`
		Size         convert.StringToFloat64 `json:"sz"`
		Price        convert.StringToFloat64 `json:"px"`
		Timestamp    convert.ExchangeTime    `json:"ts"`
	} `json:"tradeInfo"`
	OptionType string `json:"optType"`
}

// OptionTrade holds option trade item.
type OptionTrade struct {
	FillVolume       convert.StringToFloat64 `json:"fillVol"`
	ForwardPrice     convert.StringToFloat64 `json:"fwdPx"`
	IndexPrice       convert.StringToFloat64 `json:"idxPx"`
	MarkPrice        convert.StringToFloat64 `json:"markPx"`
	Price            convert.StringToFloat64 `json:"px"`
	Size             convert.StringToFloat64 `json:"sz"`
	InstrumentFamily string                  `json:"instFamily"`
	InstrumentID     string                  `json:"instId"`
	OptionType       string                  `json:"optType"`
	Side             string                  `json:"side"`
	TradeID          string                  `json:"tradeId"`
	Timestamp        convert.ExchangeTime    `json:"ts"`
}

// TradingVolumeIn24HR response model.
type TradingVolumeIn24HR struct {
	BlockVolumeInCNY   convert.StringToFloat64 `json:"blockVolCny"`
	BlockVolumeInUSD   convert.StringToFloat64 `json:"blockVolUsd"`
	TradingVolumeInUSD float64                 `json:"volUsd,string"`
	TradingVolumeInCny float64                 `json:"volCny,string"`
	Timestamp          convert.ExchangeTime    `json:"ts"`
}

// OracleSmartContractResponse returns the crypto price of signing using Open Oracle smart contract.
type OracleSmartContractResponse struct {
	Messages   []string             `json:"messages"`
	Prices     map[string]string    `json:"prices"`
	Signatures []string             `json:"signatures"`
	Timestamp  convert.ExchangeTime `json:"timestamp"`
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
	Timestamp  convert.ExchangeTime `json:"ts"`
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
	InstrumentType                  string                  `json:"instType"`
	InstrumentID                    string                  `json:"instId"`
	InstrumentFamily                string                  `json:"instFamily"`
	Underlying                      string                  `json:"uly"`
	Category                        string                  `json:"category"`
	BaseCurrency                    string                  `json:"baseCcy"`
	QuoteCurrency                   string                  `json:"quoteCcy"`
	SettlementCurrency              string                  `json:"settleCcy"`
	ContractValue                   convert.StringToFloat64 `json:"ctVal"`
	ContractMultiplier              convert.StringToFloat64 `json:"ctMult"`
	ContractValueCurrency           string                  `json:"ctValCcy"`
	OptionType                      string                  `json:"optType"`
	StrikePrice                     string                  `json:"stk"`
	ListTime                        convert.ExchangeTime    `json:"listTime"`
	ExpTime                         convert.ExchangeTime    `json:"expTime"`
	MaxLeverage                     convert.StringToFloat64 `json:"lever"`
	TickSize                        convert.StringToFloat64 `json:"tickSz"`
	LotSize                         convert.StringToFloat64 `json:"lotSz"`
	MinimumOrderSize                convert.StringToFloat64 `json:"minSz"`
	ContractType                    string                  `json:"ctType"`
	Alias                           string                  `json:"alias"`
	State                           string                  `json:"state"`
	MaxQuantityOfSpotLimitOrder     convert.StringToFloat64 `json:"maxLmtSz"`
	MaxQuantityOfMarketLimitOrder   convert.StringToFloat64 `json:"maxMktSz"`
	MaxQuantityOfSpotTwapLimitOrder convert.StringToFloat64 `json:"maxTwapSz"`
	MaxSpotIcebergSize              convert.StringToFloat64 `json:"maxIcebergSz"`
	MaxTriggerSize                  convert.StringToFloat64 `json:"maxTriggerSz"`
	MaxStopSize                     convert.StringToFloat64 `json:"maxStopSz"`
}

// DeliveryHistoryDetail holds instrument id and delivery price information detail
type DeliveryHistoryDetail struct {
	Type          string  `json:"type"`
	InstrumentID  string  `json:"insId"`
	DeliveryPrice float64 `json:"px,string"`
}

// DeliveryHistory represents list of delivery history detail items and timestamp information
type DeliveryHistory struct {
	Timestamp convert.ExchangeTime    `json:"ts"`
	Details   []DeliveryHistoryDetail `json:"details"`
}

// OpenInterest Retrieve the total open interest for contracts on OKX.
type OpenInterest struct {
	InstrumentType       asset.Item           `json:"instType"`
	InstrumentID         string               `json:"instId"`
	OpenInterest         float64              `json:"oi,string"`
	OpenInterestCurrency float64              `json:"oiCcy,string"`
	Timestamp            convert.ExchangeTime `json:"ts"`
}

// FundingRateResponse response data for the Funding Rate for an instruction type
type FundingRateResponse struct {
	FundingRate     convert.StringToFloat64 `json:"fundingRate"`
	RealisedRate    convert.StringToFloat64 `json:"realizedRate"`
	FundingTime     convert.ExchangeTime    `json:"fundingTime"`
	InstrumentID    string                  `json:"instId"`
	InstrumentType  string                  `json:"instType"`
	NextFundingRate convert.StringToFloat64 `json:"nextFundingRate"`
	NextFundingTime convert.ExchangeTime    `json:"nextFundingTime"`
}

// LimitPriceResponse hold an information for
type LimitPriceResponse struct {
	InstrumentType string                  `json:"instType"`
	InstrumentID   string                  `json:"instId"`
	BuyLimit       convert.StringToFloat64 `json:"buyLmt"`
	SellLimit      convert.StringToFloat64 `json:"sellLmt"`
	Timestamp      convert.ExchangeTime    `json:"ts"`
}

// OptionMarketDataResponse holds response data for option market data
type OptionMarketDataResponse struct {
	InstrumentType string                  `json:"instType"`
	InstrumentID   string                  `json:"instId"`
	Underlying     string                  `json:"uly"`
	Delta          convert.StringToFloat64 `json:"delta"`
	Gamma          convert.StringToFloat64 `json:"gamma"`
	Theta          convert.StringToFloat64 `json:"theta"`
	Vega           convert.StringToFloat64 `json:"vega"`
	DeltaBS        convert.StringToFloat64 `json:"deltaBS"`
	GammaBS        convert.StringToFloat64 `json:"gammaBS"`
	ThetaBS        convert.StringToFloat64 `json:"thetaBS"`
	VegaBS         convert.StringToFloat64 `json:"vegaBS"`
	RealVol        string                  `json:"realVol"`
	BidVolatility  convert.StringToFloat64 `json:"bidVol"`
	AskVolatility  convert.StringToFloat64 `json:"askVol"`
	MarkVolatility convert.StringToFloat64 `json:"markVol"`
	Leverage       convert.StringToFloat64 `json:"lever"`
	ForwardPrice   convert.StringToFloat64 `json:"fwdPx"`
	Timestamp      convert.ExchangeTime    `json:"ts"`
}

// DeliveryEstimatedPrice holds an estimated delivery or exercise price response.
type DeliveryEstimatedPrice struct {
	InstrumentType         string               `json:"instType"`
	InstrumentID           string               `json:"instId"`
	EstimatedDeliveryPrice string               `json:"settlePx"`
	Timestamp              convert.ExchangeTime `json:"ts"`
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
	DiscountRate string                  `json:"discountRate"`
	MaxAmount    convert.StringToFloat64 `json:"maxAmt"`
	MinAmount    convert.StringToFloat64 `json:"minAmt"`
}

// ServerTime returning  the server time instance.
type ServerTime struct {
	Timestamp convert.ExchangeTime `json:"ts"`
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
	BankruptcyLoss        string               `json:"bkLoss"`
	BankruptcyPx          string               `json:"bkPx"`
	Currency              string               `json:"ccy"`
	PosSide               string               `json:"posSide"`
	Side                  string               `json:"side"` // May be empty
	QuantityOfLiquidation float64              `json:"sz,string"`
	Timestamp             convert.ExchangeTime `json:"ts"`
}

// MarkPrice represents a mark price information for a single instrument id
type MarkPrice struct {
	InstrumentType string               `json:"instType"`
	InstrumentID   string               `json:"instId"`
	MarkPrice      string               `json:"markPx"`
	Timestamp      convert.ExchangeTime `json:"ts"`
}

// PositionTiers represents position tier detailed information.
type PositionTiers struct {
	BaseMaxLoan                  string                  `json:"baseMaxLoan"`
	InitialMarginRequirement     string                  `json:"imr"`
	InstrumentID                 string                  `json:"instId"`
	MaximumLeverage              string                  `json:"maxLever"`
	MaximumSize                  convert.StringToFloat64 `json:"maxSz"`
	MinSize                      convert.StringToFloat64 `json:"minSz"`
	MaintenanceMarginRequirement string                  `json:"mmr"`
	OptionalMarginFactor         string                  `json:"optMgnFactor"`
	QuoteMaxLoan                 string                  `json:"quoteMaxLoan"`
	Tier                         string                  `json:"tier"`
	Underlying                   string                  `json:"uly"`
}

// InterestRateLoanQuotaBasic holds the basic Currency, loan,and interest rate information.
type InterestRateLoanQuotaBasic struct {
	Currency     string                  `json:"ccy"`
	LoanQuota    string                  `json:"quota"`
	InterestRate convert.StringToFloat64 `json:"rate"`
}

// InterestRateLoanQuotaItem holds the basic Currency, loan,interest rate, and other level and VIP related information.
type InterestRateLoanQuotaItem struct {
	InterestRateLoanQuotaBasic
	InterestRateDiscount convert.StringToFloat64 `json:"irDiscount"`
	LoanQuotaCoefficient convert.StringToFloat64 `json:"loanQuotaCoef"`
	Level                string                  `json:"level"`
}

// VIPInterestRateAndLoanQuotaInformation holds interest rate and loan quoata information for VIP users.
type VIPInterestRateAndLoanQuotaInformation struct {
	InterestRateLoanQuotaBasic
	LevelList []struct {
		Level     string                  `json:"level"`
		LoanQuota convert.StringToFloat64 `json:"loanQuota"`
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
	Total   convert.StringToFloat64          `json:"total"`
}

// InsuranceFundInformationDetail represents an Insurance fund information item for a
// single currency and type
type InsuranceFundInformationDetail struct {
	Timestamp convert.ExchangeTime    `json:"ts"`
	Amount    convert.StringToFloat64 `json:"amt"`
	Balance   convert.StringToFloat64 `json:"balance"`
	Currency  string                  `json:"ccy"`
	Type      string                  `json:"type"`
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
	AssetType     asset.Item `json:"-"`
	InstrumentID  string     `json:"instId"`
	TradeMode     string     `json:"tdMode,omitempty"` // cash isolated
	ClientOrderID string     `json:"clOrdId,omitempty"`
	Currency      string     `json:"ccy,omitempty"` // Only applicable to cross MARGIN orders in Single-currency margin.
	OrderTag      string     `json:"tag,omitempty"`
	Side          string     `json:"side,omitempty"`
	PositionSide  string     `json:"posSide,omitempty"`
	OrderType     string     `json:"ordType,omitempty"`
	Amount        float64    `json:"sz,string,omitempty"`
	Price         float64    `json:"px,string,omitempty"`
	ReduceOnly    bool       `json:"reduceOnly,string,omitempty"`
	QuantityType  string     `json:"tgtCcy,omitempty"` // values base_ccy and quote_ccy
	// Added in the websocket requests
	BanAmend   bool                 `json:"banAmend,omitempty"` // Whether the SPOT Market Order size can be amended by the system.
	ExpiryTime convert.ExchangeTime `json:"expTime,omitempty"`
}

// OrderData response message for place, cancel, and amend an order requests.
// implements the StatusCodeHolder interface
type OrderData struct {
	OrderID       string `json:"ordId,omitempty"`
	RequestID     string `json:"reqId,omitempty"`
	ClientOrderID string `json:"clOrdId,omitempty"`
	Tag           string `json:"tag,omitempty"`
	SCode         string `json:"sCode,omitempty"`
	SMessage      string `json:"sMsg,omitempty"`
}

// GetSCode returns a status code value
func (a *OrderData) GetSCode() string { return a.SCode }

// GetSMsg returns a status message value
func (a *OrderData) GetSMsg() string { return a.SMessage }

// WsCancelSpreadOrders used to hold response for cancelling all spread orders.
// implements the StatusCodeHolder interface
type WsCancelSpreadOrders struct {
	Result   bool   `json:"result"`
	SCode    string `json:"sCode,omitempty"`
	SMessage string `json:"sMsg,omitempty"`
}

// GetSCode returns a status code value
func (a *WsCancelSpreadOrders) GetSCode() string { return a.SCode }

// GetSMsg returns a status message value
func (a *WsCancelSpreadOrders) GetSMsg() string { return a.SMessage }

// CancelOrderRequestParam represents order parameters to cancel an order.
type CancelOrderRequestParam struct {
	InstrumentID  string `json:"instId"`
	OrderID       string `json:"ordId"`
	ClientOrderID string `json:"clOrdId,omitempty"`
}

// AmendOrderRequestParams represents amend order requesting parameters.
type AmendOrderRequestParams struct {
	InstrumentID    string  `json:"instId"`
	CancelOnFail    bool    `json:"cxlOnFail,omitempty"`
	OrderID         string  `json:"ordId,omitempty"`
	ClientOrderID   string  `json:"clOrdId,omitempty"`
	ClientRequestID string  `json:"reqId,omitempty"`
	NewQuantity     float64 `json:"newSz,string,omitempty"`
	NewPrice        float64 `json:"newPx,string,omitempty"`
}

// ClosePositionsRequestParams input parameters for close position endpoints
type ClosePositionsRequestParams struct {
	InstrumentID          string `json:"instId"` // REQUIRED
	PositionSide          string `json:"posSide"`
	MarginMode            string `json:"mgnMode"` // cross or isolated
	Currency              string `json:"ccy"`
	AutomaticallyCanceled bool   `json:"autoCxl"`
	ClientID              string `json:"clOrdId,omitempty"`
	Tag                   string `json:"tag,omitempty"`
}

// ClosePositionResponse response data for close position.
type ClosePositionResponse struct {
	InstrumentID string `json:"instId"`
	PositionSide string `json:"posSide"`
}

// OrderDetailRequestParam payload data to request order detail
type OrderDetailRequestParam struct {
	InstrumentID  string `json:"instId"`
	OrderID       string `json:"ordId"`
	ClientOrderID string `json:"clOrdId"`
}

// OrderDetail returns a order detail information
type OrderDetail struct {
	InstrumentType             string                  `json:"instType"`
	InstrumentID               string                  `json:"instId"`
	Currency                   string                  `json:"ccy"`
	OrderID                    string                  `json:"ordId"`
	ClientOrderID              string                  `json:"clOrdId"`
	Tag                        string                  `json:"tag"`
	ProfitAndLoss              string                  `json:"pnl"`
	OrderType                  string                  `json:"ordType"`
	Side                       order.Side              `json:"side"`
	PositionSide               string                  `json:"posSide"`
	TradeMode                  string                  `json:"tdMode"`
	TradeID                    string                  `json:"tradeId"`
	FillTime                   convert.ExchangeTime    `json:"fillTime"`
	Source                     string                  `json:"source"`
	State                      string                  `json:"state"`
	TakeProfitTriggerPriceType string                  `json:"tpTriggerPxType"`
	StopLossTriggerPriceType   string                  `json:"slTriggerPxType"`
	StopLossOrderPrice         convert.StringToFloat64 `json:"slOrdPx"`
	RebateCurrency             string                  `json:"rebateCcy"`
	QuantityType               string                  `json:"tgtCcy"`   // base_ccy and quote_ccy
	Category                   string                  `json:"category"` // normal, twap, adl, full_liquidation, partial_liquidation, delivery, ddh
	AccumulatedFillSize        convert.StringToFloat64 `json:"accFillSz"`
	FillPrice                  convert.StringToFloat64 `json:"fillPx"`
	FillSize                   convert.StringToFloat64 `json:"fillSz"`
	RebateAmount               convert.StringToFloat64 `json:"rebate"`
	FeeCurrency                string                  `json:"feeCcy"`
	TransactionFee             convert.StringToFloat64 `json:"fee"`
	AveragePrice               convert.StringToFloat64 `json:"avgPx"`
	Leverage                   convert.StringToFloat64 `json:"lever"`
	Price                      convert.StringToFloat64 `json:"px"`
	Size                       convert.StringToFloat64 `json:"sz"`
	TakeProfitTriggerPrice     convert.StringToFloat64 `json:"tpTriggerPx"`
	TakeProfitOrderPrice       convert.StringToFloat64 `json:"tpOrdPx"`
	StopLossTriggerPrice       convert.StringToFloat64 `json:"slTriggerPx"`
	UpdateTime                 convert.ExchangeTime    `json:"uTime"`
	CreationTime               convert.ExchangeTime    `json:"cTime"`
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
	AccumulatedFillSize        convert.StringToFloat64 `json:"accFillSz"`
	AveragePrice               convert.StringToFloat64 `json:"avgPx"`
	CreationTime               convert.ExchangeTime    `json:"cTime"`
	Category                   string                  `json:"category"`
	Currency                   string                  `json:"ccy"`
	ClientOrderID              string                  `json:"clOrdId"`
	Fee                        convert.StringToFloat64 `json:"fee"`
	FeeCurrency                currency.Code           `json:"feeCcy"`
	LastFilledPrice            convert.StringToFloat64 `json:"fillPx"`
	LastFilledSize             convert.StringToFloat64 `json:"fillSz"`
	FillTime                   convert.ExchangeTime    `json:"fillTime"`
	InstrumentID               string                  `json:"instId"`
	InstrumentType             string                  `json:"instType"`
	Leverage                   convert.StringToFloat64 `json:"lever"`
	OrderID                    string                  `json:"ordId"`
	OrderType                  string                  `json:"ordType"`
	ProfitAndLoss              string                  `json:"pnl"`
	PositionSide               string                  `json:"posSide"`
	RebateAmount               convert.StringToFloat64 `json:"rebate"`
	RebateCurrency             string                  `json:"rebateCcy"`
	Side                       order.Side              `json:"side"`
	StopLossOrdPrice           convert.StringToFloat64 `json:"slOrdPx"`
	StopLossTriggerPrice       convert.StringToFloat64 `json:"slTriggerPx"`
	StopLossTriggerPriceType   string                  `json:"slTriggerPxType"`
	State                      string                  `json:"state"`
	Price                      convert.StringToFloat64 `json:"px"`
	Size                       convert.StringToFloat64 `json:"sz"`
	Tag                        string                  `json:"tag"`
	SizeType                   string                  `json:"tgtCcy"`
	TradeMode                  string                  `json:"tdMode"`
	Source                     string                  `json:"source"`
	TakeProfitOrdPrice         convert.StringToFloat64 `json:"tpOrdPx"`
	TakeProfitTriggerPrice     convert.StringToFloat64 `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string                  `json:"tpTriggerPxType"`
	TradeID                    string                  `json:"tradeId"`
	UpdateTime                 convert.ExchangeTime    `json:"uTime"`
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

// FillArchiveParam transaction detail param for 2 year.
type FillArchiveParam struct {
	Year    int64  `json:"year,string"`
	Quarter string `json:"quarter"`
}

// ArchiveReference holds recently-filled transaction details archive link and timestamp information.
type ArchiveReference struct {
	FileHref  string               `json:"fileHref"`
	State     string               `json:"state"`
	Timestamp convert.ExchangeTime `json:"ts"`
}

// TransactionDetail holds ecently-filled transaction detail data.
type TransactionDetail struct {
	InstrumentType string                  `json:"instType"`
	InstrumentID   string                  `json:"instId"`
	TradeID        string                  `json:"tradeId"`
	OrderID        string                  `json:"ordId"`
	ClientOrderID  string                  `json:"clOrdId"`
	BillID         string                  `json:"billId"`
	Tag            string                  `json:"tag"`
	FillPrice      convert.StringToFloat64 `json:"fillPx"`
	FillSize       convert.StringToFloat64 `json:"fillSz"`
	Side           order.Side              `json:"side"`
	PositionSide   string                  `json:"posSide"`
	ExecType       string                  `json:"execType"`
	FeeCurrency    string                  `json:"feeCcy"`
	Fee            convert.StringToFloat64 `json:"fee"`
	Timestamp      convert.ExchangeTime    `json:"ts"`
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

// AlgoOrder algo order requests response.
type AlgoOrder struct {
	AlgoID     string `json:"algoId"`
	StatusCode string `json:"sCode"`
	StatusMsg  string `json:"sMsg"`
}

// AmendAlgoOrderParam request parameter to amend an algo order.
type AmendAlgoOrderParam struct {
	InstrumentID                  string  `json:"instId"`
	AlgoID                        string  `json:"algoId,omitempty"`
	ClientSuppliedAlgoOrderID     string  `json:"algoClOrdId,omitempty"`
	CancelOrderWhenFail           bool    `json:"cxlOnFail,omitempty"` // Whether the order needs to be automatically canceled when the order amendment fails Valid options: false or true, the default is false.
	RequestID                     string  `json:"reqId,omitempty"`
	NewSize                       float64 `json:"newSz,omitempty,string"`
	NewTakeProfitTriggerPrice     float64 `json:"newTpTriggerPx,omitempty,string"`
	NewTakeProfitOrderPrice       float64 `json:"newTpOrdPx,omitempty,string"`
	NewStopLossTriggerPrice       float64 `json:"newSlTriggerPx,omitempty,string"`
	NewStopLossOrderPrice         float64 `json:"newSlOrdPx,omitempty,string"`  // Stop-loss order price If the price is -1, stop-loss will be executed at the market price.
	NewTakeProfitTriggerPriceType string  `json:"newTpTriggerPxType,omitempty"` // Take-profit trigger price type'last': last price 'index': index price 'mark': mark price
	NewStopLossTriggerPriceType   string  `json:"newSlTriggerPxType,omitempty"` // Stop-loss trigger price type 'last': last price  'index': index price  'mark': mark price
}

// AmendAlgoResponse holds response information of amending an algo order.
type AmendAlgoResponse struct {
	AlgoClientOrderID string `json:"algoClOrdId"`
	AlgoID            string `json:"algoId"`
	ReqID             string `json:"reqId"`
	SCode             string `json:"sCode"`
	SMsg              string `json:"sMsg"`
}

// AlgoOrderDetail represents an algo order detail.
type AlgoOrderDetail struct {
	InstrumentType          string                  `json:"instType"`
	InstrumentID            string                  `json:"instId"`
	OrderID                 string                  `json:"ordId"`
	OrderIDList             []string                `json:"ordIdList"`
	Currency                string                  `json:"ccy"`
	ClientOrderID           string                  `json:"clOrdId"`
	AlgoID                  string                  `json:"algoId"`
	AttachAlgoOrds          []string                `json:"attachAlgoOrds"`
	Size                    convert.StringToFloat64 `json:"sz"`
	CloseFraction           string                  `json:"closeFraction"`
	OrderType               string                  `json:"ordType"`
	Side                    string                  `json:"side"`
	PosSide                 string                  `json:"posSide"`
	TradeMode               string                  `json:"tdMode"`
	TargetCurrency          string                  `json:"tgtCcy"`
	State                   string                  `json:"state"`
	Leverage                convert.StringToFloat64 `json:"lever"`
	TpTriggerPrice          convert.StringToFloat64 `json:"tpTriggerPx"`
	TpTriggerPriceType      string                  `json:"tpTriggerPxType"`
	TpOrdPrice              convert.StringToFloat64 `json:"tpOrdPx"`
	SlTriggerPrice          convert.StringToFloat64 `json:"slTriggerPx"`
	SlTriggerPriceType      string                  `json:"slTriggerPxType"`
	TriggerPrice            convert.StringToFloat64 `json:"triggerPx"`
	TriggerPriceType        string                  `json:"triggerPxType"`
	OrderPrice              convert.StringToFloat64 `json:"ordPx"`
	ActualSize              convert.StringToFloat64 `json:"actualSz"`
	ActualPrice             convert.StringToFloat64 `json:"actualPx"`
	ActualSide              string                  `json:"actualSide"`
	PriceVar                string                  `json:"pxVar"`
	PriceSpread             string                  `json:"pxSpread"`
	PriceLimit              convert.StringToFloat64 `json:"pxLimit"`
	SizeLimit               convert.StringToFloat64 `json:"szLimit"`
	Tag                     string                  `json:"tag"`
	TimeInterval            string                  `json:"timeInterval"`
	CallbackRatio           string                  `json:"callbackRatio"`
	CallbackSpread          string                  `json:"callbackSpread"`
	ActivePrice             convert.StringToFloat64 `json:"activePx"`
	MoveTriggerPrice        convert.StringToFloat64 `json:"moveTriggerPx"`
	ReduceOnly              string                  `json:"reduceOnly"`
	TriggerTime             convert.ExchangeTime    `json:"triggerTime"`
	Last                    convert.StringToFloat64 `json:"last"` // Last filled price while placing
	FailCode                string                  `json:"failCode"`
	AlgoClOrdID             string                  `json:"algoClOrdId"`
	AmendPriceOnTriggerType string                  `json:"amendPxOnTriggerType"`
	CreationTime            convert.ExchangeTime    `json:"cTime"`
}

// AlgoOrderCancelParams algo order request parameter
type AlgoOrderCancelParams struct {
	AlgoOrderID  string `json:"algoId"`
	InstrumentID string `json:"instId"`
}

// AlgoOrderResponse holds algo order information.
type AlgoOrderResponse struct {
	InstrumentType             string                  `json:"instType"`
	InstrumentID               string                  `json:"instId"`
	OrderID                    string                  `json:"ordId"`
	Currency                   string                  `json:"ccy"`
	AlgoOrderID                string                  `json:"algoId"`
	Quantity                   string                  `json:"sz"`
	OrderType                  string                  `json:"ordType"`
	Side                       order.Side              `json:"side"`
	PositionSide               string                  `json:"posSide"`
	TradeMode                  string                  `json:"tdMode"`
	QuantityType               string                  `json:"tgtCcy"`
	State                      string                  `json:"state"`
	Lever                      string                  `json:"lever"`
	TakeProfitTriggerPrice     convert.StringToFloat64 `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string                  `json:"tpTriggerPxType"`
	TakeProfitOrdPrice         convert.StringToFloat64 `json:"tpOrdPx"`
	StopLossTriggerPriceType   string                  `json:"slTriggerPxType"`
	StopLossTriggerPrice       convert.StringToFloat64 `json:"slTriggerPx"`
	TriggerPrice               convert.StringToFloat64 `json:"triggerPx"`
	TriggerPriceType           string                  `json:"triggerPxType"`
	OrderPrice                 convert.StringToFloat64 `json:"ordPx"`
	ActualSize                 string                  `json:"actualSz"`
	ActualPrice                convert.StringToFloat64 `json:"actualPx"`
	ActualSide                 string                  `json:"actualSide"`
	PriceVar                   string                  `json:"pxVar"`
	PriceSpread                string                  `json:"pxSpread"`
	PriceLimit                 string                  `json:"pxLimit"`
	SizeLimit                  string                  `json:"szLimit"`
	TimeInterval               string                  `json:"timeInterval"`
	TriggerTime                convert.ExchangeTime    `json:"triggerTime"`
	CallbackRatio              convert.StringToFloat64 `json:"callbackRatio"`
	CallbackSpread             string                  `json:"callbackSpread"`
	ActivePrice                convert.StringToFloat64 `json:"activePx"`
	MoveTriggerPrice           convert.StringToFloat64 `json:"moveTriggerPx"`
	CreationTime               convert.ExchangeTime    `json:"cTime"`
}

// CurrencyResponse represents a currency item detail response data.
type CurrencyResponse struct {
	CanDeposit          bool                    `json:"canDep"`      // Availability to deposit from chain. false: not available true: available
	CanInternalTransfer bool                    `json:"canInternal"` // Availability to internal transfer.
	CanWithdraw         bool                    `json:"canWd"`       // Availability to withdraw to chain.
	Currency            string                  `json:"ccy"`         //
	Chain               string                  `json:"chain"`       //
	LogoLink            string                  `json:"logoLink"`    // Logo link of currency
	MainNet             bool                    `json:"mainNet"`     // If current chain is main net then return true, otherwise return false
	MaxFee              convert.StringToFloat64 `json:"maxFee"`      // Minimum withdrawal fee
	MaxWithdrawal       convert.StringToFloat64 `json:"maxWd"`       // Minimum amount of currency withdrawal in a single transaction
	MinFee              convert.StringToFloat64 `json:"minFee"`      // Minimum withdrawal fee
	MinWithdrawal       convert.StringToFloat64 `json:"minWd"`       // Minimum amount of currency withdrawal in a single transaction
	Name                string                  `json:"name"`        // Chinese name of currency
	UsedWithdrawalQuota string                  `json:"usedWdQuota"` // Amount of currency withdrawal used in the past 24 hours, unit in BTC
	WithdrawalQuota     string                  `json:"wdQuota"`     // Minimum amount of currency withdrawal in a single transaction
	WithdrawalTickSize  convert.StringToFloat64 `json:"wdTickSz"`    // Withdrawal precision, indicating the number of digits after the decimal point
}

// AssetBalance represents account owner asset balance
type AssetBalance struct {
	AvailBal      convert.StringToFloat64 `json:"availBal"`
	Balance       convert.StringToFloat64 `json:"bal"`
	Currency      string                  `json:"ccy"`
	FrozenBalance convert.StringToFloat64 `json:"frozenBal"`
}

// NonTradableAsset holds non-tradable asset detail.
type NonTradableAsset struct {
	Balance    convert.StringToFloat64 `json:"bal"`
	CanWd      bool                    `json:"canWd"`
	Currency   string                  `json:"ccy"`
	Chain      string                  `json:"chain"`
	CtAddr     string                  `json:"ctAddr"`
	Fee        convert.StringToFloat64 `json:"fee"`
	LogoLink   string                  `json:"logoLink"`
	MinWd      string                  `json:"minWd"`
	Name       string                  `json:"name"`
	NeedTag    bool                    `json:"needTag"`
	WdAll      bool                    `json:"wdAll"`
	WdTickSize convert.StringToFloat64 `json:"wdTickSz"`
}

// AccountAssetValuation represents view account asset valuation data
type AccountAssetValuation struct {
	Details struct {
		Classic convert.StringToFloat64 `json:"classic"`
		Earn    convert.StringToFloat64 `json:"earn"`
		Funding convert.StringToFloat64 `json:"funding"`
		Trading convert.StringToFloat64 `json:"trading"`
	} `json:"details"`
	TotalBalance convert.StringToFloat64 `json:"totalBal"`
	Timestamp    convert.ExchangeTime    `json:"ts"`
}

// FundingTransferRequestInput represents funding account request input.
type FundingTransferRequestInput struct {
	Currency     string  `json:"ccy"`
	Type         int64   `json:"type,string"`
	Amount       float64 `json:"amt,string"`
	From         string  `json:"from"` // "6": Funding account, "18": Trading account
	To           string  `json:"to"`
	SubAccount   string  `json:"subAcct"`
	LoanTransfer bool    `json:"loanTrans,string"`
	ClientID     string  `json:"clientId"` // Client-supplied ID A combination of case-sensitive alphanumerics, all numbers, or all letters of up to 32 characters.
}

// FundingTransferResponse represents funding transfer and trading account transfer response.
type FundingTransferResponse struct {
	TransferID string                  `json:"transId"`
	Currency   string                  `json:"ccy"`
	ClientID   string                  `json:"clientId"`
	From       int64                   `json:"from,string"`
	Amount     convert.StringToFloat64 `json:"amt"`
	To         int64                   `json:"to,string"`
}

// TransferFundRateResponse represents funcing transfer rate response
type TransferFundRateResponse struct {
	Amount         convert.StringToFloat64 `json:"amt"`
	Currency       string                  `json:"ccy"`
	ClientID       string                  `json:"clientId"`
	From           string                  `json:"from"`
	InstrumentID   string                  `json:"instId"`
	State          string                  `json:"state"`
	SubAccount     string                  `json:"subAcct"`
	To             string                  `json:"to"`
	ToInstrumentID string                  `json:"toInstId"`
	TransferID     string                  `json:"transId"`
	Type           int64                   `json:"type,string"`
}

// AssetBillDetail represents  the billing record
type AssetBillDetail struct {
	BillID         string               `json:"billId"`
	Currency       string               `json:"ccy"`
	ClientID       string               `json:"clientId"`
	BalanceChange  string               `json:"balChg"`
	AccountBalance string               `json:"bal"`
	Type           int64                `json:"type,string"`
	Timestamp      convert.ExchangeTime `json:"ts"`
}

// LightningDepositItem for creating an invoice.
type LightningDepositItem struct {
	CreationTime convert.ExchangeTime `json:"cTime"`
	Invoice      string               `json:"invoice"`
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
	Amount           convert.StringToFloat64 `json:"amt"`
	TransactionID    string                  `json:"txId"` // Hash record of the deposit
	Currency         string                  `json:"ccy"`
	Chain            string                  `json:"chain"`
	From             string                  `json:"from"`
	ToDepositAddress string                  `json:"to"`
	Timestamp        convert.ExchangeTime    `json:"ts"`
	State            int64                   `json:"state,string"`
	DepositID        string                  `json:"depId"`
}

// WithdrawalInput represents request parameters for cryptocurrency withdrawal
type WithdrawalInput struct {
	Amount                float64 `json:"amt,string"`
	TransactionFee        float64 `json:"fee,string"`
	WithdrawalDestination string  `json:"dest"`
	Currency              string  `json:"ccy"`
	ChainName             string  `json:"chain"`
	ToAddress             string  `json:"toAddr"`
	ClientID              string  `json:"clientId"`
}

// WithdrawalResponse cryptocurrency withdrawal response
type WithdrawalResponse struct {
	Amount       convert.StringToFloat64 `json:"amt"`
	WithdrawalID string                  `json:"wdId"`
	Currency     string                  `json:"ccy"`
	ClientID     string                  `json:"clientId"`
	Chain        string                  `json:"chain"`
}

// LightningWithdrawalRequestInput to request Lightning Withdrawal requests.
type LightningWithdrawalRequestInput struct {
	Currency string `json:"ccy"`     // REQUIRED Token symbol. Currently only BTC is supported.
	Invoice  string `json:"invoice"` // REQUIRED Invoice text
	Memo     string `json:"memo"`    // Lightning withdrawal memo
}

// LightningWithdrawalResponse response item for holding lightning withdrawal requests.
type LightningWithdrawalResponse struct {
	WithdrawalID string               `json:"wdId"`
	CreationTime convert.ExchangeTime `json:"cTime"`
}

// WithdrawalHistoryResponse represents the withdrawal response history.
type WithdrawalHistoryResponse struct {
	ChainName            string                  `json:"chain"`
	WithdrawalFee        convert.StringToFloat64 `json:"fee"`
	Currency             string                  `json:"ccy"`
	ClientID             string                  `json:"clientId"`
	Amount               convert.StringToFloat64 `json:"amt"`
	TransactionID        string                  `json:"txId"` // Hash record of the withdrawal. This parameter will not be returned for internal transfers.
	FromRemittingAddress string                  `json:"from"`
	ToReceivingAddress   string                  `json:"to"`
	StateOfWithdrawal    string                  `json:"state"`
	Timestamp            convert.ExchangeTime    `json:"ts"`
	WithdrawalID         string                  `json:"wdId"`
	PaymentID            string                  `json:"pmtId,omitempty"`
	Memo                 string                  `json:"memo"`
}

// DepositWithdrawStatus holds deposit withdraw status info.
type DepositWithdrawStatus struct {
	WithdrawID      string               `json:"wdId"`
	TransactionID   string               `json:"txId"`
	State           string               `json:"state"`
	EstCompleteTime convert.ExchangeTime `json:"estCompleteTime"`
}

// ExchangeInfo represents exchange information
type ExchangeInfo struct {
	ExchID       string `json:"exchId"`
	ExchangeName string `json:"exchName"`
}

// SmallAssetConvertResponse represents a response of converting a small asset to OKB.
type SmallAssetConvertResponse struct {
	Details []struct {
		Amount        convert.StringToFloat64 `json:"amt"`    // Quantity of currency assets before conversion
		Currency      string                  `json:"ccy"`    //
		ConvertAmount convert.StringToFloat64 `json:"cnvAmt"` // Quantity of OKB after conversion
		ConversionFee convert.StringToFloat64 `json:"fee"`    // Fee for conversion, unit in OKB
	} `json:"details"`
	TotalConvertAmount convert.StringToFloat64 `json:"totalCnvAmt"` // Total quantity of OKB after conversion
}

// SavingBalanceResponse returns a saving response.
type SavingBalanceResponse struct {
	Currency      string                  `json:"ccy"`
	Earnings      convert.StringToFloat64 `json:"earnings"`
	RedemptAmount convert.StringToFloat64 `json:"redemptAmt"`
	Rate          convert.StringToFloat64 `json:"rate"`
	Amount        convert.StringToFloat64 `json:"amt"`
	LoanAmount    convert.StringToFloat64 `json:"loanAmt"`
	PendingAmount convert.StringToFloat64 `json:"pendingAmt"`
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
	Currency   string                  `json:"ccy"`
	Amount     convert.StringToFloat64 `json:"amt"`
	ActionType string                  `json:"side"`
	Rate       convert.StringToFloat64 `json:"rate"`
}

// LendingRate represents lending rate response
type LendingRate struct {
	Currency string                  `json:"ccy"`
	Rate     convert.StringToFloat64 `json:"rate"`
}

// LendingHistory holds lending history responses
type LendingHistory struct {
	Currency  string               `json:"ccy"`
	Amount    float64              `json:"amt,string"`
	Earnings  float64              `json:"earnings,string,omitempty"`
	Rate      float64              `json:"rate,string"`
	Timestamp convert.ExchangeTime `json:"ts"`
}

// PublicBorrowInfo holds a currency's borrow info.
type PublicBorrowInfo struct {
	Currency         string                  `json:"ccy"`
	AverageAmount    convert.StringToFloat64 `json:"avgAmt"`
	AverageAmountUSD convert.StringToFloat64 `json:"avgAmtUsd"`
	AverageRate      convert.StringToFloat64 `json:"avgRate"`
	PreviousRate     convert.StringToFloat64 `json:"preRate"`
	EstimatedRate    convert.StringToFloat64 `json:"estRate"`
}

// PublicBorrowHistory holds a currencies borrow history.
type PublicBorrowHistory struct {
	Amount    convert.StringToFloat64 `json:"amt"`
	Currency  string                  `json:"ccy"`
	Rate      convert.StringToFloat64 `json:"rate"`
	Timestamp convert.ExchangeTime    `json:"ts"`
}

// ConvertCurrency represents currency conversion detailed data.
type ConvertCurrency struct {
	Currency string                  `json:"currency"`
	Min      convert.StringToFloat64 `json:"min"`
	Max      convert.StringToFloat64 `json:"max"`
}

// ConvertCurrencyPair holds information related to conversion between two pairs.
type ConvertCurrencyPair struct {
	InstrumentID     string                  `json:"instId"`
	BaseCurrency     string                  `json:"baseCcy"`
	BaseCurrencyMax  convert.StringToFloat64 `json:"baseCcyMax"`
	BaseCurrencyMin  convert.StringToFloat64 `json:"baseCcyMin"`
	QuoteCurrency    string                  `json:"quoteCcy,omitempty"`
	QuoteCurrencyMax convert.StringToFloat64 `json:"quoteCcyMax"`
	QuoteCurrencyMin convert.StringToFloat64 `json:"quoteCcyMin"`
}

// EstimateQuoteRequestInput represents estimate quote request parameters
type EstimateQuoteRequestInput struct {
	BaseCurrency         string  `json:"baseCcy,omitempty"`
	QuoteCurrency        string  `json:"quoteCcy,omitempty"`
	Side                 string  `json:"side,omitempty"`
	RfqAmount            float64 `json:"rfqSz,omitempty"`
	RfqSzCurrency        string  `json:"rfqSzCcy,omitempty"`
	ClientRequestOrderID string  `json:"clQReqId,string,omitempty"`
	Tag                  string  `json:"tag,omitempty"`
}

// EstimateQuoteResponse represents estimate quote response data.
type EstimateQuoteResponse struct {
	BaseCurrency    string               `json:"baseCcy"`
	BaseSize        string               `json:"baseSz"`
	ClientRequestID string               `json:"clQReqId"`
	ConvertPrice    string               `json:"cnvtPx"`
	OrigRfqSize     string               `json:"origRfqSz"`
	QuoteCurrency   string               `json:"quoteCcy"`
	QuoteID         string               `json:"quoteId"`
	QuoteSize       string               `json:"quoteSz"`
	QuoteTime       convert.ExchangeTime `json:"quoteTime"`
	RfqSize         string               `json:"rfqSz"`
	RfqSizeCurrency string               `json:"rfqSzCcy"`
	Side            order.Side           `json:"side"`
	TTLMs           string               `json:"ttlMs"` // Validity period of quotation in milliseconds
}

// ConvertTradeInput represents convert trade request input
type ConvertTradeInput struct {
	BaseCurrency  string  `json:"baseCcy"`
	QuoteCurrency string  `json:"quoteCcy"`
	Side          string  `json:"side"`
	Size          float64 `json:"sz,string"`
	SizeCurrency  string  `json:"szCcy"`
	QuoteID       string  `json:"quoteId"`
	ClientOrderID string  `json:"clTReqId"`
	Tag           string  `json:"tag"`
}

// ConvertTradeResponse represents convert trade response
type ConvertTradeResponse struct {
	BaseCurrency  string                  `json:"baseCcy"`
	ClientOrderID string                  `json:"clTReqId"`
	FillBaseSize  convert.StringToFloat64 `json:"fillBaseSz"`
	FillPrice     convert.StringToFloat64 `json:"fillPx"`
	FillQuoteSize convert.StringToFloat64 `json:"fillQuoteSz"`
	InstrumentID  string                  `json:"instId"`
	QuoteCurrency string                  `json:"quoteCcy"`
	QuoteID       string                  `json:"quoteId"`
	Side          order.Side              `json:"side"`
	State         string                  `json:"state"`
	TradeID       string                  `json:"tradeId"`
	Timestamp     convert.ExchangeTime    `json:"ts"`
}

// ConvertHistory holds convert trade history response
type ConvertHistory struct {
	InstrumentID  string                  `json:"instId"`
	Side          order.Side              `json:"side"`
	FillPrice     convert.StringToFloat64 `json:"fillPx"`
	BaseCurrency  string                  `json:"baseCcy"`
	QuoteCurrency string                  `json:"quoteCcy"`
	FillBaseSize  convert.StringToFloat64 `json:"fillBaseSz"`
	State         string                  `json:"state"`
	TradeID       string                  `json:"tradeId"`
	FillQuoteSize convert.StringToFloat64 `json:"fillQuoteSz"`
	Timestamp     convert.ExchangeTime    `json:"ts"`
}

// Account holds currency account balance and related information
type Account struct {
	AdjEq       convert.StringToFloat64 `json:"adjEq"`
	Details     []AccountDetail         `json:"details"`
	Imr         convert.StringToFloat64 `json:"imr"` // Frozen equity for open positions and pending orders in USD level Applicable to Multi-currency margin and Portfolio margin
	IsoEq       convert.StringToFloat64 `json:"isoEq"`
	MgnRatio    convert.StringToFloat64 `json:"mgnRatio"`
	Mmr         convert.StringToFloat64 `json:"mmr"` // Maintenance margin requirement in USD level Applicable to Multi-currency margin and Portfolio margin
	NotionalUsd convert.StringToFloat64 `json:"notionalUsd"`
	OrdFroz     convert.StringToFloat64 `json:"ordFroz"` // Margin frozen for pending orders in USD level Applicable to Multi-currency margin and Portfolio margin
	TotalEquity convert.StringToFloat64 `json:"totalEq"` // Total Equity in USD level
	UpdateTime  convert.ExchangeTime    `json:"uTime"`   // UpdateTime
}

// AccountDetail account detail information.
type AccountDetail struct {
	AvailableBalance              convert.StringToFloat64 `json:"availBal"`
	AvailableEquity               convert.StringToFloat64 `json:"availEq"`
	CashBalance                   convert.StringToFloat64 `json:"cashBal"` // Cash Balance
	Currency                      string                  `json:"ccy"`
	CrossLiab                     convert.StringToFloat64 `json:"crossLiab"`
	DiscountEquity                convert.StringToFloat64 `json:"disEq"`
	EquityOfCurrency              convert.StringToFloat64 `json:"eq"`
	EquityUsd                     convert.StringToFloat64 `json:"eqUsd"`
	FrozenBalance                 convert.StringToFloat64 `json:"frozenBal"`
	Interest                      convert.StringToFloat64 `json:"interest"`
	IsoEquity                     convert.StringToFloat64 `json:"isoEq"`
	IsolatedLiabilities           convert.StringToFloat64 `json:"isoLiab"`
	IsoUpl                        convert.StringToFloat64 `json:"isoUpl"` // Isolated unrealized profit and loss of the currency applicable to Single-currency margin and Multi-currency margin and Portfolio margin
	LiabilitiesOfCurrency         convert.StringToFloat64 `json:"liab"`
	MaxLoan                       convert.StringToFloat64 `json:"maxLoan"`
	MarginRatio                   convert.StringToFloat64 `json:"mgnRatio"`      // Equity of the currency
	NotionalLever                 convert.StringToFloat64 `json:"notionalLever"` // Leverage of the currency applicable to Single-currency margin
	OpenOrdersMarginFrozen        convert.StringToFloat64 `json:"ordFrozen"`
	Twap                          convert.StringToFloat64 `json:"twap"`
	UpdateTime                    convert.ExchangeTime    `json:"uTime"`
	UnrealizedProfit              convert.StringToFloat64 `json:"upl"`
	UnrealizedCurrencyLiabilities convert.StringToFloat64 `json:"uplLiab"`
	StrategyEquity                convert.StringToFloat64 `json:"stgyEq"`  // strategy equity
	TotalEquity                   convert.StringToFloat64 `json:"totalEq"` // Total equity in USD level. Appears unused
}

// AccountPosition account position.
type AccountPosition struct {
	AutoDeleveraging             string                  `json:"adl"`      // Auto-deleveraging (ADL) indicator Divided into 5 levels, from 1 to 5, the smaller the number, the weaker the adl intensity.
	AvailablePosition            string                  `json:"availPos"` // Position that can be closed Only applicable to MARGIN, FUTURES/SWAP in the long-short mode, OPTION in Simple and isolated OPTION in margin Account.
	AveragePrice                 convert.StringToFloat64 `json:"avgPx"`
	CreationTime                 convert.ExchangeTime    `json:"cTime"`
	Currency                     string                  `json:"ccy"`
	DeltaBS                      string                  `json:"deltaBS"` // delta：Black-Scholes Greeks in dollars,only applicable to OPTION
	DeltaPA                      string                  `json:"deltaPA"` // delta：Greeks in coins,only applicable to OPTION
	GammaBS                      string                  `json:"gammaBS"` // gamma：Black-Scholes Greeks in dollars,only applicable to OPTION
	GammaPA                      string                  `json:"gammaPA"` // gamma：Greeks in coins,only applicable to OPTION
	InitialMarginRequirement     convert.StringToFloat64 `json:"imr"`     // Initial margin requirement, only applicable to cross.
	InstrumentID                 string                  `json:"instId"`
	InstrumentType               asset.Item              `json:"instType"`
	Interest                     convert.StringToFloat64 `json:"interest"`
	USDPrice                     convert.StringToFloat64 `json:"usdPx"`
	LastTradePrice               convert.StringToFloat64 `json:"last"`
	Leverage                     convert.StringToFloat64 `json:"lever"`   // Leverage, not applicable to OPTION seller
	Liabilities                  string                  `json:"liab"`    // Liabilities, only applicable to MARGIN.
	LiabilitiesCurrency          string                  `json:"liabCcy"` // Liabilities currency, only applicable to MARGIN.
	LiquidationPrice             convert.StringToFloat64 `json:"liqPx"`   // Estimated liquidation price Not applicable to OPTION
	MarkPrice                    convert.StringToFloat64 `json:"markPx"`
	Margin                       convert.StringToFloat64 `json:"margin"`
	MarginMode                   string                  `json:"mgnMode"`
	MarginRatio                  convert.StringToFloat64 `json:"mgnRatio"`
	MaintenanceMarginRequirement convert.StringToFloat64 `json:"mmr"`         // Maintenance margin requirement in USD level Applicable to Multi-currency margin and Portfolio margin
	NotionalUsd                  convert.StringToFloat64 `json:"notionalUsd"` // Quality of Positions -- usd
	OptionValue                  convert.StringToFloat64 `json:"optVal"`      // Option Value, only application to position.
	QuantityOfPosition           convert.StringToFloat64 `json:"pos"`         // Quantity of positions,In the mode of autonomous transfer from position to position, after the deposit is transferred, a position with pos of 0 will be generated
	PositionCurrency             string                  `json:"posCcy"`
	PositionID                   string                  `json:"posId"`
	PositionSide                 string                  `json:"posSide"`
	ThetaBS                      string                  `json:"thetaBS"` // theta：Black-Scholes Greeks in dollars,only applicable to OPTION
	ThetaPA                      string                  `json:"thetaPA"` // theta：Greeks in coins,only applicable to OPTION
	TradeID                      string                  `json:"tradeId"`
	UpdatedTime                  convert.ExchangeTime    `json:"uTime"`    // Latest time position was adjusted,
	UPNL                         convert.StringToFloat64 `json:"upl"`      // Unrealized profit and loss
	UPLRatio                     convert.StringToFloat64 `json:"uplRatio"` // Unrealized profit and loss ratio
	VegaBS                       string                  `json:"vegaBS"`   // vega：Black-Scholes Greeks in dollars,only applicable to OPTION
	VegaPA                       string                  `json:"vegaPA"`   // vega：Greeks in coins,only applicable to OPTION

	// PushTime added feature in the websocket push data.

	PushTime convert.ExchangeTime `json:"pTime"` // The time when the account position data is pushed.
}

// AccountPositionHistory hold account position history.
type AccountPositionHistory struct {
	CreationTime       convert.ExchangeTime    `json:"cTime"`
	Currency           string                  `json:"ccy"`
	CloseAveragePrice  convert.StringToFloat64 `json:"closeAvgPx"`
	CloseTotalPosition convert.StringToFloat64 `json:"closeTotalPos"`
	InstrumentID       string                  `json:"instId"`
	InstrumentType     string                  `json:"instType"`
	Leverage           string                  `json:"lever"`
	ManagementMode     string                  `json:"mgnMode"`
	OpenAveragePrice   string                  `json:"openAvgPx"`
	OpenMaxPosition    string                  `json:"openMaxPos"`
	ProfitAndLoss      convert.StringToFloat64 `json:"pnl"`
	ProfitAndLossRatio convert.StringToFloat64 `json:"pnlRatio"`
	PositionID         string                  `json:"posId"`
	PositionSide       string                  `json:"posSide"`
	TriggerPrice       string                  `json:"triggerPx"`
	Type               string                  `json:"type"`
	UpdateTime         convert.ExchangeTime    `json:"uTime"`
	Underlying         string                  `json:"uly"`
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
	Timestamp          convert.ExchangeTime `json:"ts"`
}

// BillsDetailQueryParameter represents bills detail query parameter
type BillsDetailQueryParameter struct {
	InstrumentType string // Instrument type "SPOT" "MARGIN" "SWAP" "FUTURES" "OPTION"
	Currency       string
	MarginMode     string // Margin mode "isolated" "cross"
	ContractType   string // Contract type "linear" & "inverse" Only applicable to FUTURES/SWAP
	BillType       uint64 // Bill type 1: Transfer 2: Trade 3: Delivery 4: Auto token conversion 5: Liquidation 6: Margin transfer 7: Interest deduction 8: Funding fee 9: ADL 10: Clawback 11: System token conversion 12: Strategy transfer 13: ddh
	BillSubType    int64  // allowed bill subtype values are [ 1,2,3,4,5,6,9,11,12,14,160,161,162,110,111,118,119,100,101,102,103,104,105,106,110,125,126,127,128,131,132,170,171,172,112,113,117,173,174,200,201,202,203 ], link: https://www.okx.com/docs-v5/en/#rest-api-account-get-bills-details-last-7-days
	After          string
	Before         string
	BeginTime      time.Time
	EndTime        time.Time
	Limit          int64
}

// BillsDetailResponse represents account bills information.
type BillsDetailResponse struct {
	Balance                    convert.StringToFloat64 `json:"bal"`
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
	Timestamp                  convert.ExchangeTime    `json:"ts"`
	Type                       string                  `json:"type"`
}

// AccountConfigurationResponse represents account configuration response.
type AccountConfigurationResponse struct {
	AccountLevel         uint64 `json:"acctLv,string"` // 1: Simple 2: Single-currency margin 3: Multi-currency margin 4：Portfolio margin
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
	Leverage     float64 `json:"lever,string"`     // set leverage for isolated
	MarginMode   string  `json:"mgnMode"`          // Margin Mode "cross" and "isolated"
	InstrumentID string  `json:"instId,omitempty"` // Optional:
	Currency     string  `json:"ccy,omitempty"`    // Optional:
	PositionSide string  `json:"posSide,omitempty"`
}

// SetLeverageResponse represents set leverage response
type SetLeverageResponse struct {
	Leverage     convert.StringToFloat64 `json:"lever"`
	MarginMode   string                  `json:"mgnMode"` // Margin Mode "cross" and "isolated"
	InstrumentID string                  `json:"instId"`
	PositionSide string                  `json:"posSide"` // "long", "short", and "net"
}

// MaximumBuyAndSell get maximum buy , sell amount or open amount
type MaximumBuyAndSell struct {
	Currency     string                  `json:"ccy"`
	InstrumentID string                  `json:"instId"`
	MaximumBuy   convert.StringToFloat64 `json:"maxBuy"`
	MaximumSell  convert.StringToFloat64 `json:"maxSell"`
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
	Amount       convert.StringToFloat64 `json:"amt"`
	Currency     string                  `json:"ccy"`
	InstrumentID string                  `json:"instId"`
	Leverage     convert.StringToFloat64 `json:"leverage"`
	PosSide      string                  `json:"posSide"`
	Type         string                  `json:"type"`
}

// LeverageResponse instrument id leverage response.
type LeverageResponse struct {
	InstrumentID string                  `json:"instId"`
	MarginMode   string                  `json:"mgnMode"`
	PositionSide string                  `json:"posSide"`
	Leverage     convert.StringToFloat64 `json:"lever"`
}

// LeverageEstimatedInfo leverage estimated info response.
type LeverageEstimatedInfo struct {
	EstimatedAvailQuoteTrans string                  `json:"estAvailQuoteTrans"`
	EstimatedAvailTrans      string                  `json:"estAvailTrans"`
	EstimatedLiqPrice        convert.StringToFloat64 `json:"estLiqPx"`
	EstimatedMaxAmount       convert.StringToFloat64 `json:"estMaxAmt"`
	EstimatedMargin          string                  `json:"estMgn"`
	EstimatedQuoteMaxAmount  convert.StringToFloat64 `json:"estQuoteMaxAmt"`
	EstimatedQuoteMgn        string                  `json:"estQuoteMgn"`
	ExistOrd                 bool                    `json:"existOrd"` // Whether there is pending orders
	MaxLeverage              convert.StringToFloat64 `json:"maxLever"`
	MinLeverage              convert.StringToFloat64 `json:"minLever"`
}

// MaximumLoanInstrument represents maximum loan of an instrument id.
type MaximumLoanInstrument struct {
	InstrumentID string     `json:"instId"`
	MgnMode      string     `json:"mgnMode"`
	MgnCcy       string     `json:"mgnCcy"`
	MaxLoan      string     `json:"maxLoan"`
	Currency     string     `json:"ccy"`
	Side         order.Side `json:"side"`
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
	Timestamp        convert.ExchangeTime    `json:"ts"`
}

// InterestAccruedData represents interest rate accrued response
type InterestAccruedData struct {
	Currency     string               `json:"ccy"`
	InstrumentID string               `json:"instId"`
	Interest     string               `json:"interest"`
	InterestRate string               `json:"interestRate"` // Interest rate in an hour.
	Liability    string               `json:"liab"`
	MarginMode   string               `json:"mgnMode"` //  	Margin mode "cross" "isolated"
	Timestamp    convert.ExchangeTime `json:"ts"`
	LoanType     string               `json:"type"`
}

// VIPInterestData holds interest accrued/deducted data
type VIPInterestData struct {
	Currency     string                  `json:"ccy"`
	Interest     convert.StringToFloat64 `json:"interest"`
	InterestRate convert.StringToFloat64 `json:"interestRate"`
	Liability    convert.StringToFloat64 `json:"liab"`
	OrderID      string                  `json:"ordId"`
	Timestamp    convert.ExchangeTime    `json:"ts"`
}

// VIPLoanOrder holds VIP loan items
type VIPLoanOrder struct {
	BorrowAmount    convert.StringToFloat64 `json:"borrowAmt"`
	Currency        string                  `json:"ccy"`
	CurrentRate     convert.StringToFloat64 `json:"curRate"`
	DueAmount       convert.StringToFloat64 `json:"dueAmt"`
	NextRefreshTime convert.ExchangeTime    `json:"nextRefreshTime"`
	OrderID         string                  `json:"ordId"`
	OriginalRate    convert.StringToFloat64 `json:"origRate"`
	RepayAmount     convert.StringToFloat64 `json:"repayAmt"`
	State           string                  `json:"state"`
	Timestamp       convert.ExchangeTime    `json:"ts"`
}

// VIPLoanOrderDetail holds vip loan order detail
type VIPLoanOrderDetail struct {
	Amount     convert.StringToFloat64 `json:"amt"`
	Currency   string                  `json:"ccy"`
	FailReason string                  `json:"failReason"`
	Rate       convert.StringToFloat64 `json:"rate"`
	Timestamp  convert.ExchangeTime    `json:"ts"`
	Type       string                  `json:"type"` // Operation Type: 1:Borrow 2:Repayment 3:System Repayment 4:Interest Rate Refresh
}

// InterestRateResponse represents interest rate response.
type InterestRateResponse struct {
	InterestRate convert.StringToFloat64 `json:"interestRate"`
	Currency     string                  `json:"ccy"`
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

// BorrowAndRepay manual holds manual borrow and repay in quick margin mode.
type BorrowAndRepay struct {
	Amount       float64 `json:"amt,string"`
	InstrumentID string  `json:"instId"`
	LoanCcy      string  `json:"ccy"`
	Side         string  `json:"side"` // possible values: 'borrow' and 'repay'
}

// BorrowRepayHistoryItem holds borrow or repay history item information
type BorrowRepayHistoryItem struct {
	InstID          string                  `json:"instId"`
	Ccy             string                  `json:"ccy"`
	Side            string                  `json:"side"`
	AccBorrowAmount convert.StringToFloat64 `json:"accBorrowed"`
	Amount          convert.StringToFloat64 `json:"amt"`
	RefID           string                  `json:"refId"`
	Timestamp       convert.ExchangeTime    `json:"ts"`
}

// MaximumWithdrawal represents maximum withdrawal amount query response.
type MaximumWithdrawal struct {
	Currency            string `json:"ccy"`
	MaximumWithdrawal   string `json:"maxWd"`   // Max withdrawal (not allowing borrowed crypto transfer out under Multi-currency margin)
	MaximumWithdrawalEx string `json:"maxWdEx"` // Max withdrawal (allowing borrowed crypto transfer out under Multi-currency margin)
}

// AccountRiskState represents account risk state.
type AccountRiskState struct {
	IsTheAccountAtRisk bool                 `json:"atRisk"`
	AtRiskIdx          []interface{}        `json:"atRiskIdx"` // derivatives risk unit list
	AtRiskMgn          []interface{}        `json:"atRiskMgn"` // margin risk unit list
	Timestamp          convert.ExchangeTime `json:"ts"`
}

// LoanBorrowAndReplayInput represents currency VIP borrow or repay request params.
type LoanBorrowAndReplayInput struct {
	Currency string  `json:"ccy"`
	Side     string  `json:"side,omitempty"`
	Amount   float64 `json:"amt,string,omitempty"`
}

// LoanBorrowAndReplay loans borrow and repay
type LoanBorrowAndReplay struct {
	Amount        string                  `json:"amt"`
	AvailableLoan convert.StringToFloat64 `json:"availLoan"`
	Currency      string                  `json:"ccy"`
	LoanQuota     convert.StringToFloat64 `json:"loanQuota"`
	PosLoan       string                  `json:"posLoan"`
	Side          string                  `json:"side"` // borrow or repay
	UsedLoan      string                  `json:"usedLoan"`
}

// BorrowRepayHistory represents borrow and repay history item data
type BorrowRepayHistory struct {
	Currency   string               `json:"ccy"`
	TradedLoan string               `json:"tradedLoan"`
	Timestamp  convert.ExchangeTime `json:"ts"`
	Type       string               `json:"type"`
	UsedLoan   string               `json:"usedLoan"`
}

// BorrowInterestAndLimitResponse represents borrow interest and limit rate for different loan type.
type BorrowInterestAndLimitResponse struct {
	Debt             string               `json:"debt"`
	Interest         string               `json:"interest"`
	NextDiscountTime convert.ExchangeTime `json:"nextDiscountTime"`
	NextInterestTime convert.ExchangeTime `json:"nextInterestTime"`
	Records          []struct {
		AvailLoan  string                  `json:"availLoan"`
		Currency   string                  `json:"ccy"`
		Interest   string                  `json:"interest"`
		LoanQuota  string                  `json:"loanQuota"`
		PosLoan    string                  `json:"posLoan"` // Frozen amount for current account Only applicable to VIP loans
		Rate       convert.StringToFloat64 `json:"rate"`
		SurplusLmt string                  `json:"surplusLmt"`
		UsedLmt    convert.StringToFloat64 `json:"usedLmt"`
		UsedLoan   string                  `json:"usedLoan"`
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
	PositionsCount         uint64         `json:"pos,omitempty"`
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
	Timestamp                    convert.ExchangeTime  `json:"ts"`
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
	ThetaBS   string               `json:"thetaBS"`
	ThetaPA   string               `json:"thetaPA"`
	DeltaBS   string               `json:"deltaBS"`
	DeltaPA   string               `json:"deltaPA"`
	GammaBS   string               `json:"gammaBS"`
	GammaPA   string               `json:"gammaPA"`
	VegaBS    string               `json:"vegaBS"`
	VegaPA    string               `json:"vegaPA"`
	Currency  string               `json:"ccy"`
	Timestamp convert.ExchangeTime `json:"ts"`
}

// CounterpartiesResponse represents
type CounterpartiesResponse struct {
	TraderName string `json:"traderName"`
	TraderCode string `json:"traderCode"`
	Type       string `json:"type"`
}

// RfqOrderLeg represents Rfq Order responses leg.
type RfqOrderLeg struct {
	Size         string `json:"sz"`
	Side         string `json:"side"`
	InstrumentID string `json:"instId"`
	TgtCurrency  string `json:"tgtCcy,omitempty"`
}

// CreateRfqInput Rfq create method input.
type CreateRfqInput struct {
	Anonymous      bool          `json:"anonymous"`
	CounterParties []string      `json:"counterparties"`
	ClientRfqID    string        `json:"clRfqId"`
	Legs           []RfqOrderLeg `json:"legs"`
}

// CancelRfqRequestParam represents cancel Rfq order request params
type CancelRfqRequestParam struct {
	RfqID       string `json:"rfqId"`
	ClientRfqID string `json:"clRfqId"`
}

// CancelRfqRequestsParam represents cancel multiple Rfq orders request params
type CancelRfqRequestsParam struct {
	RfqIDs       []string `json:"rfqIds"`
	ClientRfqIDs []string `json:"clRfqIds"`
}

// CancelRfqResponse represents cancel Rfq orders response
type CancelRfqResponse struct {
	RfqID       string `json:"rfqId"`
	ClientRfqID string `json:"clRfqId"`
	StatusCode  string `json:"sCode"`
	StatusMsg   string `json:"sMsg"`
}

// TimestampResponse holds timestamp response only.
type TimestampResponse struct {
	Timestamp convert.ExchangeTime `json:"ts"`
}

// MMPStatusResponse holds MMP reset status response
type MMPStatusResponse struct {
	Result bool `json:"result"`
}

// MMPConfig holds request/response structure to set Market Maker Protection (MMP)
type MMPConfig struct {
	InstrumentFamily string  `json:"instFamily"`
	TimeInterval     int64   `json:"timeInterval,string"`
	FrozenInterval   int64   `json:"frozenInterval,string"` // Frozen period (ms). "0" means the trade will remain frozen until you request "Reset MMP Status" to unfrozen
	QuantityLimit    float64 `json:"qtyLimit,string"`
}

// MMPConfigDetail holds MMP config details.
type MMPConfigDetail struct {
	FrozenInterval   int64                   `json:"frozenInterval,string"`
	InstrumentFamily string                  `json:"instFamily"`
	MMPFrozen        bool                    `json:"mmpFrozen"`
	MMPFrozenUntil   string                  `json:"mmpFrozenUntil"`
	QuantityLimit    convert.StringToFloat64 `json:"qtyLimit"`
	TimeInterval     int64                   `json:"timeInterval"`
}

// ExecuteQuoteParams represents Execute quote request params
type ExecuteQuoteParams struct {
	RfqID   string `json:"rfqId"`
	QuoteID string `json:"quoteId"`
}

// ExecuteQuoteResponse represents execute quote response.
type ExecuteQuoteResponse struct {
	BlockTradedID   string               `json:"blockTdId"`
	RfqID           string               `json:"rfqId"`
	ClientRfqID     string               `json:"clRfqId"`
	QuoteID         string               `json:"quoteId"`
	ClientQuoteID   string               `json:"clQuoteId"`
	TraderCode      string               `json:"tTraderCode"`
	MakerTraderCode string               `json:"mTraderCode"`
	CreationTime    convert.ExchangeTime `json:"cTime"`
	Legs            []OrderLeg           `json:"legs"`
}

// QuoteProduct represents products which makers want to quote and receive RFQs for
type QuoteProduct struct {
	InstrumentType string `json:"instType,omitempty"`
	IncludeALL     bool   `json:"includeALL"`
	Data           []struct {
		Underlying     string                  `json:"uly"`
		MaxBlockSize   convert.StringToFloat64 `json:"maxBlockSz"`
		MakerPriceBand convert.StringToFloat64 `json:"makerPxBand"`
	} `json:"data"`
	InstrumentType0 string `json:"instType:,omitempty"`
}

// OrderLeg represents legs information for both websocket and REST available Quote information.
type OrderLeg struct {
	Price          string `json:"px"`
	Size           string `json:"sz"`
	InstrumentID   string `json:"instId"`
	Side           string `json:"side"`
	TargetCurrency string `json:"tgtCcy"`

	// available in REST only
	Fee         convert.StringToFloat64 `json:"fee"`
	FeeCurrency string                  `json:"feeCcy"`
	TradeID     string                  `json:"tradeId"`
}

// CreateQuoteParams holds information related to create quote.
type CreateQuoteParams struct {
	RfqID         string     `json:"rfqId"`
	ClientQuoteID string     `json:"clQuoteId"`
	QuoteSide     order.Side `json:"quoteSide"`
	Legs          []QuoteLeg `json:"legs"`
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
	CreationTime  convert.ExchangeTime `json:"cTime"`
	UpdateTime    convert.ExchangeTime `json:"uTime"`
	ValidUntil    convert.ExchangeTime `json:"validUntil"`
	QuoteID       string               `json:"quoteId"`
	ClientQuoteID string               `json:"clQuoteId"`
	RfqID         string               `json:"rfqId"`
	QuoteSide     string               `json:"quoteSide"`
	ClientRfqID   string               `json:"clRfqId"`
	TraderCode    string               `json:"traderCode"`
	State         string               `json:"state"`
	Legs          []QuoteLeg           `json:"legs"`
}

// CancelQuoteRequestParams represents cancel quote request params
type CancelQuoteRequestParams struct {
	QuoteID       string `json:"quoteId"`
	ClientQuoteID string `json:"clQuoteId"`
}

// CancelQuotesRequestParams represents cancel multiple quotes request params
type CancelQuotesRequestParams struct {
	QuoteIDs       []string `json:"quoteIds,omitempty"`
	ClientQuoteIDs []string `json:"clQuoteIds,omitempty"`
}

// CancelQuoteResponse represents cancel quote response
type CancelQuoteResponse struct {
	QuoteID       string `json:"quoteId"`
	ClientQuoteID string `json:"clQuoteId"`
	SCode         string `json:"sCode"`
	SMsg          string `json:"sMsg"`
}

// RfqRequestParams represents get Rfq orders param
type RfqRequestParams struct {
	RfqID       string
	ClientRfqID string
	State       string
	BeginningID string
	EndID       string
	Limit       int64
}

// RfqResponse Rfq response detail.
type RfqResponse struct {
	CreateTime     convert.ExchangeTime `json:"cTime"`
	UpdateTime     convert.ExchangeTime `json:"uTime"`
	ValidUntil     convert.ExchangeTime `json:"validUntil"`
	TraderCode     string               `json:"traderCode"`
	RfqID          string               `json:"rfqId"`
	ClientRfqID    string               `json:"clRfqId"`
	State          string               `json:"state"`
	Counterparties []string             `json:"counterparties"`
	Legs           []struct {
		InstrumentID string                  `json:"instId"`
		Size         convert.StringToFloat64 `json:"sz"`
		Side         string                  `json:"side"`
		TgtCcy       string                  `json:"tgtCcy"`
	} `json:"legs"`
}

// QuoteRequestParams request params.
type QuoteRequestParams struct {
	RfqID         string
	ClientRfqID   string
	QuoteID       string
	ClientQuoteID string
	State         string
	BeginID       string
	EndID         string
	Limit         int64
}

// RfqTradesRequestParams represents Rfq trades request param
type RfqTradesRequestParams struct {
	RfqID         string
	ClientRfqID   string
	QuoteID       string
	BlockTradeID  string
	ClientQuoteID string
	State         string
	BeginID       string
	EndID         string
	Limit         int64
}

// RfqTradeResponse Rfq trade response
type RfqTradeResponse struct {
	RfqID           string               `json:"rfqId"`
	ClientRfqID     string               `json:"clRfqId"`
	QuoteID         string               `json:"quoteId"`
	ClientQuoteID   string               `json:"clQuoteId"`
	BlockTradeID    string               `json:"blockTdId"`
	Legs            []RfqTradeLeg        `json:"legs"`
	CreationTime    convert.ExchangeTime `json:"cTime"`
	TakerTraderCode string               `json:"tTraderCode"`
	MakerTraderCode string               `json:"mTraderCode"`
}

// RfqTradeLeg RFQ trade response leg.
type RfqTradeLeg struct {
	InstrumentID string                  `json:"instId"`
	Side         string                  `json:"side"`
	Size         string                  `json:"sz"`
	Price        convert.StringToFloat64 `json:"px"`
	TradeID      string                  `json:"tradeId"`

	Fee         convert.StringToFloat64 `json:"fee"`
	FeeCurrency string                  `json:"feeCcy"`
}

// PublicTradesResponse represents data will be pushed whenever there is a block trade.
type PublicTradesResponse struct {
	BlockTradeID string               `json:"blockTdId"`
	CreationTime convert.ExchangeTime `json:"cTime"`
	Legs         []RfqTradeLeg        `json:"legs"`
}

// SubaccountInfo represents subaccount information detail.
type SubaccountInfo struct {
	Enable          bool                 `json:"enable"`
	SubAccountName  string               `json:"subAcct"`
	SubaccountType  string               `json:"type"` // sub-account note
	SubaccountLabel string               `json:"label"`
	MobileNumber    string               `json:"mobile"`      // Mobile number that linked with the sub-account.
	GoogleAuth      bool                 `json:"gAuth"`       // If the sub-account switches on the Google Authenticator for login authentication.
	CanTransferOut  bool                 `json:"canTransOut"` // If can transfer out, false: can not transfer out, true: can transfer.
	Timestamp       convert.ExchangeTime `json:"ts"`
}

// SubaccountBalanceDetail represents subaccount balance detail
type SubaccountBalanceDetail struct {
	AvailableBalance               convert.StringToFloat64 `json:"availBal"`
	AvailableEquity                convert.StringToFloat64 `json:"availEq"`
	CashBalance                    convert.StringToFloat64 `json:"cashBal"`
	Currency                       string                  `json:"ccy"`
	CrossLiability                 string                  `json:"crossLiab"`
	DiscountEquity                 string                  `json:"disEq"`
	Equity                         string                  `json:"eq"`
	EquityUsd                      string                  `json:"eqUsd"`
	FrozenBalance                  convert.StringToFloat64 `json:"frozenBal"`
	Interest                       string                  `json:"interest"`
	IsoEquity                      string                  `json:"isoEq"`
	IsolatedLiabilities            string                  `json:"isoLiab"`
	LiabilitiesOfCurrency          string                  `json:"liab"`
	MaxLoan                        string                  `json:"maxLoan"`
	MarginRatio                    convert.StringToFloat64 `json:"mgnRatio"`
	NotionalLeverage               string                  `json:"notionalLever"`
	OrdFrozen                      string                  `json:"ordFrozen"`
	Twap                           string                  `json:"twap"`
	UpdateTime                     convert.ExchangeTime    `json:"uTime"`
	UnrealizedProfitAndLoss        convert.StringToFloat64 `json:"upl"`
	UnrealizedProfitAndLiabilities string                  `json:"uplLiab"`
}

// SubaccountBalanceResponse represents subaccount balance response
type SubaccountBalanceResponse struct {
	AdjustedEffectiveEquity      string                    `json:"adjEq"`
	Details                      []SubaccountBalanceDetail `json:"details"`
	Imr                          string                    `json:"imr"`
	IsolatedMarginEquity         string                    `json:"isoEq"`
	MarginRatio                  convert.StringToFloat64   `json:"mgnRatio"`
	MaintenanceMarginRequirement string                    `json:"mmr"`
	NotionalUsd                  string                    `json:"notionalUsd"`
	OrdFroz                      string                    `json:"ordFroz"`
	TotalEq                      string                    `json:"totalEq"`
	UpdateTime                   convert.ExchangeTime      `json:"uTime"`
}

// FundingBalance holds function balance.
type FundingBalance struct {
	AvailableBalance convert.StringToFloat64 `json:"availBal"`
	Balance          convert.StringToFloat64 `json:"bal"`
	Currency         string                  `json:"ccy"`
	FrozenBalance    convert.StringToFloat64 `json:"frozenBal"`
}

// SubAccountMaximumWithdrawal holds sub-account maximum withdrawal information
type SubAccountMaximumWithdrawal struct {
	Currency          string                  `json:"ccy"`
	MaxWd             convert.StringToFloat64 `json:"maxWd"`
	MaxWdEx           string                  `json:"maxWdEx"`
	SpotOffsetMaxWd   convert.StringToFloat64 `json:"spotOffsetMaxWd"`
	SpotOffsetMaxWdEx convert.StringToFloat64 `json:"spotOffsetMaxWdEx"`
}

// SubaccountBillItem represents subaccount balance bill item
type SubaccountBillItem struct {
	BillID                 string                  `json:"billId"`
	Type                   string                  `json:"type"`
	AccountCurrencyBalance string                  `json:"ccy"`
	Amount                 convert.StringToFloat64 `json:"amt"`
	SubAccount             string                  `json:"subAcct"`
	Timestamp              convert.ExchangeTime    `json:"ts"`
}

// SubAccountTransfer holds sub-account transfer instance.
type SubAccountTransfer struct {
	BillID     string                  `json:"billId"`
	Type       string                  `json:"type"`
	Currency   string                  `json:"ccy"`
	Amount     convert.StringToFloat64 `json:"amt"`
	SubAccount string                  `json:"subAcct"`
	SubUID     string                  `json:"subUid"`
	Timestamp  convert.ExchangeTime    `json:"ts"`
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

// SubAccountLoanAllocationParam holds parameter for VIP sub-account loan allocation
type SubAccountLoanAllocationParam struct {
	Enable bool                              `json:"enable"`
	Alloc  []subAccountVIPLoanAllocationInfo `json:"alloc"`
}

type subAccountVIPLoanAllocationInfo struct {
	SubAcct   string  `json:"subAcct"`
	LoanAlloc float64 `json:"loanAlloc,string"`
}

// SubAccounBorrowInterestAndLimit represents sub-account borrow interest and limit
type SubAccounBorrowInterestAndLimit struct {
	SubAcct          string                  `json:"subAcct"`
	Debt             convert.StringToFloat64 `json:"debt"`
	Interest         convert.StringToFloat64 `json:"interest"`
	NextDiscountTime convert.ExchangeTime    `json:"nextDiscountTime"`
	NextInterestTime convert.ExchangeTime    `json:"nextInterestTime"`
	LoanAlloc        convert.StringToFloat64 `json:"loanAlloc"`
	Records          []struct {
		AvailLoan         string                  `json:"availLoan"`
		Ccy               string                  `json:"ccy"`
		Interest          convert.StringToFloat64 `json:"interest"`
		LoanQuota         convert.StringToFloat64 `json:"loanQuota"`
		PosLoan           string                  `json:"posLoan"`
		Rate              convert.StringToFloat64 `json:"rate"`
		SurplusLmt        string                  `json:"surplusLmt"`
		SurplusLmtDetails struct {
			AllAcctRemainingQuota convert.StringToFloat64 `json:"allAcctRemainingQuota"`
			CurAcctRemainingQuota convert.StringToFloat64 `json:"curAcctRemainingQuota"`
			PlatRemainingQuota    convert.StringToFloat64 `json:"platRemainingQuota"`
		} `json:"surplusLmtDetails"`
		UsedLmt  convert.StringToFloat64 `json:"usedLmt"`
		UsedLoan convert.StringToFloat64 `json:"usedLoan"`
	} `json:"records"`
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

// StopGridAlgoOrderParam holds stop grid algo order parameter
type StopGridAlgoOrderParam struct {
	AlgoID        string `json:"algoId"`
	InstrumentID  string `json:"instId"`
	StopType      string `json:"stopType"`
	AlgoOrderType string `json:"algoOrdType"`
}

// ClosePositionParams holds close position parameters
type ClosePositionParams struct {
	AlgoID                  string  `json:"algoId"`
	MarketCloseAllPositions bool    `json:"mktClose"` // true: Market close all position, false：Close part of position
	Size                    float64 `json:"sz,omitempty,string"`
	Price                   float64 `json:"px,omitempty,string"`
}

// ClosePositionContractGridResponse holds contract grid close position response data
type ClosePositionContractGridResponse struct {
	AlgoClientOrderID string `json:"algoClOrdId"`
	AlgoID            string `json:"algoId"`
	OrderID           string `json:"ordId"`
	Tag               string `json:"tag"`
}

// CancelClosePositionOrder holds close position order parameter cancellation parameter
type CancelClosePositionOrder struct {
	AlgoID  string `json:"algoId"`
	OrderID string `json:"ordId"` // Close position order ID
}

// TriggeredGridAlgoOrderInfo holds grid algo order info
type TriggeredGridAlgoOrderInfo struct {
	AlgoClientOrderID string `json:"algoClOrdId"`
	AlgoID            string `json:"algoId"`
}

// GridAlgoOrderAmend represents amend algo order response
type GridAlgoOrderAmend struct {
	AlgoID                 string                  `json:"algoId"`
	InstrumentID           string                  `json:"instId"`
	StopLossTriggerPrice   convert.StringToFloat64 `json:"slTriggerPx"`
	TakeProfitTriggerPrice convert.StringToFloat64 `json:"tpTriggerPx"`
}

// StopGridAlgoOrderRequest represents stop grid algo order request parameter
type StopGridAlgoOrderRequest struct {
	AlgoID        string `json:"algoId"`
	InstrumentID  string `json:"instId"`
	StopType      uint64 `json:"stopType,string"` // Spot grid "1": Sell base currency "2": Keep base currency | Contract grid "1": Market Close All positions "2": Keep positions
	AlgoOrderType string `json:"algoOrdType"`
}

// GridAlgoOrderResponse a complete information of grid algo order item response.
type GridAlgoOrderResponse struct {
	ActualLever               string                  `json:"actualLever"`
	AlgoID                    string                  `json:"algoId"`
	AlgoOrderType             string                  `json:"algoOrdType"`
	ArbitrageNumber           string                  `json:"arbitrageNum"`
	BasePosition              bool                    `json:"basePos"`
	BaseSize                  convert.StringToFloat64 `json:"baseSz"`
	CancelType                string                  `json:"cancelType"`
	Direction                 string                  `json:"direction"`
	FloatProfit               string                  `json:"floatProfit"`
	GridQuantity              convert.StringToFloat64 `json:"gridNum"`
	GridProfit                string                  `json:"gridProfit"`
	InstrumentID              string                  `json:"instId"`
	InstrumentType            string                  `json:"instType"`
	Investment                string                  `json:"investment"`
	Leverage                  string                  `json:"lever"`
	EstimatedLiquidationPrice convert.StringToFloat64 `json:"liqPx"`
	MaximumPrice              convert.StringToFloat64 `json:"maxPx"`
	MinimumPrice              convert.StringToFloat64 `json:"minPx"`
	ProfitAndLossRatio        convert.StringToFloat64 `json:"pnlRatio"`
	QuoteSize                 convert.StringToFloat64 `json:"quoteSz"`
	RunType                   string                  `json:"runType"`
	StopLossTriggerPrice      convert.StringToFloat64 `json:"slTriggerPx"`
	State                     string                  `json:"state"`
	StopResult                string                  `json:"stopResult,omitempty"`
	StopType                  string                  `json:"stopType"`
	Size                      string                  `json:"sz"`
	Tag                       string                  `json:"tag"`
	TotalProfitAndLoss        string                  `json:"totalPnl"`
	TakeProfitTriggerPrice    convert.StringToFloat64 `json:"tpTriggerPx"`
	CreationTime              convert.ExchangeTime    `json:"cTime"`
	UpdateTime                convert.ExchangeTime    `json:"uTime"`
	Underlying                string                  `json:"uly"`

	// Added in Detail

	EquityOfStrength    string                  `json:"eq,omitempty"`
	PerMaxProfitRate    convert.StringToFloat64 `json:"perMaxProfitRate,omitempty"`
	PerMinProfitRate    convert.StringToFloat64 `json:"perMinProfitRate,omitempty"`
	Profit              string                  `json:"profit,omitempty"`
	Runpx               string                  `json:"runpx,omitempty"`
	SingleAmt           convert.StringToFloat64 `json:"singleAmt,omitempty"`
	TotalAnnualizedRate convert.StringToFloat64 `json:"totalAnnualizedRate,omitempty"`
	TradeNumber         string                  `json:"tradeNum,omitempty"`

	// Suborders Detail

	AnnualizedRate convert.StringToFloat64 `json:"annualizedRate,omitempty"`
	CurBaseSize    convert.StringToFloat64 `json:"curBaseSz,omitempty"`
	CurQuoteSize   convert.StringToFloat64 `json:"curQuoteSz,omitempty"`
}

// AlgoOrderPosition represents algo order position detailed data.
type AlgoOrderPosition struct {
	AutoDecreasingLine           string                  `json:"adl"`
	AlgoID                       string                  `json:"algoId"`
	AveragePrice                 convert.StringToFloat64 `json:"avgPx"`
	Currency                     string                  `json:"ccy"`
	InitialMarginRequirement     string                  `json:"imr"`
	InstrumentID                 string                  `json:"instId"`
	InstrumentType               string                  `json:"instType"`
	LastTradedPrice              convert.StringToFloat64 `json:"last"`
	Leverage                     convert.StringToFloat64 `json:"lever"`
	LiquidationPrice             convert.StringToFloat64 `json:"liqPx"`
	MarkPrice                    convert.StringToFloat64 `json:"markPx"`
	MarginMode                   string                  `json:"mgnMode"`
	MarginRatio                  convert.StringToFloat64 `json:"mgnRatio"`
	MaintenanceMarginRequirement string                  `json:"mmr"`
	NotionalUSD                  string                  `json:"notionalUsd"`
	QuantityPosition             string                  `json:"pos"`
	PositionSide                 string                  `json:"posSide"`
	UnrealizedProfitAndLoss      convert.StringToFloat64 `json:"upl"`
	UnrealizedProfitAndLossRatio convert.StringToFloat64 `json:"uplRatio"`
	UpdateTime                   convert.ExchangeTime    `json:"uTime"`
	CreationTime                 convert.ExchangeTime    `json:"cTime"`
}

// AlgoOrderWithdrawalProfit algo withdrawal order profit info.
type AlgoOrderWithdrawalProfit struct {
	AlgoID         string `json:"algoId"`
	WithdrawProfit string `json:"profit"`
}

// SystemStatusResponse represents the system status and other details.
type SystemStatusResponse struct {
	Title               string               `json:"title"`
	State               string               `json:"state"`
	Begin               convert.ExchangeTime `json:"begin"` // Begin time of system maintenance,
	End                 convert.ExchangeTime `json:"end"`   // Time of resuming trading totally.
	Href                string               `json:"href"`  // Hyperlink for system maintenance details
	ServiceType         string               `json:"serviceType"`
	System              string               `json:"system"`
	ScheduleDescription string               `json:"scheDesc"`

	// PushTime timestamp information when the data is pushed
	PushTime convert.ExchangeTime `json:"ts"`
}

// BlockTicker holds block trading information.
type BlockTicker struct {
	InstrumentType           string                  `json:"instType"`
	InstrumentID             string                  `json:"instId"`
	TradingVolumeInCCY24Hour convert.StringToFloat64 `json:"volCcy24h"`
	TradingVolumeInUSD24Hour convert.StringToFloat64 `json:"vol24h"`
	Timestamp                convert.ExchangeTime    `json:"ts"`
}

// BlockTrade represents a block trade.
type BlockTrade struct {
	FillVolume   convert.StringToFloat64 `json:"fillVol"`
	ForwardPrice convert.StringToFloat64 `json:"fwdPx"`
	InedexPrice  convert.StringToFloat64 `json:"idxPx"`
	MarkPrice    convert.StringToFloat64 `json:"markPx"`
	Side         string                  `json:"side"`
	InstrumentID string                  `json:"instId"`
	TradeID      string                  `json:"tradeId"`
	Price        convert.StringToFloat64 `json:"px"`
	Size         convert.StringToFloat64 `json:"sz"`
	Timestamp    convert.ExchangeTime    `json:"ts"`
}

// SpreadOrderParam holds parameters for spread orders.
type SpreadOrderParam struct {
	InstrumentID  string  `json:"instId"`
	SpreadID      string  `json:"sprdId,omitempty"`
	ClientOrderID string  `json:"clOrdId,omitempty"`
	Side          string  `json:"side"`    // Order side, buy sell
	OrderType     string  `json:"ordType"` // Order type  'limit': Limit order  'post_only': Post-only order 'ioc': Immediate-or-cancel order
	Size          float64 `json:"sz,string"`
	Price         float64 `json:"px,string"`
	Tag           string  `json:"tag,omitempty"`
}

// SpreadOrderResponse represents a spread create order response
type SpreadOrderResponse struct {
	SCode         string `json:"sCode"`
	SMsg          string `json:"sMsg"`
	ClientOrderID string `json:"clOrdId"`
	OrderID       string `json:"ordId"`
	Tag           string `json:"tag"`

	// Added when amending spread order through websocket
	RequestID string `json:"reqId"`
}

// GetSCode returns a status code value
func (a *SpreadOrderResponse) GetSCode() string { return a.SCode }

// GetSMsg returns a status message value
func (a *SpreadOrderResponse) GetSMsg() string { return a.SMsg }

// StatusCodeHolder interface to represent structs which has a status code information
type StatusCodeHolder interface {
	GetSCode() string
	GetSMsg() string
}

// AmendSpreadOrderParam holds amend parameters for spread order
type AmendSpreadOrderParam struct {
	OrderID       string  `json:"ordId"`
	ClientOrderID string  `json:"clOrdId"`
	RequestID     string  `json:"reqId"`
	NewSize       float64 `json:"newSz,omitempty,string"`
	NewPrice      float64 `json:"newPx,omitempty,string"`
}

// SpreadOrder holds spread order details.
type SpreadOrder struct {
	TradeID           string                  `json:"tradeId"`
	InstrumentID      string                  `json:"instId"`
	OrderID           string                  `json:"ordId"`
	SpreadID          string                  `json:"sprdId"`
	ClientOrderID     string                  `json:"clOrdId"`
	Tag               string                  `json:"tag"`
	Price             convert.StringToFloat64 `json:"px"`
	Size              convert.StringToFloat64 `json:"sz"`
	OrderType         string                  `json:"ordType"`
	Side              string                  `json:"side"`
	FillSize          convert.StringToFloat64 `json:"fillSz"`
	FillPrice         convert.StringToFloat64 `json:"fillPx"`
	AccFillSize       convert.StringToFloat64 `json:"accFillSz"`
	PendingFillSize   convert.StringToFloat64 `json:"pendingFillSz"`
	PendingSettleSize convert.StringToFloat64 `json:"pendingSettleSz"`
	CanceledSize      convert.StringToFloat64 `json:"canceledSz"`
	State             string                  `json:"state"`
	AveragePrice      convert.StringToFloat64 `json:"avgPx"`
	CancelSource      string                  `json:"cancelSource"`
	UpdateTime        convert.ExchangeTime    `json:"uTime"`
	CreationTime      convert.ExchangeTime    `json:"cTime"`
}

// SpreadTrade holds spread trade transaction instance
type SpreadTrade struct {
	SpreadID      string                  `json:"sprdId"`
	TradeID       string                  `json:"tradeId"`
	OrderID       string                  `json:"ordId"`
	ClientOrderID string                  `json:"clOrdId"`
	Tag           string                  `json:"tag"`
	FillPrice     convert.StringToFloat64 `json:"fillPx"`
	FillSize      convert.StringToFloat64 `json:"fillSz"`
	State         string                  `json:"state"`
	Side          string                  `json:"side"`
	ExecType      string                  `json:"execType"`
	Timestamp     string                  `json:"ts"`
	Legs          []struct {
		InstrumentID string                  `json:"instId"`
		Price        convert.StringToFloat64 `json:"px"`
		Size         convert.StringToFloat64 `json:"sz"`
		Side         string                  `json:"side"`
		Fee          convert.StringToFloat64 `json:"fee"`
		FeeCcy       string                  `json:"feeCcy"`
		TradeID      string                  `json:"tradeId"`
	} `json:"legs"`
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

// SpreadTradeOrder retrieve all available spreads based on the request parameters.
type SpreadTradeOrder struct {
	SpreadID      string                  `json:"sprdId"`
	SpreadType    string                  `json:"sprdType"`
	State         string                  `json:"state"`
	BaseCurrency  string                  `json:"baseCcy"`
	SizeCurrency  string                  `json:"szCcy"`
	QuoteCurrency string                  `json:"quoteCcy"`
	TickSize      convert.StringToFloat64 `json:"tickSz"`
	MinSize       convert.StringToFloat64 `json:"minSz"`
	LotSize       convert.StringToFloat64 `json:"lotSz"`
	ListTime      string                  `json:"listTime"`
	Legs          []struct {
		InstrumentID string `json:"instId"`
		Side         string `json:"side"`
	} `json:"legs"`
	ExpTime    convert.ExchangeTime `json:"expTime"`
	UpdateTime convert.ExchangeTime `json:"uTime"`
}

// SpreadOrderbook holds spread orderbook information.
type SpreadOrderbook struct {
	// Asks and Bids are [3]string; price, quantity, and # number of orders at the price
	Asks      [][]string           `json:"asks"`
	Bids      [][]string           `json:"bids"`
	Timestamp convert.ExchangeTime `json:"ts"`
}

// SpreadTicker represents a ticker instance.
type SpreadTicker struct {
	SpreadID  string                  `json:"sprdId"`
	Last      convert.StringToFloat64 `json:"last"`
	LastSize  convert.StringToFloat64 `json:"lastSz"`
	AskPrice  convert.StringToFloat64 `json:"askPx"`
	AskSize   convert.StringToFloat64 `json:"askSz"`
	BidPrice  convert.StringToFloat64 `json:"bidPx"`
	BidSize   convert.StringToFloat64 `json:"bidSz"`
	Timestamp convert.ExchangeTime    `json:"ts"`
}

// SpreadPublicTradeItem represents publicly available trade order instance
type SpreadPublicTradeItem struct {
	SprdID    string                  `json:"sprdId"`
	Side      string                  `json:"side"`
	Size      convert.StringToFloat64 `json:"sz"`
	Price     convert.StringToFloat64 `json:"px"`
	TradeID   string                  `json:"tradeId"`
	Timestamp convert.ExchangeTime    `json:"ts"`
}

// UnitConvertResponse unit convert response.
type UnitConvertResponse struct {
	InstrumentID string                  `json:"instId"`
	Price        convert.StringToFloat64 `json:"px"`
	Size         convert.StringToFloat64 `json:"sz"`
	ConvertType  int64                   `json:"type,string"`
	Unit         string                  `json:"unit"`
}

// OptionTickBand holds option band information
type OptionTickBand struct {
	InstrumentType   string `json:"instType"`
	InstrumentFamily string `json:"instFamily"`
	TickBand         []struct {
		MinPrice convert.StringToFloat64 `json:"minPx"`
		MaxPrice convert.StringToFloat64 `json:"maxPx"`
		TickSize convert.StringToFloat64 `json:"tickSz"`
	} `json:"tickBand"`
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

// SubscriptionInfo holds the channel and instrument IDs.
type SubscriptionInfo struct {
	Channel          string `json:"channel"`
	InstrumentID     string `json:"instId,omitempty"`
	InstrumentFamily string `json:"instFamily,omitempty"`
	InstrumentType   string `json:"instType,omitempty"`
	Underlying       string `json:"uly,omitempty"`
	UID              string `json:"uid,omitempty"` // user identifier

	// For Algo Orders
	AlgoID   string `json:"algoId,omitempty"`
	Currency string `json:"ccy,omitempty"`
	SpreadID string `json:"sprdId,omitempty"`
}

// WSSubscriptionInformationList websocket subscription and unsubscription operation inputs.
type WSSubscriptionInformationList struct {
	Operation string             `json:"op"`
	Arguments []SubscriptionInfo `json:"args"`
}

// OperationResponse holds common operation identification
type OperationResponse struct {
	ID        string `json:"id"`
	Operation string `json:"op"`
	Code      string `json:"code"`
	Msg       string `json:"msg"`
}

// WsPlaceOrderResponse place order response thought the websocket connection.
type WsPlaceOrderResponse struct {
	OperationResponse
	Data []OrderData `json:"data"`
}

// SpreadOrderInfo holds spread order response information
// implements the StatusCodeHolder interface
type SpreadOrderInfo struct {
	ClientOrderID string `json:"clOrdId"`
	OrderID       string `json:"ordId"`
	Tag           string `json:"tag"`
	SCode         string `json:"sCode"`
	SMsg          string `json:"sMsg"`
}

// GetSCode returns a status code value
func (a *SpreadOrderInfo) GetSCode() string { return a.SCode }

// GetSMsg returns a status message value
func (a *SpreadOrderInfo) GetSMsg() string { return a.SMsg }

// WsSpreadOrderResponse holds websocket spread order response.
type WsSpreadOrderResponse struct {
	OperationResponse
	Data []SpreadOrderInfo `json:"data"`
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
func (w *wsIncomingData) copyToPlaceOrderResponse() (*WsPlaceOrderResponse, error) {
	if len(w.Data) == 0 {
		return nil, errEmptyPlaceOrderResponse
	}
	var placeOrds []OrderData
	err := json.Unmarshal(w.Data, &placeOrds)
	if err != nil {
		return nil, err
	}
	return &WsPlaceOrderResponse{
		OperationResponse: OperationResponse{
			Operation: w.Operation,
			ID:        w.ID,
		},
		Data: placeOrds,
	}, nil
}

// copyResponseToInterface returns unmarshals the response data into the dataHolder interface.
func (w *wsIncomingData) copyResponseToInterface(dataHolder interface{}) error {
	rv := reflect.ValueOf(dataHolder)
	if rv.Kind() != reflect.Pointer /* TODO: || rv.IsNil()*/ {
		return errInvalidResponseParam
	}
	if len(w.Data) == 0 {
		return errEmptyPlaceOrderResponse
	}
	var spreadOrderResps []SpreadOrderInfo
	return json.Unmarshal(w.Data, &spreadOrderResps)
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

// WSOpenInterestResponse represents an open interest instance.
type WSOpenInterestResponse struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []OpenInterest   `json:"data"`
}

// WsOperationInput for all websocket request inputs.
type WsOperationInput struct {
	ID        string      `json:"id"`
	Operation string      `json:"op"`
	Arguments interface{} `json:"args"`
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
	PositionID       string                  `json:"posId"`
	TradeID          string                  `json:"tradeId"`
	InstrumentID     string                  `json:"instId"`
	InstrumentType   string                  `json:"instType"`
	MarginMode       string                  `json:"mgnMode"`
	PositionSide     string                  `json:"posSide"`
	Position         string                  `json:"pos"`
	Currency         string                  `json:"ccy"`
	PositionCurrency string                  `json:"posCcy"`
	AveragePrice     convert.StringToFloat64 `json:"avgPx"`
	UpdateTime       convert.ExchangeTime    `json:"uTIme"`
}

// BalanceData represents currency and it's Cash balance with the update time.
type BalanceData struct {
	Currency    string                  `json:"ccy"`
	CashBalance convert.StringToFloat64 `json:"cashBal"`
	UpdateTime  convert.ExchangeTime    `json:"uTime"`
}

// BalanceAndPositionData represents balance and position data with the push time.
type BalanceAndPositionData struct {
	PushTime     convert.ExchangeTime `json:"pTime"`
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
	AmendResult     string                  `json:"amendResult"`
	Code            string                  `json:"code"`
	ExecType        string                  `json:"execType"`
	FillFee         convert.StringToFloat64 `json:"fillFee"`
	FillFeeCurrency string                  `json:"fillFeeCcy"`
	FillNotionalUsd convert.StringToFloat64 `json:"fillNotionalUsd"`
	Msg             string                  `json:"msg"`
	NotionalUSD     convert.StringToFloat64 `json:"notionalUsd"`
	ReduceOnly      bool                    `json:"reduceOnly,string"`
	RequestID       string                  `json:"reqId"`
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
	InstrumentType             string                  `json:"instType"`
	InstrumentID               string                  `json:"instId"`
	OrderID                    string                  `json:"ordId"`
	Currency                   string                  `json:"ccy"`
	AlgoID                     string                  `json:"algoId"`
	Price                      convert.StringToFloat64 `json:"px"`
	Size                       convert.StringToFloat64 `json:"sz"`
	TradeMode                  string                  `json:"tdMode"`
	TargetCurrency             string                  `json:"tgtCcy"`
	NotionalUsd                string                  `json:"notionalUsd"`
	OrderType                  string                  `json:"ordType"`
	Side                       order.Side              `json:"side"`
	PositionSide               string                  `json:"posSide"`
	State                      string                  `json:"state"`
	Leverage                   string                  `json:"lever"`
	TakeProfitTriggerPrice     convert.StringToFloat64 `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string                  `json:"tpTriggerPxType"`
	TakeProfitOrdPrice         convert.StringToFloat64 `json:"tpOrdPx"`
	StopLossTriggerPrice       convert.StringToFloat64 `json:"slTriggerPx"`
	StopLossTriggerPriceType   string                  `json:"slTriggerPxType"`
	TriggerPrice               convert.StringToFloat64 `json:"triggerPx"`
	TriggerPriceType           string                  `json:"triggerPxType"`
	OrderPrice                 convert.StringToFloat64 `json:"ordPx"`
	ActualSize                 convert.StringToFloat64 `json:"actualSz"`
	ActualPrice                convert.StringToFloat64 `json:"actualPx"`
	Tag                        string                  `json:"tag"`
	ActualSide                 string                  `json:"actualSide"`
	TriggerTime                convert.ExchangeTime    `json:"triggerTime"`
	CreationTime               convert.ExchangeTime    `json:"cTime"`
}

// WsAdvancedAlgoOrder advanced algo order response.
type WsAdvancedAlgoOrder struct {
	Argument SubscriptionInfo            `json:"arg"`
	Data     []WsAdvancedAlgoOrderDetail `json:"data"`
}

// WsAdvancedAlgoOrderDetail advanced algo order response pushed through the websocket conn
type WsAdvancedAlgoOrderDetail struct {
	ActualPrice            string               `json:"actualPx"`
	ActualSide             string               `json:"actualSide"`
	ActualSize             string               `json:"actualSz"`
	AlgoID                 string               `json:"algoId"`
	Currency               string               `json:"ccy"`
	Count                  string               `json:"count"`
	InstrumentID           string               `json:"instId"`
	InstrumentType         string               `json:"instType"`
	Leverage               string               `json:"lever"`
	NotionalUsd            string               `json:"notionalUsd"`
	OrderPrice             string               `json:"ordPx"`
	OrdType                string               `json:"ordType"`
	PositionSide           string               `json:"posSide"`
	PriceLimit             string               `json:"pxLimit"`
	PriceSpread            string               `json:"pxSpread"`
	PriceVariation         string               `json:"pxVar"`
	Side                   order.Side           `json:"side"`
	StopLossOrderPrice     string               `json:"slOrdPx"`
	StopLossTriggerPrice   string               `json:"slTriggerPx"`
	State                  string               `json:"state"`
	Size                   string               `json:"sz"`
	SizeLimit              string               `json:"szLimit"`
	TradeMode              string               `json:"tdMode"`
	TimeInterval           string               `json:"timeInterval"`
	TakeProfitOrderPrice   string               `json:"tpOrdPx"`
	TakeProfitTriggerPrice string               `json:"tpTriggerPx"`
	Tag                    string               `json:"tag"`
	TriggerPrice           string               `json:"triggerPx"`
	CallbackRatio          string               `json:"callbackRatio"`
	CallbackSpread         string               `json:"callbackSpread"`
	ActivePrice            string               `json:"activePx"`
	MoveTriggerPrice       string               `json:"moveTriggerPx"`
	CreationTime           convert.ExchangeTime `json:"cTime"`
	PushTime               convert.ExchangeTime `json:"pTime"`
	TriggerTime            convert.ExchangeTime `json:"triggerTime"`
}

// WsGreeks greeks push data with the subscription info through websocket channel
type WsGreeks struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []WsGreekData    `json:"data"`
}

// WsGreekData greeks push data through websocket channel
type WsGreekData struct {
	ThetaBS   string               `json:"thetaBS"`
	ThetaPA   string               `json:"thetaPA"`
	DeltaBS   string               `json:"deltaBS"`
	DeltaPA   string               `json:"deltaPA"`
	GammaBS   string               `json:"gammaBS"`
	GammaPA   string               `json:"gammaPA"`
	VegaBS    string               `json:"vegaBS"`
	VegaPA    string               `json:"vegaPA"`
	Currency  string               `json:"ccy"`
	Timestamp convert.ExchangeTime `json:"ts"`
}

// WsRfq represents websocket push data for "rfqs" subscription
type WsRfq struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []WsRfqData      `json:"data"`
}

// WsRfqData represents rfq order response data streamed through the websocket channel
type WsRfqData struct {
	CreationTime   time.Time     `json:"cTime"`
	UpdateTime     time.Time     `json:"uTime"`
	TraderCode     string        `json:"traderCode"`
	RfqID          string        `json:"rfqId"`
	ClientRfqID    string        `json:"clRfqId"`
	State          string        `json:"state"`
	ValidUntil     string        `json:"validUntil"`
	Counterparties []string      `json:"counterparties"`
	Legs           []RfqOrderLeg `json:"legs"`
}

// WsQuote represents websocket push data for "quotes" subscription
type WsQuote struct {
	Arguments SubscriptionInfo `json:"arg"`
	Data      []WsQuoteData    `json:"data"`
}

// WsQuoteData represents a single quote order information
type WsQuoteData struct {
	ValidUntil    convert.ExchangeTime `json:"validUntil"`
	UpdatedTime   convert.ExchangeTime `json:"uTime"`
	CreationTime  convert.ExchangeTime `json:"cTime"`
	Legs          []OrderLeg           `json:"legs"`
	QuoteID       string               `json:"quoteId"`
	RfqID         string               `json:"rfqId"`
	TraderCode    string               `json:"traderCode"`
	QuoteSide     string               `json:"quoteSide"`
	State         string               `json:"state"`
	ClientQuoteID string               `json:"clQuoteId"`
}

// WsStructureBlocTrade represents websocket push data for "struc-block-trades" subscription
type WsStructureBlocTrade struct {
	Argument SubscriptionInfo       `json:"arg"`
	Data     []WsBlockTradeResponse `json:"data"`
}

// WsBlockTradeResponse represents a structure block order information
type WsBlockTradeResponse struct {
	CreationTime    convert.ExchangeTime `json:"cTime"`
	RfqID           string               `json:"rfqId"`
	ClientRfqID     string               `json:"clRfqId"`
	QuoteID         string               `json:"quoteId"`
	ClientQuoteID   string               `json:"clQuoteId"`
	BlockTradeID    string               `json:"blockTdId"`
	TakerTraderCode string               `json:"tTraderCode"`
	MakerTraderCode string               `json:"mTraderCode"`
	Legs            []OrderLeg           `json:"legs"`
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
	StopType               string               `json:"stopType"`
	TotalAnnualizedRate    string               `json:"totalAnnualizedRate"`
	TotalProfitAndLoss     string               `json:"totalPnl"`
	TakeProfitTriggerPrice string               `json:"tpTriggerPx"`
	TradeNum               string               `json:"tradeNum"`
	TriggerTime            convert.ExchangeTime `json:"triggerTime"`
	CreationTime           convert.ExchangeTime `json:"cTime"`
	PushTime               convert.ExchangeTime `json:"pTime"`
	UpdateTime             convert.ExchangeTime `json:"uTime"`
}

// WsContractGridAlgoOrder represents websocket push data for "grid-orders-contract" subscription
type WsContractGridAlgoOrder struct {
	Argument SubscriptionInfo        `json:"arg"`
	Data     []ContractGridAlgoOrder `json:"data"`
}

// ContractGridAlgoOrder represents contract grid algo order
type ContractGridAlgoOrder struct {
	ActualLever            string                  `json:"actualLever"`
	AlgoID                 string                  `json:"algoId"`
	AlgoOrderType          string                  `json:"algoOrdType"`
	AnnualizedRate         convert.StringToFloat64 `json:"annualizedRate"`
	ArbitrageNumber        string                  `json:"arbitrageNum"`
	BasePosition           bool                    `json:"basePos"`
	CancelType             string                  `json:"cancelType"`
	Direction              string                  `json:"direction"`
	Eq                     string                  `json:"eq"`
	FloatProfit            string                  `json:"floatProfit"`
	GridQuantity           string                  `json:"gridNum"`
	GridProfit             string                  `json:"gridProfit"`
	InstrumentID           string                  `json:"instId"`
	InstrumentType         string                  `json:"instType"`
	Investment             string                  `json:"investment"`
	Leverage               string                  `json:"lever"`
	LiqPrice               convert.StringToFloat64 `json:"liqPx"`
	MaxPrice               convert.StringToFloat64 `json:"maxPx"`
	MinPrice               convert.StringToFloat64 `json:"minPx"`
	CreationTime           convert.ExchangeTime    `json:"cTime"`
	PushTime               convert.ExchangeTime    `json:"pTime"`
	PerMaxProfitRate       convert.StringToFloat64 `json:"perMaxProfitRate"`
	PerMinProfitRate       convert.StringToFloat64 `json:"perMinProfitRate"`
	ProfitAndLossRatio     convert.StringToFloat64 `json:"pnlRatio"`
	RunPrice               convert.StringToFloat64 `json:"runPx"`
	RunType                string                  `json:"runType"`
	SingleAmount           convert.StringToFloat64 `json:"singleAmt"`
	SlTriggerPrice         convert.StringToFloat64 `json:"slTriggerPx"`
	State                  string                  `json:"state"`
	StopType               string                  `json:"stopType"`
	Size                   convert.StringToFloat64 `json:"sz"`
	Tag                    string                  `json:"tag"`
	TotalAnnualizedRate    string                  `json:"totalAnnualizedRate"`
	TotalProfitAndLoss     convert.StringToFloat64 `json:"totalPnl"`
	TakeProfitTriggerPrice string                  `json:"tpTriggerPx"`
	TradeNumber            string                  `json:"tradeNum"`
	TriggerTime            convert.ExchangeTime    `json:"triggerTime"`
	UpdateTime             convert.ExchangeTime    `json:"uTime"`
	Underlying             string                  `json:"uly"`
}

// WsGridSubOrderData to retrieve grid sub orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing order.
type WsGridSubOrderData struct {
	Argument SubscriptionInfo   `json:"arg"`
	Data     []GridSubOrderData `json:"data"`
}

// GridSubOrderData represents a single sub order detailed info
type GridSubOrderData struct {
	AccumulatedFillSize convert.StringToFloat64 `json:"accFillSz"`
	AlgoID              string                  `json:"algoId"`
	AlgoOrderType       string                  `json:"algoOrdType"`
	AveragePrice        convert.StringToFloat64 `json:"avgPx"`
	CreationTime        convert.ExchangeTime    `json:"cTime"`
	ContractValue       string                  `json:"ctVal"`
	Fee                 convert.StringToFloat64 `json:"fee"`
	FeeCurrency         string                  `json:"feeCcy"`
	GroupID             string                  `json:"groupId"`
	InstrumentID        string                  `json:"instId"`
	InstrumentType      string                  `json:"instType"`
	Leverage            string                  `json:"lever"`
	OrderID             string                  `json:"ordId"`
	OrderType           string                  `json:"ordType"`
	PushTime            convert.ExchangeTime    `json:"pTime"`
	ProfitAdLoss        string                  `json:"pnl"`
	PositionSide        string                  `json:"posSide"`
	Price               convert.StringToFloat64 `json:"px"`
	Side                order.Side              `json:"side"`
	State               string                  `json:"state"`
	Size                convert.StringToFloat64 `json:"sz"`
	Tag                 string                  `json:"tag"`
	TradeMode           string                  `json:"tdMode"`
	UpdateTime          convert.ExchangeTime    `json:"uTime"`
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

// WsOptionTrades represents option trade data
type WsOptionTrades struct {
	Arg  SubscriptionInfo `json:"arg"`
	Data []PublicTrade    `json:"data"`
}

// PublicTrade represents public trade item for option, block, and others
type PublicTrade struct {
	FillVolume       convert.StringToFloat64 `json:"fillVol"`
	ForwardPrice     convert.StringToFloat64 `json:"fwdPx"`
	IndexPrice       convert.StringToFloat64 `json:"idxPx"`
	InstrumentFamily string                  `json:"instFamily"`
	InstrumentID     string                  `json:"instId"`
	MarkPrice        convert.StringToFloat64 `json:"markPx"`
	OptionType       string                  `json:"optType"`
	Price            convert.StringToFloat64 `json:"px"`
	Side             string                  `json:"side"`
	Size             convert.StringToFloat64 `json:"sz"`
	TradeID          string                  `json:"tradeId"`
	Timestamp        convert.ExchangeTime    `json:"ts"`
}

// WsOrderBookData represents a book order push data.
type WsOrderBookData struct {
	Asks      [][4]string          `json:"asks"`
	Bids      [][4]string          `json:"bids"`
	Timestamp convert.ExchangeTime `json:"ts"`
	Checksum  int32                `json:"checksum,omitempty"`
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

// PublicBlockTrades holds public block trades
type PublicBlockTrades struct {
	Arg  SubscriptionInfo `json:"arg"`
	Data []PublicTrade    `json:"data"`
}

// PMLimitationResponse represents portfolio margin mode limitation for specific underlying
type PMLimitationResponse struct {
	MaximumSize  convert.StringToFloat64 `json:"maxSz"`
	PositionType string                  `json:"postType"`
	Underlying   string                  `json:"uly"`
}

// RiskOffsetType represents risk offset type value
type RiskOffsetType struct {
	Type string `json:"type"`
}

// AutoLoan holds auto loan information
type AutoLoan struct {
	AutoLoan bool `json:"autoLoan"`
}

// AccountMode holds account mode
type AccountMode struct {
	IsoMode string `json:"isoMode"`
}

// EasyConvertDetail represents easy convert currencies list and their detail.
type EasyConvertDetail struct {
	FromData   []EasyConvertFromData `json:"fromData"`
	ToCurrency []string              `json:"toCcy"`
}

// EasyConvertFromData represents convert currency from detail
type EasyConvertFromData struct {
	FromAmount   convert.StringToFloat64 `json:"fromAmt"`
	FromCurrency string                  `json:"fromCcy"`
}

// PlaceEasyConvertParam represents easy convert request params
type PlaceEasyConvertParam struct {
	FromCurrency []string `json:"fromCcy"`
	ToCurrency   string   `json:"toCcy"`
}

// EasyConvertItem represents easy convert place order response.
type EasyConvertItem struct {
	FilFromSize  convert.StringToFloat64 `json:"fillFromSz"`
	FillToSize   convert.StringToFloat64 `json:"fillToSz"`
	FromCurrency string                  `json:"fromCcy"`
	Status       string                  `json:"status"`
	ToCurrency   string                  `json:"toCcy"`
	UpdateTime   convert.ExchangeTime    `json:"uTime"`
}

// TradeOneClickRepayParam represents click one repay param
type TradeOneClickRepayParam struct {
	DebtCurrency  []string `json:"debtCcy"`
	RepayCurrency string   `json:"repayCcy"`
}

// CurrencyOneClickRepay represents one click repay currency
type CurrencyOneClickRepay struct {
	DebtCurrency  string                  `json:"debtCcy"`
	FillFromSize  convert.StringToFloat64 `json:"fillFromSz"`
	FillRepaySize convert.StringToFloat64 `json:"fillRepaySz"`
	FillToSize    convert.StringToFloat64 `json:"fillToSz"`
	RepayCurrency string                  `json:"repayCcy"`
	Status        string                  `json:"status"`
	UpdateTime    convert.ExchangeTime    `json:"uTime"`
}

// CancelMMPResponse holds cancel MMP response result
type CancelMMPResponse struct {
	Result bool `json:"result"`
}

// CancelMMPAfterCountdownResponse returns list of
type CancelMMPAfterCountdownResponse struct {
	TriggerTime convert.ExchangeTime `json:"triggerTime"` // The time the cancellation is triggered. triggerTime=0 means Cancel All After is disabled.
	Timestamp   convert.ExchangeTime `json:"ts"`          // The time the request is sent.
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
	IP               string               `json:"ip"`
	SubAccountName   string               `json:"subAcct"`
	APIKey           string               `json:"apiKey"`
	Label            string               `json:"label"`
	APIKeyPermission string               `json:"perm"`
	Timestamp        convert.ExchangeTime `json:"ts"`
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
	Leverage      convert.StringToFloat64 `json:"lever"`
	MaximumAmount convert.StringToFloat64 `json:"maxAmt"`
}

// AdjustMarginBalanceResponse represents algo id for response for margin balance adjust request.
type AdjustMarginBalanceResponse struct {
	AlgoID string `json:"algoId"`
}

// GridAIParameterResponse represents gri AI parameter response.
type GridAIParameterResponse struct {
	AlgoOrderType        string                  `json:"algoOrdType"`
	AnnualizedRate       string                  `json:"annualizedRate"`
	Currency             string                  `json:"ccy"`
	Direction            string                  `json:"direction"`
	Duration             string                  `json:"duration"`
	GridNum              string                  `json:"gridNum"`
	InstrumentID         string                  `json:"instId"`
	Leverage             convert.StringToFloat64 `json:"lever"`
	MaximumPrice         convert.StringToFloat64 `json:"maxPx"`
	MinimumInvestment    convert.StringToFloat64 `json:"minInvestment"`
	MinimumPrice         convert.StringToFloat64 `json:"minPx"`
	PerMaximumProfitRate convert.StringToFloat64 `json:"perMaxProfitRate"`
	PerMinimumProfitRate convert.StringToFloat64 `json:"perMinProfitRate"`
	RunType              string                  `json:"runType"`
}

// InvestmentData holds investment data parameter
type InvestmentData struct {
	Amount   float64 `json:"amt,string"`
	Currency string  `json:"ccy"`
}

// ComputeInvestmentDataParam holds parameter values for computing investment data
type ComputeInvestmentDataParam struct {
	InstrumentID   string           `json:"instId"`
	AlgoOrderType  string           `json:"algoOrdType"` // Algo order type 'grid': Spot grid 'contract_grid': Contract grid
	GridNumber     float64          `json:"gridNum,string"`
	Direction      string           `json:"direction"` // Contract grid type 'long','short', 'neutral' Only applicable to contract grid
	MaxPrice       float64          `json:"maxPx,string"`
	MinPrice       float64          `json:"minPx,string"`
	RunType        string           `json:"runType"` // Grid type 1: Arithmetic, 2: Geometric
	Leverage       float64          `json:"lever,omitempty,string"`
	BasePosition   bool             `json:"basePos"`
	InvestmentData []InvestmentData `json:"investmentData"`
}

// InvestmentResult holds investment response
type InvestmentResult struct {
	MinInvestmentData []InvestmentData        `json:"minInvestmentData"`
	SingleAmount      convert.StringToFloat64 `json:"singleAmt"`
}

// RSIBacktestingResponse holds response for relative strength index(RSI) backtesting
type RSIBacktestingResponse struct {
	TriggerNumber string `json:"triggerNum"`
}

// SignalBotOrderDetail holds detail of signal bot order.
type SignalBotOrderDetail struct {
	AlgoID               string                  `json:"algoId"`
	ClientSuppliedAlgoID string                  `json:"algoClOrdId"`
	AlgoOrderType        string                  `json:"algoOrdType"`
	InstrumentType       string                  `json:"instType"`
	InstrumentIds        []string                `json:"instIds"`
	CreationTime         convert.ExchangeTime    `json:"cTime"`
	UpdateTime           convert.ExchangeTime    `json:"uTime"`
	State                string                  `json:"state"`
	CancelType           string                  `json:"cancelType"`
	TotalPnl             convert.StringToFloat64 `json:"totalPnl"`
	ProfitAndLossRatio   convert.StringToFloat64 `json:"pnlRatio"`
	TotalEq              convert.StringToFloat64 `json:"totalEq"`
	FloatPnl             string                  `json:"floatPnl"`
	FrozenBal            string                  `json:"frozenBal"`
	AvailableBalance     convert.StringToFloat64 `json:"availBal"`
	Lever                convert.StringToFloat64 `json:"lever"`
	InvestAmount         convert.StringToFloat64 `json:"investAmt"`
	SubOrdType           string                  `json:"subOrdType"`
	Ratio                convert.StringToFloat64 `json:"ratio"`
	EntrySettingParam    struct {
		AllowMultipleEntry bool                    `json:"allowMultipleEntry"`
		Amount             convert.StringToFloat64 `json:"amt"`
		EntryType          string                  `json:"entryType"`
		Ratio              convert.StringToFloat64 `json:"ratio"`
	} `json:"entrySettingParam"`
	ExitSettingParam struct {
		StopLossPercentage   string `json:"slPct"`
		TakeProfitPercentage string `json:"tpPct"`
		TakeProfitSlType     string `json:"tpSlType"`
	} `json:"exitSettingParam"`
	SignalChanID     string `json:"signalChanId"`
	SignalChanName   string `json:"signalChanName"`
	SignalSourceType string `json:"signalSourceType"`

	TotalPnlRatio convert.StringToFloat64 `json:"totalPnlRatio"`
	RealizedPnl   string                  `json:"realizedPnl"`
}

// SignalBotPosition holds signal bot position information
type SignalBotPosition struct {
	AutoDecreaseLine             string                  `json:"adl"`
	AlgoClientOrderID            string                  `json:"algoClOrdId"`
	AlgoID                       string                  `json:"algoId"`
	AveragePrice                 convert.StringToFloat64 `json:"avgPx"`
	CreationTime                 convert.ExchangeTime    `json:"cTime"`
	Currency                     string                  `json:"ccy"`
	InitialMarginRequirement     string                  `json:"imr"`
	InstrumentID                 string                  `json:"instId"`
	InstrumentType               string                  `json:"instType"`
	Last                         convert.StringToFloat64 `json:"last"`
	Lever                        convert.StringToFloat64 `json:"lever"`
	LiquidationPrice             convert.StringToFloat64 `json:"liqPx"`
	MarkPrice                    convert.StringToFloat64 `json:"markPx"`
	MgnMode                      string                  `json:"mgnMode"`
	MgnRatio                     convert.StringToFloat64 `json:"mgnRatio"` // Margin mode 'cross' 'isolated'
	MaintenanceMarginRequirement string                  `json:"mmr"`
	NotionalUsd                  string                  `json:"notionalUsd"`
	Position                     string                  `json:"pos"`
	PositionSide                 string                  `json:"posSide"` // Position side 'net'
	UpdateTime                   convert.ExchangeTime    `json:"uTime"`
	UnrealizedProfitAndLoss      string                  `json:"upl"`
	UplRatio                     convert.StringToFloat64 `json:"uplRatio"` // Unrealized profit and loss ratio
}

// SubOrder holds signal bot sub orders
type SubOrder struct {
	AccountFillSize   string                  `json:"accFillSz"`
	AlgoClientOrderID string                  `json:"algoClOrdId"`
	AlgoID            string                  `json:"algoId"`
	AlgoOrdType       string                  `json:"algoOrdType"`
	AveragePrice      convert.StringToFloat64 `json:"avgPx"`
	CreationTime      convert.ExchangeTime    `json:"cTime"`
	Currency          string                  `json:"ccy"`
	ClientOrderID     string                  `json:"clOrdId"`
	CtVal             string                  `json:"ctVal"`
	Fee               convert.StringToFloat64 `json:"fee"`
	FeeCurrency       string                  `json:"feeCcy"`
	InstrumentID      string                  `json:"instId"`
	InstrumentType    string                  `json:"instType"`
	Leverage          convert.StringToFloat64 `json:"lever"`
	OrderID           string                  `json:"ordId"`
	OrderType         string                  `json:"ordType"`
	ProfitAndLoss     convert.StringToFloat64 `json:"pnl"`
	PosSide           string                  `json:"posSide"`
	Price             convert.StringToFloat64 `json:"px"`
	Side              string                  `json:"side"`
	State             string                  `json:"state"`
	Size              convert.StringToFloat64 `json:"sz"`
	Tag               string                  `json:"tag"`
	TdMode            string                  `json:"tdMode"`
	UpdateTime        convert.ExchangeTime    `json:"uTime"`
}

// SignalBotEventHistory holds history information for signal bot
type SignalBotEventHistory struct {
	AlertMsg         time.Time            `json:"alertMsg"`
	AlgoID           string               `json:"algoId"`
	EventCtime       convert.ExchangeTime `json:"eventCtime"`
	EventProcessMsg  string               `json:"eventProcessMsg"`
	EventStatus      string               `json:"eventStatus"`
	EventUtime       convert.ExchangeTime `json:"eventUtime"`
	EventType        string               `json:"eventType"`
	TriggeredOrdData []struct {
		ClientOrderID string `json:"clOrdId"`
	} `json:"triggeredOrdData"`
}

// PlaceRecurringBuyOrderParam holds parameters for placing recurring order
type PlaceRecurringBuyOrderParam struct {
	Tag                       string              `json:"tag"`
	ClientSuppliedAlgoOrderID string              `json:"algoClOrdId"`
	StrategyName              string              `json:"stgyName"` // Custom name for trading bot
	Amount                    float64             `json:"amt,string"`
	RecurringList             []RecurringListItem `json:"recurringList"`
	Period                    string              `json:"period"` // Period 'monthly' 'weekly' 'daily'

	// Recurring buy date
	// When the period is monthly, the value range is an integer of [1,28]
	// When the period is weekly, the value range is an integer of [1,7]
	// When the period is daily, the value is 1
	RecurringDay       string `json:"recurringDay"`
	RecurringTime      int64  `json:"recurringTime,string"` // Recurring buy time, the value range is an integer of [0,23]
	TimeZone           string `json:"timeZone"`
	TradeMode          string `json:"tdMode"` // Trading mode Margin mode: 'cross' Non-Margin mode: 'cash'
	InvestmentCurrency string `json:"investmentCcy"`
}

// RecurringListItem holds recurring list item
type RecurringListItem struct {
	Currency currency.Code `json:"ccy"`
	Ratio    float64       `json:"ratio,string"`
}

// RecurringListItemDetailed holds a detailed instance of recurring list item
type RecurringListItemDetailed struct {
	AveragePrice convert.StringToFloat64 `json:"avgPx"`
	Currency     string                  `json:"ccy"`
	Profit       convert.StringToFloat64 `json:"profit"`
	Price        convert.StringToFloat64 `json:"px"`
	Ratio        convert.StringToFloat64 `json:"ratio"`
	TotalAmount  convert.StringToFloat64 `json:"totalAmt"`
}

// RecurringOrderResponse holds recurring order response.
type RecurringOrderResponse struct {
	AlgoID            string `json:"algoId"`
	AlgoClientOrderID string `json:"algoClOrdId"`
	SCode             string `json:"sCode"`
	SMsg              string `json:"sMsg"`
}

// AmendRecurringOrderParam holds recurring order params.
type AmendRecurringOrderParam struct {
	AlgoID       string `json:"algoId"`
	StrategyName string `json:"stgyName"`
}

// StopRecurringBuyOrder stop recurring order
type StopRecurringBuyOrder struct {
	AlgoID string `json:"algoId"`
}

// RecurringOrderItem holds recurring order info.
type RecurringOrderItem struct {
	AlgoClOrdID        string                  `json:"algoClOrdId"`
	AlgoID             string                  `json:"algoId"`
	AlgoOrdType        string                  `json:"algoOrdType"`
	Amount             convert.StringToFloat64 `json:"amt"`
	CreationTime       convert.ExchangeTime    `json:"cTime"`
	Cycles             string                  `json:"cycles"`
	InstrumentType     string                  `json:"instType"`
	InvestmentAmount   string                  `json:"investmentAmt"`
	InvestmentCurrency string                  `json:"investmentCcy"`
	MarketCap          string                  `json:"mktCap"`
	Period             string                  `json:"period"`
	ProfitAndLossRatio convert.StringToFloat64 `json:"pnlRatio"`
	RecurringDay       string                  `json:"recurringDay"`
	RecurringList      []RecurringListItem     `json:"recurringList"`
	RecurringTime      string                  `json:"recurringTime"`
	State              string                  `json:"state"`
	StgyName           string                  `json:"stgyName"`
	Tag                string                  `json:"tag"`
	TimeZone           string                  `json:"timeZone"`
	TotalAnnRate       string                  `json:"totalAnnRate"`
	TotalPnl           string                  `json:"totalPnl"`
	UpdateTime         convert.ExchangeTime    `json:"uTime"`
}

// RecurringOrderDeail holds detailed information about recurring order
type RecurringOrderDeail struct {
	RecurringListItem
	RecurringList []RecurringListItemDetailed `json:"recurringList"`
}

// RecurringBuySubOrder holds recurring buy sub order detail.
type RecurringBuySubOrder struct {
	AccFillSize     convert.StringToFloat64 `json:"accFillSz"`
	AlgoClientOrdID string                  `json:"algoClOrdId"`
	AlgoID          string                  `json:"algoId"`
	AlgoOrderType   string                  `json:"algoOrdType"`
	AveragePrice    convert.StringToFloat64 `json:"avgPx"`
	CreationTime    convert.ExchangeTime    `json:"cTime"`
	Fee             convert.StringToFloat64 `json:"fee"`
	FeeCcy          string                  `json:"feeCcy"`
	InstrumentID    string                  `json:"instId"`
	InstrumentType  string                  `json:"instType"`
	OrderID         string                  `json:"ordId"`
	OrderType       string                  `json:"ordType"`
	Price           convert.StringToFloat64 `json:"px"`
	Side            string                  `json:"side"`
	State           string                  `json:"state"`
	Size            convert.StringToFloat64 `json:"sz"`
	Tag             string                  `json:"tag"`
	TradeMode       string                  `json:"tdMode"`
	UpdateTime      convert.ExchangeTime    `json:"uTime"`
}

// PositionInfo represents a positions detail.
type PositionInfo struct {
	InstrumentType    string                  `json:"instType"`
	InstrumentID      string                  `json:"instId"`
	AlgoID            string                  `json:"algoId"`
	Lever             convert.StringToFloat64 `json:"lever"`
	MgnMode           string                  `json:"mgnMode"`
	OpenAvgPrice      convert.StringToFloat64 `json:"openAvgPx"`
	OpenOrdID         string                  `json:"openOrdId"`
	OpenTime          convert.ExchangeTime    `json:"openTime"`
	PosSide           string                  `json:"posSide"`
	SlTriggerPrice    convert.StringToFloat64 `json:"slTriggerPx"`
	SubPos            string                  `json:"subPos"`
	SubPosID          string                  `json:"subPosId"`
	TpTriggerPrice    convert.StringToFloat64 `json:"tpTriggerPx"`
	CloseAveragePrice convert.StringToFloat64 `json:"closeAvgPx"`
	CloseTime         convert.ExchangeTime    `json:"closeTime"`
}

// TPSLOrderParam holds Take profit and stop loss order parameters.
type TPSLOrderParam struct {
	InstrumentType            string  `json:"instType"`
	SubPositionID             string  `json:"subPosId"`
	TakeProfitTriggerPrice    float64 `json:"tpTriggerPx,omitempty,string"`
	StopLossTriggerPrice      float64 `json:"slTriggerPx,omitempty,string"`
	TakePofitTriggerPriceType string  `json:"tpTriggerPriceType"` // last: last price, 'index': index price 'mark': mark price Default is 'last'
	StopLossTriggerPriceType  string  `jsonL:"slTriggerPxType"`   // Stop-loss trigger price type 'last': last price 'index': index price 'mark': mark price Default is 'last'
	Tag                       string  `json:"tag"`
}

// PositionIDInfo holds place positions information
type PositionIDInfo struct {
	SubPosID string `json:"subPosId"`
	Tag      string `json:"tag"`
}

// CloseLeadingPositionParam request parameter for closing leading position
type CloseLeadingPositionParam struct {
	InstrumentType string `json:"instType"`
	SubPositionID  string `json:"subPosId"`
	Tag            string `json:"tag"`
}

// LeadingInstrumentItem represents leading instrument info and it's status
type LeadingInstrumentItem struct {
	Enabled      bool   `json:"enabled"`
	InstrumentID string `json:"instId"`
}

// ProfitSharingItem holds profit sharing information
type ProfitSharingItem struct {
	Ccy                 string               `json:"ccy"`
	NickName            string               `json:"nickName"`
	ProfitSharingAmount string               `json:"profitSharingAmt"`
	ProfitSharingID     string               `json:"profitSharingId"`
	InstrumentType      string               `json:"instType"`
	Timestamp           convert.ExchangeTime `json:"ts"`
}

// TotalProfitSharing holds information about total amount of profit shared since joining the platform.
type TotalProfitSharing struct {
	Currency                 string                  `json:"ccy"`
	InstrumentType           string                  `json:"instType"`
	TotalProfitSharingAmount convert.StringToFloat64 `json:"totalProfitSharingAmt"`
}

// Offer represents an investment offer information for different 'staking' and 'defi' protocols
type Offer struct {
	Currency     string                  `json:"ccy"`
	ProductID    string                  `json:"productId"`
	Protocol     string                  `json:"protocol"`
	ProtocolType string                  `json:"protocolType"`
	EarningCcy   []string                `json:"earningCcy"`
	Term         string                  `json:"term"`
	Apy          convert.StringToFloat64 `json:"apy"`
	EarlyRedeem  bool                    `json:"earlyRedeem"`
	InvestData   []OfferInvestData       `json:"investData"`
	EarningData  []struct {
		Currency    string `json:"ccy"`
		EarningType string `json:"earningType"`
	} `json:"earningData"`
}

// OfferInvestData represents currencies invest data information for an offer
type OfferInvestData struct {
	Currency      string                  `json:"ccy"`
	Balance       convert.StringToFloat64 `json:"bal"`
	MinimumAmount convert.StringToFloat64 `json:"minAmt"`
	MaximumAmount convert.StringToFloat64 `json:"maxAmt"`
}

// PurchaseRequestParam represents purchase request param specific product
type PurchaseRequestParam struct {
	ProductID  string                   `json:"productId"`
	Term       int64                    `json:"term,string,omitempty"`
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
		Currency string                  `json:"ccy"`
		Amount   convert.StringToFloat64 `json:"amt"`
	} `json:"investData"`
	EarningData []struct {
		Ccy         string                  `json:"ccy"`
		EarningType string                  `json:"earningType"`
		Earnings    convert.StringToFloat64 `json:"earnings"`
	} `json:"earningData"`
	PurchasedTime convert.ExchangeTime `json:"purchasedTime"`
}

// BETHAssetsBalance balance is a snapshot summarized all BETH assets
type BETHAssetsBalance struct {
	Currency              string                  `json:"ccy"`
	Amount                convert.StringToFloat64 `json:"amt"`
	LatestInterestAccrual convert.StringToFloat64 `json:"latestInterestAccrual"`
	TotalInterestAccrual  convert.StringToFloat64 `json:"totalInterestAccrual"`
	Timestamp             convert.ExchangeTime    `json:"ts"`
}

// PurchaseRedeemHistory holds purchase and redeem history
type PurchaseRedeemHistory struct {
	Amt              convert.StringToFloat64 `json:"amt"`
	CompletedTime    convert.ExchangeTime    `json:"completedTime"`
	EstCompletedTime convert.ExchangeTime    `json:"estCompletedTime"`
	RequestTime      convert.ExchangeTime    `json:"requestTime"`
	Status           string                  `json:"status"`
	Type             string                  `json:"type"`
}

// APYItem holds annual percentage yield record
type APYItem struct {
	Rate      convert.StringToFloat64 `json:"rate"`
	Timestamp convert.ExchangeTime    `json:"ts"`
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

// WsOrderbook5 stores the orderbook data for orderbook 5 websocket
type WsOrderbook5 struct {
	Argument struct {
		Channel      string `json:"channel"`
		InstrumentID string `json:"instId"`
	} `json:"arg"`
	Data []Book5Data `json:"data"`
}

// Book5Data stores the orderbook data for orderbook 5 websocket
type Book5Data struct {
	Asks         [][4]string          `json:"asks"`
	Bids         [][4]string          `json:"bids"`
	InstrumentID string               `json:"instId"`
	Timestamp    convert.ExchangeTime `json:"ts"`
	SequenceID   int64                `json:"seqId"`
}

// WsSpreadOrder represents spread order detail.
type WsSpreadOrder struct {
	SpreadID          string                  `json:"sprdId"`
	OrderID           string                  `json:"ordId"`
	ClientOrderID     string                  `json:"clOrdId"`
	Tag               string                  `json:"tag"`
	Price             convert.StringToFloat64 `json:"px"`
	Size              convert.StringToFloat64 `json:"sz"`
	OrderType         string                  `json:"ordType"`
	Side              string                  `json:"side"`
	FillSize          convert.StringToFloat64 `json:"fillSz"`
	FillPrice         convert.StringToFloat64 `json:"fillPx"`
	TradeID           string                  `json:"tradeId"`
	AccFillSize       convert.StringToFloat64 `json:"accFillSz"`
	PendingFillSize   convert.StringToFloat64 `json:"pendingFillSz"`
	PendingSettleSize convert.StringToFloat64 `json:"pendingSettleSz"`
	CanceledSize      convert.StringToFloat64 `json:"canceledSz"`
	State             string                  `json:"state"`
	AvgPrice          convert.StringToFloat64 `json:"avgPx"`
	CancelSource      string                  `json:"cancelSource"`
	UpdateTime        convert.ExchangeTime    `json:"uTime"`
	CreationTime      convert.ExchangeTime    `json:"cTime"`
	Code              string                  `json:"code"`
	Msg               string                  `json:"msg"`
}

// WsSpreadOrderTrade trade of an order.
type WsSpreadOrderTrade struct {
	Argument struct {
		Channel  string `json:"channel"`
		SpreadID string `json:"sprdId"`
		UID      string `json:"uid"`
	} `json:"arg"`
	Data []struct {
		SpreadID      string                  `json:"sprdId"`
		TradeID       string                  `json:"tradeId"`
		OrderID       string                  `json:"ordId"`
		ClientOrderID string                  `json:"clOrdId"`
		Tag           string                  `json:"tag"`
		FillPrice     convert.StringToFloat64 `json:"fillPx"`
		FillSize      convert.StringToFloat64 `json:"fillSz"`
		State         string                  `json:"state"`
		Side          string                  `json:"side"`
		ExecType      string                  `json:"execType"`
		Timestamp     convert.ExchangeTime    `json:"ts"`
		Legs          []struct {
			InstrumentID string                  `json:"instId"`
			Price        convert.StringToFloat64 `json:"px"`
			Size         convert.StringToFloat64 `json:"sz"`
			Side         string                  `json:"side"`
			Fee          convert.StringToFloat64 `json:"fee"`
			FeeCcy       string                  `json:"feeCcy"`
			TradeID      string                  `json:"tradeId"`
		} `json:"legs"`
		Code string `json:"code"`
		Msg  string `json:"msg"`
	} `json:"data"`
}

// WsSpreadOrderbook holds spread orderbook data.
type WsSpreadOrderbook struct {
	Arg struct {
		Channel  string `json:"channel"`
		SpreadID string `json:"sprdId"`
	} `json:"arg"`
	Data []struct {
		Asks      [][3]string          `json:"asks"`
		Bids      [][3]string          `json:"bids"`
		Timestamp convert.ExchangeTime `json:"ts"`
	} `json:"data"`
}

// ExtractSpreadOrder extracts WsSpreadOrderbookData from WsSpreadOrderbook
func (a *WsSpreadOrderbook) ExtractSpreadOrder() (*WsSpreadOrderbookData, error) {
	resp := &WsSpreadOrderbookData{
		Argument: SubscriptionInfo{
			SpreadID: a.Arg.SpreadID,
			Channel:  a.Arg.Channel,
		},
		Data: make([]WsSpreadOrderbookItem, len(a.Data)),
	}
	var err error
	for x := range a.Data {
		resp.Data[x].Timestamp = a.Data[x].Timestamp.Time()
		resp.Data[x].Asks = make([]orderbook.Item, len(a.Data[x].Asks))
		resp.Data[x].Bids = make([]orderbook.Item, len(a.Data[x].Bids))

		for as := range a.Data[x].Asks {
			resp.Data[x].Asks[as].Price, err = strconv.ParseFloat(a.Data[x].Asks[a][0])
			if err != nil {
				return nil, err
			}
			resp.Data[x].Asks[as].Amount, err = strconv.ParseFloat(a.Data[x].Asks[a][0])
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

// WsSpreadOrderbookItem represents an orderbook asks and bids details.
type WsSpreadOrderbookItem struct {
	Asks      []orderbook.Item
	Bids      []orderbook.Item
	Timestamp time.Time
}

type WsSpreadOrderbookData struct {
	Argument SubscriptionInfo `json:"arg"`
	Data     []WsSpreadOrderbookItem
}
