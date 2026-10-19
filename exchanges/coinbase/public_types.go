package coinbase

import (
	"bytes"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Contract expiry types
const (
	ContractExpiryTypeExpiring  = "EXPIRING"
	ContractExpiryTypePerpetual = "PERPETUAL"
)

// Candle granularities
const (
	GranularityOneMinute     = "ONE_MINUTE"
	GranularityFiveMinute    = "FIVE_MINUTE"
	GranularityFifteenMinute = "FIFTEEN_MINUTE"
	GranularityThirtyMinute  = "THIRTY_MINUTE"
	GranularityOneHour       = "ONE_HOUR"
	GranularityTwoHour       = "TWO_HOUR"
	GranularityFourHour      = "FOUR_HOUR"
	GranularitySixHour       = "SIX_HOUR"
	GranularityOneDay        = "ONE_DAY"
)

// Product list sort orders
const (
	ProductsSortOrderVolume24HourDescending = "PRODUCTS_SORT_ORDER_VOLUME_24H_DESCENDING"
	ProductsSortOrderListTimeDescending     = "PRODUCTS_SORT_ORDER_LIST_TIME_DESCENDING"
)

// Expiring contract statuses, which select futures in a product list
const (
	ExpiringContractStatusUnexpired = "STATUS_UNEXPIRED"
	ExpiringContractStatusExpired   = "STATUS_EXPIRED"
	ExpiringContractStatusAll       = "STATUS_ALL"
)

// ServerTime is the Advanced Trade API's clock
type ServerTime struct {
	ISO          time.Time  `json:"iso"`
	EpochSeconds types.Time `json:"epochSeconds"`
	EpochMillis  types.Time `json:"epochMillis"`
}

// Duration is a duration Coinbase sends as a number of seconds suffixed with s, such as "3600s"
type Duration time.Duration

// UnmarshalJSON decodes a duration such as "3600s"; null or an empty string leaves it zero
func (d *Duration) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*d = 0
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("%w: %s", errInvalidDuration, data)
	}
	if s == "" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("%w: %q", errInvalidDuration, s)
	}
	*d = Duration(parsed)
	return nil
}

// Duration returns the duration as a time.Duration
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// ListProductsRequest selects the products ListProducts and ListPublicProducts return
type ListProductsRequest struct {
	// Limit is how many products to return, and Offset how many to skip first; Coinbase returns every product when
	// Limit is unset
	Limit  uint64
	Offset uint64
	// ProductType selects spot, futures or equities; Coinbase returns only spot products when it is unset
	ProductType string
	ProductIDs  []string
	// ContractExpiryType selects expiring or perpetual futures, when ProductType is FUTURE
	ContractExpiryType string
	// ExpiringContractStatus selects unexpired (the default), expired or all futures
	ExpiringContractStatus string
	// GetAllProducts returns every product of every type, expired futures included
	GetAllProducts bool
	// ProductsSortOrder sorts products by 24 hour quote volume (the default) or by listing time
	ProductsSortOrder string
	// Cursor pages from where a previous page's next cursor left off
	Cursor string
	// FuturesUnderlyingType selects futures by their underlying, such as FUTURES_UNDERLYING_TYPE_SPOT
	FuturesUnderlyingType string
	// UserCountryCode localises the products' display names
	UserCountryCode string
	// ProductVenue selects the products of one venue; the docs name it only in Expired's description, and Coinbase
	// lists the Global Derivatives gateway's instruments, view only, for DERIBIT
	ProductVenue string
	// Expired returns recently expired instruments instead of active ones, for ProductVenue DERIBIT
	Expired bool
	// GetTradabilityStatus sets a spot product's ViewOnly to whether the account can trade it; ListProducts only
	GetTradabilityStatus bool
}

// ProductsResponse is a page of products
type ProductsResponse struct {
	Products    []Product          `json:"products"`
	NumProducts uint64             `json:"num_products"`
	Pagination  PaginationMetadata `json:"pagination"`
}

// PaginationMetadata is the cursor a page of products is fetched with
type PaginationMetadata struct {
	PreviousCursor string `json:"prev_cursor"`
	NextCursor     string `json:"next_cursor"`
	HasNext        bool   `json:"has_next"`
	HasPrevious    bool   `json:"has_prev"`
}

// Product is a product Coinbase lists, a spot pair, a future or an equity; a numeric field Coinbase leaves empty
// decodes as zero
type Product struct {
	ProductID                    string       `json:"product_id"`
	Price                        types.Number `json:"price"`
	PricePercentageChange24Hour  types.Number `json:"price_percentage_change_24h"`
	Volume24Hour                 types.Number `json:"volume_24h"`
	VolumePercentageChange24Hour types.Number `json:"volume_percentage_change_24h"`
	BaseIncrement                types.Number `json:"base_increment"`
	QuoteIncrement               types.Number `json:"quote_increment"`
	QuoteMinimumSize             types.Number `json:"quote_min_size"`
	QuoteMaximumSize             types.Number `json:"quote_max_size"`
	BaseMinimumSize              types.Number `json:"base_min_size"`
	BaseMaximumSize              types.Number `json:"base_max_size"`
	BaseName                     string       `json:"base_name"`
	QuoteName                    string       `json:"quote_name"`
	Watched                      bool         `json:"watched"`
	IsDisabled                   bool         `json:"is_disabled"`
	New                          bool         `json:"new"`
	// Status is online or delisted for spot, STANDARD for futures and ONLINE for the gateway's instruments
	Status          string `json:"status"`
	CancelOnly      bool   `json:"cancel_only"`
	LimitOnly       bool   `json:"limit_only"`
	PostOnly        bool   `json:"post_only"`
	TradingDisabled bool   `json:"trading_disabled"`
	AuctionMode     bool   `json:"auction_mode"`
	ProductType     string `json:"product_type"`
	QuoteCurrencyID string `json:"quote_currency_id"`
	BaseCurrencyID  string `json:"base_currency_id"`
	// FCMTradingSessionDetails is set for US derivatives, which trade in sessions
	FCMTradingSessionDetails *FCMTradingSessionDetails `json:"fcm_trading_session_details"`
	MidMarketPrice           types.Number              `json:"mid_market_price"`
	// Alias is the product whose book this product trades on, and AliasTo the products that trade on this one's
	Alias              string   `json:"alias"`
	AliasTo            []string `json:"alias_to"`
	BaseDisplaySymbol  string   `json:"base_display_symbol"`
	QuoteDisplaySymbol string   `json:"quote_display_symbol"`
	// ViewOnly is set for a US derivative that has expired, a gateway instrument, and, with GetTradabilityStatus, a
	// spot product the account cannot trade
	ViewOnly                     bool                  `json:"view_only"`
	PriceIncrement               types.Number          `json:"price_increment"`
	DisplayName                  string                `json:"display_name"`
	ProductVenue                 string                `json:"product_venue"`
	ApproximateQuote24HourVolume types.Number          `json:"approximate_quote_24h_volume"`
	NewAt                        time.Time             `json:"new_at"`
	MarketCap                    types.Number          `json:"market_cap"`
	BaseCBRN                     string                `json:"base_cbrn"`
	QuoteCBRN                    string                `json:"quote_cbrn"`
	ProductCBRN                  string                `json:"product_cbrn"`
	IconColour                   string                `json:"icon_color"`
	IconURL                      string                `json:"icon_url"`
	DisplayNameOverwrite         string                `json:"display_name_overwrite"`
	IsAlphaTesting               bool                  `json:"is_alpha_testing"`
	AboutDescription             string                `json:"about_description"`
	BestBidPrice                 types.Number          `json:"best_bid_price"`
	BestAskPrice                 types.Number          `json:"best_ask_price"`
	High24Hour                   types.Number          `json:"high_24h"`
	Low24Hour                    types.Number          `json:"low_24h"`
	HiddenFromDiscovery          bool                  `json:"hidden_from_discovery"`
	FutureProductDetails         *FutureProductDetails `json:"future_product_details"`
	EquityProductDetails         *EquityProductDetails `json:"equity_product_details"`
}

// FCMTradingSessionDetails is the trading session a US derivative trades in
type FCMTradingSessionDetails struct {
	IsSessionOpen                bool                     `json:"is_session_open"`
	OpenTime                     time.Time                `json:"open_time"`
	CloseTime                    time.Time                `json:"close_time"`
	SessionState                 string                   `json:"session_state"`
	AfterHoursOrderEntryDisabled bool                     `json:"after_hours_order_entry_disabled"`
	ClosedReason                 string                   `json:"closed_reason"`
	Maintenance                  *FCMScheduledMaintenance `json:"maintenance"`
}

// FCMScheduledMaintenance is a maintenance window of the US derivatives venue
type FCMScheduledMaintenance struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// FutureProductDetails describes a future: a US derivative, a retired INTX perpetual, or a gateway instrument
type FutureProductDetails struct {
	Venue                    string                   `json:"venue"`
	ContractCode             string                   `json:"contract_code"`
	ContractExpiry           time.Time                `json:"contract_expiry"`
	ContractSize             types.Number             `json:"contract_size"`
	ContractRootUnit         string                   `json:"contract_root_unit"`
	GroupDescription         string                   `json:"group_description"`
	ContractExpiryTimezone   string                   `json:"contract_expiry_timezone"`
	GroupShortDescription    string                   `json:"group_short_description"`
	RiskManagedBy            string                   `json:"risk_managed_by"`
	ContractExpiryType       string                   `json:"contract_expiry_type"`
	PerpetualDetails         *PerpetualProductDetails `json:"perpetual_details"`
	ContractDisplayName      string                   `json:"contract_display_name"`
	TimeToExpiryMilliseconds types.Number             `json:"time_to_expiry_ms"`
	NonCrypto                bool                     `json:"non_crypto"`
	ContractExpiryName       string                   `json:"contract_expiry_name"`
	TwentyFourBySeven        bool                     `json:"twenty_four_by_seven"`
	// FundingInterval, OpenInterest, FundingRate and FundingTime are set for a perpetual style US derivative, which
	// pays funding hourly
	FundingInterval         Duration               `json:"funding_interval"`
	OpenInterest            types.Number           `json:"open_interest"`
	FundingRate             types.Number           `json:"funding_rate"`
	FundingTime             time.Time              `json:"funding_time"`
	DisplayName             string                 `json:"display_name"`
	RegionEnabled           map[string]bool        `json:"region_enabled"`
	IntradayMarginRate      *MarginRate            `json:"intraday_margin_rate"`
	OvernightMarginRate     *MarginRate            `json:"overnight_margin_rate"`
	SettlementPrice         types.Number           `json:"settlement_price"`
	FuturesAssetType        string                 `json:"futures_asset_type"`
	IndexPrice              types.Number           `json:"index_price"`
	ContractCodeDisplayName string                 `json:"contract_code_display_name"`
	ProductGroupCBRN        string                 `json:"product_group_cbrn"`
	FuturesAssetTypes       []string               `json:"futures_asset_types"`
	TradingHoursType        string                 `json:"trading_hours_type"`
	DeribitProductDetails   *DeribitProductDetails `json:"deribit_product_details"`
}

// PerpetualProductDetails describes a perpetual's funding and leverage
type PerpetualProductDetails struct {
	OpenInterest    types.Number `json:"open_interest"`
	FundingRate     types.Number `json:"funding_rate"`
	FundingTime     time.Time    `json:"funding_time"`
	MaximumLeverage types.Number `json:"max_leverage"`
	BaseAssetUUID   string       `json:"base_asset_uuid"`
	UnderlyingType  string       `json:"underlying_type"`
}

// MarginRate is a US derivative's margin rate for long and short positions
type MarginRate struct {
	LongMarginRate  types.Number `json:"long_margin_rate"`
	ShortMarginRate types.Number `json:"short_margin_rate"`
}

// DeribitProductDetails describes a Global Derivatives gateway instrument; Coinbase documents it but sends null
type DeribitProductDetails struct {
	InstrumentID                 string         `json:"instrument_id"`
	SettlementCurrency           string         `json:"settlement_currency"`
	CounterCurrency              string         `json:"counter_currency"`
	SettlementPeriod             string         `json:"settlement_period"`
	PriceIndex                   string         `json:"price_index"`
	InstrumentType               string         `json:"instrument_type"`
	MakerCommission              types.Number   `json:"maker_commission"`
	TakerCommission              types.Number   `json:"taker_commission"`
	BlockTradeCommission         types.Number   `json:"block_trade_commission"`
	BlockTradeTickSize           types.Number   `json:"block_trade_tick_size"`
	BlockTradeMinimumTradeAmount types.Number   `json:"block_trade_min_trade_amount"`
	CreationTime                 time.Time      `json:"creation_time"`
	MaximumLeverage              types.Number   `json:"max_leverage"`
	MaximumLiquidationCommission types.Number   `json:"max_liquidation_commission"`
	TickSizeSteps                []TickSizeStep `json:"tick_size_steps"`
	QuantityTickSize             types.Number   `json:"qty_tick_size"`
	IndexID                      string         `json:"index_id"`
	ProductGroup                 string         `json:"product_group"`
	BaseCurrencyUUID             string         `json:"base_currency_uuid"`
	QuoteCurrencyUUID            string         `json:"quote_currency_uuid"`
	State                        string         `json:"state"`
	DeliveryPrice                types.Number   `json:"delivery_price"`
}

// TickSizeStep is the larger tick size an instrument's prices take above a price
type TickSizeStep struct {
	AbovePrice types.Number `json:"above_price"`
	TickSize   types.Number `json:"tick_size"`
}

// EquityProductDetails describes an equity: its trading permissions, sessions and issuer
type EquityProductDetails struct {
	EquitySubtype string `json:"equity_subtype"`
	Fractionable  bool   `json:"fractionable"`
	// LiquidateOnly is set when the equity may only be sold
	LiquidateOnly bool `json:"liquidate_only"`
	// Ticker is the equity's display ticker; orders name it by its product ID
	Ticker                             string                `json:"ticker"`
	Description                        string                `json:"description"`
	TradingHalted                      bool                  `json:"trading_halted"`
	TradingHaltedStartTime             time.Time             `json:"trading_halted_start_time"`
	TradingHaltedEndTime               time.Time             `json:"trading_halted_end_time"`
	OpenPrice                          types.Number          `json:"open_price"`
	VolumeToday                        types.Number          `json:"volume_today"`
	FractionalNotionalMinimumSize      types.Number          `json:"fractional_notional_min_size"`
	CIK                                string                `json:"cik"`
	ShortName                          string                `json:"short_name"`
	CompanyDescription                 string                `json:"company_description"`
	CompanyWebsite                     string                `json:"company_website"`
	TradingDayInfo                     *EquityTradingDayInfo `json:"trading_day_info"`
	CurrentSession                     string                `json:"current_session"`
	PricePercentageChangeMarketHours   types.Number          `json:"price_percentage_change_market_hours"`
	PricePercentageChangeExtendedHours types.Number          `json:"price_percentage_change_extended_hours"`
	// OfficialPlaceOfListing is the ISO 10383 market identifier code of the exchange that lists the equity
	OfficialPlaceOfListing           string                   `json:"opol"`
	PricePercentageChangeMarketClose types.Number             `json:"price_percentage_change_market_close"`
	LastMarketDayClosePrice          types.Number             `json:"last_market_day_close_price"`
	CurrentMarketDayClosePrice       types.Number             `json:"current_market_day_close_price"`
	RecentTradingDays                *EquityRecentTradingDays `json:"recent_trading_days"`
	EquityTradingFlags               *EquityTradingFlags      `json:"equity_trading_flags"`
	Fundamentals                     *EquityFundamentals      `json:"fundamentals"`
	HasTradableOptions               bool                     `json:"has_tradable_options"`
	OptionVolume                     types.Number             `json:"option_volume"`
	// IPODetails is undocumented and has only been seen null, so it is kept as sent
	IPODetails              json.RawMessage              `json:"ipo_details"`
	AvailableTargetSessions []EquityTradingSessionWindow `json:"available_target_sessions"`
	PutCallRatio            types.Number                 `json:"put_call_ratio"`
	Volatility30Day         types.Number                 `json:"volatility_30d"`
}

// EquityTradingDayInfo is an equity venue's schedule for a trading day
type EquityTradingDayInfo struct {
	VenueID          string                       `json:"venue_id"`
	Date             string                       `json:"date"`
	TradeDateType    string                       `json:"trade_date_type"`
	HolidayName      string                       `json:"holiday_name"`
	VersionID        string                       `json:"version_id"`
	TradingSessions  []EquityTradingSessionWindow `json:"trading_sessions"`
	MarketOpenPrice  types.Number                 `json:"market_open_price"`
	MarketClosePrice types.Number                 `json:"market_close_price"`
}

// EquityRecentTradingDays are an equity venue's recent trading days
type EquityRecentTradingDays struct {
	TradingDays []EquityTradingDayInfo `json:"trading_days"`
}

// EquityTradingSessionWindow is a trading session within a trading day
type EquityTradingSessionWindow struct {
	SessionType       string    `json:"session_type"`
	SessionStartTime  time.Time `json:"session_start_time"`
	SessionEndTime    time.Time `json:"session_end_time"`
	SupportFractional bool      `json:"support_fractional"`
	LimitOnly         bool      `json:"limit_only"`
}

// EquityTradingFlags are the order flows an equity accepts
type EquityTradingFlags struct {
	Tradable             bool `json:"tradable"`
	Searchable           bool `json:"searchable"`
	BuyEnabled           bool `json:"buy_enabled"`
	BuyWholeShares       bool `json:"buy_whole_shares"`
	BuyFractionalShares  bool `json:"buy_fractional_shares"`
	BuyNotional          bool `json:"buy_notional"`
	SellEnabled          bool `json:"sell_enabled"`
	SellWholeShares      bool `json:"sell_whole_shares"`
	SellFractionalShares bool `json:"sell_fractional_shares"`
	SellNotional         bool `json:"sell_notional"`
}

// EquityFundamentals are an equity issuer's fundamentals, which Coinbase sends undocumented
type EquityFundamentals struct {
	Industry               string             `json:"industry"`
	EmployeeCount          types.Number       `json:"employee_count"`
	HeadquartersLocation   string             `json:"headquarters_location"`
	YearFounded            string             `json:"year_founded"`
	EarningsPerShare       types.Number       `json:"eps"`
	PriceEarningsRatio     types.Number       `json:"pe_ratio"`
	CIK                    string             `json:"cik"`
	ShortName              string             `json:"short_name"`
	CompanyDescription     string             `json:"company_description"`
	CompanyWebsite         string             `json:"company_website"`
	OfficialPlaceOfListing string             `json:"opol"`
	Earnings               []EquityEarnings   `json:"earnings"`
	QuarterlyFinancials    []EquityFinancials `json:"quarterly_financials"`
	AnnualFinancials       []EquityFinancials `json:"annual_financials"`
	ForwardDividendYield   types.Number       `json:"forward_dividend_yield"`
	Sector                 string             `json:"sector"`
	IndustryCode           string             `json:"industry_code"`
	HeadquartersCity       string             `json:"headquarters_city"`
	HeadquartersState      string             `json:"headquarters_state"`
	HeadquartersStateCode  string             `json:"headquarters_state_code"`
	EarningsPerShareGrowth types.Number       `json:"eps_growth"`
	IndustryName           string             `json:"industry_name"`
}

// EquityEarnings is an equity issuer's earnings announcement for a period
type EquityEarnings struct {
	ReferencePeriod                      string       `json:"reference_period"`
	AnnouncementDate                     string       `json:"announcement_date"`
	AnnounceTime                         string       `json:"announce_time"`
	Status                               string       `json:"status"`
	EarningsPerShare                     types.Number `json:"eps"`
	EstimatedEarningsPerShare            types.Number `json:"estimated_eps"`
	ReportedDilutedEarningsPerShareFinal types.Number `json:"reported_diluted_eps_final"`
	AdjustedDilutedEarningsPerShareFinal types.Number `json:"adjusted_diluted_eps_final"`
	CurrencyCode                         string       `json:"currency_code"`
}

// EquityFinancials are an equity issuer's financials for a fiscal period
type EquityFinancials struct {
	FiscalPeriod  string       `json:"fiscal_period"`
	PeriodEndDate string       `json:"period_end_date"`
	TotalRevenue  types.Number `json:"total_revenue"`
	NetIncome     types.Number `json:"net_income"`
	GrossProfit   types.Number `json:"gross_profit"`
	NetMargin     types.Number `json:"net_margin"`
	CurrencyCode  string       `json:"currency_code"`
}

// ProductBookRequest selects a product's book
type ProductBookRequest struct {
	ProductID string
	// Limit is how many levels each side returns; Coinbase returns its default depth when it is unset
	Limit uint64
	// AggregationPriceIncrement groups the book's levels into price intervals of this size
	AggregationPriceIncrement float64
}

// ProductBookResponse is a product's book, with its last trade price, midpoint and spread
type ProductBookResponse struct {
	PriceBook PriceBook `json:"pricebook"`
	// Last is the last trade's price, which Coinbase sends only for the authenticated book
	Last           types.Number `json:"last"`
	MidMarket      types.Number `json:"mid_market"`
	SpreadBPS      types.Number `json:"spread_bps"`
	SpreadAbsolute types.Number `json:"spread_absolute"`
}

// PriceBook is a product's bids and asks at a time
type PriceBook struct {
	ProductID string      `json:"product_id"`
	Bids      []BookLevel `json:"bids"`
	Asks      []BookLevel `json:"asks"`
	Time      time.Time   `json:"time"`
}

// BookLevel is a book's price level and the size resting at it
type BookLevel struct {
	Price types.Number `json:"price"`
	Size  types.Number `json:"size"`
}

// ProductCandlesRequest selects a product's candles
type ProductCandlesRequest struct {
	ProductID   string
	Start       time.Time
	End         time.Time
	Granularity string
	// Limit is how many candles to return, at most 350, the default
	Limit uint64
}

// candlesResponse holds a product's candles
type candlesResponse struct {
	Candles []Candle `json:"candles"`
}

// Candle is a product's trading over an interval
type Candle struct {
	Start  types.Time   `json:"start"`
	Low    types.Number `json:"low"`
	High   types.Number `json:"high"`
	Open   types.Number `json:"open"`
	Close  types.Number `json:"close"`
	Volume types.Number `json:"volume"`
}

// MarketTradesRequest selects a product's recent trades
type MarketTradesRequest struct {
	ProductID string
	// Limit is how many trades to return; Coinbase requires it
	Limit uint64
	Start time.Time
	End   time.Time
}

// MarketTradesResponse is a product's recent trades and its best bid and ask
type MarketTradesResponse struct {
	Trades  []MarketTrade `json:"trades"`
	BestBid types.Number  `json:"best_bid"`
	BestAsk types.Number  `json:"best_ask"`
}

// MarketTrade is a trade in a product's market
type MarketTrade struct {
	TradeID   string       `json:"trade_id"`
	ProductID string       `json:"product_id"`
	Price     types.Number `json:"price"`
	Size      types.Number `json:"size"`
	Time      time.Time    `json:"time"`
	// Side is the maker's side of the trade
	Side     string `json:"side"`
	Exchange string `json:"exchange"`
}
