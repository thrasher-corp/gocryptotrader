package coinbase

import (
	"bytes"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// DerivativesCurrencyGrouped selects every currency's expiration labels, keyed by currency, in an expirations request
const DerivativesCurrencyGrouped = "grouped"

// Gateway funding chart lengths
const (
	DerivativesFundingChartLength8Hour    = "8h"
	DerivativesFundingChartLength24Hour   = "24h"
	DerivativesFundingChartLengthOneMonth = "1m"
)

// Gateway index chart ranges
const (
	DerivativesIndexChartRangeOneHour  = "1h"
	DerivativesIndexChartRangeOneDay   = "1d"
	DerivativesIndexChartRangeTwoDays  = "2d"
	DerivativesIndexChartRangeOneMonth = "1m"
	DerivativesIndexChartRangeOneYear  = "1y"
	DerivativesIndexChartRangeAll      = "all"
)

// Gateway price index types, which select the names a supported index names request returns
const (
	DerivativesIndexTypeAll        = "all"
	DerivativesIndexTypeSpot       = "spot"
	DerivativesIndexTypeDerivative = "derivative"
)

// Gateway settlement event types; an unfiltered settlements request also returns the exercise of expired options
const (
	DerivativesSettlementTypeSettlement = "settlement"
	DerivativesSettlementTypeDelivery   = "delivery"
	DerivativesSettlementTypeBankruptcy = "bankruptcy"
)

// DerivativesBookSummaryByCurrencyRequest selects the market summaries of a currency's instruments
type DerivativesBookSummaryByCurrencyRequest struct {
	// Currency is a settlement currency, such as BTC for the inverse instruments or USDC for the linear ones; the
	// gateway does not accept any
	Currency currency.Code
	// Kind is future, option, spot, future_combo or option_combo; every kind is summarised when it is unset
	Kind string
}

// derivativesBookSummaryByCurrencyParams are public/get_book_summary_by_currency's params
type derivativesBookSummaryByCurrencyParams struct {
	Currency string `json:"currency"`
	Kind     string `json:"kind,omitempty"`
}

// DerivativesBookSummary is an instrument's market summary; prices the gateway sends as null, such as the bid of an
// empty book or the last price of an instrument that has not traded in 24 hours, are zero
type DerivativesBookSummary struct {
	InstrumentName string        `json:"instrument_name"`
	BaseCurrency   currency.Code `json:"base_currency"`
	QuoteCurrency  currency.Code `json:"quote_currency"`
	// CreationTimestamp is when the gateway compiled the summary, not when the instrument was created
	CreationTimestamp types.Time `json:"creation_timestamp"`
	BidPrice          float64    `json:"bid_price"`
	AskPrice          float64    `json:"ask_price"`
	MidPrice          float64    `json:"mid_price"`
	MarkPrice         float64    `json:"mark_price"`
	LastPrice         float64    `json:"last"`
	// High, Low and PriceChange, a percentage, cover 24 hours
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	PriceChange float64 `json:"price_change"`
	// Volume is the 24 hour volume in the base currency, VolumeUSD in USD and VolumeNotional in the quote currency;
	// options carry no notional volume, and only some spot pairs carry either of the latter
	Volume         float64 `json:"volume"`
	VolumeUSD      float64 `json:"volume_usd"`
	VolumeNotional float64 `json:"volume_notional"`
	// OpenInterest is in USD for inverse futures and perpetuals, and in the base currency for options and linear
	// instruments
	OpenInterest           float64 `json:"open_interest"`
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
	// CurrentFunding is a perpetual's instantaneous funding rate, and Funding8Hour the rate accumulated towards its next
	// settlement
	CurrentFunding float64 `json:"current_funding"`
	Funding8Hour   float64 `json:"funding_8h"`
	// MarkImpliedVolatility, InterestRate, UnderlyingIndex and UnderlyingPrice are an option's; UnderlyingIndex is the
	// underlying future, or index_price
	MarkImpliedVolatility float64 `json:"mark_iv"`
	InterestRate          float64 `json:"interest_rate"`
	UnderlyingIndex       string  `json:"underlying_index"`
	UnderlyingPrice       float64 `json:"underlying_price"`
}

// DerivativesContractSizeResponse is an instrument's contract size, in USD for inverse futures and perpetuals and in
// the base currency for options, spot and linear instruments: 10 USD for BTC-PERPETUAL and 0.0001 BTC for
// BTC_USDC-PERPETUAL. It is not the amount increment, which is the instrument's MinimumTradeAmount
type DerivativesContractSizeResponse struct {
	ContractSize float64 `json:"contract_size"`
}

// DerivativesCurrency is a currency the gateway supports and its deposit and withdrawal parameters
type DerivativesCurrency struct {
	Currency         currency.Code `json:"currency"`
	CurrencyLongName string        `json:"currency_long"`
	CurrencyUUID     string        `json:"currency_uuid"`
	// CoinType is the currency's symbol, not the BITCOIN or ETHER the docs describe
	CoinType string `json:"coin_type"`
	Decimals uint64 `json:"decimals"`
	// OnchainOperationsPrecision is sent for every currency, although the docs leave it out
	OnchainOperationsPrecision uint64                          `json:"onchain_operations_precision"`
	MinimumConfirmations       uint64                          `json:"min_confirmations"`
	WithdrawalFee              float64                         `json:"withdrawal_fee"`
	MinimumWithdrawalFee       float64                         `json:"min_withdrawal_fee"`
	WithdrawalPriorities       []DerivativesWithdrawalPriority `json:"withdrawal_priorities"`
	NetworkFee                 float64                         `json:"network_fee"`
	NetworkCurrency            currency.Code                   `json:"network_currency"`
	InCrossCollateralPool      bool                            `json:"in_cross_collateral_pool"`
	// APR is a yield-bearing token's average daily reward over the last 7 days, as a percentage
	APR float64 `json:"apr"`
	// CoinbaseNetworks are the networks a Coinbase wallet can use the currency on, which the docs leave out
	CoinbaseNetworks []DerivativesCoinbaseNetwork `json:"coinbase_networks"`
}

// DerivativesWithdrawalPriority is a withdrawal priority and its fee rate
type DerivativesWithdrawalPriority struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// DerivativesCoinbaseNetwork is a network a Coinbase wallet can use a currency on; its resource name is the network
// wallet methods take
type DerivativesCoinbaseNetwork struct {
	ResourceName string `json:"resource_name"`
	DisplayName  string `json:"display_name"`
}

// DerivativesDeliveryPricesRequest selects a page of an index's delivery prices, newest first
type DerivativesDeliveryPricesRequest struct {
	// IndexName is a price index, such as btc_usdc; index names are lower case
	IndexName string
	Offset    uint64
	// Count is how many prices the page holds, 10 when it is unset; the gateway returns at most 100 whatever the count,
	// although the docs allow 1000
	Count uint64
}

// derivativesDeliveryPricesParams are public/get_delivery_prices's params
type derivativesDeliveryPricesParams struct {
	IndexName string `json:"index_name"`
	Offset    uint64 `json:"offset,omitempty"`
	Count     uint64 `json:"count,omitempty"`
}

// DerivativesDeliveryPricesResponse is a page of an index's delivery prices and the number of prices available
type DerivativesDeliveryPricesResponse struct {
	Data         []DerivativesDeliveryPrice `json:"data"`
	RecordsTotal uint64                     `json:"records_total"`
}

// DerivativesDeliveryPrice is the price an index's futures and options were delivered at on a day
type DerivativesDeliveryPrice struct {
	// Date is the delivery's UTC date, formatted 2006-01-02
	Date          string  `json:"date"`
	DeliveryPrice float64 `json:"delivery_price"`
}

// DerivativesExpirationsRequest selects the expiration labels of listed futures and options; spot and combos have none
type DerivativesExpirationsRequest struct {
	// Currency is a settlement currency, any for every currency's labels merged, or DerivativesCurrencyGrouped for each
	// currency's labels; the docs require it, but the gateway merges every currency's labels when it and CurrencyPair are
	// unset
	Currency currency.Code
	// Kind is future, option or any for both, the default; the docs require it
	Kind string
	// CurrencyPair is an index pair, such as btc_usdc, whose labels are returned keyed by the pair; the gateway ignores it
	// when Currency is set
	CurrencyPair string
}

// derivativesExpirationsParams are public/get_expirations's params
type derivativesExpirationsParams struct {
	Currency     string `json:"currency,omitempty"`
	Kind         string `json:"kind,omitempty"`
	CurrencyPair string `json:"currency_pair,omitempty"`
}

// DerivativesExpirationsResponse holds expiration labels, such as 27NOV26 or PERPETUAL: merged across currencies when
// every currency is requested, otherwise in ByCurrency
type DerivativesExpirationsResponse struct {
	DerivativesExpirations
	// ByCurrency holds the labels keyed by lower case currency, or by index pair when a currency pair is requested
	ByCurrency map[string]DerivativesExpirations
}

// UnmarshalJSON decodes expiration labels merged across currencies, which the gateway keys by kind, or keyed by
// currency
func (r *DerivativesExpirationsResponse) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, value := range fields {
		if value = bytes.TrimSpace(value); len(value) > 0 && value[0] == '[' {
			return json.Unmarshal(data, &r.DerivativesExpirations)
		}
	}
	return json.Unmarshal(data, &r.ByCurrency)
}

// DerivativesExpirations are the expiration labels of futures and options
type DerivativesExpirations struct {
	Future []string `json:"future"`
	Option []string `json:"option"`
}

// DerivativesFundingChartDataRequest selects a perpetual's funding chart
type DerivativesFundingChartDataRequest struct {
	InstrumentName string
	// Length is the period charted: 8h or 24h, a point a minute, or 1m for a month, a point an hour
	Length string
}

// derivativesFundingChartDataParams are public/get_funding_chart_data's params
type derivativesFundingChartDataParams struct {
	InstrumentName string `json:"instrument_name"`
	Length         string `json:"length"`
}

// DerivativesFundingChartDataResponse is a perpetual's current funding and its funding chart
type DerivativesFundingChartDataResponse struct {
	CurrentInterest float64                        `json:"current_interest"`
	Interest8Hour   float64                        `json:"interest_8h"`
	Data            []DerivativesFundingChartPoint `json:"data"`
}

// DerivativesFundingChartPoint is a perpetual's 8 hour funding and its index price at a point of its funding chart
type DerivativesFundingChartPoint struct {
	Timestamp     types.Time `json:"timestamp"`
	IndexPrice    float64    `json:"index_price"`
	Interest8Hour float64    `json:"interest_8h"`
}

// DerivativesInstrumentTimeRangeRequest selects an instrument's data over a time range, both ends of which are required
type DerivativesInstrumentTimeRangeRequest struct {
	InstrumentName string
	StartTime      time.Time
	EndTime        time.Time
}

// derivativesInstrumentTimeRangeParams are the params of a request for an instrument's data over a time range, its
// ends in Unix milliseconds
type derivativesInstrumentTimeRangeParams struct {
	InstrumentName string `json:"instrument_name"`
	StartTimestamp int64  `json:"start_timestamp"`
	EndTimestamp   int64  `json:"end_timestamp"`
}

// DerivativesFundingRateHistory is a perpetual's funding over an hour
type DerivativesFundingRateHistory struct {
	// Timestamp is the end of the hour
	Timestamp  types.Time `json:"timestamp"`
	IndexPrice float64    `json:"index_price"`
	// PreviousIndexPrice is the index price at the start of the hour
	PreviousIndexPrice float64 `json:"prev_index_price"`
	// Interest1Hour is the hour's funding rate, and Interest8Hour the 8 hour rate at the end of the hour
	Interest1Hour float64 `json:"interest_1h"`
	Interest8Hour float64 `json:"interest_8h"`
}

// DerivativesHistoricalVolatility is a currency's realised volatility at an hour, as a percentage
type DerivativesHistoricalVolatility struct {
	Timestamp  types.Time
	Volatility float64
}

// UnmarshalJSON decodes a [timestamp, volatility] tuple
func (h *DerivativesHistoricalVolatility) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &h.Timestamp, &h.Volatility)
}

// DerivativesIndexChartDataRequest selects an index's price chart
type DerivativesIndexChartDataRequest struct {
	// IndexName is a price index, such as btc_usdc
	IndexName string
	// Range is the period charted: 1h, 1d, 2d, 1m, 1y or all
	Range string
}

// derivativesIndexChartDataParams are public/get_index_chart_data's params
type derivativesIndexChartDataParams struct {
	IndexName string `json:"index_name"`
	Range     string `json:"range"`
}

// DerivativesIndexChartPoint is an index's average price at a point of its chart
type DerivativesIndexChartPoint struct {
	Timestamp types.Time
	Price     float64
}

// UnmarshalJSON decodes a [timestamp, price] tuple
func (i *DerivativesIndexChartPoint) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &i.Timestamp, &i.Price)
}

// derivativesIndexNameParams are the params of a request for a price index
type derivativesIndexNameParams struct {
	IndexName string `json:"index_name"`
}

// DerivativesIndexPriceResponse is an index's current price and the price its futures and options would deliver at
type DerivativesIndexPriceResponse struct {
	IndexPrice             float64 `json:"index_price"`
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
}

// derivativesMarketDataExtendedParams are the params of a request whose extended flag adds details to its result
type derivativesMarketDataExtendedParams struct {
	Extended bool `json:"extended,omitempty"`
}

// DerivativesIndexPriceName is a price index's name and, when extended details are requested, whether future and option
// combos can be created on it; the gateway only sends a setting that is enabled
type DerivativesIndexPriceName struct {
	Name                       string `json:"name"`
	FutureComboCreationEnabled bool   `json:"future_combo_creation_enabled"`
	OptionComboCreationEnabled bool   `json:"option_combo_creation_enabled"`
}

// UnmarshalJSON decodes an index's name, which the gateway sends as a bare string unless extended details are requested
func (i *DerivativesIndexPriceName) UnmarshalJSON(data []byte) error {
	if data = bytes.TrimSpace(data); len(data) > 0 && data[0] == '"' {
		return json.Unmarshal(data, &i.Name)
	}
	type indexPriceName DerivativesIndexPriceName
	return json.Unmarshal(data, (*indexPriceName)(i))
}

// DerivativesInstrument is an instrument's specification; fields only some kinds carry are zero for the others
type DerivativesInstrument struct {
	InstrumentName string `json:"instrument_name"`
	InstrumentID   uint64 `json:"instrument_id"`
	// Kind is future, option, spot, future_combo or option_combo
	Kind string `json:"kind"`
	// InstrumentType is linear or reversed, for every kind; FutureType is its deprecated equivalent for futures
	InstrumentType string `json:"instrument_type"`
	FutureType     string `json:"future_type"`
	// OptionType, call or put, and Strike are an option's
	OptionType string  `json:"option_type"`
	Strike     float64 `json:"strike"`
	// State is the book's state: open, settlement, delivered, inactive, locked, halted or archivized. Only an open book
	// accepts orders, while IsActive is set whenever the book is visible
	State    string `json:"state"`
	IsActive bool   `json:"is_active"`
	// BaseCurrency is the underlying currency, QuoteCurrency the currency prices are quoted in, which is the base currency
	// for inverse options, and CounterCurrency the quote currency of the underlying pair
	BaseCurrency      currency.Code `json:"base_currency"`
	QuoteCurrency     currency.Code `json:"quote_currency"`
	CounterCurrency   currency.Code `json:"counter_currency"`
	BaseCurrencyUUID  string        `json:"base_currency_uuid"`
	QuoteCurrencyUUID string        `json:"quote_currency_uuid"`
	// SettlementCurrency and SettlementPeriod, day, week, month or perpetual, are left out for spot
	SettlementCurrency currency.Code `json:"settlement_currency"`
	SettlementPeriod   string        `json:"settlement_period"`
	PriceIndex         string        `json:"price_index"`
	IndexID            uint64        `json:"index_id"`
	// ProductGroup is BTC, ETH, TIER_2, TIER_3 or RWA
	ProductGroup string `json:"product_group"`
	// UnderlyingType is crypto, equity, commodity, preipo, equity_etf, crypto_index, adr, foreign_equity, equity_index,
	// commodity_index, commodity_etf or otc
	UnderlyingType    string     `json:"underlying_type"`
	CreationTimestamp types.Time `json:"creation_timestamp"`
	// ExpirationTimestamp is in the year 3000 for perpetuals and spot
	ExpirationTimestamp types.Time `json:"expiration_timestamp"`
	TickSize            float64    `json:"tick_size"`
	// TickSizeSteps are the larger tick sizes applying above a price
	TickSizeSteps []DerivativesTickSizeStep `json:"tick_size_steps"`
	// ContractSize converts between amounts and contracts, in USD for inverse futures and perpetuals and in the base
	// currency otherwise; MinimumTradeAmount is the minimum amount and its increment
	ContractSize       float64 `json:"contract_size"`
	MinimumTradeAmount float64 `json:"min_trade_amount"`
	// QuantityTickSize is the amount increment the docs describe, which the gateway does not send
	QuantityTickSize float64 `json:"qty_tick_size"`
	// LotSize is the unit fees are counted in
	LotSize                      float64 `json:"lot_size"`
	MakerCommission              float64 `json:"maker_commission"`
	TakerCommission              float64 `json:"taker_commission"`
	MaximumLiquidationCommission float64 `json:"max_liquidation_commission"`
	// The maximum commissions are only sent with extended details
	MakerMaximumCommission       float64 `json:"maker_max_commission"`
	TakerMaximumCommission       float64 `json:"taker_max_commission"`
	LiquidationMaximumCommission float64 `json:"liquidation_max_commission"`
	SettlementMaximumCommission  float64 `json:"settlement_max_commission"`
	// MarginParameters and TradingBandwidthFraction are a future's, and PerpetualDampener and PerpetualMaximumFunding a
	// perpetual's, only sent with extended details
	MarginParameters         *DerivativesMarginParameters `json:"margin_params"`
	TradingBandwidthFraction float64                      `json:"trading_bandwidth_fraction"`
	PerpetualDampener        float64                      `json:"perpetual_dempener"`
	PerpetualMaximumFunding  float64                      `json:"perpetual_max_funding"`
	// The block trade fields are left out for Coinbase-routed spot, which does not support block trades
	BlockTradeCommission         float64 `json:"block_trade_commission"`
	BlockTradeTickSize           float64 `json:"block_trade_tick_size"`
	BlockTradeMinimumTradeAmount float64 `json:"block_trade_min_trade_amount"`
	// MaximumLeverage is a future's, or a spot pair's with margin loans, for the default risk profile, and
	// MaximumNonDefaultLeverage a future's for the non-default risk profile, which the docs leave out
	MaximumLeverage           uint64  `json:"max_leverage"`
	MaximumNonDefaultLeverage float64 `json:"max_non_default_leverage"`
	// IsCoinbaseSpotRouted and its alias IsCoinbaseExchangeRouted are set for spot whose orders Coinbase Exchange
	// matches, whose trades the gateway does not serve
	IsCoinbaseSpotRouted     bool `json:"is_csr"`
	IsCoinbaseExchangeRouted bool `json:"is_cbe_routed"`
}

// UnmarshalJSON decodes an instrument, reading its expiration timestamp as milliseconds; types.Time reads the 14 digit
// timestamp of the year 3000, which perpetuals and spot expire at, as microseconds
func (i *DerivativesInstrument) UnmarshalJSON(data []byte) error {
	type instrument DerivativesInstrument
	aux := struct {
		*instrument
		ExpirationTimestamp *int64 `json:"expiration_timestamp"`
	}{instrument: (*instrument)(i)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	switch {
	case aux.ExpirationTimestamp == nil:
	case *aux.ExpirationTimestamp == 0:
		i.ExpirationTimestamp = types.Time{}
	default:
		i.ExpirationTimestamp = types.Time(time.UnixMilli(*aux.ExpirationTimestamp))
	}
	return nil
}

// DerivativesTickSizeStep is the tick size applying above a price
type DerivativesTickSizeStep struct {
	AbovePrice float64 `json:"above_price"`
	TickSize   float64 `json:"tick_size"`
}

// DerivativesMarginParameters are a future's margin formula parameters, which the gateway names only by their
// abbreviations
type DerivativesMarginParameters struct {
	InitialIncrement float64 `json:"i_inc"`
	MarginFee        float64 `json:"m_fee"`
	MarginIncrement  float64 `json:"m_inc"`
	MarginInitial    float64 `json:"m_init"`
	MarginPower      float64 `json:"m_pow"`
}

// DerivativesInstrumentsRequest selects a settlement currency's instruments, which the gateway lists perpetuals first,
// then by expiration, and caches for about a minute
type DerivativesInstrumentsRequest struct {
	// Currency is a settlement currency, or any for every currency; a named currency also selects the unexpired spot
	// pairs it is either side of
	Currency currency.Code
	// Kind is future, option, spot, future_combo, option_combo or any, the default, which the docs leave out
	Kind string
	// Expired selects recently delivered instruments instead of visible and inactive ones
	Expired bool
	// Extended adds the maximum commissions and, for futures, the margin and perpetual funding parameters; the docs leave
	// it out, but the gateway honours it
	Extended bool
}

// derivativesInstrumentsParams are public/get_instruments's params
type derivativesInstrumentsParams struct {
	Currency string `json:"currency"`
	Kind     string `json:"kind,omitempty"`
	Expired  bool   `json:"expired,omitempty"`
	Extended bool   `json:"extended,omitempty"`
}

// DerivativesLastSettlementsByCurrencyRequest selects a page of a currency's settlement events, newest first
type DerivativesLastSettlementsByCurrencyRequest struct {
	// Currency is a settlement currency; the gateway does not accept any
	Currency currency.Code
	// Type is settlement, delivery or bankruptcy; every type, the exercise of expired options included, is returned when
	// it is unset
	Type string
	// Count is how many events the page holds, 20 when it is unset and at most 1000
	Count uint64
	// Continuation is a previous page's continuation token
	Continuation string
	// SearchStartTime returns the events up to this time
	SearchStartTime time.Time
}

// derivativesLastSettlementsByCurrencyParams are public/get_last_settlements_by_currency's params
type derivativesLastSettlementsByCurrencyParams struct {
	Currency             string `json:"currency"`
	Type                 string `json:"type,omitempty"`
	Count                uint64 `json:"count,omitempty"`
	Continuation         string `json:"continuation,omitempty"`
	SearchStartTimestamp int64  `json:"search_start_timestamp,omitempty"`
}

// DerivativesLastSettlementsByInstrumentRequest selects a page of an instrument's settlement events, newest first
type DerivativesLastSettlementsByInstrumentRequest struct {
	InstrumentName string
	// Type is settlement, delivery or bankruptcy; every type is returned when it is unset
	Type string
	// Count is how many events the page holds, 20 when it is unset and at most 1000
	Count uint64
	// Continuation is a previous page's continuation token
	Continuation string
	// SearchStartTime returns the events up to this time
	SearchStartTime time.Time
}

// derivativesLastSettlementsByInstrumentParams are public/get_last_settlements_by_instrument's params
type derivativesLastSettlementsByInstrumentParams struct {
	InstrumentName       string `json:"instrument_name"`
	Type                 string `json:"type,omitempty"`
	Count                uint64 `json:"count,omitempty"`
	Continuation         string `json:"continuation,omitempty"`
	SearchStartTimestamp int64  `json:"search_start_timestamp,omitempty"`
}

// DerivativesLastTradesByCurrencyRequest selects a currency's latest trades, optionally within a trade ID or time range
type DerivativesLastTradesByCurrencyRequest struct {
	// Currency is a settlement currency
	Currency currency.Code
	// Kind is future, option, spot, future_combo, option_combo, combo or any, the default; spot and any fail for a
	// currency on either side of a Coinbase-routed spot pair, as BTC and USDC are
	Kind string
	// StartID and EndID bound the trade IDs returned
	StartID   string
	EndID     string
	StartTime time.Time
	EndTime   time.Time
	// Count is how many trades are returned, 10 when it is unset and at most 1000
	Count uint64
	// Sorting is asc, desc or default, the order trades left the database in
	Sorting string
}

// derivativesLastTradesByCurrencyParams are public/get_last_trades_by_currency's params
type derivativesLastTradesByCurrencyParams struct {
	Currency       string `json:"currency"`
	Kind           string `json:"kind,omitempty"`
	StartID        string `json:"start_id,omitempty"`
	EndID          string `json:"end_id,omitempty"`
	StartTimestamp int64  `json:"start_timestamp,omitempty"`
	EndTimestamp   int64  `json:"end_timestamp,omitempty"`
	Count          uint64 `json:"count,omitempty"`
	Sorting        string `json:"sorting,omitempty"`
}

// DerivativesLastTradesByCurrencyAndTimeRequest selects a currency's latest trades within a time range, which needs a
// start or an end time; the docs require both, but the gateway takes either alone
type DerivativesLastTradesByCurrencyAndTimeRequest struct {
	// Currency is a settlement currency
	Currency currency.Code
	// Kind is future, option, spot, future_combo, option_combo, combo or any, the default; spot and any fail for a
	// currency on either side of a Coinbase-routed spot pair, as BTC and USDC are
	Kind      string
	StartTime time.Time
	EndTime   time.Time
	// Count is how many trades are returned, 10 when it is unset and at most 1000
	Count uint64
	// Sorting is asc, desc or default, the order trades left the database in
	Sorting string
}

// derivativesLastTradesByCurrencyAndTimeParams are public/get_last_trades_by_currency_and_time's params
type derivativesLastTradesByCurrencyAndTimeParams struct {
	Currency       string `json:"currency"`
	Kind           string `json:"kind,omitempty"`
	StartTimestamp int64  `json:"start_timestamp,omitempty"`
	EndTimestamp   int64  `json:"end_timestamp,omitempty"`
	Count          uint64 `json:"count,omitempty"`
	Sorting        string `json:"sorting,omitempty"`
}

// DerivativesLastTradesByInstrumentRequest selects an instrument's latest trades, optionally within a trade sequence or
// time range; Coinbase-routed spot is not supported
type DerivativesLastTradesByInstrumentRequest struct {
	InstrumentName string
	// StartSequence and EndSequence bound the trade sequence numbers returned
	StartSequence uint64
	EndSequence   uint64
	StartTime     time.Time
	EndTime       time.Time
	// Count is how many trades are returned, 10 when it is unset and at most 1000
	Count uint64
	// Sorting is asc, desc or default, the order trades left the database in
	Sorting string
}

// derivativesLastTradesByInstrumentParams are public/get_last_trades_by_instrument's params
type derivativesLastTradesByInstrumentParams struct {
	InstrumentName string `json:"instrument_name"`
	StartSequence  uint64 `json:"start_seq,omitempty"`
	EndSequence    uint64 `json:"end_seq,omitempty"`
	StartTimestamp int64  `json:"start_timestamp,omitempty"`
	EndTimestamp   int64  `json:"end_timestamp,omitempty"`
	Count          uint64 `json:"count,omitempty"`
	Sorting        string `json:"sorting,omitempty"`
}

// DerivativesLastTradesByInstrumentAndTimeRequest selects an instrument's latest trades within a time range, which needs
// a start or an end time; the docs require both, but the gateway takes either alone. Coinbase-routed spot is not
// supported
type DerivativesLastTradesByInstrumentAndTimeRequest struct {
	InstrumentName string
	StartTime      time.Time
	EndTime        time.Time
	// Count is how many trades are returned, 10 when it is unset and at most 1000
	Count uint64
	// Sorting is asc, desc or default, the order trades left the database in
	Sorting string
}

// derivativesLastTradesByInstrumentAndTimeParams are public/get_last_trades_by_instrument_and_time's params
type derivativesLastTradesByInstrumentAndTimeParams struct {
	InstrumentName string `json:"instrument_name"`
	StartTimestamp int64  `json:"start_timestamp,omitempty"`
	EndTimestamp   int64  `json:"end_timestamp,omitempty"`
	Count          uint64 `json:"count,omitempty"`
	Sorting        string `json:"sorting,omitempty"`
}

// DerivativesLastTradesResponse is a page of public trades and whether more match the request
type DerivativesLastTradesResponse struct {
	Trades  []DerivativesPublicTrade `json:"trades"`
	HasMore bool                     `json:"has_more"`
}

// DerivativesPublicTrade is a trade on the gateway's books
type DerivativesPublicTrade struct {
	// TradeID is a number for inverse instruments' trades and prefixed by the settlement currency for others', such as
	// USDC-80635730
	TradeID string `json:"trade_id"`
	// TradeSequence is the trade's sequence number within its instrument
	TradeSequence  uint64 `json:"trade_seq"`
	InstrumentName string `json:"instrument_name"`
	// Timestamp is in milliseconds, and StarbaseTimestamp, the Starbase causal time of a trade matched in Starbase, in
	// nanoseconds
	Timestamp         types.Time `json:"timestamp"`
	StarbaseTimestamp types.Time `json:"starbase_timestamp"`
	StarbaseMatchID   uint64     `json:"starbase_match_id"`
	// Direction is the taker's direction
	Direction string `json:"direction"`
	// TickDirection is 0 for a plus tick, 1 for a zero plus tick, 2 for a minus tick and 3 for a zero minus tick
	TickDirection uint64  `json:"tick_direction"`
	Price         float64 `json:"price"`
	// Amount is in USD for inverse futures and perpetuals, and in the base currency for options and linear instruments
	Amount            float64 `json:"amount"`
	Contracts         float64 `json:"contracts"`
	MarkPrice         float64 `json:"mark_price"`
	IndexPrice        float64 `json:"index_price"`
	ImpliedVolatility float64 `json:"iv"`
	// Liquidation is M, T or MT when the maker, the taker or both were being liquidated
	Liquidation        string `json:"liquidation"`
	BlockTradeID       string `json:"block_trade_id"`
	BlockTradeLegCount uint64 `json:"block_trade_leg_count"`
	BlockRFQID         uint64 `json:"block_rfq_id"`
	// ComboID and ComboTradeID identify the combo trade a leg trade belongs to
	ComboID      string `json:"combo_id"`
	ComboTradeID string `json:"combo_trade_id"`
	// Legs holds a combo trade's leg trades, which the gateway only sends when combo trades alone are requested and the
	// docs leave out
	Legs []DerivativesPublicTrade `json:"legs"`
}

// DerivativesMarkPrice is an option's mark price at a point in time
type DerivativesMarkPrice struct {
	Timestamp types.Time
	MarkPrice float64
}

// UnmarshalJSON decodes a [timestamp, mark price] tuple
func (m *DerivativesMarkPrice) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &m.Timestamp, &m.MarkPrice)
}

// DerivativesOrderBookRequest selects an instrument's order book
type DerivativesOrderBookRequest struct {
	InstrumentName string
	// Depth is how many price levels each side returns, 20 when it is unset and at most 10000; the gateway rounds it up
	// to 1, 5, 10, 20, 50, 100, 1000 or 10000
	Depth uint64
}

// derivativesOrderBookParams are public/get_order_book's params
type derivativesOrderBookParams struct {
	InstrumentName string `json:"instrument_name"`
	Depth          uint64 `json:"depth,omitempty"`
}

// DerivativesOrderBookByInstrumentIDRequest selects the order book of an instrument identified by its ID
type DerivativesOrderBookByInstrumentIDRequest struct {
	InstrumentID uint64
	// Depth is how many price levels each side returns, 20 when it is unset and at most 10000; the gateway rounds it up
	// to 1, 5, 10, 20, 50, 100, 1000 or 10000
	Depth uint64
}

// derivativesOrderBookByInstrumentIDParams are public/get_order_book_by_instrument_id's params
type derivativesOrderBookByInstrumentIDParams struct {
	InstrumentID uint64 `json:"instrument_id"`
	Depth        uint64 `json:"depth,omitempty"`
}

// DerivativesOrderBook is an instrument's order book, bids and asks best first, and its ticker
type DerivativesOrderBook struct {
	DerivativesTicker
	// ChangeID identifies the book's state, as the book subscriptions number their changes
	ChangeID uint64                      `json:"change_id"`
	Bids     []DerivativesOrderBookLevel `json:"bids"`
	Asks     []DerivativesOrderBookLevel `json:"asks"`
}

// DerivativesOrderBookLevel is the amount offered at a price
type DerivativesOrderBookLevel struct {
	Price  float64
	Amount float64
}

// UnmarshalJSON decodes a [price, amount] tuple
func (o *DerivativesOrderBookLevel) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &o.Price, &o.Amount)
}

// derivativesSupportedIndexNamesParams are public/get_supported_index_names's params
type derivativesSupportedIndexNamesParams struct {
	Type string `json:"type,omitempty"`
}

// DerivativesTradeVolume is a currency's traded volumes in the currency, including block trades but not position moves
type DerivativesTradeVolume struct {
	Currency currency.Code `json:"currency"`
	// CurrencyPair is the currency's USD index, which the docs leave out
	CurrencyPair  string  `json:"currency_pair"`
	CallsVolume   float64 `json:"calls_volume"`
	PutsVolume    float64 `json:"puts_volume"`
	FuturesVolume float64 `json:"futures_volume"`
	// SpotVolume and its 7 and 30 day volumes are left out for a currency of Coinbase-routed spot pairs
	SpotVolume float64 `json:"spot_volume"`
	// The 7 and 30 day volumes are only sent with extended statistics
	CallsVolume7Day    float64 `json:"calls_volume_7d"`
	PutsVolume7Day     float64 `json:"puts_volume_7d"`
	FuturesVolume7Day  float64 `json:"futures_volume_7d"`
	SpotVolume7Day     float64 `json:"spot_volume_7d"`
	CallsVolume30Day   float64 `json:"calls_volume_30d"`
	PutsVolume30Day    float64 `json:"puts_volume_30d"`
	FuturesVolume30Day float64 `json:"futures_volume_30d"`
	SpotVolume30Day    float64 `json:"spot_volume_30d"`
}

// DerivativesTradingViewChartDataRequest selects an instrument's candles opening within a time range; the docs call
// Coinbase-routed spot unsupported, but the gateway answers with candles of its own book
type DerivativesTradingViewChartDataRequest struct {
	InstrumentName string
	StartTime      time.Time
	EndTime        time.Time
	// Resolution is the candle interval: 1, 3, 5, 10, 15 or 30 minutes, 1, 2, 3, 6 or 12 hours, or 1 day
	Resolution kline.Interval
}

// derivativesTradingViewChartDataParams are public/get_tradingview_chart_data's params, its times in Unix
// milliseconds and its resolution as the gateway names it
type derivativesTradingViewChartDataParams struct {
	InstrumentName string `json:"instrument_name"`
	StartTimestamp int64  `json:"start_timestamp"`
	EndTimestamp   int64  `json:"end_timestamp"`
	Resolution     string `json:"resolution"`
}

// DerivativesTradingViewChartDataResponse is an instrument's candles as parallel lists, one entry a candle
type DerivativesTradingViewChartDataResponse struct {
	// Status is ok, or no_data when the range holds no candles
	Status string `json:"status"`
	// Ticks are the candles' open times
	Ticks []types.Time `json:"ticks"`
	Open  []float64    `json:"open"`
	High  []float64    `json:"high"`
	Low   []float64    `json:"low"`
	Close []float64    `json:"close"`
	// Volume is in the base currency, and Cost in the quote currency
	Volume []float64 `json:"volume"`
	Cost   []float64 `json:"cost"`
}

// DerivativesVolatilityIndexDataRequest selects a currency's volatility index candles
type DerivativesVolatilityIndexDataRequest struct {
	// Currency is a volatility index's currency, BTC or ETH; other currencies return no candles
	Currency  currency.Code
	StartTime time.Time
	EndTime   time.Time
	// Resolution is the candle interval: 1 second, 1 minute, 1 hour, 12 hours or 1 day
	Resolution kline.Interval
}

// derivativesVolatilityIndexDataParams are public/get_volatility_index_data's params, its times in Unix milliseconds
// and its resolution as the gateway names it
type derivativesVolatilityIndexDataParams struct {
	Currency       string `json:"currency"`
	StartTimestamp int64  `json:"start_timestamp"`
	EndTimestamp   int64  `json:"end_timestamp"`
	Resolution     string `json:"resolution"`
}

// DerivativesVolatilityIndexDataResponse is up to 1000 volatility index candles, the latest of the range, oldest first
type DerivativesVolatilityIndexDataResponse struct {
	Data []DerivativesVolatilityIndexCandle `json:"data"`
	// Continuation is the end time of the next, earlier page, and zero once the range has no more candles
	Continuation types.Time `json:"continuation"`
}

// DerivativesVolatilityIndexCandle is a volatility index candle
type DerivativesVolatilityIndexCandle struct {
	Timestamp types.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
}

// UnmarshalJSON decodes a [timestamp, open, high, low, close] tuple
func (v *DerivativesVolatilityIndexCandle) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &v.Timestamp, &v.Open, &v.High, &v.Low, &v.Close)
}

// DerivativesTicker is an instrument's ticker; fields only some kinds carry are zero for the others, as are prices the
// gateway sends as null, such as the best bid of an empty book
type DerivativesTicker struct {
	InstrumentName string     `json:"instrument_name"`
	Timestamp      types.Time `json:"timestamp"`
	// State is the book's state: open, settlement, delivered, inactive, locked, halted or archivized
	State         string                 `json:"state"`
	Stats         DerivativesTickerStats `json:"stats"`
	BestBidPrice  float64                `json:"best_bid_price"`
	BestBidAmount float64                `json:"best_bid_amount"`
	BestAskPrice  float64                `json:"best_ask_price"`
	BestAskAmount float64                `json:"best_ask_amount"`
	LastPrice     float64                `json:"last_price"`
	MarkPrice     float64                `json:"mark_price"`
	IndexPrice    float64                `json:"index_price"`
	// MinimumPrice and MaximumPrice are the prices sell and buy orders are clamped to
	MinimumPrice float64 `json:"min_price"`
	MaximumPrice float64 `json:"max_price"`
	// OpenInterest is in USD for inverse futures and perpetuals, and in the base currency for options and linear
	// instruments
	OpenInterest           float64 `json:"open_interest"`
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
	// SettlementPrice is the last settlement's price while the book is open, and DeliveryPrice the delivery's once it has
	// closed; spot has neither
	SettlementPrice float64 `json:"settlement_price"`
	DeliveryPrice   float64 `json:"delivery_price"`
	// CurrentFunding, Funding8Hour and InterestValue are a perpetual's; InterestValue is used to calculate the realised
	// funding of positions
	CurrentFunding float64 `json:"current_funding"`
	Funding8Hour   float64 `json:"funding_8h"`
	InterestValue  float64 `json:"interest_value"`
	// Greeks, the implied volatilities, InterestRate, UnderlyingIndex and UnderlyingPrice are an option's or option
	// combo's; UnderlyingIndex is the underlying future, or index_price
	Greeks                *DerivativesGreeks `json:"greeks"`
	BidImpliedVolatility  float64            `json:"bid_iv"`
	AskImpliedVolatility  float64            `json:"ask_iv"`
	MarkImpliedVolatility float64            `json:"mark_iv"`
	InterestRate          float64            `json:"interest_rate"`
	UnderlyingIndex       string             `json:"underlying_index"`
	UnderlyingPrice       float64            `json:"underlying_price"`
	// ComboState, active or inactive, ImpliedBid and ImpliedAsk are a combo's, which the docs leave out; the implied
	// prices are those its legs' books imply
	ComboState string  `json:"combo_state"`
	ImpliedBid float64 `json:"implied_bid"`
	ImpliedAsk float64 `json:"implied_ask"`
	// IsAnchorBreached, AnchorMinimumPrice and AnchorMaximumPrice are a real world asset perpetual's while an anchor
	// price band applies to its mark price
	IsAnchorBreached   bool    `json:"is_anchor_breached"`
	AnchorMinimumPrice float64 `json:"anchor_min_price"`
	AnchorMaximumPrice float64 `json:"anchor_max_price"`
}

// DerivativesTickerStats are an instrument's 24 hour statistics; prices are zero when it has not traded
type DerivativesTickerStats struct {
	High float64 `json:"high"`
	Low  float64 `json:"low"`
	// PriceChange is a percentage
	PriceChange float64 `json:"price_change"`
	// Volume is in the base currency, VolumeUSD in USD and VolumeNotional, which the docs leave out, in the quote
	// currency; Coinbase-routed spot carries neither of the latter
	Volume         float64 `json:"volume"`
	VolumeUSD      float64 `json:"volume_usd"`
	VolumeNotional float64 `json:"volume_notional"`
}

// DerivativesGreeks are an option's Black-Scholes sensitivities; Theta is the lesser of the one day and lifetime theta
type DerivativesGreeks struct {
	Delta float64 `json:"delta"`
	Gamma float64 `json:"gamma"`
	Vega  float64 `json:"vega"`
	Theta float64 `json:"theta"`
	Rho   float64 `json:"rho"`
}

// DerivativesTickersByCurrencyRequest selects the tickers of a currency's open instruments
type DerivativesTickersByCurrencyRequest struct {
	// Currency is a settlement currency; the gateway does not accept any
	Currency currency.Code
	// Kinds are future, option, spot, future_combo, option_combo or any, the default; tickers of every kind listed are
	// returned
	Kinds []string
	// Depth is how many price levels each side of each ticker's book holds: 1, the default, which returns no levels,
	// 3 or 10
	Depth uint64
}

// derivativesTickersByCurrencyParams are public/tickers_by_currency's params; the gateway takes a list of kinds
type derivativesTickersByCurrencyParams struct {
	Currency string   `json:"currency"`
	Kinds    []string `json:"kind,omitempty"`
	Depth    uint64   `json:"depth,omitempty"`
}
