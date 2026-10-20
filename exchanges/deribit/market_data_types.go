package deribit

import (
	"bytes"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Funding chart lengths
const (
	FundingChartLength8Hour    = "8h"
	FundingChartLength24Hour   = "24h"
	FundingChartLengthOneMonth = "1m"
)

// Index chart ranges
const (
	IndexChartRangeOneHour  = "1h"
	IndexChartRangeOneDay   = "1d"
	IndexChartRangeTwoDays  = "2d"
	IndexChartRangeOneMonth = "1m"
	IndexChartRangeOneYear  = "1y"
	IndexChartRangeAll      = "all"
)

// Index types of a supported index names request
const (
	IndexTypeAll        = "all"
	IndexTypeSpot       = "spot"
	IndexTypeDerivative = "derivative"
)

// APRHistoryRequest requests a yield-bearing token's daily APRs, newest first
type APRHistoryRequest struct {
	// Currency is a yield-bearing token, such as USDE, STETH or USDC
	Currency currency.Code `json:"currency"`
	// Limit is the number of days returned, 365 when left out
	Limit uint64 `json:"limit,omitempty"`
	// Before returns the days before this day, counted from the Unix epoch, such as a previous page's Continuation
	Before uint64 `json:"before,omitempty"`
}

// APRHistoryResponse contains a page of a yield-bearing token's daily APRs, newest first
type APRHistoryResponse struct {
	Data []APRRecord `json:"data"`
	// Continuation is the Before of the next page, and zero on the last page; Deribit sends it as a number, although
	// its documentation types it as a string
	Continuation uint64 `json:"continuation"`
}

// APRRecord contains a day's APR, as a percentage
type APRRecord struct {
	// Day is counted from the Unix epoch
	Day uint64  `json:"day"`
	APR float64 `json:"apr"`
}

// IndexNameRequest contains the index of a request
type IndexNameRequest struct {
	IndexName string `json:"index_name"`
}

// BasketIndexCompositionResponse contains a basket index's latest composition, which Deribit forwards from its index
// composition service as received, numbers as strings
type BasketIndexCompositionResponse struct {
	ProductID string `json:"product_id"`
	// Divisor scales the constituents' market cap into the published index level
	Divisor            types.Number             `json:"divisor"`
	Timestamp          time.Time                `json:"timestamp"`
	InceptionTimestamp time.Time                `json:"inception_timestamp"`
	LastRebalance      time.Time                `json:"last_rebalance"`
	Constituents       []BasketIndexConstituent `json:"constituents"`
}

// BasketIndexConstituent contains an asset of a basket index and its weight
type BasketIndexConstituent struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Rank   uint64 `json:"rank"`
	// CapFactor scales the asset's market cap down to its capped share of the index
	CapFactor      types.Number `json:"cap_factor"`
	Amount         types.Number `json:"amount"`
	MarketCap      types.Number `json:"market_cap"`
	IndexMarketCap types.Number `json:"index_market_cap"`
	Weight         types.Number `json:"weight"`
	// RunningWeight is the combined weight of this and every higher ranked constituent
	RunningWeight types.Number `json:"running_weight"`
}

// BookSummaryByCurrencyRequest requests the market summaries of a currency's instruments
type BookSummaryByCurrencyRequest struct {
	Currency currency.Code `json:"currency"`
	// Kind is future, option, spot, future_combo or option_combo; every kind is summarised when it is left out
	Kind string `json:"kind,omitempty"`
}

// BookSummary contains an instrument's market summary; prices Deribit sends as null, such as the bid of an empty book
// or the last price of an instrument that has not traded in 24 hours, are zero
type BookSummary struct {
	InstrumentName string        `json:"instrument_name"`
	BaseCurrency   currency.Code `json:"base_currency"`
	QuoteCurrency  currency.Code `json:"quote_currency"`
	// CreationTimestamp is when Deribit compiled the summary, not when the instrument was created
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
	// Volume is the 24 hour volume in the base currency, VolumeUSD in USD and VolumeNotional, which options lack, in
	// the quote currency
	Volume         float64 `json:"volume"`
	VolumeUSD      float64 `json:"volume_usd"`
	VolumeNotional float64 `json:"volume_notional"`
	// OpenInterest is in USD for perpetuals and inverse futures, and in the base currency for options and linear
	// futures
	OpenInterest           float64 `json:"open_interest"`
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
	// CurrentFunding is a perpetual's instantaneous funding rate, and Funding8Hour the rate accumulated towards its
	// next settlement
	CurrentFunding float64 `json:"current_funding"`
	Funding8Hour   float64 `json:"funding_8h"`
	// MarkImpliedVolatility, InterestRate, UnderlyingIndex and UnderlyingPrice are an option's; UnderlyingIndex is the
	// underlying future, or index_price
	MarkImpliedVolatility float64 `json:"mark_iv"`
	InterestRate          float64 `json:"interest_rate"`
	UnderlyingIndex       string  `json:"underlying_index"`
	UnderlyingPrice       float64 `json:"underlying_price"`
}

// ContractSizeResponse contains an instrument's contract size, in USD for inverse futures and perpetuals and in the
// base currency for options, spot and linear futures and perpetuals; it is not the amount increment, which is the
// instrument's MinTradeAmount
type ContractSizeResponse struct {
	ContractSize float64 `json:"contract_size"`
}

// CurrencyDetail contains a currency Deribit supports and its deposit and withdrawal parameters
type CurrencyDetail struct {
	Currency     currency.Code `json:"currency"`
	CurrencyLong string        `json:"currency_long"`
	CurrencyUUID string        `json:"currency_uuid"`
	// CoinType is sent as the currency's symbol, not the BITCOIN or ETHER Deribit documents
	CoinType string `json:"coin_type"`
	Decimals uint64 `json:"decimals"`
	// OnchainOperationsPrecision is sent for every currency, although Deribit does not document it
	OnchainOperationsPrecision uint64                       `json:"onchain_operations_precision"`
	MinConfirmations           uint64                       `json:"min_confirmations"`
	WithdrawalFee              float64                      `json:"withdrawal_fee"`
	MinWithdrawalFee           float64                      `json:"min_withdrawal_fee"`
	WithdrawalPriorities       []CurrencyWithdrawalPriority `json:"withdrawal_priorities"`
	NetworkFee                 float64                      `json:"network_fee"`
	NetworkCurrency            currency.Code                `json:"network_currency"`
	InCrossCollateralPool      bool                         `json:"in_cross_collateral_pool"`
	// APR is a yield-bearing token's average daily reward over the last 7 days, as a percentage
	APR float64 `json:"apr"`
	// CoinbaseNetworks are the networks a Coinbase wallet account can use the currency on
	CoinbaseNetworks []CurrencyCoinbaseNetwork `json:"coinbase_networks"`
}

// CurrencyWithdrawalPriority contains a withdrawal priority and its fee rate
type CurrencyWithdrawalPriority struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// CurrencyCoinbaseNetwork contains a network a Coinbase wallet account can use a currency on; its resource name is the
// network parameter of wallet methods
type CurrencyCoinbaseNetwork struct {
	ResourceName string `json:"resource_name"`
	DisplayName  string `json:"display_name"`
}

// DeliveryPricesRequest requests an index's delivery prices, newest first
type DeliveryPricesRequest struct {
	IndexName string `json:"index_name"`
	Offset    uint64 `json:"offset,omitempty"`
	// Count is the number of prices returned, 10 when left out
	Count uint64 `json:"count,omitempty"`
}

// DeliveryPricesResponse contains a page of an index's delivery prices and the number of prices available
type DeliveryPricesResponse struct {
	Data         []DeliveryPrice `json:"data"`
	RecordsTotal uint64          `json:"records_total"`
}

// DeliveryPrice contains the price an index's futures and options were delivered at on a day
type DeliveryPrice struct {
	// Date is the delivery's UTC date, formatted 2006-01-02
	Date          string  `json:"date"`
	DeliveryPrice float64 `json:"delivery_price"`
}

// ExpirationsRequest requests the expiration labels of listed futures and options; spot and combos have none
type ExpirationsRequest struct {
	// Currency is a settlement currency, currency.ANY for every currency's labels merged, or grouped for each
	// currency's labels; CurrencyPair is used when it is left out
	Currency currency.Code `json:"currency,omitzero"`
	// Kind is future, option or any, the default, for both
	Kind string `json:"kind,omitempty"`
	// CurrencyPair is an index pair, such as btc_usd, whose labels are returned keyed by the pair; Deribit ignores it
	// when Currency is set, and merges every pair's labels when both are left out
	CurrencyPair string `json:"currency_pair,omitempty"`
}

// ExpirationsResponse contains expiration labels, such as 27NOV26 or PERPETUAL: merged across currencies when every
// currency is requested, otherwise in ByCurrency
type ExpirationsResponse struct {
	Expirations
	// ByCurrency holds the labels keyed by lowercase currency, or by index pair when a currency pair is requested
	ByCurrency map[string]Expirations
}

// UnmarshalJSON decodes expiration labels merged across currencies, which Deribit keys by kind, or keyed by currency
func (r *ExpirationsResponse) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, value := range fields {
		if value = bytes.TrimSpace(value); len(value) > 0 && value[0] == '[' {
			return json.Unmarshal(data, &r.Expirations)
		}
	}
	return json.Unmarshal(data, &r.ByCurrency)
}

// Expirations contains the expiration labels of futures and options
type Expirations struct {
	Future []string `json:"future"`
	Option []string `json:"option"`
}

// FundingChartDataRequest requests a perpetual's funding chart
type FundingChartDataRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Length is the period charted: 8h, 24h or 1m for a month
	Length string `json:"length"`
}

// FundingChartDataResponse contains a perpetual's current funding and its funding chart
type FundingChartDataResponse struct {
	CurrentInterest float64             `json:"current_interest"`
	Interest8Hour   float64             `json:"interest_8h"`
	Data            []FundingChartPoint `json:"data"`
}

// FundingChartPoint contains a perpetual's 8 hour funding and its index price at a point of its funding chart
type FundingChartPoint struct {
	Timestamp     types.Time `json:"timestamp"`
	IndexPrice    float64    `json:"index_price"`
	Interest8Hour float64    `json:"interest_8h"`
}

// InstrumentTimeRangeRequest requests an instrument's data over a time range
type InstrumentTimeRangeRequest struct {
	InstrumentName string    `json:"instrument_name"`
	StartTime      time.Time `json:"-"`
	EndTime        time.Time `json:"-"`
}

// InstrumentTimeRangeParams contains an instrument time range request's params, with its times in Unix milliseconds
type InstrumentTimeRangeParams struct {
	*InstrumentTimeRangeRequest
	StartTimestamp int64 `json:"start_timestamp"`
	EndTimestamp   int64 `json:"end_timestamp"`
}

// FundingRateHistory contains a perpetual's funding over an hour
type FundingRateHistory struct {
	// Timestamp is the end of the hour
	Timestamp  types.Time `json:"timestamp"`
	IndexPrice float64    `json:"index_price"`
	// PreviousIndexPrice is the index price at the start of the hour
	PreviousIndexPrice float64 `json:"prev_index_price"`
	// Interest1Hour is the hour's funding rate, and Interest8Hour the 8 hour rate at the end of the hour
	Interest1Hour float64 `json:"interest_1h"`
	Interest8Hour float64 `json:"interest_8h"`
}

// HistoricalVolatilityPoint contains a currency's realised volatility at an hour, as a percentage
type HistoricalVolatilityPoint struct {
	Timestamp  types.Time
	Volatility float64
}

// UnmarshalJSON decodes a [timestamp, volatility] tuple
func (h *HistoricalVolatilityPoint) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &h.Timestamp, &h.Volatility)
}

// IndexChartDataRequest requests an index's price chart
type IndexChartDataRequest struct {
	IndexName string `json:"index_name"`
	// Range is the period charted: 1h, 1d, 2d, 1m, 1y or all
	Range string `json:"range"`
}

// IndexChartPoint contains an index's average price at a point of its chart
type IndexChartPoint struct {
	Timestamp types.Time
	Price     float64
}

// UnmarshalJSON decodes a [timestamp, price] tuple
func (i *IndexChartPoint) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &i.Timestamp, &i.Price)
}

// IndexPriceResponse contains an index's current price
type IndexPriceResponse struct {
	IndexPrice             float64 `json:"index_price"`
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
}

// IndexPriceNamesRequest contains whether an index price names request asks for each index's combo creation settings
type IndexPriceNamesRequest struct {
	Extended bool `json:"extended,omitempty"`
}

// IndexPriceName contains an index's name and, when extended details are requested, whether future and option combos
// can be created on it; Deribit only sends a setting that is enabled
type IndexPriceName struct {
	Name                       string `json:"name"`
	FutureComboCreationEnabled bool   `json:"future_combo_creation_enabled"`
	OptionComboCreationEnabled bool   `json:"option_combo_creation_enabled"`
}

// UnmarshalJSON decodes an index's name, which Deribit sends as a bare string unless extended details are requested
func (i *IndexPriceName) UnmarshalJSON(data []byte) error {
	if data = bytes.TrimSpace(data); len(data) > 0 && data[0] == '"' {
		return json.Unmarshal(data, &i.Name)
	}
	type indexPriceName IndexPriceName
	return json.Unmarshal(data, (*indexPriceName)(i))
}

// Instrument contains an instrument's specification; fields only some kinds carry are zero for the others
type Instrument struct {
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
	// State is the order book's state: open, settlement, delivered, inactive, locked, halted or archivized. Only an
	// open book accepts orders, while IsActive is set whenever the book is visible
	State    string `json:"state"`
	IsActive bool   `json:"is_active"`
	// BaseCurrency is the underlying currency, QuoteCurrency the currency prices are quoted in, which is the base
	// currency for inverse options, and CounterCurrency the quote currency of the underlying currency pair
	BaseCurrency      currency.Code `json:"base_currency"`
	QuoteCurrency     currency.Code `json:"quote_currency"`
	CounterCurrency   currency.Code `json:"counter_currency"`
	BaseCurrencyUUID  string        `json:"base_currency_uuid"`
	QuoteCurrencyUUID string        `json:"quote_currency_uuid"`
	// SettlementCurrency and SettlementPeriod, minute, hour, day, week, month or perpetual, are omitted for spot
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
	// ExpirationTimestamp is far in the future for perpetuals and spot
	ExpirationTimestamp types.Time `json:"expiration_timestamp"`
	TickSize            float64    `json:"tick_size"`
	// TickSizeSteps are the larger tick sizes applying above a price
	TickSizeSteps []TickSizeStep `json:"tick_size_steps"`
	// ContractSize converts between amounts and contracts, in USD for inverse futures and perpetuals and in the base
	// currency otherwise; MinTradeAmount is the minimum amount and its increment
	ContractSize   float64 `json:"contract_size"`
	MinTradeAmount float64 `json:"min_trade_amount"`
	// LotSize is the unit fees are counted in
	LotSize                  float64 `json:"lot_size"`
	MakerCommission          float64 `json:"maker_commission"`
	TakerCommission          float64 `json:"taker_commission"`
	MaxLiquidationCommission float64 `json:"max_liquidation_commission"`
	// MakerMaxCommission, TakerMaxCommission, LiquidationMaxCommission and SettlementMaxCommission are only sent with
	// extended instrument details
	MakerMaxCommission       float64 `json:"maker_max_commission"`
	TakerMaxCommission       float64 `json:"taker_max_commission"`
	LiquidationMaxCommission float64 `json:"liquidation_max_commission"`
	SettlementMaxCommission  float64 `json:"settlement_max_commission"`
	// MarginParameters and TradingBandwidthFraction are a future's, and PerpetualDampener and PerpetualMaxFunding a
	// perpetual's, only sent with extended instrument details
	MarginParameters         *MarginParameters `json:"margin_params"`
	TradingBandwidthFraction float64           `json:"trading_bandwidth_fraction"`
	PerpetualDampener        float64           `json:"perpetual_dempener"`
	PerpetualMaxFunding      float64           `json:"perpetual_max_funding"`
	// The block trade fields are omitted for Coinbase-routed spot, which does not support block trades
	BlockTradeCommission     float64 `json:"block_trade_commission"`
	BlockTradeTickSize       float64 `json:"block_trade_tick_size"`
	BlockTradeMinTradeAmount float64 `json:"block_trade_min_trade_amount"`
	// MaxLeverage is a future's, or a spot pair's with margin loans, for the default risk profile, and
	// MaxNonDefaultLeverage a future's for the non-default risk profile; Deribit sends the latter as a decimal
	MaxLeverage           float64 `json:"max_leverage"`
	MaxNonDefaultLeverage float64 `json:"max_non_default_leverage"`
	// IsCoinbaseSpotRouted and its alias IsCoinbaseExchangeRouted are set for spot whose orders are routed to Coinbase
	// Exchange for matching
	IsCoinbaseSpotRouted     bool `json:"is_csr"`
	IsCoinbaseExchangeRouted bool `json:"is_cbe_routed"`
}

// UnmarshalJSON decodes an instrument, reading its expiration timestamp as milliseconds; types.Time reads the 14 digit
// timestamp of the year 3000, which perpetuals and spot expire at, as microseconds
func (i *Instrument) UnmarshalJSON(data []byte) error {
	type instrument Instrument
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

// TickSizeStep contains the tick size applying above a price
type TickSizeStep struct {
	AbovePrice float64 `json:"above_price"`
	TickSize   float64 `json:"tick_size"`
}

// MarginParameters contains a future's margin formula parameters, which Deribit documents only by their abbreviated
// names
type MarginParameters struct {
	InitialIncrement float64 `json:"i_inc"`
	MarginFee        float64 `json:"m_fee"`
	MarginIncrement  float64 `json:"m_inc"`
	MarginInitial    float64 `json:"m_init"`
	MarginPower      float64 `json:"m_pow"`
}

// InstrumentsRequest requests the instruments of a settlement currency, sorted with perpetuals first, then by
// expiration; Deribit caches the list for about a minute
type InstrumentsRequest struct {
	// Currency is a settlement currency, or currency.ANY for every currency; a named currency also selects the
	// unexpired spot pairs it is either side of
	Currency currency.Code `json:"currency"`
	// Kind is future, option, spot, future_combo, option_combo or any, the default, for every kind
	Kind string `json:"kind,omitempty"`
	// Expired selects recently delivered instruments instead of visible and inactive ones
	Expired bool `json:"expired,omitempty"`
	// Extended adds the maximum commissions and, for futures, the margin and perpetual funding parameters
	Extended bool `json:"extended,omitempty"`
}

// LastSettlementsByCurrencyRequest requests a currency's settlement, delivery and bankruptcy events, newest first
type LastSettlementsByCurrencyRequest struct {
	Currency currency.Code `json:"currency"`
	// Type is settlement, delivery or bankruptcy; every type, including the exercise of options, is returned when it is
	// left out
	Type string `json:"type,omitempty"`
	// Count is the number of events returned, 20 when left out
	Count        uint64 `json:"count,omitempty"`
	Continuation string `json:"continuation,omitempty"`
	// SearchStartTime returns the events up to this time
	SearchStartTime time.Time `json:"-"`
}

// LastSettlementsByCurrencyParams contains a currency's last settlements request params, with its search start time in
// Unix milliseconds
type LastSettlementsByCurrencyParams struct {
	*LastSettlementsByCurrencyRequest
	SearchStartTimestamp int64 `json:"search_start_timestamp,omitempty"`
}

// LastSettlementsByInstrumentRequest requests an instrument's settlement, delivery and bankruptcy events, newest first
type LastSettlementsByInstrumentRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Type is settlement, delivery or bankruptcy; every type is returned when it is left out
	Type string `json:"type,omitempty"`
	// Count is the number of events returned, 20 when left out
	Count        uint64 `json:"count,omitempty"`
	Continuation string `json:"continuation,omitempty"`
	// SearchStartTime returns the events up to this time
	SearchStartTime time.Time `json:"-"`
}

// LastSettlementsByInstrumentParams contains an instrument's last settlements request params, with its search start
// time in Unix milliseconds
type LastSettlementsByInstrumentParams struct {
	*LastSettlementsByInstrumentRequest
	SearchStartTimestamp int64 `json:"search_start_timestamp,omitempty"`
}

// LastTradesByCurrencyRequest requests a currency's latest trades, optionally within a trade ID or time range
type LastTradesByCurrencyRequest struct {
	Currency currency.Code `json:"currency"`
	// Kind is future, option, spot, future_combo, option_combo, combo or any, the default; spot and any fail for a
	// currency on either side of a Coinbase-routed spot pair
	Kind string `json:"kind,omitempty"`
	// StartID and EndID bound the trade IDs returned
	StartID   string    `json:"start_id,omitempty"`
	EndID     string    `json:"end_id,omitempty"`
	StartTime time.Time `json:"-"`
	EndTime   time.Time `json:"-"`
	// Count is the number of trades returned, 10 when left out
	Count uint64 `json:"count,omitempty"`
	// Sorting is asc, desc or default, the database's order
	Sorting string `json:"sorting,omitempty"`
}

// LastTradesByCurrencyParams contains a currency's last trades request params, with its times in Unix milliseconds
type LastTradesByCurrencyParams struct {
	*LastTradesByCurrencyRequest
	StartTimestamp int64 `json:"start_timestamp,omitempty"`
	EndTimestamp   int64 `json:"end_timestamp,omitempty"`
}

// LastTradesByCurrencyAndTimeRequest requests a currency's latest trades within a time range, which needs a start or an
// end time
type LastTradesByCurrencyAndTimeRequest struct {
	Currency currency.Code `json:"currency"`
	// Kind is future, option, spot, future_combo, option_combo, combo or any, the default; spot and any fail for a
	// currency on either side of a Coinbase-routed spot pair
	Kind      string    `json:"kind,omitempty"`
	StartTime time.Time `json:"-"`
	EndTime   time.Time `json:"-"`
	// Count is the number of trades returned, 10 when left out
	Count uint64 `json:"count,omitempty"`
	// Sorting is asc, desc or default, the database's order
	Sorting string `json:"sorting,omitempty"`
}

// LastTradesByCurrencyAndTimeParams contains a currency's last trades by time request params, with its times in Unix
// milliseconds
type LastTradesByCurrencyAndTimeParams struct {
	*LastTradesByCurrencyAndTimeRequest
	StartTimestamp int64 `json:"start_timestamp,omitempty"`
	EndTimestamp   int64 `json:"end_timestamp,omitempty"`
}

// LastTradesByInstrumentRequest requests an instrument's latest trades, optionally within a trade sequence or time
// range; Coinbase-routed spot is not supported
type LastTradesByInstrumentRequest struct {
	InstrumentName string `json:"instrument_name"`
	// StartSequence and EndSequence bound the trade sequence numbers returned
	StartSequence uint64    `json:"start_seq,omitempty"`
	EndSequence   uint64    `json:"end_seq,omitempty"`
	StartTime     time.Time `json:"-"`
	EndTime       time.Time `json:"-"`
	// Count is the number of trades returned, 10 when left out
	Count uint64 `json:"count,omitempty"`
	// Sorting is asc, desc or default, the database's order
	Sorting string `json:"sorting,omitempty"`
}

// LastTradesByInstrumentParams contains an instrument's last trades request params, with its times in Unix milliseconds
type LastTradesByInstrumentParams struct {
	*LastTradesByInstrumentRequest
	StartTimestamp int64 `json:"start_timestamp,omitempty"`
	EndTimestamp   int64 `json:"end_timestamp,omitempty"`
}

// LastTradesByInstrumentAndTimeRequest requests an instrument's latest trades within a time range, which needs a start
// or an end time; Coinbase-routed spot is not supported
type LastTradesByInstrumentAndTimeRequest struct {
	InstrumentName string    `json:"instrument_name"`
	StartTime      time.Time `json:"-"`
	EndTime        time.Time `json:"-"`
	// Count is the number of trades returned, 10 when left out
	Count uint64 `json:"count,omitempty"`
	// Sorting is asc, desc or default, the database's order
	Sorting string `json:"sorting,omitempty"`
}

// LastTradesByInstrumentAndTimeParams contains an instrument's last trades by time request params, with its times in
// Unix milliseconds
type LastTradesByInstrumentAndTimeParams struct {
	*LastTradesByInstrumentAndTimeRequest
	StartTimestamp int64 `json:"start_timestamp,omitempty"`
	EndTimestamp   int64 `json:"end_timestamp,omitempty"`
}

// LastTradesResponse contains a page of public trades and whether more match the request
type LastTradesResponse struct {
	Trades  []PublicTrade `json:"trades"`
	HasMore bool          `json:"has_more"`
}

// PublicTrade contains a trade on Deribit's books
type PublicTrade struct {
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
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures
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
	// Legs holds a combo trade's leg trades, which Deribit only sends when combo trades alone are requested
	Legs []PublicTrade `json:"legs"`
}

// MarkPricePoint contains an option's mark price at a point in time
type MarkPricePoint struct {
	Timestamp types.Time
	MarkPrice float64
}

// UnmarshalJSON decodes a [timestamp, mark price] tuple
func (m *MarkPricePoint) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &m.Timestamp, &m.MarkPrice)
}

// OrderBookRequest requests an instrument's order book
type OrderBookRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Depth is the number of price levels returned on each side, 20 when left out and at most 10000; Deribit rounds it
	// up to 1, 5, 10, 20, 50, 100, 1000 or 10000
	Depth uint64 `json:"depth,omitempty"`
}

// OrderBookByInstrumentIDRequest requests the order book of an instrument identified by its ID
type OrderBookByInstrumentIDRequest struct {
	InstrumentID uint64 `json:"instrument_id"`
	// Depth is the number of price levels returned on each side, 20 when left out and at most 10000; Deribit rounds it
	// up to 1, 5, 10, 20, 50, 100, 1000 or 10000
	Depth uint64 `json:"depth,omitempty"`
}

// OrderBookResponse contains an instrument's order book, bids best first and asks best first, and its ticker
type OrderBookResponse struct {
	TickerResponse
	ChangeID uint64           `json:"change_id"`
	Bids     []OrderBookLevel `json:"bids"`
	Asks     []OrderBookLevel `json:"asks"`
}

// OrderBookLevel contains the amount offered at a price
type OrderBookLevel struct {
	Price  float64
	Amount float64
}

// UnmarshalJSON decodes a [price, amount] tuple
func (o *OrderBookLevel) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &o.Price, &o.Amount)
}

// SupportedIndexNamesRequest contains the index type of a supported index names request
type SupportedIndexNamesRequest struct {
	Type string `json:"type,omitempty"`
}

// TradeVolumesRequest contains whether a trade volumes request asks for 7 and 30 day volumes
type TradeVolumesRequest struct {
	Extended bool `json:"extended,omitempty"`
}

// TradeVolume contains a currency's traded volumes in the currency, including block trades but not position moves
type TradeVolume struct {
	Currency currency.Code `json:"currency"`
	// CurrencyPair is the currency's USD index, which Deribit sends although it does not document it
	CurrencyPair  string  `json:"currency_pair"`
	CallsVolume   float64 `json:"calls_volume"`
	PutsVolume    float64 `json:"puts_volume"`
	FuturesVolume float64 `json:"futures_volume"`
	// SpotVolume is left out for a currency of Coinbase-routed spot pairs
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

// TradingViewChartDataRequest requests an instrument's candles; Deribit documents Coinbase-routed spot as unsupported,
// although it answers with candles of its own book
type TradingViewChartDataRequest struct {
	InstrumentName string    `json:"instrument_name"`
	StartTime      time.Time `json:"-"`
	EndTime        time.Time `json:"-"`
	// Resolution is the candle interval: 1, 3, 5, 10, 15 or 30 minutes, 1, 2, 3, 6 or 12 hours, or 1 day
	Resolution kline.Interval `json:"-"`
}

// TradingViewChartDataParams contains a TradingView chart data request's params, with its times in Unix milliseconds
// and its resolution as Deribit names it
type TradingViewChartDataParams struct {
	*TradingViewChartDataRequest
	StartTimestamp  int64  `json:"start_timestamp"`
	EndTimestamp    int64  `json:"end_timestamp"`
	ChartResolution string `json:"resolution"`
}

// TradingViewChartDataResponse contains an instrument's candles as parallel lists, one entry per candle
type TradingViewChartDataResponse struct {
	// Status is ok, or no_data when the range holds no candles
	Status string `json:"status"`
	// Ticks are the candles' open times
	Ticks []types.Time `json:"ticks"`
	Open  []float64    `json:"open"`
	High  []float64    `json:"high"`
	Low   []float64    `json:"low"`
	Close []float64    `json:"close"`
	// Volume is in the base currency, and Cost in the quote currency, which Deribit sends as zero for option combos
	Volume []float64 `json:"volume"`
	Cost   []float64 `json:"cost"`
}

// VolatilityIndexDataRequest requests a currency's volatility index candles
type VolatilityIndexDataRequest struct {
	Currency  currency.Code `json:"currency"`
	StartTime time.Time     `json:"-"`
	EndTime   time.Time     `json:"-"`
	// Resolution is the candle interval: 1 second, 1 minute, 1 hour, 12 hours or 1 day
	Resolution kline.Interval `json:"-"`
}

// VolatilityIndexDataParams contains a volatility index data request's params, with its times in Unix milliseconds and
// its resolution as Deribit names it
type VolatilityIndexDataParams struct {
	*VolatilityIndexDataRequest
	StartTimestamp  int64  `json:"start_timestamp"`
	EndTimestamp    int64  `json:"end_timestamp"`
	IndexResolution string `json:"resolution"`
}

// VolatilityIndexDataResponse contains up to 1000 volatility index candles, the latest of the range, oldest first
type VolatilityIndexDataResponse struct {
	Data []VolatilityIndexCandle `json:"data"`
	// Continuation is the end time of the next, earlier page, and zero once the range has no more candles
	Continuation types.Time `json:"continuation"`
}

// VolatilityIndexCandle contains a volatility index candle
type VolatilityIndexCandle struct {
	Timestamp types.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
}

// UnmarshalJSON decodes a [timestamp, open, high, low, close] tuple
func (v *VolatilityIndexCandle) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &v.Timestamp, &v.Open, &v.High, &v.Low, &v.Close)
}

// TickerResponse contains an instrument's ticker; fields only some kinds carry are zero for the others, as are prices
// Deribit sends as null, such as the best bid of an empty book
type TickerResponse struct {
	InstrumentName string     `json:"instrument_name"`
	Timestamp      types.Time `json:"timestamp"`
	// State is the order book's state: open, settlement, delivered, inactive, locked, halted or archivized
	State         string      `json:"state"`
	Stats         TickerStats `json:"stats"`
	BestBidPrice  float64     `json:"best_bid_price"`
	BestBidAmount float64     `json:"best_bid_amount"`
	BestAskPrice  float64     `json:"best_ask_price"`
	BestAskAmount float64     `json:"best_ask_amount"`
	LastPrice     float64     `json:"last_price"`
	MarkPrice     float64     `json:"mark_price"`
	IndexPrice    float64     `json:"index_price"`
	// MinPrice and MaxPrice are the prices sell and buy orders are clamped to
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
	// OpenInterest is in USD for perpetuals and inverse futures, and in the base currency for options and linear
	// futures
	OpenInterest           float64 `json:"open_interest"`
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
	// SettlementPrice is the last settlement's price while the book is open, and DeliveryPrice the delivery's once it
	// has closed; spot has neither
	SettlementPrice float64 `json:"settlement_price"`
	DeliveryPrice   float64 `json:"delivery_price"`
	// CurrentFunding, Funding8Hour and InterestValue are a perpetual's; InterestValue is used to calculate the realised
	// funding of positions
	CurrentFunding float64 `json:"current_funding"`
	Funding8Hour   float64 `json:"funding_8h"`
	InterestValue  float64 `json:"interest_value"`
	// Greeks, the implied volatilities, InterestRate, UnderlyingIndex and UnderlyingPrice are an option's or option
	// combo's; UnderlyingIndex is the underlying future, or index_price
	Greeks                *Greeks `json:"greeks"`
	BidImpliedVolatility  float64 `json:"bid_iv"`
	AskImpliedVolatility  float64 `json:"ask_iv"`
	MarkImpliedVolatility float64 `json:"mark_iv"`
	InterestRate          float64 `json:"interest_rate"`
	UnderlyingIndex       string  `json:"underlying_index"`
	UnderlyingPrice       float64 `json:"underlying_price"`
	// ComboState, active or inactive, ImpliedBid and ImpliedAsk are a combo's; the implied prices are those its legs'
	// books imply
	ComboState string  `json:"combo_state"`
	ImpliedBid float64 `json:"implied_bid"`
	ImpliedAsk float64 `json:"implied_ask"`
	// IsAnchorBreached, AnchorMinPrice and AnchorMaxPrice are a real world asset perpetual's while an anchor price band
	// applies to its mark price
	IsAnchorBreached bool    `json:"is_anchor_breached"`
	AnchorMinPrice   float64 `json:"anchor_min_price"`
	AnchorMaxPrice   float64 `json:"anchor_max_price"`
}

// TickerStats contains an instrument's 24 hour statistics; prices are zero when it has not traded
type TickerStats struct {
	High float64 `json:"high"`
	Low  float64 `json:"low"`
	// PriceChange is a percentage
	PriceChange float64 `json:"price_change"`
	// Volume is in the base currency, VolumeUSD in USD and VolumeNotional in the quote currency; Coinbase-routed spot
	// carries neither of the latter
	Volume         float64 `json:"volume"`
	VolumeUSD      float64 `json:"volume_usd"`
	VolumeNotional float64 `json:"volume_notional"`
}

// Greeks contains an option's Black-Scholes sensitivities; Theta is the lesser of the one day and lifetime theta
type Greeks struct {
	Delta float64 `json:"delta"`
	Gamma float64 `json:"gamma"`
	Vega  float64 `json:"vega"`
	Theta float64 `json:"theta"`
	Rho   float64 `json:"rho"`
}
