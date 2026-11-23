package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// EventContractSeriesRequest holds the parameters for GetEventContractSeries
type EventContractSeriesRequest struct {
	SeriesID string
}

// EventContractSeriesResponse holds one prediction market series
type EventContractSeriesResponse struct {
	SeriesID   string                  `json:"seriesId"`
	Frequency  string                  `json:"freq"`
	Title      string                  `json:"title"`
	Category   string                  `json:"category"`
	Settlement EventContractSettlement `json:"settlement"`
}

// EventContractSettlement holds a prediction market series' settlement rules
type EventContractSettlement struct {
	Method     string `json:"method"`
	CloseEarly bool   `json:"closeEarly"`
	SourceName string `json:"srcName"`
	Underlying string `json:"underlying"`
}

// EventContractEventsRequest holds the parameters for GetEventContractEvents
type EventContractEventsRequest struct {
	SeriesID string
	EventID  string
	State    string
	Limit    uint64
	Before   time.Time // returns events whose expiry time is later than this time
	After    time.Time // returns events whose expiry time is earlier than this time
}

// EventContractEventResponse holds one prediction market event
type EventContractEventResponse struct {
	SeriesID   string     `json:"seriesId"`
	EventID    string     `json:"eventId"`
	FixTime    types.Time `json:"fixTime"`
	ExpiryTime types.Time `json:"expTime"`
	State      string     `json:"state"`
}

// CorporateActionsRequest holds the parameters for GetCorporateActions
type CorporateActionsRequest struct {
	Ticker       string
	InstrumentID string
	Limit        uint64
	After        string // the NextAfter cursor of the previous page
	Before       string
}

// CorporateActionsResponse holds a page of corporate action events
type CorporateActionsResponse struct {
	DataStatus string                 `json:"dataStatus"`
	DataAsOf   time.Time              `json:"dataAsOf"`
	NextAfter  string                 `json:"nextAfter"`
	Events     []CorporateActionEvent `json:"events"`
}

// CorporateActionEvent holds one corporate action of an underlying stock
type CorporateActionEvent struct {
	EventID            string                      `json:"caEventId"`
	EventVersion       uint64                      `json:"eventVersion"`
	Underlying         string                      `json:"underlying"`
	BloombergGlobalID  string                      `json:"bbGlobalId"`
	Exchange           string                      `json:"exchange"`
	ActionType         string                      `json:"actionType"`
	Class              string                      `json:"caClass"`
	DividendSize       string                      `json:"dividendSize"`
	CashDividend       string                      `json:"cashDividend"`
	RatioFrom          types.Number                `json:"ratioFrom"`
	RatioTo            types.Number                `json:"ratioTo"`
	Status             string                      `json:"status"`
	PlannedCutoverTime time.Time                   `json:"plannedCutoverAt"`
	Instruments        []CorporateActionInstrument `json:"instruments"`
}

// CorporateActionInstrument holds the trading state of an instrument a corporate action affects
type CorporateActionInstrument struct {
	InstrumentID string `json:"instId"`
	State        string `json:"state"`
}

// EventContractMarketsRequest holds the parameters for GetEventContractMarkets
type EventContractMarketsRequest struct {
	SeriesID     string
	EventID      string
	InstrumentID string
	State        string
	Limit        uint64
	Before       time.Time // returns markets whose expiry time is later than this time
	After        time.Time // returns markets whose expiry time is earlier than this time
}

// EventContractMarketResponse holds one prediction market
type EventContractMarketResponse struct {
	SeriesID     string       `json:"seriesId"`
	EventID      string       `json:"eventId"`
	InstrumentID string       `json:"instId"`
	ListTime     types.Time   `json:"listTime"`
	FixTime      types.Time   `json:"fixTime"`
	ExpiryTime   types.Time   `json:"expTime"`
	State        string       `json:"state"`
	Disputed     bool         `json:"disputed"`
	Outcome      string       `json:"outcome"`
	FloorStrike  types.Number `json:"floorStrike"`
	CapStrike    types.Number `json:"capStrike"` // OKX's INF, no upper bound, decodes as positive infinity
	SettleValue  types.Number `json:"settleValue"`
	HitDirection string       `json:"hitDir"`
}

// EstimatedDeliveryPriceResponse holds an instrument's estimated delivery or exercise price
type EstimatedDeliveryPriceResponse struct {
	InstrumentType  string       `json:"instType"`
	InstrumentID    string       `json:"instId"`
	SettlementType  string       `json:"settleType"` // sent only by the estimated-price websocket channel
	SettlementPrice types.Number `json:"settlePx"`
	Timestamp       types.Time   `json:"ts"`
}

// DeliveryHistoryRequest holds the parameters for GetDeliveryHistory
type DeliveryHistoryRequest struct {
	InstrumentType   string
	InstrumentFamily string
	After            time.Time // returns records earlier than this time
	Before           time.Time // returns records newer than this time
	Limit            uint64
}

// DeliveryHistoryResponse holds the deliveries or exercises at one time
type DeliveryHistoryResponse struct {
	Timestamp types.Time              `json:"ts"`
	Details   []DeliveryHistoryDetail `json:"details"`
}

// DeliveryHistoryDetail holds one contract's delivery or exercise
type DeliveryHistoryDetail struct {
	InstrumentID string       `json:"insId"`
	Price        types.Number `json:"px"`
	Type         string       `json:"type"`
}

// EstimatedSettlementPriceResponse holds a futures contract's estimated settlement price
type EstimatedSettlementPriceResponse struct {
	InstrumentID             string       `json:"instId"`
	NextSettlementTime       types.Time   `json:"nextSettleTime"`
	EstimatedSettlementPrice types.Number `json:"estSettlePx"`
	Timestamp                types.Time   `json:"ts"`
}

// FuturesSettlementHistoryRequest holds the parameters for GetFuturesSettlementHistory
type FuturesSettlementHistoryRequest struct {
	InstrumentFamily string
	After            time.Time // returns records earlier than this time
	Before           time.Time // returns records newer than this time
	Limit            uint64
}

// FuturesSettlementHistoryResponse holds the futures settlements at one time
type FuturesSettlementHistoryResponse struct {
	Timestamp types.Time                `json:"ts"`
	Details   []FuturesSettlementDetail `json:"details"`
}

// FuturesSettlementDetail holds one futures contract's settlement
type FuturesSettlementDetail struct {
	InstrumentID    string       `json:"instId"`
	SettlementPrice types.Number `json:"settlePx"`
}

// FundingRateHistoryRequest holds the parameters for GetFundingRateHistory
type FundingRateHistoryRequest struct {
	InstrumentID string
	Before       time.Time // returns funding times later than this time
	After        time.Time // returns funding times earlier than this time
	Limit        uint64
}

// OpenInterestRequest holds the parameters for GetOpenInterestData
type OpenInterestRequest struct {
	InstrumentType   string
	InstrumentFamily string
	InstrumentID     string
}

// LimitPriceResponse holds an instrument's highest buy and lowest sell limit prices
type LimitPriceResponse struct {
	InstrumentType string       `json:"instType"`
	InstrumentID   string       `json:"instId"`
	BuyLimit       types.Number `json:"buyLmt"`
	SellLimit      types.Number `json:"sellLmt"`
	Timestamp      types.Time   `json:"ts"`
	Enabled        bool         `json:"enabled"`
}

// OptionMarketDataRequest holds the parameters for GetOptionMarketData
type OptionMarketDataRequest struct {
	InstrumentFamily string
	ExpiryTime       time.Time // the expiry date in UTC
}

// OptionMarketDataResponse holds an option's market data
type OptionMarketDataResponse struct {
	InstrumentType       string       `json:"instType"`
	InstrumentID         string       `json:"instId"`
	Underlying           string       `json:"uly"`
	Delta                types.Number `json:"delta"`
	Gamma                types.Number `json:"gamma"`
	Vega                 types.Number `json:"vega"`
	Theta                types.Number `json:"theta"`
	DeltaBlackScholes    types.Number `json:"deltaBS"`
	GammaBlackScholes    types.Number `json:"gammaBS"`
	VegaBlackScholes     types.Number `json:"vegaBS"`
	ThetaBlackScholes    types.Number `json:"thetaBS"`
	Leverage             types.Number `json:"lever"`
	MarkVolatility       types.Number `json:"markVol"`
	BidVolatility        types.Number `json:"bidVol"`
	AskVolatility        types.Number `json:"askVol"`
	RealisedVolatility   types.Number `json:"realVol"`
	AtTheMoneyVolatility types.Number `json:"volLv"`
	ForwardPrice         types.Number `json:"fwdPx"`
	BuyAPR               types.Number `json:"buyApr"`
	SellAPR              types.Number `json:"sellApr"`
	Distance             types.Number `json:"distance"`
	Timestamp            types.Time   `json:"ts"`
}

// DiscountRateRequest holds the parameters for GetDiscountRateAndInterestFreeQuota
type DiscountRateRequest struct {
	Currency      currency.Code
	DiscountLevel string // deprecated by OKX
}

// DiscountRateResponse holds a currency's collateral discount rate tiers and interest-free quota
type DiscountRateResponse struct {
	Currency                    currency.Code        `json:"ccy"`
	CollateralRestrictionStatus string               `json:"colRes"`
	CollateralRestricted        bool                 `json:"collateralRestrict"`
	InterestFreeQuota           types.Number         `json:"amt"`
	DiscountLevel               string               `json:"discountLv"`
	MinDiscountRate             types.Number         `json:"minDiscountRate"`
	EffectiveTime               types.Time           `json:"effTime"`
	Details                     []DiscountRateDetail `json:"details"`
}

// DiscountRateDetail holds one discount rate tier
type DiscountRateDetail struct {
	DiscountRate           types.Number `json:"discountRate"`
	MaxAmount              types.Number `json:"maxAmt"` // empty, decoding as zero, is unbounded
	MinAmount              types.Number `json:"minAmt"`
	Tier                   string       `json:"tier"`
	LiquidationPenaltyRate types.Number `json:"liqPenaltyRate"`
	DiscountCurrencyEquity types.Number `json:"disCcyEq"`
}

// SystemTimeResponse holds the API server time
type SystemTimeResponse struct {
	Timestamp types.Time `json:"ts"`
}

// MarkPriceRequest holds the parameters for GetMarkPrice
type MarkPriceRequest struct {
	InstrumentType   string
	InstrumentFamily string
	InstrumentID     string
}

// PositionTiersRequest holds the parameters for GetPositionTiers
type PositionTiersRequest struct {
	InstrumentType     string
	TradeMode          string
	InstrumentFamilies []string
	InstrumentIDs      []string
	Currency           currency.Code
	Tier               uint64
}

// PositionTierResponse holds one position tier
type PositionTierResponse struct {
	Underlying                   string       `json:"uly"`
	InstrumentFamily             string       `json:"instFamily"`
	InstrumentID                 string       `json:"instId"`
	Tier                         string       `json:"tier"`
	MinSize                      types.Number `json:"minSz"`
	MaxSize                      types.Number `json:"maxSz"`
	MaintenanceMarginRequirement types.Number `json:"mmr"`
	InitialMarginRequirement     types.Number `json:"imr"`
	MaxLeverage                  types.Number `json:"maxLever"`
	OptionMarginFactor           types.Number `json:"optMgnFactor"`
	QuoteMaxLoan                 types.Number `json:"quoteMaxLoan"`
	BaseMaxLoan                  types.Number `json:"baseMaxLoan"`
}

// InterestRateLoanQuotaResponse holds the borrowing interest rates and loan quotas
type InterestRateLoanQuotaResponse struct {
	Basic                  []InterestRateLoanQuotaBasic    `json:"basic"`
	VIP                    []InterestRateLoanQuotaLevel    `json:"vip"`
	Regular                []InterestRateLoanQuotaLevel    `json:"regular"`
	ConfiguredCurrencyList []InterestRateLoanQuotaCurrency `json:"configCcyList"`
	Configurations         []InterestRateLoanQuotaConfig   `json:"config"`
}

// InterestRateLoanQuotaBasic holds a currency's basic daily borrowing rate and maximum borrow
type InterestRateLoanQuotaBasic struct {
	Currency     currency.Code `json:"ccy"`
	InterestRate types.Number  `json:"rate"`
	LoanQuota    types.Number  `json:"quota"`
}

// InterestRateLoanQuotaLevel holds a user level's loan quota coefficient
type InterestRateLoanQuotaLevel struct {
	UserLevel            string       `json:"level"`
	LoanQuotaCoefficient types.Number `json:"loanQuotaCoef"`
	InterestRateDiscount types.Number `json:"irDiscount"` // deprecated by OKX
}

// InterestRateLoanQuotaCurrency holds a currency whose loan quota is a configured absolute value. OKX documents the
// list as strings but sends objects carrying the daily rate
type InterestRateLoanQuotaCurrency struct {
	Currency     currency.Code `json:"ccy"`
	InterestRate types.Number  `json:"rate"`
}

// InterestRateLoanQuotaConfig holds a currency's loan quota configured as an absolute value
type InterestRateLoanQuotaConfig struct {
	Currency     currency.Code `json:"ccy"`
	StrategyType string        `json:"stgyType"`
	LoanQuota    types.Number  `json:"quota"`
	UserLevel    string        `json:"level"`
}

// InsuranceFundRequest holds the parameters for GetInsuranceFundInformation
type InsuranceFundRequest struct {
	InstrumentType   string
	Type             string
	InstrumentFamily string
	Currency         currency.Code
	Before           time.Time // returns records newer than this time
	After            time.Time // returns records earlier than this time
	Limit            uint64
}

// InsuranceFundResponse holds an instrument type's or family's security fund balance
type InsuranceFundResponse struct {
	Total            types.Number          `json:"total"`
	InstrumentFamily string                `json:"instFamily"`
	InstrumentType   string                `json:"instType"`
	Details          []InsuranceFundDetail `json:"details"`
}

// InsuranceFundDetail holds one security fund balance record
type InsuranceFundDetail struct {
	Balance             types.Number  `json:"balance"`
	Amount              types.Number  `json:"amt"`
	Currency            currency.Code `json:"ccy"`
	Type                string        `json:"type"`
	MaxBalance          types.Number  `json:"maxBal"`
	MaxBalanceTimestamp types.Time    `json:"maxBalTs"`
	DeclineRate         types.Number  `json:"decRate"`
	ADLType             string        `json:"adlType"`
	Timestamp           types.Time    `json:"ts"`
}

// UnitConvertRequest holds the parameters for CurrencyUnitConvert
type UnitConvertRequest struct {
	ConvertType   string // 1: currency to contract, the default; 2: contract to currency
	InstrumentID  string
	Size          float64
	Price         float64
	Unit          string // coin, the default, or usds
	OperationType string // open or close, the default
}

// UnitConvertResponse holds a currency to contract or contract to currency conversion
type UnitConvertResponse struct {
	ConvertType  string       `json:"type"`
	InstrumentID string       `json:"instId"`
	Price        types.Number `json:"px"`
	Size         types.Number `json:"sz"`
	Unit         string       `json:"unit"`
}

// InstrumentTickBandsRequest holds the parameters for GetInstrumentTickBands
type InstrumentTickBandsRequest struct {
	InstrumentType   string
	InstrumentFamily string
}

// InstrumentTickBandResponse holds an instrument family's or event series' tick size bands
type InstrumentTickBandResponse struct {
	InstrumentType   string     `json:"instType"`
	InstrumentFamily string     `json:"instFamily"`
	SeriesID         string     `json:"seriesId"`
	TickBands        []TickBand `json:"tickBand"`
}

// TickBand holds the tick size of a price band
type TickBand struct {
	MinPrice types.Number `json:"minPx"`
	MaxPrice types.Number `json:"maxPx"`
	TickSize types.Number `json:"tickSz"`
}

// PremiumHistoryRequest holds the parameters for GetPremiumHistory
type PremiumHistoryRequest struct {
	InstrumentID string
	After        time.Time // returns records earlier than this time
	Before       time.Time // returns records newer than this time
	Limit        uint64
}

// PremiumHistoryResponse holds a perpetual swap's premium index at one time
type PremiumHistoryResponse struct {
	InstrumentID string       `json:"instId"`
	Premium      types.Number `json:"premium"`
	Timestamp    types.Time   `json:"ts"`
}

// IndexTickersRequest holds the parameters for GetIndexTickers
type IndexTickersRequest struct {
	QuoteCurrency currency.Code
	InstrumentID  string
}

// IndexTickerResponse holds an index ticker
type IndexTickerResponse struct {
	InstrumentID       string       `json:"instId"`
	IndexPrice         types.Number `json:"idxPx"`
	HighestPrice24Hour types.Number `json:"high24h"`
	LowestPrice24Hour  types.Number `json:"low24h"`
	OpenPrice24Hour    types.Number `json:"open24h"`
	OpenPriceUTC0      types.Number `json:"sodUtc0"`
	OpenPriceUTC8      types.Number `json:"sodUtc8"`
	Timestamp          types.Time   `json:"ts"`
}

// PriceCandlesticksRequest holds the parameters for the index and mark price candlestick endpoints
type PriceCandlesticksRequest struct {
	InstrumentID string // an index such as BTC-USD for the index candlesticks
	Bar          kline.Interval
	After        time.Time // returns records earlier than this time
	Before       time.Time // returns records newer than this time
	Limit        uint64
}

// PriceCandlestickResponse holds one index or mark price candlestick, which carries no volume
type PriceCandlestickResponse struct {
	OpenTime     types.Time
	OpenPrice    types.Number
	HighestPrice types.Number
	LowestPrice  types.Number
	ClosePrice   types.Number
	Confirmed    bool
}

// UnmarshalJSON decodes an index or mark price candlestick from its array form
func (c *PriceCandlestickResponse) UnmarshalJSON(data []byte) error {
	var confirm string
	if err := json.Unmarshal(data, &[6]any{&c.OpenTime, &c.OpenPrice, &c.HighestPrice, &c.LowestPrice, &c.ClosePrice, &confirm}); err != nil {
		return err
	}
	c.Confirmed = confirm == "1"
	return nil
}

// ExchangeRateResponse holds the two week average USD to CNY exchange rate
type ExchangeRateResponse struct {
	USDCNY types.Number `json:"usdCny"`
}

// IndexComponentsResponse holds an index's latest price and components
type IndexComponentsResponse struct {
	Index      string           `json:"index"`
	LastPrice  types.Number     `json:"last"`
	Timestamp  types.Time       `json:"ts"`
	Components []IndexComponent `json:"components"`
}

// IndexComponent holds one exchange trading pair of an index
type IndexComponent struct {
	ExchangeName    string       `json:"exch"`
	Symbol          string       `json:"symbol"`
	SymbolPairPrice types.Number `json:"symPx"`
	Weight          types.Number `json:"wgt"`
	ConvertedPrice  types.Number `json:"cnvPx"`
}

// EconomicCalendarRequest holds the parameters for GetEconomicCalendarData
type EconomicCalendarRequest struct {
	Region     string
	Importance string    // 1: low, 2: medium, 3: high
	Before     time.Time // returns records newer than this time
	After      time.Time // returns records earlier than this time
	Limit      uint64
}

// EconomicCalendarEventResponse holds one macro-economic calendar event
type EconomicCalendarEventResponse struct {
	CalendarID           string        `json:"calendarId"`
	Date                 types.Time    `json:"date"`
	Region               string        `json:"region"`
	Category             string        `json:"category"`
	Event                string        `json:"event"`
	ReferenceDate        types.Time    `json:"refDate"`
	Actual               string        `json:"actual"`
	Previous             string        `json:"previous"`
	Forecast             string        `json:"forecast"`
	DateSpan             string        `json:"dateSpan"`
	Importance           string        `json:"importance"`
	UpdateTime           types.Time    `json:"uTime"`
	PreviousInitialValue string        `json:"prevInitial"`
	Currency             currency.Code `json:"ccy"`
	Unit                 string        `json:"unit"`
}

// HistoricalMarketDataRequest holds the parameters for GetHistoricalMarketData
type HistoricalMarketDataRequest struct {
	Module              string
	InstrumentType      string
	InstrumentIDs       []string // spot only
	InstrumentFamilies  []string // all instrument types but spot
	DateAggregationType string   // daily or monthly
	Begin               time.Time
	End                 time.Time
}

// HistoricalMarketDataResponse holds the historical market data files of a query
type HistoricalMarketDataResponse struct {
	Timestamp           types.Time                   `json:"ts"`
	TotalSizeMB         types.Number                 `json:"totalSizeMB"`
	DateAggregationType string                       `json:"dateAggrType"`
	Details             []HistoricalMarketDataDetail `json:"details"`
}

// HistoricalMarketDataDetail holds the historical market data files of one instrument or family
type HistoricalMarketDataDetail struct {
	InstrumentID     string                     `json:"instId"`
	InstrumentFamily string                     `json:"instFamily"`
	InstrumentType   string                     `json:"instType"`
	Currency         currency.Code              `json:"ccy"`
	DateRangeStart   types.Time                 `json:"dateRangeStart"`
	DateRangeEnd     types.Time                 `json:"dateRangeEnd"`
	GroupSizeMB      types.Number               `json:"groupSizeMB"`
	GroupDetails     []HistoricalMarketDataFile `json:"groupDetails"`
}

// HistoricalMarketDataFile holds one historical market data file
type HistoricalMarketDataFile struct {
	Filename string       `json:"filename"`
	DataTime types.Time   `json:"dateTs"`
	SizeMB   types.Number `json:"sizeMB"`
	URL      string       `json:"url"`
}

// UnmarshalJSON decodes a historical market data file, taking its timestamp from dateTs, which OKX sends and its
// example shows, or else from dataTs, which its response table names
func (f *HistoricalMarketDataFile) UnmarshalJSON(data []byte) error {
	type file HistoricalMarketDataFile
	type response struct {
		*file
		DateTimestamp types.Time `json:"dateTs"`
		DataTimestamp types.Time `json:"dataTs"`
	}
	decoded := response{file: (*file)(f)}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	switch {
	case !decoded.DateTimestamp.Time().IsZero():
		f.DataTime = decoded.DateTimestamp
	case !decoded.DataTimestamp.Time().IsZero():
		f.DataTime = decoded.DataTimestamp
	}
	return nil
}

// MMInstrumentTypesRequest holds the parameters for GetMMInstrumentTypes
type MMInstrumentTypesRequest struct {
	InstrumentType string
	InstrumentID   string
}

// MMInstrumentTypeResponse holds an instrument's market maker program classification
type MMInstrumentTypeResponse struct {
	InstrumentID   string `json:"instId"`
	InstrumentType string `json:"instType"`
	PairType       string `json:"pairType"`
}

// DeltaHedgeCurrenciesRequest holds the parameters for GetDeltaHedgeCurrencies
type DeltaHedgeCurrenciesRequest struct {
	Currency currency.Code
}

// DeltaHedgeCurrencyResponse holds the currencies sharing an underlying asset with a currency
type DeltaHedgeCurrencyResponse struct {
	Currency        currency.Code   `json:"ccy"`
	HedgeCurrencies []currency.Code `json:"hedgeCcy"`
}
