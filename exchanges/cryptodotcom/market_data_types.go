package cryptodotcom

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Deposit instrument types besides InstrumentTypeCurrency: tokens a balance is held as collateral, credit or a credit
// limit in, which Crypto.com lists although its docs only show CCY
const (
	InstrumentTypeCollateral  = "COLLATERAL"
	InstrumentTypeCredit      = "CREDIT"
	InstrumentTypeCreditLimit = "CREDITLIMIT"
)

// Product types, the market an instrument trades
const (
	ProductTypeCommodities       = "COMMODITIES"
	ProductTypeEquityIndex       = "EQUITY_IND"
	ProductTypeCurrencies        = "CURRENCIES"
	ProductTypeDigitalCurrencies = "DIGITAL_CURRENCIES"
	ProductTypeEvents            = "EVENTS"
	ProductTypeEquity            = "EQUITY"
	ProductTypePreIPO            = "PRE_IPO"
)

// Valuation types: the index and mark prices, the hourly funding rates settled, the rate settling at the end of the
// current hour and the rate estimated for the next interval
const (
	ValuationTypeIndexPrice           = "index_price"
	ValuationTypeMarkPrice            = "mark_price"
	ValuationTypeFundingHistory       = "funding_hist"
	ValuationTypeFundingRate          = "funding_rate"
	ValuationTypeEstimatedFundingRate = "estimated_funding_rate"
)

// RiskParametersResponse holds Smart Cross Margin's risk parameters: the defaults, and the parameters each base
// currency overrides them with. A position limit of -1 means none applies
type RiskParametersResponse struct {
	DefaultMaxProductLeverageForSpot             types.Number             `json:"default_max_product_leverage_for_spot"`
	DefaultMaxProductLeverageForPerpetuals       types.Number             `json:"default_max_product_leverage_for_perps"`
	DefaultMaxProductLeverageForFutures          types.Number             `json:"default_max_product_leverage_for_futures"`
	DefaultUnitMarginRateMultiplierForSpot       types.Number             `json:"default_umr_multiplier_for_spot"`
	DefaultUnitMarginRateMultiplierForPerpetuals types.Number             `json:"default_umr_multiplier_for_perps"`
	DefaultUnitMarginRateMultiplierForFutures    types.Number             `json:"default_umr_multiplier_for_futures"`
	DefaultLongPositionLimitPerpetuals           types.Number             `json:"default_long_pos_limit_perps"`
	DefaultShortPositionLimitPerpetuals          types.Number             `json:"default_short_pos_limit_perps"`
	DefaultLongPositionLimitFutures              types.Number             `json:"default_long_pos_limit_futures"`
	DefaultShortPositionLimitFutures             types.Number             `json:"default_short_pos_limit_futures"`
	DefaultUnitMarginRate                        types.Number             `json:"default_unit_margin_rate"`
	DefaultCollateralCap                         types.Number             `json:"default_collateral_cap"`
	UpdateTime                                   types.Time               `json:"update_timestamp_ms"`
	BaseCurrencyConfig                           []BaseCurrencyRiskConfig `json:"base_currency_config"`
	// PerpetualSwapConfig and FuturesConfig, which Crypto.com sends although its docs leave them out, cap the notional
	// of each perpetual swap and futures order on a base currency
	PerpetualSwapConfig []BaseCurrencyOrderLimit `json:"perpetual_swap_config"`
	FuturesConfig       []BaseCurrencyOrderLimit `json:"futures_config"`
}

// BaseCurrencyRiskConfig holds the risk parameters of a base currency; a parameter it leaves out takes its default, and
// a currency without a collateral cap or minimum haircut is not eligible as collateral
type BaseCurrencyRiskConfig struct {
	// InstrumentName is a base currency, or a group such as USD_Stable_Coin, whose case a currency code would not keep
	InstrumentName                  string       `json:"instrument_name"`
	CollateralCapNotional           types.Number `json:"collateral_cap_notional"`
	MinimumHaircut                  types.Number `json:"minimum_haircut"`
	MaxProductLeverageForSpot       types.Number `json:"max_product_leverage_for_spot"`
	MaxProductLeverageForPerpetuals types.Number `json:"max_product_leverage_for_perps"`
	MaxProductLeverageForFutures    types.Number `json:"max_product_leverage_for_futures"`
	UnitMarginRate                  types.Number `json:"unit_margin_rate"`
	// MaxShortSellLimit is zero when the currency cannot be sold short
	MaxShortSellLimit types.Number `json:"max_short_sell_limit"`
	// DailyNotionalLimit caps the spot order notional in any 24 hours; zero means no limit
	DailyNotionalLimit           types.Number `json:"daily_notional_limit"`
	OrderLimit                   types.Number `json:"order_limit"`
	MaxOrderNotionalUSD          types.Number `json:"max_order_notional_usd"`
	MinOrderNotionalUSD          types.Number `json:"min_order_notional_usd"`
	LongPositionLimitPerpetuals  types.Number `json:"long_pos_limit_perps"`
	ShortPositionLimitPerpetuals types.Number `json:"short_pos_limit_perps"`
	LongPositionLimitFutures     types.Number `json:"long_pos_limit_futures"`
	ShortPositionLimitFutures    types.Number `json:"short_pos_limit_futures"`
}

// BaseCurrencyOrderLimit caps the notional of an order on a base currency
type BaseCurrencyOrderLimit struct {
	// InstrumentName is a base currency, as in BaseCurrencyRiskConfig
	InstrumentName      string       `json:"instrument_name"`
	MaxOrderNotionalUSD types.Number `json:"max_order_notional_usd"`
}

// InstrumentResponse is an instrument Crypto.com trades
type InstrumentResponse struct {
	// Symbol is the instrument name, such as BTC_USD, BTCUSD-PERP or BTCUSD-261030
	Symbol         string        `json:"symbol"`
	InstrumentType string        `json:"inst_type"`
	DisplayName    string        `json:"display_name"`
	BaseCurrency   currency.Code `json:"base_ccy"`
	QuoteCurrency  currency.Code `json:"quote_ccy"`
	// QuoteDecimals and QuantityDecimals are numbers live although the docs show them quoted
	QuoteDecimals     types.Number `json:"quote_decimals"`
	QuantityDecimals  types.Number `json:"quantity_decimals"`
	PriceTickSize     types.Number `json:"price_tick_size"`
	QuantityTickSize  types.Number `json:"qty_tick_size"`
	MaxLeverage       types.Number `json:"max_leverage"`
	Tradable          bool         `json:"tradable"`
	ExpiryTime        types.Time   `json:"expiry_timestamp_ms"`
	BetaProduct       bool         `json:"beta_product"`
	UnderlyingSymbol  string       `json:"underlying_symbol"`
	ProductType       string       `json:"product_type"`
	ContractSize      types.Number `json:"contract_size"`
	MarginBuyEnabled  bool         `json:"margin_buy_enabled"`
	MarginSellEnabled bool         `json:"margin_sell_enabled"`
}

// OrderbookRequest is the params of a public/get-book request
type OrderbookRequest struct {
	InstrumentName string
	// Depth is how many bids and asks to return, up to 150; zero returns 150
	Depth uint64
}

// OrderbookResponse is an instrument's order book
type OrderbookResponse struct {
	Depth          types.Number        `json:"depth"`
	InstrumentName string              `json:"instrument_name"`
	Data           []OrderbookSnapshot `json:"data"`
}

// OrderbookSnapshot holds an order book's bids, best first, and asks, best first
type OrderbookSnapshot struct {
	Bids      []OrderbookLevel `json:"bids"`
	Asks      []OrderbookLevel `json:"asks"`
	Timestamp types.Time       `json:"t"`
}

// OrderbookLevel is a price level of an order book, which Crypto.com sends as a [price, quantity, number of orders]
// array
type OrderbookLevel struct {
	Price      types.Number
	Quantity   types.Number
	OrderCount uint64
}

// CandlestickRequest is the params of a public/get-candlestick request
type CandlestickRequest struct {
	InstrumentName string
	// Interval is each candle's period; zero requests Crypto.com's default of one minute
	Interval kline.Interval
	// Count is how many candles to return, up to 300; zero returns 25
	Count     uint64
	StartTime time.Time
	EndTime   time.Time
}

// CandlestickResponse holds an instrument's candles
type CandlestickResponse struct {
	// Interval is the timeframe requested, such as 1h or 1D
	Interval       string   `json:"interval"`
	InstrumentName string   `json:"instrument_name"`
	Data           []Candle `json:"data"`
}

// Candle is an instrument's prices and volume over a candle's period
type Candle struct {
	Open      types.Number `json:"o"`
	High      types.Number `json:"h"`
	Low       types.Number `json:"l"`
	Close     types.Number `json:"c"`
	Volume    types.Number `json:"v"`
	StartTime types.Time   `json:"t"`
}

// TradesRequest is the params of a public/get-trades request
type TradesRequest struct {
	InstrumentName string
	// Count is how many trades to return, up to 150; zero returns 25
	Count     uint64
	StartTime time.Time
	EndTime   time.Time
}

// TradeResponse is a public trade
type TradeResponse struct {
	// TradeID is unique, but trades filling one order share a TradeTimeNano and a MatchID
	TradeID       string       `json:"d"`
	TradeTime     types.Time   `json:"t"`
	TradeTimeNano types.Time   `json:"tn"`
	Quantity      types.Number `json:"q"`
	Price         types.Number `json:"p"`
	// Side is the taker's, which REST sends in lower case although the docs show BUY and SELL
	Side           string `json:"s"`
	InstrumentName string `json:"i"`
	MatchID        string `json:"m"`
}

// TickersRequest is the params of a public/get-tickers request; an empty instrument name requests every instrument's
// ticker
type TickersRequest struct {
	InstrumentName string
}

// TickerResponse is an instrument's ticker; the prices are zero when there have been no trades, bids or asks
type TickerResponse struct {
	InstrumentName string       `json:"i"`
	High24Hour     types.Number `json:"h"`
	Low24Hour      types.Number `json:"l"`
	LastPrice      types.Number `json:"a"`
	Volume24Hour   types.Number `json:"v"`
	// USDVolume24Hour is the volume's value in USD, whatever the instrument's quote currency
	USDVolume24Hour types.Number `json:"vv"`
	// PriceChange24Hour is a fraction of the price 24 hours before: 0.0005 is a rise of 0.05%
	PriceChange24Hour types.Number `json:"c"`
	BestBidPrice      types.Number `json:"b"`
	BestAskPrice      types.Number `json:"k"`
	OpenInterest      types.Number `json:"oi"`
	Timestamp         types.Time   `json:"t"`
}

// ValuationsRequest is the params of a public/get-valuations request
type ValuationsRequest struct {
	InstrumentName string
	ValuationType  string
	// Count is how many values to return, up to 1000; zero returns 25
	Count     uint64
	StartTime time.Time
	EndTime   time.Time
}

// ValuationsResponse holds an instrument's values of a valuation type
type ValuationsResponse struct {
	InstrumentName string      `json:"instrument_name"`
	Data           []Valuation `json:"data"`
}

// Valuation is a price or rate at a time
type Valuation struct {
	Value     types.Number `json:"v"`
	Timestamp types.Time   `json:"t"`
}

// ExpiredSettlementPriceRequest is the params of a public/get-expired-settlement-price request
type ExpiredSettlementPriceRequest struct {
	InstrumentType string
	// Page is the page of ten prices to return, counting from 1 for the latest expiries; zero returns the first
	Page uint64
}

// ExpiredSettlementPriceResponse is the price an expired instrument settled at
type ExpiredSettlementPriceResponse struct {
	InstrumentName  string       `json:"i"`
	ExpiryTime      types.Time   `json:"x"`
	SettlementPrice types.Number `json:"v"`
	Timestamp       types.Time   `json:"t"`
}

// DepositInstrumentResponse is a currency or token that can be deposited, and whether it counts as collateral
type DepositInstrumentResponse struct {
	// Symbol is a currency, or a collateral, credit or staked token such as SOL.staked-COLLATERAL, whose case a currency
	// code would not keep
	Symbol             string        `json:"symbol"`
	InstrumentType     string        `json:"inst_type"`
	DisplayName        string        `json:"display_name"`
	BaseCurrency       currency.Code `json:"base_ccy"`
	QuoteCurrency      currency.Code `json:"quote_ccy"`
	QuoteDecimals      uint64        `json:"quote_decimals"`
	QuantityDecimals   uint64        `json:"quantity_decimals"`
	Haircut            types.Number  `json:"haircut"`
	CollateralEligible bool          `json:"collateral_eligible"`
	PriceTickSize      types.Number  `json:"price_tick_size"`
	QuantityTickSize   types.Number  `json:"qty_tick_size"`
	MaxLeverage        types.Number  `json:"max_leverage"`
	Tradable           bool          `json:"tradable"`
	ExpiryTime         types.Time    `json:"expiry_timestamp_ms"`
	BetaProduct        bool          `json:"beta_product"`
	ConversionFee      types.Number  `json:"conversion_fee"`
	MarginBuyEnabled   bool          `json:"margin_buy_enabled"`
	MarginSellEnabled  bool          `json:"margin_sell_enabled"`
	InstrumentID       uint64        `json:"inst_id"`
	// ProductType and UnderlyingSymbol are sent for some tokens although the docs leave them out; UnderlyingSymbol can
	// name a group such as USD_Stable_Coin
	ProductType      string `json:"product_type"`
	UnderlyingSymbol string `json:"underlying_symbol"`
}

// InsuranceRequest is the params of a public/get-insurance request
type InsuranceRequest struct {
	Currency currency.Code
	// Count is how many balances to return; zero returns 25
	Count     uint64
	StartTime time.Time
	EndTime   time.Time
}

// InsuranceResponse holds a currency's insurance fund balances
type InsuranceResponse struct {
	Currency currency.Code          `json:"instrument_name"`
	Data     []InsuranceFundBalance `json:"data"`
}

// InsuranceFundBalance is the insurance fund's balance at a time
type InsuranceFundBalance struct {
	Balance   types.Number `json:"v"`
	Timestamp types.Time   `json:"t"`
}
