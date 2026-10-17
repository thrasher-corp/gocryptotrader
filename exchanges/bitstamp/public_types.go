package bitstamp

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Market types
const (
	MarketTypeSpot      = "SPOT"
	MarketTypePerpetual = "PERPETUAL"
)

// Public transaction time periods
const (
	TransactionPeriodMinute = "minute"
	TransactionPeriodHour   = "hour"
	TransactionPeriodDay    = "day"
)

// Market broker entities used to filter markets
const (
	EntityUKLtd     = "UK_LTD"
	EntityEuropeSA  = "EUROPE_SA"
	EntityUSInc     = "US_INC"
	EntityBVILtd    = "BVI_LTD"
	EntityEuropeBEL = "EUROPE_BEL"
	EntityUK        = "UK"
	EntitySIBroker  = "SI_BROKER"
	EntitySIMTF     = "SI_MTF"
	EntityAsiaSG    = "ASIA_SG"
)

// Availability of market trading and of currency or network deposits and withdrawals
const (
	StatusEnabled  = "Enabled"
	StatusDisabled = "Disabled"
)

// Currency types
const (
	CurrencyTypeCrypto = "crypto"
	CurrencyTypeFiat   = "fiat"
)

// Market asset classes
const (
	AssetClassCrypto      = "CRYPTO"
	AssetClassETF         = "ETF"
	AssetClassCommodities = "COMMODITIES"
	AssetClassFX          = "FX"
)

// OrderbookGrouping determines how orders are aggregated in orderbook responses
type OrderbookGrouping uint8

// Orderbook groupings
const (
	OrderbookUngrouped OrderbookGrouping = iota
	OrderbookGroupedByPrice
	OrderbookUngroupedWithOrderIDs
)

// CurrencyResponse holds information for a listed currency
type CurrencyResponse struct {
	Name            string            `json:"name"`
	Currency        currency.Code     `json:"currency"`
	Type            string            `json:"type"`
	Symbol          string            `json:"symbol"`
	Decimals        uint8             `json:"decimals"`
	Logo            string            `json:"logo"`
	AvailableSupply types.Number      `json:"available_supply"`
	Deposit         string            `json:"deposit"`
	Withdrawal      string            `json:"withdrawal"`
	Networks        []CurrencyNetwork `json:"networks"`
}

// CurrencyNetwork holds deposit and withdrawal details for a currency network
type CurrencyNetwork struct {
	Network                 string       `json:"network"`
	WithdrawalMinimumAmount types.Number `json:"withdrawal_minimum_amount"`
	WithdrawalDecimals      uint8        `json:"withdrawal_decimals"`
	Deposit                 string       `json:"deposit"`
	Withdrawal              string       `json:"withdrawal"`
}

// TickerResponse holds ticker information for a market
type TickerResponse struct {
	Last                       types.Number `json:"last"`
	High                       types.Number `json:"high"`
	Low                        types.Number `json:"low"`
	VolumeWeightedAveragePrice types.Number `json:"vwap"`
	Volume                     types.Number `json:"volume"`
	BestBid                    types.Number `json:"bid"`
	BestAsk                    types.Number `json:"ask"`
	Timestamp                  types.Time   `json:"timestamp"`
	Open                       types.Number `json:"open"`
	Open24Hour                 types.Number `json:"open_24"`
	PercentChange24Hour        types.Number `json:"percent_change_24"`
	Side                       OrderSide    `json:"side"`
	MarketType                 string       `json:"market_type"`
	MarkPrice                  types.Number `json:"mark_price"`
	IndexPrice                 types.Number `json:"index_price"`
	OpenInterest               types.Number `json:"open_interest"`
	OpenInterestValue          types.Number `json:"open_interest_value"`
	// Market is only populated by the all markets ticker endpoint
	Market string `json:"market"`
}

// OrderbookResponse holds orderbook information
type OrderbookResponse struct {
	Timestamp      types.Time       `json:"timestamp"`
	Microtimestamp types.Time       `json:"microtimestamp"`
	Bids           []OrderbookLevel `json:"bids"`
	Asks           []OrderbookLevel `json:"asks"`
}

// OrderbookLevel holds an orderbook price level
// OrderID is only populated when the orderbook is requested with OrderbookUngroupedWithOrderIDs
type OrderbookLevel struct {
	Price   float64
	Amount  float64
	OrderID uint64
}

// TransactionResponse holds a public market trade
type TransactionResponse struct {
	Date    types.Time   `json:"date"`
	TradeID uint64       `json:"tid,string"`
	Price   types.Number `json:"price"`
	Side    OrderSide    `json:"type"`
	Amount  types.Number `json:"amount"`
}

// MarketResponse holds market information
type MarketResponse struct {
	Name                        string        `json:"name"`
	MarketSymbol                string        `json:"market_symbol"`
	BaseCurrency                currency.Code `json:"base_currency"`
	BaseDecimals                uint8         `json:"base_decimals"`
	CounterCurrency             currency.Code `json:"counter_currency"`
	CounterDecimals             uint8         `json:"counter_decimals"`
	MinimumOrderValue           types.Number  `json:"minimum_order_value"`
	MaximumOrderValue           types.Number  `json:"maximum_order_value"`
	MinimumOrderAmount          types.Number  `json:"minimum_order_amount"`
	MaximumOrderAmount          types.Number  `json:"maximum_order_amount"`
	Trading                     string        `json:"trading"`
	InstantOrderCounterDecimals uint8         `json:"instant_order_counter_decimals"`
	InstantAndMarketOrders      string        `json:"instant_and_market_orders"`
	Description                 string        `json:"description"`
	MarketType                  string        `json:"market_type"`
	UnderlyingAsset             string        `json:"underlying_asset"`
	PayoffType                  string        `json:"payoff_type"`
	ContractSize                types.Number  `json:"contract_size"`
	MaxLeverage                 types.Number  `json:"max_leverage"`
	TickSize                    types.Number  `json:"tick_size"`
	ISIN                        string        `json:"isin"`
	AssetClass                  string        `json:"asset_class"`
	HasMarketHours              bool          `json:"has_market_hours"`
	Exchange                    string        `json:"exchange"`
}

// OHLCRequest holds the parameters for fetching candle data
// When both Start and End are set the API ignores Start and returns up to Limit candles ending at End
type OHLCRequest struct {
	Pair                 currency.Pair
	Step                 kline.Interval
	Limit                uint64
	Start                time.Time
	End                  time.Time
	ExcludeCurrentCandle bool
}

// OHLCResponse holds candle data
type OHLCResponse struct {
	Data OHLCData `json:"data"`
}

// OHLCData holds the candles for a market
type OHLCData struct {
	// Pair is only populated for spot markets and Market is only populated for perpetual markets
	Pair   string   `json:"pair"`
	Market string   `json:"market"`
	OHLC   []Candle `json:"ohlc"`
}

// Candle holds an individual candle
type Candle struct {
	Timestamp types.Time   `json:"timestamp"`
	Open      types.Number `json:"open"`
	High      types.Number `json:"high"`
	Low       types.Number `json:"low"`
	Close     types.Number `json:"close"`
	Volume    types.Number `json:"volume"`
}

// EURUSDConversionRateResponse holds the EUR/USD buy and sell conversion rates
type EURUSDConversionRateResponse struct {
	Buy  types.Number `json:"buy"`
	Sell types.Number `json:"sell"`
}

// FundingRateResponse holds the current funding rate for a perpetual market
type FundingRateResponse struct {
	FundingRate     types.Number `json:"funding_rate"`
	Timestamp       types.Time   `json:"timestamp"`
	Market          string       `json:"market"`
	NextFundingTime types.Time   `json:"next_funding_time"`
}

// FundingRateHistoryRequest holds the parameters for fetching funding rate history
type FundingRateHistoryRequest struct {
	Pair  currency.Pair
	Limit uint64
	Since time.Time
	Until time.Time
}

// FundingRateHistoryResponse holds the funding rate history for a perpetual market
type FundingRateHistoryResponse struct {
	Market             string        `json:"market"`
	FundingRateHistory []FundingRate `json:"funding_rate_history"`
}

// FundingRate holds a historical funding rate
type FundingRate struct {
	FundingRate types.Number `json:"funding_rate"`
	Timestamp   types.Time   `json:"timestamp"`
}

// VASPListResponse holds Virtual Asset Service Providers used for Travel Rule compliance
type VASPListResponse struct {
	Data       []VASP     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// VASP holds a Virtual Asset Service Provider
type VASP struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// Pagination holds paging information
type Pagination struct {
	Page  uint64 `json:"page"`
	Size  uint64 `json:"size"`
	Count uint64 `json:"count"`
}
