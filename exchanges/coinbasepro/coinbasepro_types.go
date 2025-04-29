package coinbasepro

import (
	"net/url"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

type jwtStruct struct {
	jwt       string
	jwtExpire time.Time
	m         sync.RWMutex
}

type pairAliases struct {
	associatedAliases map[currency.Pair]currency.Pairs
	m                 sync.RWMutex
}

// CoinbasePro is the overarching type across the coinbasepro package
type CoinbasePro struct {
	exchange.Base
	jwtStruct   jwtStruct
	pairAliases pairAliases
}

// Version is used for the niche cases where the Version of the API must be specified and passed around for proper functionality
type Version bool

// FiatTransferType is used so that we don't need to duplicate the four fiat transfer-related endpoints under version 2 of the API
type FiatTransferType bool

// ValueWithCurrency is a sub-struct used in the types Account, NativeAndRaw, DetailedPortfolioResponse, ErrorMetadata, SubscriptionInfo, FuturesBalanceSummary, ListFuturesSweepsResponse, PerpetualsPortfolioSummary, PerpPositionDetail, FeeStruct, AmScale, and ConvertResponse
type ValueWithCurrency struct {
	Value    float64 `json:"value,string"`
	Currency string  `json:"currency"`
}

// Account holds details for a trading account, returned by GetAccountByID and used as a sub-struct in the type AllAccountsResponse
type Account struct {
	UUID              string            `json:"uuid"`
	Name              string            `json:"name"`
	Currency          string            `json:"currency"`
	AvailableBalance  ValueWithCurrency `json:"available_balance"`
	Default           bool              `json:"default"`
	Active            bool              `json:"active"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	DeletedAt         time.Time         `json:"deleted_at"`
	Type              string            `json:"type"`
	Ready             bool              `json:"ready"`
	Hold              ValueWithCurrency `json:"hold"`
	RetailPortfolioID string            `json:"retail_portfolio_id"`
	Platform          string            `json:"platform"`
}

// AllAccountsResponse holds many Account structs, as well as pagination information, returned by ListAccounts
type AllAccountsResponse struct {
	Accounts []Account `json:"accounts"`
	HasNext  bool      `json:"has_next"`
	Cursor   string    `json:"cursor"`
	Size     uint8     `json:"size"`
}

// Params is used within functions to make the setting of parameters easier
type Params struct {
	url.Values
}

// PriceSize is a sub-struct used in the type ProductBook
type PriceSize struct {
	Price float64 `json:"price,string"`
	Size  float64 `json:"size,string"`
}

// ProductBook holds bid and ask prices for a particular product, returned by GetBestBidAsk and used in ProductBookResp
type ProductBook struct {
	ProductID currency.Pair `json:"product_id"`
	Bids      []PriceSize   `json:"bids"`
	Asks      []PriceSize   `json:"asks"`
	Time      time.Time     `json:"time"`
}

// ProductBookResp holds a ProductBook struct, and associated information, returned by GetProductBookV3
type ProductBookResp struct {
	Pricebook      ProductBook `json:"pricebook"`
	Last           float64     `json:"last,string"`
	MidMarket      float64     `json:"mid_market,string"`
	SpreadBPs      float64     `json:"spread_bps,string"`
	SpreadAbsolute float64     `json:"spread_absolute,string"`
}

// FCMTradingSessionDetails is a sub-struct used in the type Product
type FCMTradingSessionDetails struct {
	IsSessionOpen                bool      `json:"is_session_open"`
	OpenTime                     time.Time `json:"open_time"`
	CloseTime                    time.Time `json:"close_time"`
	SessionState                 string    `json:"session_state"`
	AfterHoursOrderEntryDisabled bool      `json:"after_hours_order_entry_disabled"`
}

// PerpetualDetails is a sub-struct used in the type FutureProductDetails
type PerpetualDetails struct {
	OpenInterest  types.Number `json:"open_interest"`
	FundingRate   types.Number `json:"funding_rate"`
	FundingTime   time.Time    `json:"funding_time"`
	MaxLeverage   types.Number `json:"max_leverage"`
	BaseAssetUUID string       `json:"base_asset_uuid"`
}

// FutureProductDetails is a sub-struct used in the type Product
type FutureProductDetails struct {
	Venue                  string           `json:"venue"`
	ContractCode           string           `json:"contract_code"`
	ContractExpiry         time.Time        `json:"contract_expiry"`
	ContractSize           types.Number     `json:"contract_size"`
	ContractRootUnit       string           `json:"contract_root_unit"`
	GroupDescription       string           `json:"group_description"`
	ContractExpiryTimezone string           `json:"contract_expiry_timezone"`
	GroupShortDescription  string           `json:"group_short_description"`
	RiskManagedBy          string           `json:"risk_managed_by"`
	ContractExpiryType     string           `json:"contract_expiry_type"`
	PerpetualDetails       PerpetualDetails `json:"perpetual_details"`
	ContractDisplayName    string           `json:"contract_display_name"`
	TimeToExpiry           time.Duration    `json:"time_to_expiry_ms,string"`
	NonCrypto              bool             `json:"non_crypto"`
	ContractExpiryName     string           `json:"contract_expiry_name"`
}

// Product holds product information, returned by GetProductByID, and used as a sub-struct in the type AllProducts
type Product struct {
	ID                        currency.Pair            `json:"product_id"`
	Price                     types.Number             `json:"price"`
	PricePercentageChange24H  types.Number             `json:"price_percentage_change_24h"`
	Volume24H                 types.Number             `json:"volume_24h"`
	VolumePercentageChange24H types.Number             `json:"volume_percentage_change_24h"`
	BaseIncrement             types.Number             `json:"base_increment"`
	QuoteIncrement            types.Number             `json:"quote_increment"`
	QuoteMinSize              types.Number             `json:"quote_min_size"`
	QuoteMaxSize              types.Number             `json:"quote_max_size"`
	BaseMinSize               types.Number             `json:"base_min_size"`
	BaseMaxSize               types.Number             `json:"base_max_size"`
	BaseName                  string                   `json:"base_name"`
	QuoteName                 string                   `json:"quote_name"`
	Watched                   bool                     `json:"watched"`
	IsDisabled                bool                     `json:"is_disabled"`
	New                       bool                     `json:"new"`
	Status                    string                   `json:"status"`
	CancelOnly                bool                     `json:"cancel_only"`
	LimitOnly                 bool                     `json:"limit_only"`
	PostOnly                  bool                     `json:"post_only"`
	TradingDisabled           bool                     `json:"trading_disabled"`
	AuctionMode               bool                     `json:"auction_mode"`
	ProductType               string                   `json:"product_type"`
	QuoteCurrencyID           currency.Code            `json:"quote_currency_id"`
	BaseCurrencyID            currency.Code            `json:"base_currency_id"`
	FCMTradingSessionDetails  FCMTradingSessionDetails `json:"fcm_trading_session_details"`
	MidMarketPrice            types.Number             `json:"mid_market_price"`
	Alias                     currency.Pair            `json:"alias"`
	AliasTo                   []currency.Pair          `json:"alias_to"`
	BaseDisplaySymbol         string                   `json:"base_display_symbol"`
	QuoteDisplaySymbol        string                   `json:"quote_display_symbol"`
	ViewOnly                  bool                     `json:"view_only"`
	PriceIncrement            types.Number             `json:"price_increment"`
	DisplayName               string                   `json:"display_name"`
	ProductVenue              string                   `json:"product_venue"`
	ApproximateQuote24HVolume types.Number             `json:"approximate_quote_24h_volume"`
	FutureProductDetails      FutureProductDetails     `json:"future_product_details"`
}

// AllProducts holds information on a lot of available currency pairs, returned by GetAllProducts
type AllProducts struct {
	Products    []Product `json:"products"`
	NumProducts int32     `json:"num_products"`
}

// Klines holds historic trade information, returned by GetHistoricKlines
type Klines struct {
	Start  types.Time `json:"start"`
	Low    float64    `json:"low,string"`
	High   float64    `json:"high,string"`
	Open   float64    `json:"open,string"`
	Close  float64    `json:"close,string"`
	Volume float64    `json:"volume,string"`
}

// Trades is a sub-struct used in the type Ticker
type Trades struct {
	TradeID   string        `json:"trade_id"`
	ProductID currency.Pair `json:"product_id"`
	Price     float64       `json:"price,string"`
	Size      float64       `json:"size,string"`
	Time      time.Time     `json:"time"`
	Side      string        `json:"side"`
	Bid       types.Number  `json:"bid"`
	Ask       types.Number  `json:"ask"`
}

// Ticker holds basic ticker information, returned by GetTicker
type Ticker struct {
	Trades  []Trades     `json:"trades"`
	BestBid types.Number `json:"best_bid"`
	BestAsk types.Number `json:"best_ask"`
}

// MarketMarketIOC is a sub-struct used in the type OrderConfiguration
type MarketMarketIOC struct {
	QuoteSize types.Number `json:"quote_size,omitempty"`
	BaseSize  types.Number `json:"base_size,omitempty"`
}

// LimitLimitGTC is a sub-struct used in the type OrderConfiguration
type LimitLimitGTC struct {
	BaseSize   types.Number `json:"base_size,omitempty"`
	QuoteSize  types.Number `json:"quote_size,omitempty"`
	LimitPrice types.Number `json:"limit_price"`
	PostOnly   bool         `json:"post_only"`
}

// LimitLimitGTD is a sub-struct used in the type OrderConfiguration
type LimitLimitGTD struct {
	BaseSize   types.Number `json:"base_size,omitempty"`
	QuoteSize  types.Number `json:"quote_size,omitempty"`
	LimitPrice types.Number `json:"limit_price"`
	EndTime    time.Time    `json:"end_time"`
	PostOnly   bool         `json:"post_only"`
}

// StopLimitStopLimitGTC is a sub-struct used in the type OrderConfiguration
type StopLimitStopLimitGTC struct {
	BaseSize      types.Number `json:"base_size,omitempty"`
	QuoteSize     types.Number `json:"quote_size,omitempty"`
	LimitPrice    types.Number `json:"limit_price"`
	StopPrice     types.Number `json:"stop_price"`
	StopDirection string       `json:"stop_direction"`
}

// StopLimitStopLimitGTD is a sub-struct used in the type OrderConfiguration
type StopLimitStopLimitGTD struct {
	BaseSize      types.Number `json:"base_size,omitempty"`
	QuoteSize     types.Number `json:"quote_size,omitempty"`
	LimitPrice    types.Number `json:"limit_price"`
	StopPrice     types.Number `json:"stop_price"`
	EndTime       time.Time    `json:"end_time"`
	StopDirection string       `json:"stop_direction"`
}

// OrderConfiguration is a struct used in the formation of requests in PrepareOrderConfig, and is a sub-struct used in the types PlaceOrderResp and GetOrderResponse
type OrderConfiguration struct {
	MarketMarketIOC       *MarketMarketIOC       `json:"market_market_ioc,omitempty"`
	LimitLimitGTC         *LimitLimitGTC         `json:"limit_limit_gtc,omitempty"`
	LimitLimitGTD         *LimitLimitGTD         `json:"limit_limit_gtd,omitempty"`
	StopLimitStopLimitGTC *StopLimitStopLimitGTC `json:"stop_limit_stop_limit_gtc,omitempty"`
	StopLimitStopLimitGTD *StopLimitStopLimitGTD `json:"stop_limit_stop_limit_gtd,omitempty"`
}

// SuccessResponse is a sub-struct used in the type PlaceOrderResp
type SuccessResponse struct {
	OrderID       string        `json:"order_id"`
	ProductID     currency.Pair `json:"product_id"`
	Side          string        `json:"side"`
	ClientOrderID string        `json:"client_oid"`
}

// PlaceOrderInfo is a struct used in the formation of requests in PlaceOrder
type PlaceOrderInfo struct {
	ClientOID             string
	ProductID             string
	Side                  string
	StopDirection         string
	OrderType             string
	SelfTradePreventionID string
	MarginType            string
	RetailPortfolioID     string
	BaseAmount            float64
	QuoteAmount           float64
	LimitPrice            float64
	StopPrice             float64
	Leverage              float64
	PostOnly              bool
	EndTime               time.Time
}

// PlaceOrderResp contains information on an order, returned by PlaceOrder
type PlaceOrderResp struct {
	Success            bool               `json:"success"`
	FailureReason      string             `json:"failure_reason"`
	SuccessResponse    SuccessResponse    `json:"success_response"`
	OrderConfiguration OrderConfiguration `json:"order_configuration"`
}

// OrderCancelDetail contains information on attempted order cancellations, returned by CancelOrders
type OrderCancelDetail struct {
	Success       bool   `json:"success"`
	FailureReason string `json:"failure_reason"`
	OrderID       string `json:"order_id"`
}

// EditOrderPreviewResp contains information on the effects of editing an order, returned by EditOrderPreview
type EditOrderPreviewResp struct {
	Slippage           float64 `json:"slippage,string"`
	OrderTotal         float64 `json:"order_total,string"`
	CommissionTotal    float64 `json:"commission_total,string"`
	QuoteSize          float64 `json:"quote_size,string"`
	BaseSize           float64 `json:"base_size,string"`
	BestBid            float64 `json:"best_bid,string"`
	BestAsk            float64 `json:"best_ask,string"`
	AverageFilledPrice float64 `json:"average_filled_price,string"`
}

// EditHistory is a sub-struct used in the type GetOrderResponse
type EditHistory struct {
	Price                  float64   `json:"price,string"`
	Size                   float64   `json:"size,string"`
	ReplaceAcceptTimestamp time.Time `json:"replace_accept_timestamp"`
}

// GetOrderResponse contains information on an order, returned by GetOrderByID IterativeGetAllOrders, and used in GetAllOrdersResp
type GetOrderResponse struct {
	OrderID               string             `json:"order_id"`
	ProductID             currency.Pair      `json:"product_id"`
	UserID                string             `json:"user_id"`
	OrderConfiguration    OrderConfiguration `json:"order_configuration"`
	Side                  string             `json:"side"`
	ClientOID             string             `json:"client_order_id"`
	Status                string             `json:"status"`
	TimeInForce           string             `json:"time_in_force"`
	CreatedTime           time.Time          `json:"created_time"`
	CompletionPercentage  float64            `json:"completion_percentage,string"`
	FilledSize            float64            `json:"filled_size,string"`
	AverageFilledPrice    float64            `json:"average_filled_price,string"`
	Fee                   types.Number       `json:"fee"`
	NumberOfFills         int64              `json:"num_fills,string"`
	FilledValue           float64            `json:"filled_value,string"`
	PendingCancel         bool               `json:"pending_cancel"`
	SizeInQuote           bool               `json:"size_in_quote"`
	TotalFees             float64            `json:"total_fees,string"`
	SizeInclusiveOfFees   bool               `json:"size_inclusive_of_fees"`
	TotalValueAfterFees   float64            `json:"total_value_after_fees,string"`
	TriggerStatus         string             `json:"trigger_status"`
	OrderType             string             `json:"order_type"`
	RejectReason          string             `json:"reject_reason"`
	Settled               bool               `json:"settled"`
	ProductType           string             `json:"product_type"`
	RejectMessage         string             `json:"reject_message"`
	CancelMessage         string             `json:"cancel_message"`
	OrderPlacementSource  string             `json:"order_placement_source"`
	OutstandingHoldAmount float64            `json:"outstanding_hold_amount,string"`
	IsLiquidation         bool               `json:"is_liquidation"`
	LastFillTime          time.Time          `json:"last_fill_time"`
	EditHistory           []EditHistory      `json:"edit_history"`
	Leverage              types.Number       `json:"leverage"`
	MarginType            string             `json:"margin_type"`
	RetailPortfolioID     string             `json:"retail_portfolio_id"`
}

// Fills is a sub-struct used in the type FillResponse
type Fills struct {
	EntryID            string        `json:"entry_id"`
	TradeID            string        `json:"trade_id"`
	OrderID            string        `json:"order_id"`
	TradeTime          time.Time     `json:"trade_time"`
	TradeType          string        `json:"trade_type"`
	Price              float64       `json:"price,string"`
	Size               float64       `json:"size,string"`
	Commission         float64       `json:"commission,string"`
	ProductID          currency.Pair `json:"product_id"`
	SequenceTimestamp  time.Time     `json:"sequence_timestamp"`
	LiquidityIndicator string        `json:"liquidity_indicator"`
	SizeInQuote        bool          `json:"size_in_quote"`
	UserID             string        `json:"user_id"`
	Side               string        `json:"side"`
}

// FillResponse contains fill information, returned by ListFills
type FillResponse struct {
	Fills  []Fills `json:"fills"`
	Cursor string  `json:"cursor"`
}

// PreviewOrderInfo is a struct used in the formation of requests in PreviewOrder
type PreviewOrderInfo struct {
	ProductID        string
	Side             string
	OrderType        string
	StopDirection    string
	MarginType       string
	CommissionValue  float64
	BaseAmount       float64
	QuoteAmount      float64
	LimitPrice       float64
	StopPrice        float64
	TradableBalance  float64
	Leverage         float64
	PostOnly         bool
	IsMax            bool
	SkipFCMRiskCheck bool
	EndTime          time.Time
}

// PreviewOrderResp contains information on the effects of placing an order, returned by PreviewOrder
type PreviewOrderResp struct {
	OrderTotal       float64  `json:"order_total,string"`
	CommissionTotal  float64  `json:"commission_total,string"`
	Errs             []string `json:"errs"`
	Warning          []string `json:"warning"`
	QuoteSize        float64  `json:"quote_size,string"`
	BaseSize         float64  `json:"base_size,string"`
	BestBid          float64  `json:"best_bid,string"`
	BestAsk          float64  `json:"best_ask,string"`
	IsMax            bool     `json:"is_max"`
	OrderMarginTotal float64  `json:"order_margin_total,string"`
	Leverage         float64  `json:"leverage,string"`
	LongLeverage     float64  `json:"long_leverage,string"`
	ShortLeverage    float64  `json:"short_leverage,string"`
	Slippage         float64  `json:"slippage,string"`
}

// SimplePortfolioData is a sub-struct used in the type DetailedPortfolioResponse
type SimplePortfolioData struct {
	Name    string `json:"name"`
	UUID    string `json:"uuid"`
	Type    string `json:"type"`
	Deleted bool   `json:"deleted"`
}

// MovePortfolioFundsResponse contains the UUIDs of the portfolios involved. Returned by MovePortfolioFunds
type MovePortfolioFundsResponse struct {
	SourcePortfolioUUID string `json:"source_portfolio_uuid"`
	TargetPortfolioUUID string `json:"target_portfolio_uuid"`
}

// FundsData is used internally when preparing a request in MovePortfolioFunds
type FundsData struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

// NativeAndRaw is a sub-struct used in the type DetailedPortfolioResponse
type NativeAndRaw struct {
	UserNativeCurrency ValueWithCurrency `json:"userNativeCurrency"`
	RawCurrency        ValueWithCurrency `json:"rawCurrency"`
}

// PortfolioBalances is a sub-struct used in the type DetailedPortfolioResponse
type PortfolioBalances struct {
	TotalBalance               ValueWithCurrency `json:"total_balance"`
	TotalFuturesBalance        ValueWithCurrency `json:"total_futures_balance"`
	TotalCashEquivalentBalance ValueWithCurrency `json:"total_cash_equivalent_balance"`
	TotalCryptoBalance         ValueWithCurrency `json:"total_crypto_balance"`
	FuturesUnrealizedPNL       ValueWithCurrency `json:"futures_unrealized_pnl"`
	PerpUnrealizedPNL          ValueWithCurrency `json:"perp_unrealized_pnl"`
}

// SpotPositions is a sub-struct used in the type DetailedPortfolioResponse
type SpotPositions struct {
	Asset                 string            `json:"asset"`
	AccountUUID           string            `json:"account_uuid"`
	TotalBalanceFiat      float64           `json:"total_balance_fiat"`
	TotalBalanceCrypto    float64           `json:"total_balance_crypto"`
	AvailableToTreadeFiat float64           `json:"available_to_trade_fiat"`
	Allocation            float64           `json:"allocation"`
	OneDayChange          float64           `json:"one_day_change"`
	CostBasis             ValueWithCurrency `json:"cost_basis"`
	AssetImgURL           string            `json:"asset_img_url"`
	IsCash                bool              `json:"is_cash"`
}

// PerpPositions is a sub-struct used in the type DetailedPortfolioResponse
type PerpPositions struct {
	ProductID             currency.Pair `json:"product_id"`
	ProductUUID           string        `json:"product_uuid"`
	Symbol                string        `json:"symbol"`
	AssetImageURL         string        `json:"asset_image_url"`
	VWAP                  NativeAndRaw  `json:"vwap"`
	PositionSide          string        `json:"position_side"`
	NetSize               float64       `json:"net_size,string"`
	BuyOrderSize          float64       `json:"buy_order_size,string"`
	SellOrderSize         float64       `json:"sell_order_size,string"`
	IMContribution        float64       `json:"im_contribution,string"`
	UnrealizedPNL         NativeAndRaw  `json:"unrealized_pnl"`
	MarkPrice             NativeAndRaw  `json:"mark_price"`
	LiquidationPrice      NativeAndRaw  `json:"liquidation_price"`
	Leverage              float64       `json:"leverage,string"`
	IMNotional            NativeAndRaw  `json:"im_notional"`
	MMNotional            NativeAndRaw  `json:"mm_notional"`
	PositionNotional      NativeAndRaw  `json:"position_notional"`
	MarginType            string        `json:"margin_type"`
	LiquidationBuffer     float64       `json:"liquidation_buffer,string"`
	LiquidationPercentage float64       `json:"liquidation_percentage,string"`
}

// FuturesPositions is a sub-struct used in the type DetailedPortfolioResponse
type FuturesPositions []struct {
	ProductID       currency.Pair `json:"product_id"`
	ContractSize    float64       `json:"contract_size,string"`
	Side            string        `json:"side"`
	Amount          float64       `json:"amount,string"`
	AvgEntryPrice   float64       `json:"avg_entry_price,string"`
	CurrentPrice    float64       `json:"current_price,string"`
	UnrealizedPNL   float64       `json:"unrealized_pnl,string"`
	Expiry          time.Time     `json:"expiry"`
	UnderlyingAsset string        `json:"underlying_asset"`
	AssetImgURL     string        `json:"asset_img_url"`
	ProductName     string        `json:"product_name"`
	Venue           string        `json:"venue"`
	NotionalValue   float64       `json:"notional_value,string"`
}

// DetailedPortfolioResponse contains a great deal of information on a single portfolio. Returned by GetPortfolioByID
type DetailedPortfolioResponse struct {
	Portfolio         SimplePortfolioData `json:"portfolio"`
	PortfolioBalances PortfolioBalances   `json:"portfolio_balances"`
	SpotPositions     []SpotPositions     `json:"spot_positions"`
	PerpPositions     []PerpPositions     `json:"perp_positions"`
	FuturesPositions  []FuturesPositions  `json:"futures_positions"`
}

// FuturesBalanceSummary contains information on futures balances, returned by GetFuturesBalanceSummary
type FuturesBalanceSummary struct {
	FuturesBuyingPower          ValueWithCurrency `json:"futures_buying_power"`
	TotalUSDBalance             ValueWithCurrency `json:"total_usd_balance"`
	CBIUSDBalance               ValueWithCurrency `json:"cbi_usd_balance"`
	CFMUSDBalance               ValueWithCurrency `json:"cfm_usd_balance"`
	TotalOpenOrdersHoldAmount   ValueWithCurrency `json:"total_open_orders_hold_amount"`
	UnrealizedPNL               ValueWithCurrency `json:"unrealized_pnl"`
	DailyRealizedPNL            ValueWithCurrency `json:"daily_realized_pnl"`
	InitialMargin               ValueWithCurrency `json:"initial_margin"`
	AvailableMargin             ValueWithCurrency `json:"available_margin"`
	LiquidationThreshold        ValueWithCurrency `json:"liquidation_threshold"`
	LiquidationBufferAmount     ValueWithCurrency `json:"liquidation_buffer_amount"`
	LiquidationBufferPercentage float64           `json:"liquidation_buffer_percentage,string"`
}

// FuturesPosition contains information on a single futures position, returned by GetFuturesPositionByID and ListFuturesPositions
type FuturesPosition struct {
	// This may belong in a struct of its own called "position", requiring a bit more abstraction, but for the moment I'll assume it doesn't
	ProductID         currency.Pair `json:"product_id"`
	ExpirationTime    time.Time     `json:"expiration_time"`
	Side              string        `json:"side"`
	NumberOfContracts float64       `json:"number_of_contracts,string"`
	CurrentPrice      float64       `json:"current_price,string"`
	AverageEntryPrice float64       `json:"avg_entry_price,string"`
	UnrealizedPNL     float64       `json:"unrealized_pnl,string"`
	DailyRealizedPNL  float64       `json:"daily_realized_pnl,string"`
}

// SweepData contains information on pending and processing sweep requests, returned by ListFuturesSweeps
type SweepData struct {
	ID              string            `json:"id"`
	RequestedAmount ValueWithCurrency `json:"requested_amount"`
	ShouldSweepAll  bool              `json:"should_sweep_all"`
	Status          string            `json:"status"`
	ScheduledTime   time.Time         `json:"scheduled_time"`
}

// PerpetualsPortfolioSummary contains information on perpetuals portfolio balances, used as a sub-struct in the types PerpPositionDetail, AllPerpPosResponse, and OnePerpPosResponse
type PerpetualsPortfolioSummary struct {
	PortfolioUUID              string            `json:"portfolio_uuid"`
	Collateral                 float64           `json:"collateral,string"`
	PositionNotional           float64           `json:"position_notional,string"`
	OpenPositionNotional       float64           `json:"open_position_notional,string"`
	PendingFees                float64           `json:"pending_fees,string"`
	Borrow                     float64           `json:"borrow,string"`
	AccruedInterest            float64           `json:"accrued_interest,string"`
	RollingDebt                float64           `json:"rolling_debt,string"`
	PortfolioInitialMargin     float64           `json:"portfolio_initial_margin,string"`
	PortfolioIMNotional        ValueWithCurrency `json:"portfolio_im_notional"`
	PortfolioMaintenanceMargin float64           `json:"portfolio_maintenance_margin,string"`
	PortfolioMMNotional        ValueWithCurrency `json:"portfolio_mm_notional"`
	LiquidationPercentage      float64           `json:"liquidation_percentage,string"`
	LiquidationBuffer          float64           `json:"liquidation_buffer,string"`
	MarginType                 string            `json:"margin_type"`
	MarginFlags                string            `json:"margin_flags"`
	LiquidationStatus          string            `json:"liquidation_status"`
	UnrealizedPNL              ValueWithCurrency `json:"unrealized_pnl"`
	BuyingPower                ValueWithCurrency `json:"buying_power"`
	TotalBalance               ValueWithCurrency `json:"total_balance"`
	MaxWithDrawal              ValueWithCurrency `json:"max_withdrawal"`
}

// PerpPositionDetail contains information on a single perpetuals position, used as a sub-struct in the types AllPerpPosResponse and OnePerpPosResponse
type PerpPositionDetail struct {
	ProductID             currency.Pair              `json:"product_id"`
	ProductUUID           string                     `json:"product_uuid"`
	Symbol                string                     `json:"symbol"`
	VWAP                  ValueWithCurrency          `json:"vwap"`
	PositionSide          string                     `json:"position_side"`
	NetSize               float64                    `json:"net_size,string"`
	BuyOrderSize          float64                    `json:"buy_order_size,string"`
	SellOrderSize         float64                    `json:"sell_order_size,string"`
	IMContribution        float64                    `json:"im_contribution,string"`
	UnrealizedPNL         ValueWithCurrency          `json:"unrealized_pnl"`
	MarkPrice             ValueWithCurrency          `json:"mark_price"`
	LiquidationPrice      ValueWithCurrency          `json:"liquidation_price"`
	Leverage              float64                    `json:"leverage,string"`
	IMNotional            ValueWithCurrency          `json:"im_notional"`
	MMNotional            ValueWithCurrency          `json:"mm_notional"`
	PositionNotional      ValueWithCurrency          `json:"position_notional"`
	MarginType            string                     `json:"margin_type"`
	LiquidationBuffer     float64                    `json:"liquidation_buffer,string"`
	LiquidationPercentage float64                    `json:"liquidation_percentage,string"`
	PortfolioSummary      PerpetualsPortfolioSummary `json:"portfolio_summary"`
}

// AllPerpPosResponse contains information on perpetuals positions, returned by GetAllPerpetualsPositions
type AllPerpPosResponse struct {
	Positions        []PerpPositionDetail       `json:"positions"`
	PortfolioSummary PerpetualsPortfolioSummary `json:"portfolio_summary"`
}

// OnePerpPosResponse contains information on a single perpetuals position, returned by GetPerpetualsPositionByID
type OnePerpPosResponse struct {
	Position         PerpPositionDetail         `json:"position"`
	PortfolioSummary PerpetualsPortfolioSummary `json:"portfolio_summary"`
}

// FeeTier is a sub-struct used in the type TransactionSummary
type FeeTier struct {
	PricingTier  string       `json:"pricing_tier"`
	USDFrom      float64      `json:"usd_from,string"`
	USDTo        float64      `json:"usd_to,string"`
	TakerFeeRate float64      `json:"taker_fee_rate,string"`
	MakerFeeRate float64      `json:"maker_fee_rate,string"`
	AOPFrom      types.Number `json:"aop_from"`
	AOPTo        types.Number `json:"aop_to"`
}

// MarginRate is a sub-struct used in the type TransactionSummary
type MarginRate struct {
	Value float64 `json:"value,string"`
}

// GoodsAndServicesTax is a sub-struct used in the type TransactionSummary
type GoodsAndServicesTax struct {
	Rate float64 `json:"rate,string"`
	Type string  `json:"type"`
}

// TransactionSummary contains a summary of transaction fees, volume, and the like. Returned by GetTransactionSummary
type TransactionSummary struct {
	TotalVolume             float64             `json:"total_volume"`
	TotalFees               float64             `json:"total_fees"`
	FeeTier                 FeeTier             `json:"fee_tier"`
	MarginRate              MarginRate          `json:"margin_rate"`
	GoodsAndServicesTax     GoodsAndServicesTax `json:"goods_and_services_tax"`
	AdvancedTradeOnlyVolume float64             `json:"advanced_trade_only_volume"`
	AdvancedTradeOnlyFees   float64             `json:"advanced_trade_only_fees"`
	CoinbaseProVolume       float64             `json:"coinbase_pro_volume"`
	CoinbaseProFees         float64             `json:"coinbase_pro_fees"`
	TotalBalance            types.Number        `json:"total_balance"`
	HasPromoFee             bool                `json:"has_promo_fee"`
}

// ListOrdersResp contains information on a lot of orders, returned by ListOrders
type ListOrdersResp struct {
	Orders   []GetOrderResponse `json:"orders"`
	Sequence int64              `json:"sequence,string"`
	HasNext  bool               `json:"has_next"`
	Cursor   string             `json:"cursor"`
}

// LinkStruct is a sub-struct storing information on links, used in Disclosure and ConvertResponse
type LinkStruct struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

// Disclosure is a sub-struct used in FeeStruct
type Disclosure struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Link        LinkStruct `json:"link"`
}

// WaivedDetails is a sub-struct used in FeeStruct
type WaivedDetails struct {
	Amount ValueWithCurrency `json:"amount"`
	Source string            `json:"source"`
}

// FeeStruct is a sub-struct storing information on fees, used in ConvertResponse
type FeeStruct struct {
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Amount        ValueWithCurrency `json:"amount"`
	Label         string            `json:"label"`
	Disclosure    Disclosure        `json:"disclosure"`
	WaivedDetails WaivedDetails     `json:"waived_details"`
}

// AccountID is a sub-struct, used in AccountStruct
type AccountID struct {
	AccountID string `json:"account_id"`
}

// HashWithHeight is a sub-struct, used in AccountStruct
type HashWithHeight struct {
	Hash   string `json:"hsh"`
	Height int32  `json:"height"`
}

// AddressHolder is a sub-struct, used in AccountStruct and FedWireInstitution
type AddressHolder struct {
	Lines       []string `json:"lines"`
	CountryCode string   `json:"country_code"`
}

// FedAccountHolder is a sub-struct, used in Fedwire
type FedAccountHolder struct {
	LegalName     string        `json:"legal_name"`
	AccountNumber string        `json:"account_number"`
	Address       AddressHolder `json:"address"`
}

// FedWireInstitution is a sub-struct, used in Fedwire
type FedWireInstitution struct {
	Name           string        `json:"name"`
	Address        AddressHolder `json:"address"`
	Identifier     string        `json:"identifier"`
	Type           string        `json:"type"`
	IdentifierCode string        `json:"identifier_code"`
}

// Fedwire is a sub-struct, used in AccountStruct
type Fedwire struct {
	RoutingNumber    string             `json:"routing_number"`
	AccountHolder    FedAccountHolder   `json:"account_holder"`
	Bank             FedWireInstitution `json:"bank"`
	IntermediaryBank FedWireInstitution `json:"intermediary_bank"`
}

// SwiftAccountHolder is a sub-struct, used in Swift
type SwiftAccountHolder struct {
	LegalName                  string `json:"legal_name"`
	IBAN                       string `json:"iban"`
	BBAN                       string `json:"bban"`
	DomesticAccountID          string `json:"domestic_account_id"`
	CustomerPaymentAddress1    string `json:"customer_payment_address1"`
	CustomerPaymentAddress2    string `json:"customer_payment_address2"`
	CustomerPaymentAddress3    string `json:"customer_payment_address3"`
	CustomerPaymentCountryCode string `json:"customer_payment_country_code"`
}

// SwiftInstitution is a sub-struct, used in Swift
type SwiftInstitution struct {
	BIC                 string `json:"bic"`
	Name                string `json:"name"`
	BankAddress1        string `json:"bank_address1"`
	BankAddress2        string `json:"bank_address2"`
	BankAddress3        string `json:"bank_address3"`
	BankCountryCode     string `json:"bank_country_code"`
	DomesticBankID      string `json:"domestic_bank_id"`
	InternationalBankID string `json:"international_bank_id"`
}

// Swift is a sub-struct, used in AccountStruct
type Swift struct {
	AccountHolder SwiftAccountHolder `json:"account_holder"`
	Institution   SwiftInstitution   `json:"institution"`
	Intermediary  SwiftInstitution   `json:"intermediary"`
}

// ValueWithStoreID is a sub-struct, used in CardInfo
type ValueWithStoreID struct {
	Value   string `json:"value"`
	StoreID string `json:"store_id"`
}

// MonthYear is a sub-struct, used in CardInfo
type MonthYear struct {
	Month string `json:"month"`
	Year  string `json:"year"`
}

// MerchantID is a sub-struct, used in CardInfo
type MerchantID struct {
	MerchantID string `json:"mid"`
}

// VaultToken is a sub-struct, used in CardInfo
type VaultToken struct {
	Value   string `json:"value"`
	VaultID string `json:"vault_id"`
}

// WorldplayParams is a sub-struct, used in CardInfo
type WorldplayParams struct {
	TokenValue        string `json:"token_value"`
	UsesMerchantToken bool   `json:"uses_merchant_token"`
	AcceptHeader      string `json:"accept_header"`
	UserAgentHeader   string `json:"user_agent_header"`
	ShopperIP         string `json:"shopper_ip"`
	ShopperSessionID  string `json:"shopper_session_id"`
}

// FullAddress is a sub-struct, used in CardInfo and UKAccountHolder
type FullAddress struct {
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// FullDate is a sub-struct, used in CardInfo
type FullDate struct {
	Month string `json:"month"`
	Day   string `json:"day"`
	Year  string `json:"year"`
}

// SourceID is a sub-struct, used in CardInfo
type SourceID struct {
	SourceID string `json:"source_id"`
}

// CardInfo is a sub-struct, used in AccountStruct
type CardInfo struct {
	FirstDataToken              ValueWithStoreID `json:"first_data_token"`
	ExpiryDate                  MonthYear        `json:"expiry_date"`
	PostalCode                  string           `json:"postal_code"`
	Merchant                    MerchantID       `json:"merchant"`
	VaultToken                  VaultToken       `json:"vault_token"`
	WorldpayParams              WorldplayParams  `json:"worldpay_params"`
	PreviousSchemeTransactionID string           `json:"previous_scheme_tx_id"`
	CustomerName                string           `json:"customer_name"`
	Address                     FullAddress      `json:"address"`
	PhoneNumber                 string           `json:"phone_number"`
	UserID                      string           `json:"user_id"`
	CustomerFirstName           string           `json:"customer_first_name"`
	CustomerLastName            string           `json:"customer_last_name"`
	SixDigitBin                 string           `json:"six_digit_bin"`
	CustomerDateOfBirth         FullDate         `json:"customer_dob"`
	Scheme                      string           `json:"scheme"`
	EightDigitBin               string           `json:"eight_digit_bin"`
	CheckoutToken               SourceID         `json:"checkout_token"`
}

// ZenginAccountHolder is a sub-struct, used in Zengin
type ZenginAccountHolder struct {
	LegalName  string `json:"legal_name"`
	Identifier string `json:"identifier"`
	Type       string `json:"type"`
}

// BankAndBranchCode is a sub-struct, used in Zengin
type BankAndBranchCode struct {
	BankCode   string `json:"bank_code"`
	BranchCode string `json:"branch_code"`
}

// Zengin is a sub-struct, used in AccountStruct
type Zengin struct {
	AccountHolder ZenginAccountHolder `json:"account_holder"`
	Institution   BankAndBranchCode   `json:"institution"`
}

// UKAccountHolder is a sub-struct, used in UKAccount
type UKAccountHolder struct {
	LegalName     string      `json:"legal_name"`
	BBAN          string      `json:"bban"`
	SortCode      string      `json:"sort_code"`
	AccountNumber string      `json:"account_number"`
	Address       FullAddress `json:"address"`
}

// Name is a sub-struct, used in UKAccount and AccountStruct
type Name struct {
	Name string `json:"name"`
}

// UKAccount is a sub-struct, used in AccountStruct
type UKAccount struct {
	AccountHolder     UKAccountHolder `json:"account_holder"`
	Institution       Name            `json:"institution"`
	CustomerFirstName string          `json:"customer_first_name"`
	CustomerLastName  string          `json:"customer_last_name"`
	Email             string          `json:"email"`
	PhoneNumber       string          `json:"phone_number"`
}

// SEPAAccountHolder is a sub-struct, used in SEPA
type SEPAAccountHolder struct {
	LegalName string `json:"legal_name"`
	IBAN      string `json:"iban"`
	BBAN      string `json:"bban"`
}

// BICWithName is a sub-struct, used in SEPA
type BICWithName struct {
	BIC  string `json:"bic"`
	Name string `json:"name"`
}

// SEPA is a sub-struct, used in AccountStruct
type SEPA struct {
	AccountHolder     SEPAAccountHolder `json:"account_holder"`
	Institution       BICWithName       `json:"institution"`
	CustomerFirstName string            `json:"customer_first_name"`
	CustomerLastName  string            `json:"customer_last_name"`
	Email             string            `json:"email"`
	PhoneNumber       string            `json:"phone_number"`
}

// PayPalAccountHolder is a sub-struct, used in PayPal
type PayPalAccountHolder struct {
	PayPalID   string `json:"paypal_id"`
	PayPalPMID string `json:"paypal_pm_id"`
}

// MerchantAccountID is a sub-struct, used in PayPal
type MerchantAccountID struct {
	MerchantAccountID string `json:"merchant_account_id"`
}

// PayPalCorrelationID is a sub-struct, used in PayPal
type PayPalCorrelationID struct {
	PayPalCorrelationID string `json:"paypal_correlation_id"`
}

// PayPal is a sub-struct, used in AccountStruct
type PayPal struct {
	AccountHolder PayPalAccountHolder `json:"account_holder"`
	Merchant      MerchantAccountID   `json:"merchant"`
	Metadata      PayPalCorrelationID `json:"metadata"`
}

// Owner is a sub-struct, used in LedgerAccount
type Owner struct {
	ID       string `json:"id"`
	UUID     string `json:"uuid"`
	UserUUID string `json:"user_uuid"`
	Type     string `json:"type"`
}

// LedgerAccount is a sub-struct, used in AccountStruct
type LedgerAccount struct {
	AccountID string `json:"account_id"`
	Currency  string `json:"currency"`
	Owner     Owner  `json:"owner"`
}

// PaymentMethodID is a sub-struct, used in AccountStruct
type PaymentMethodID struct {
	PaymentMethodID string `json:"payment_method_id"`
}

// ProAccount is a sub-struct, used in AccountStruct
type ProAccount struct {
	AccountID         string `json:"account_id"`
	CoinbaseAccountID string `json:"coinbase_account_id"`
	UserID            string `json:"user_id"`
	Currency          string `json:"currency"`
	PortfolioID       string `json:"portfolio_id"`
}

// RTPAccountHolder is a sub-struct, used in RTP
type RTPAccountHolder struct {
	LegalName  string `json:"legal_name"`
	Identifier string `json:"identifier"`
}

// RoutingNumber is a sub-struct, used in RTP
type RoutingNumber struct {
	RoutingNumber string `json:"routing_number"`
}

// RTP is a sub-struct, used in AccountStruct
type RTP struct {
	AccountHolder RTPAccountHolder `json:"account_holder"`
	Institution   RoutingNumber    `json:"institution"`
}

// LedgerNamedAccount is a sub-struct, used in AccountStruct
type LedgerNamedAccount struct {
	Name           string `json:"name"`
	Currency       string `json:"currency"`
	ForeignNetwork string `json:"foreign_network"`
}

// CustodialPool is a sub-struct, used in AccountStruct
type CustodialPool struct {
	Name    string `json:"name"`
	Network string `json:"network"`
	FiatID  string `json:"fiat_id"`
}

// NonceWithCorrelation is a sub-struct, used in ApplePay and GooglePay
type NonceWithCorrelation struct {
	Nonce         string `json:"nonce"`
	CorrelationID string `json:"correlation_id"`
}

// ApplePay is a sub-struct, used in AccountStruct
type ApplePay struct {
	BrainTree      NonceWithCorrelation `json:"brain_tree"`
	ApplePay       NonceWithCorrelation `json:"apple_pay"`
	UserID         string               `json:"user_id"`
	PostalCode     string               `json:"postal_code"`
	CustomerName   string               `json:"customer_name"`
	Address        FullAddress          `json:"address"`
	SixDigitBin    string               `json:"six_digit_bin"`
	LastFour       string               `json:"last_four"`
	IssuingCountry string               `json:"issuing_country"`
	IssuingBank    string               `json:"issuing_bank"`
	ProductID      string               `json:"product_id"`
	Scheme         string               `json:"scheme"`
	Prepaid        string               `json:"prepaid"`
	Debit          string               `json:"debit"`
}

// UserUUIDWithCurrency is a sub-struct, used in AccountStruct
type UserUUIDWithCurrency struct {
	UserUUID string `json:"user_uuid"`
	Currency string `json:"currency"`
}

// RemitlyAccountHolder is a sub-struct, used in Remitly
type RemitlyAccountHolder struct {
	RecipientID      string `json:"recipient_id"`
	PayoutMethodType string `json:"payout_method_type"`
}

// Remitly is a sub-struct, used in AccountStruct
type Remitly struct {
	AccountHolder RemitlyAccountHolder `json:"account_holder"`
}

// UserIDWithCurrency is a sub-struct, used in AccountStruct
type UserIDWithCurrency struct {
	UserID   string `json:"user_id"`
	Currency string `json:"currency"`
}

// DAPP is a sub-struct, used in AccountStruct
type DAPP struct {
	UserUUID       string `json:"user_uuid"`
	Network        string `json:"network"`
	CohortID       string `json:"cohort_id"`
	SigningBackend string `json:"signing_backend"`
	Currency       string `json:"currency"`
}

// GooglePay is a sub-struct, used in AccountStruct
type GooglePay struct {
	BrainTree      NonceWithCorrelation `json:"brain_tree"`
	GooglePay      NonceWithCorrelation `json:"google_pay"`
	UserID         string               `json:"user_id"`
	PostalCode     string               `json:"postal_code"`
	CustomerName   string               `json:"customer_name"`
	Address        FullAddress          `json:"address"`
	SixDigitBin    string               `json:"six_digit_bin"`
	LastFour       string               `json:"last_four"`
	IssuingCountry string               `json:"issuing_country"`
	ProductID      string               `json:"product_id"`
	Scheme         string               `json:"scheme"`
	Prepaid        string               `json:"prepaid"`
	Debit          string               `json:"debit"`
}

// DAPPBlockchain is a sub-struct, used in AccountStruct
type DAPPBlockchain struct {
	Network  string `json:"network"`
	Address  string `json:"address"`
	CohortID string `json:"cohort_id"`
	UserUUID string `json:"user_uuid"`
	Pool     string `json:"pool"`
}

// PhoneNumber is a sub-struct, used in AccountStruct and BancomatPay
type PhoneNumber struct {
	PhoneNumber string `json:"phone_number"`
}

// DenebUPI is a sub-struct, used in AccountStruct
type DenebUPI struct {
	VPAID             string      `json:"vpa_id"`
	CustomerFirstName string      `json:"customer_first_name"`
	CustomerLastName  string      `json:"customer_last_name"`
	Email             string      `json:"email"`
	PhoneNumber       PhoneNumber `json:"phone_number"`
	PAN               string      `json:"pan"`
	Address           FullAddress `json:"address"`
}

// BankAccount is a sub-struct, used in AccountStruct
type BankAccount struct {
	CustomerAccountType   string `json:"customer_account_type"`
	CustomerAccountNumber string `json:"customer_account_number"`
	CustomerRoutingNumber string `json:"customer_routing_number"`
	CustomerName          string `json:"customer_name"`
}

// NetworkWithAddress is a sub-struct, used in AccountStruct
type NetworkWithAddress struct {
	Network string `json:"network"`
	Address string `json:"address"`
}

// DenebIMPS is a sub-struct, used in AccountStruct
type DenebIMPS struct {
	IFSCCode          string      `json:"ifsc_code"`
	AccountNumber     string      `json:"account_number"`
	CustomerFirstName string      `json:"customer_first_name"`
	CustomerLastName  string      `json:"customer_last_name"`
	Email             string      `json:"email"`
	PhoneNumber       PhoneNumber `json:"phone_number"`
	PAN               string      `json:"pan"`
	Address           FullAddress `json:"address"`
}

// Movements is a sub-struct, used in Legs
type Movements struct {
	ID                 string             `json:"id"`
	SourceAccount      LedgerAccount      `json:"source_account"`
	DestinationAccount LedgerAccount      `json:"destination_account"`
	Amount             AmountWithCurrency `json:"amount"`
}

// Legs is a sub-struct, used in Allocation
type Legs struct {
	ID        string      `json:"id"`
	Movements []Movements `json:"movements"`
	IsNetted  bool        `json:"is_netted"`
}

// Allocation is a sub-struct, used in AccountStruct
type Allocation struct {
	ID       string `json:"id"`
	Legs     []Legs `json:"legs"`
	IsNetted bool   `json:"is_netted"`
}

// LiquidityPool is a sub-struct, used in AccountStruct
type LiquidityPool struct {
	Network     string `json:"network"`
	Pool        string `json:"pool"`
	Currency    string `json:"currency"`
	AccountID   string `json:"account_id"`
	FromAddress string `json:"from_address"`
}

// DirectDeposit is a sub-struct, used in AccountStruct
type DirectDeposit struct {
	DirectDepositAccount string `json:"direct_deposit_account"`
}

// NameAndIBAN is a sub-struct, used in SEPAV2
type NameAndIBAN struct {
	LegalName string `json:"legal_name"`
	IBAN      string `json:"iban"`
}

// SEPAV2 is a sub-struct, used in AccountStruct
type SEPAV2 struct {
	Account             NameAndIBAN `json:"account"`
	CustomerFirstName   string      `json:"customer_first_name"`
	CustomerLastName    string      `json:"customer_last_name"`
	Email               string      `json:"email"`
	PhoneNumber         PhoneNumber `json:"phone_number"`
	CustomerCountry     string      `json:"customer_country"`
	Address             FullAddress `json:"address"`
	SupportsOpenBanking bool        `json:"supports_open_banking"`
}

// ZeptoAccount is a sub-struct, used in Zepto
type ZeptoAccount struct {
	ContactID     string `json:"contact_id"`
	BankAccountID string `json:"bank_account_id"`
}

// Zepto is a sub-struct, used in AccountStruct
type Zepto struct {
	Account ZeptoAccount `json:"account"`
}

// TransactionWithAccount is a sub-struct, used in PixEBANX
type TransactionWithAccount struct {
	TransactionID string `json:"transaction_id"`
	AccountID     string `json:"account_id"`
}

// PixWithdrawal is a sub-struct, used in PixEBANX
type PixWithdrawal struct {
	AccountNumber string `json:"account_number"`
	AccountType   string `json:"account_type"`
	BankCode      string `json:"bank_code"`
	BranchNumber  string `json:"branch_number"`
	PixKey        string `json:"pix_key"`
}

// PixEBANX is a sub-struct, used in AccountStruct
type PixEBANX struct {
	PaymentMethodID string                 `json:"payment_method_id"`
	UserUUID        string                 `json:"user_uuid"`
	Deposit         TransactionWithAccount `json:"deposit"`
	Withdrawal      PixWithdrawal          `json:"withdrawal"`
}

// Signet is a sub-struct, used in AccountStruct
type Signet struct {
	SignetWalletID string `json:"signet_wallet_id"`
}

// Settlement is a sub-struct, used in DerivativeSettlement
type Settlement struct {
	Amount                   AmountWithCurrency `json:"amount"`
	SourceLedgerAccount      LedgerAccount      `json:"source_ledger_account"`
	SourceLedgerNamedAccount LedgerNamedAccount `json:"source_ledger_named_account"`
	TargetLedgerAccount      LedgerAccount      `json:"target_ledger_account"`
	TargetLedgerNamedAccount LedgerNamedAccount `json:"target_ledger_named_account"`
	HoldIDToReplace          string             `json:"hold_id_to_replace"`
	NewHoldID                string             `json:"new_hold_id"`
	NewHoldAmount            AmountWithCurrency `json:"new_hold_amount"`
	ExistingHoldID           string             `json:"existing_hold_id"`
}

// EquityReset is a sub-struct, used in DerivativeSettlement
type EquityReset struct {
	Amount        AmountWithCurrency `json:"amount"`
	EquityAccount LedgerAccount      `json:"equity_account"`
}

// DerivativeSettlement is a sub-struct, used in AccountStruct
type DerivativeSettlement struct {
	AccountSettlements []Settlement `json:"account_settlements"`
	EquityReset        EquityReset  `json:"equity_reset"`
}

// UserUUID is a sub-struct, used in AccountStruct
type UserUUID struct {
	UserUUID string `json:"user_uuid"`
}

// NameWithAccount is a sub-struct, used in SgFAST
type NameWithAccount struct {
	CustomerName  string `json:"customer_name"`
	AccountNumber string `json:"account_number"`
}

// BankCode is a sub-struct, used in SgFAST
type BankCode struct {
	BankCode string `json:"bank_code"`
}

// SgFAST is a sub-struct, used in AccountStruct
type SgFAST struct {
	Account     NameWithAccount `json:"account"`
	Institution BankCode        `json:"institution"`
}

// InteracAccount is a sub-struct, used in Interac
type InteracAccount struct {
	AccountName       string `json:"account_name"`
	InstitutionNumber string `json:"institution_number"`
	TransitNumber     string `json:"transit_number"`
	AccountNumber     string `json:"account_number"`
}

// Interac is a sub-struct, used in AccountStruct
type Interac struct {
	PmsvcID string         `json:"pmsvc_id"`
	Account InteracAccount `json:"account"`
}

// IntraBank is a sub-struct, used in AccountStruct
type IntraBank struct {
	Currency      string `json:"currency"`
	AccountNumber string `json:"account_number"`
	RoutingNumber string `json:"routing_number"`
	CustomerName  string `json:"customer_name"`
	FiatID        string `json:"fiat_id"`
}

// Cbit is a sub-struct, used in AccountStruct
type Cbit struct {
	CbitWalletAddress      string `json:"cbit_wallet_address"`
	CusomtersBankAccountID string `json:"customers_bank_account_id"`
}

// CustomerPaymentInfo is a sub-struct, used in AccountStruct
type CustomerPaymentInfo struct {
	Currency            string `json:"currency"`
	IBAN                string `json:"iban"`
	BIC                 string `json:"bic"`
	BankName            string `json:"bank_name"`
	CustomerPaymentName string `json:"customer_payment_name"`
	CustomerCountryCode string `json:"customer_country_code"`
}

// SgPayNow is a sub-struct, used in AccountStruct
type SgPayNow struct {
	IdentifierType string `json:"identifier_type"`
	Identifier     string `json:"identifier"`
	CustomerName   string `json:"customer_name"`
}

// PaymentLink is a sub-struct, used in AccountStruct
type PaymentLink struct {
	PaymentLinkID string `json:"payment_link_id"`
}

// StringValue is a sub-struct, used in AccountStruct
type StringValue struct {
	Value string `json:"value"`
}

// VendorPayment is a sub-struct, used in AccountStruct
type VendorPayment struct {
	VendorName      string `json:"vendor_name"`
	VendorPaymentID string `json:"vendor_payment_id"`
}

// IDString is a sub-struct, used in AccountStruct
type IDString struct {
	ID string `json:"id"`
}

// BancomatPay is a sub-struct, used in AccountStruct
type BancomatPay struct {
	CustomerName string      `json:"customer_name"`
	Account      PhoneNumber `json:"account"`
}

// NovaAccount is a sub-struct, used in AccountStruct
type NovaAccount struct {
	Network               string `json:"network"`
	NovaAccountID         string `json:"nova_account_id"`
	PoolName              string `json:"pool_name"`
	AccountIdempotencyKey string `json:"account_idempotency_key"`
}

// IdempotencyString is a sub-struct, used in AccountStruct
type IdempotencyString struct {
	Idempotency string `json:"idem"`
}

// EFTAccount is a sub-struct, used in EFT
type EFTAccount struct {
	AccountName        string `json:"account_name"`
	AccountPhoneNumber string `json:"account_phone_number"`
	AccountEmail       string `json:"account_email"`
	InstitutionNumber  string `json:"institution_number"`
	TransitNumber      string `json:"transit_number"`
	AccountNumber      string `json:"account_number"`
}

// EFT is a sub-struct, used in AccountStruct
type EFT struct {
	Account EFTAccount `json:"account"`
}

// WallaceAccount is a sub-struct, used in AccountStruct
type WallaceAccount struct {
	WallaceAccountID string `json:"wallace_account_id"`
	PoolName         string `json:"pool_name"`
}

// ManualSettlement is a sub-struct, used in AccountStruct
type ManualSettlement struct {
	SettlementBankName      string `json:"settlement_bank_name"`
	SettlementAccountNumber string `json:"settlement_account_number"`
	Reference               string `json:"reference"`
}

// TaxWithCBU is a sub-struct, used in AccountStruct
type TaxWithCBU struct {
	TaxID string `json:"tax_id"`
	CBU   string `json:"cbu"`
}

// CurrencyString is a sub-struct, used in AccountStruct
type CurrencyString struct {
	Currency string `json:"currency"`
}

// BankingCircleNow is a sub-struct, used in AccountStruct
type BankingCircleNow struct {
	IBAN                string `json:"iban"`
	Currency            string `json:"currency"`
	CustomerPaymentName string `json:"customer_payment_name"`
}

// Trustly is a sub-struct, used in AccountStruct
type Trustly struct {
	Country              string `json:"country"`
	IBAN                 string `json:"iban"`
	AccountHolder        string `json:"account_holder"`
	BankCode             string `json:"bank_code"`
	AccountNumber        string `json:"account_number"`
	PartialAccountNumber string `json:"partial_account_number"`
	BankName             string `json:"bank_name"`
	Email                string `json:"email"`
}

// Blik is a sub-struct, used in AccountStruct
type Blik struct {
	Email         string `json:"email"`
	Country       string `json:"country"`
	AccountHolder string `json:"account_holder"`
}

// PIX is a sub-struct, used in AccountStruct
type PIX struct {
	AccountNumber       string `json:"account_number"`
	AccountType         string `json:"account_type"`
	BankCode            string `json:"bank_code"`
	BankName            string `json:"bank_name"`
	BranchNumber        string `json:"branch_number"`
	CustomerPaymentName string `json:"customer_payment_name"`
	SenderDocument      string `json:"sender_document"`
	PIXKey              string `json:"pix_key"`
}

// AccountStruct is a sub-struct storing information on accounts, used in ConvertResponse
type AccountStruct struct {
	Type                        string               `json:"type"`
	Network                     string               `json:"network"`
	PaymentMethodID             string               `json:"payment_method_id"`
	BlockchainAddress           AddressInfo          `json:"blockchain_address"`
	CoinbaseAccount             AccountID            `json:"coinbase_account"`
	BlockchainTransaction       HashWithHeight       `json:"blockchain_transaction"`
	Fedwire                     Fedwire              `json:"fedwire"`
	Swift                       Swift                `json:"swift"`
	Card                        CardInfo             `json:"card"`
	Zengin                      Zengin               `json:"zengin"`
	UK                          UKAccount            `json:"uk"`
	SEPA                        SEPA                 `json:"sepa"`
	PayPal                      PayPal               `json:"paypal"`
	LedgerAccount               LedgerAccount        `json:"ledger_account"`
	ExternalPaymentMethod       PaymentMethodID      `json:"external_payment_method"`
	ProAccount                  ProAccount           `json:"pro_account"`
	RTP                         RTP                  `json:"rtp"`
	Venue                       Name                 `json:"venue"`
	LedgerNamedAccount          LedgerNamedAccount   `json:"ledger_named_account"`
	CustodialPool               CustodialPool        `json:"custodial_pool"`
	ApplePay                    ApplePay             `json:"apple_pay"`
	DefaultAccount              UserUUIDWithCurrency `json:"default_account"`
	Remitly                     Remitly              `json:"remitly"`
	ProInternalAccount          UserIDWithCurrency   `json:"pro_internal_account"`
	DAPPWalletAccount           DAPP                 `json:"dapp_wallet_account"`
	GooglePay                   GooglePay            `json:"google_pay"`
	DAPPWalletBlockchainAddress DAPPBlockchain       `json:"dapp_wallet_blockchain_address"`
	ZaakpayMobikwik             PhoneNumber          `json:"zaakpay_mobikwik"`
	DenebUPI                    DenebUPI             `json:"deneb_upi"`
	BankAccount                 BankAccount          `json:"bank_account"`
	IdentityContractCall        NetworkWithAddress   `json:"identity_contract_call"`
	DenebIMPS                   DenebIMPS            `json:"deneb_imps"`
	Allocation                  Allocation           `json:"allocation"`
	LiquidityPool               LiquidityPool        `json:"liquidity_pool"`
	ZenginV2                    Zengin               `json:"zengin_v2"`
	DirectDeposit               DirectDeposit        `json:"direct_deposit"`
	SEPAV2                      SEPAV2               `json:"sepa_v2"`
	Zepto                       Zepto                `json:"zepto"`
	PixEBANX                    PixEBANX             `json:"pix_ebanx"`
	Signet                      Signet               `json:"signet"`
	DerivativeSettlement        DerivativeSettlement `json:"derivative_settlement"`
	User                        UserUUID             `json:"user"`
	SgFAST                      SgFAST               `json:"sg_fast"`
	Interac                     Interac              `json:"interac"`
	IntraBank                   IntraBank            `json:"intra_bank"`
	Cbit                        Cbit                 `json:"cbit"`
	Ideal                       CustomerPaymentInfo  `json:"ideal"`
	Sofort                      CustomerPaymentInfo  `json:"sofort"`
	SgPayNow                    SgPayNow             `json:"sg_paynow"`
	CheckoutPaymentLink         PaymentLink          `json:"checkout_payment_link"`
	EmailAddress                StringValue          `json:"email_address"`
	PhoneNumber                 StringValue          `json:"phone_number"`
	VendorPayment               VendorPayment        `json:"vendor_payment"`
	CTN                         IDString             `json:"ctn"`
	BancomatPay                 BancomatPay          `json:"bancomat_pay"`
	HotWallet                   NetworkWithAddress   `json:"hot_wallet"`
	NovaAccount                 NovaAccount          `json:"nova_account"`
	MagicSpendBlockchainAddress AddressInfo          `json:"magic_spend_blockchain_address"`
	TransferPointer             IdempotencyString    `json:"transfer_pointer"`
	EFT                         EFT                  `json:"eft"`
	WallaceAccount              WallaceAccount       `json:"wallace_account"`
	Manual                      ManualSettlement     `json:"manual"`
	ArgentineBankAccount        TaxWithCBU           `json:"argentine_bank_account"`
	Representment               CurrencyString       `json:"representment"`
	BankingCircleNow            BankingCircleNow     `json:"banking_circle_now"`
	Trustly                     Trustly              `json:"trustly"`
	Blik                        Blik                 `json:"blik"`
	MBWay                       json.RawMessage      `json:"mbway"`
	PIX                         PIX                  `json:"pix"`
}

// AmScale is a sub-struct storing information on amounts and scales, used in ConvertResponse
type AmScale struct {
	Amount ValueWithCurrency `json:"amount"`
	Scale  int32             `json:"scale"`
}

// UnitPrice is a sub-struct used in ConvertResponse
type UnitPrice struct {
	TargetToFiat   AmScale `json:"target_to_fiat"`
	TargetToSource AmScale `json:"target_to_source"`
	SourceToFiat   AmScale `json:"source_to_fiat"`
}

// Context is a sub-struct used in UserWarnings
type Context struct {
	Details  []string `json:"details"`
	Title    string   `json:"title"`
	LinkText string   `json:"link_text"`
}

// UserWarnings is a sub-struct used in ConvertResponse
type UserWarnings struct {
	ID      string     `json:"id"`
	Link    LinkStruct `json:"link"`
	Context Context    `json:"context"`
	Code    string     `json:"code"`
	Message string     `json:"message"`
}

// ErrorMetadata is a sub-struct used in CancellationReason
type ErrorMetadata struct {
	LimitAmount ValueWithCurrency `json:"limit_amount"`
}

// CancellationReason is a sub-struct used in ConvertResponse
type CancellationReason struct {
	Message       string        `json:"message"`
	Code          string        `json:"code"`
	ErrorCode     string        `json:"error_code"`
	ErrorCTA      string        `json:"error_cta"`
	ErrorMetadata ErrorMetadata `json:"error_metadata"`
	Title         string        `json:"title"`
}

// SubscriptionInfo is a sub-struct used in ConvertResponse
type SubscriptionInfo struct {
	FreeTradingResetDate                       time.Time         `json:"free_trading_reset_date"`
	UsedZeroFeeTrading                         ValueWithCurrency `json:"used_zero_fee_trading"`
	RemainingFreeTradingVolume                 ValueWithCurrency `json:"remaining_free_trading_volume"`
	MaxFreeTradingVolume                       ValueWithCurrency `json:"max_free_trading_volume"`
	HasBenefitCap                              bool              `json:"has_benefit_cap"`
	AppliedSubscriptionBenefit                 bool              `json:"applied_subscription_benefit"`
	FeeWithoutSubscriptionBenefit              ValueWithCurrency `json:"fee_without_subscription_benefit"`
	PaymentMethodFeeWithoutSubscriptionBenefit ValueWithCurrency `json:"payment_method_fee_without_subscription_benefit"`
}

// TaxDetails is a sub-struct used in ConvertResponse
type TaxDetails struct {
	Name   string            `json:"name"`
	Amount ValueWithCurrency `json:"amount"`
}

// TradeIncentiveInfo is a sub-struct used in ConvertResponse
type TradeIncentiveInfo struct {
	AppliedIncentive    bool              `json:"applied_incentive"`
	UserIncentiveID     string            `json:"user_incentive_id"`
	CodeVal             string            `json:"code_val"`
	EndsAt              time.Time         `json:"ends_at"`
	FeeWithoutIncentive ValueWithCurrency `json:"fee_without_incentive"`
	Redeemed            bool              `json:"redeemed"`
}

// ConvertResponse contains information on a convert trade, returned by CreateConvertQuote, CommitConvertTrade, and GetConvertTradeByID
type ConvertResponse struct {
	ID                 string             `json:"id"`
	Status             string             `json:"status"`
	UserEnteredAmount  ValueWithCurrency  `json:"user_entered_amount"`
	Amount             ValueWithCurrency  `json:"amount"`
	Subtotal           ValueWithCurrency  `json:"subtotal"`
	Total              ValueWithCurrency  `json:"total"`
	Fees               []FeeStruct        `json:"fees"`
	TotalFee           FeeStruct          `json:"total_fee"`
	Source             AccountStruct      `json:"source"`
	Target             AccountStruct      `json:"target"`
	UnitPrice          UnitPrice          `json:"unit_price"`
	UserWarnings       []UserWarnings     `json:"user_warnings"`
	UserReference      string             `json:"user_reference"`
	SourceCurrency     string             `json:"source_currency"`
	TargetCurrency     string             `json:"target_currency"`
	CancellationReason CancellationReason `json:"cancellation_reason"`
	SourceID           string             `json:"source_id"`
	TargetID           string             `json:"target_id"`
	SubscriptionInfo   SubscriptionInfo   `json:"subscription_info"`
	ExchangeRate       ValueWithCurrency  `json:"exchange_rate"`
	TaxDetails         []TaxDetails       `json:"tax_details"`
	TradeIncentiveInfo TradeIncentiveInfo `json:"trade_incentive_info"`
	TotalFeeWithoutTax FeeStruct          `json:"total_fee_without_tax"`
	FiatDenotedTotal   ValueWithCurrency  `json:"fiat_denoted_total"`
}

// ServerTimeV3 holds information on the server's time, returned by GetV3Time
type ServerTimeV3 struct {
	Iso               time.Time  `json:"iso"`
	EpochSeconds      types.Time `json:"epochSeconds"`
	EpochMilliseconds types.Time `json:"epochMillis"`
}

// PaymentMethodData is a sub-type that holds information on a payment method
type PaymentMethodData struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Name          string    `json:"name"`
	Currency      string    `json:"currency"`
	Verified      bool      `json:"verified"`
	AllowBuy      bool      `json:"allow_buy"`
	AllowSell     bool      `json:"allow_sell"`
	AllowDeposit  bool      `json:"allow_deposit"`
	AllowWithdraw bool      `json:"allow_withdraw"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// IDResource holds an ID, resource type, and associated data, used in ListNotificationsResponse, TransactionData, DeposWithdrData, and PaymentMethodData
type IDResource struct {
	ID           string `json:"id"`
	Resource     string `json:"resource"`
	ResourcePath string `json:"resource_path"`
	Email        string `json:"email"`
}

// PaginationResp holds pagination information, used in ListNotificationsResponse, GetAllWalletsResponse, GetAllAddrResponse, ManyTransactionsResp, and ManyDeposWithdrResp
type PaginationResp struct {
	EndingBefore         string `json:"ending_before"`
	StartingAfter        string `json:"starting_after"`
	PreviousEndingBefore string `json:"previous_ending_before"`
	NextStartingAfter    string `json:"next_starting_after"`
	Limit                uint8  `json:"limit"`
	Order                string `json:"order"`
	PreviousURI          string `json:"previous_uri"`
	NextURI              string `json:"next_uri"`
}

// PaginationInp holds information needed to engage in pagination with Sign in With Coinbase. Used in ListNotifications, GetAllWallets, GetAllAddresses, GetAddressTransactions, GetAllTransactions, GetAllFiatTransfers, ListPaymentMethods, and preparePagination
type PaginationInp struct {
	Limit         uint8
	OrderAscend   bool
	StartingAfter string
	EndingBefore  string
}

// AmountWithCurrency is a sub-struct used in ListNotificationsSubData, WalletData, TransactionData, DeposWithdrData, Settlement, EquityReset, and PaymentMethodData
type AmountWithCurrency struct {
	Amount   float64 `json:"amount,string"`
	Currency string  `json:"currency"`
}

// Fees is a sub-struct used in ListNotificationsSubData
type Fees []struct {
	Type   string             `json:"type"`
	Amount AmountWithCurrency `json:"amount"`
}

// ListNotificationsSubData is a sub-struct used in ListNotificationsData
type ListNotificationsSubData struct {
	ID            string             `json:"id"`
	Address       string             `json:"address"`
	Name          string             `json:"name"`
	Status        string             `json:"status"`
	PaymentMethod IDResource         `json:"payment_method"`
	Transaction   IDResource         `json:"transaction"`
	Amount        AmountWithCurrency `json:"amount"`
	Total         AmountWithCurrency `json:"total"`
	Subtotal      AmountWithCurrency `json:"subtotal"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	Resource      string             `json:"resource"`
	ResourcePath  string             `json:"resource_path"`
	Committed     bool               `json:"committed"`
	Instant       bool               `json:"instant"`
	Fee           AmountWithCurrency `json:"fee"`
	Fees          []Fees             `json:"fees"`
	PayoutAt      time.Time          `json:"payout_at"`
}

// AdditionalData is a sub-struct used in ListNotificationsData
type AdditionalData struct {
	Hash   string             `json:"hash"`
	Amount AmountWithCurrency `json:"amount"`
}

// ListNotificationsData is a sub-struct used in ListNotificationsResponse
type ListNotificationsData struct {
	ID               string                   `json:"id"`
	Type             string                   `json:"type"`
	Data             ListNotificationsSubData `json:"data"`
	AdditionalData   AdditionalData           `json:"additional_data"`
	User             IDResource               `json:"user"`
	Account          IDResource               `json:"account"`
	DeliveryAttempts int32                    `json:"delivery_attempts"`
	CreatedAt        time.Time                `json:"created_at"`
	Resource         string                   `json:"resource"`
	ResourcePath     string                   `json:"resource_path"`
	Transaction      IDResource               `json:"transaction"`
}

// ListNotificationsResponse holds information on notifications that the user is subscribed to. Returned by ListNotifications
type ListNotificationsResponse struct {
	Pagination PaginationResp          `json:"pagination"`
	Data       []ListNotificationsData `json:"data"`
}

// CodeName is a sub-struct holding a code and a name, used in UserResponse
type CodeName struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// Country is a sub-struct, used in UserResponse
type Country struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	IsInEurope bool   `json:"is_in_europe"`
}

// Tiers is a sub-struct, used in UserResponse
type Tiers struct {
	CompletedDescription string `json:"completed_description"`
	UpgradeButtonText    string `json:"upgrade_button_text"`
	Header               string `json:"header"`
	Body                 string `json:"body"`
}

// ReferralMoney is a sub-struct, used in UserResponse
type ReferralMoney struct {
	Amount            float64 `json:"amount,string"`
	Currency          string  `json:"currency"`
	CurrencySymbol    string  `json:"currency_symbol"`
	ReferralThreshold float64 `json:"referral_threshold,string"`
}

// UserResponse holds information on a user, returned by GetCurrentUser
type UserResponse struct {
	ID                                    string        `json:"id"`
	Name                                  string        `json:"name"`
	Username                              string        `json:"username"`
	ProfileLocation                       string        `json:"profile_location"`
	ProfileBio                            string        `json:"profile_bio"`
	ProfileURL                            string        `json:"profile_url"`
	AvatarURL                             string        `json:"avatar_url"`
	Resource                              string        `json:"resource"`
	ResourcePath                          string        `json:"resource_path"`
	LegacyID                              string        `json:"legacy_id"`
	TimeZone                              string        `json:"time_zone"`
	NativeCurrency                        string        `json:"native_currency"`
	BitcoinUnit                           string        `json:"bitcoin_unit"`
	State                                 string        `json:"state"`
	Country                               Country       `json:"country"`
	Nationality                           CodeName      `json:"nationality"`
	RegionSupportsFiatTransfers           bool          `json:"region_supports_fiat_transfers"`
	RegionSupportsCryptoToCryptoTransfers bool          `json:"region_supports_crypto_to_crypto_transfers"`
	CreatedAt                             time.Time     `json:"created_at"`
	SupportsRewards                       bool          `json:"supports_rewards"`
	Tiers                                 Tiers         `json:"tiers"`
	ReferralMoney                         ReferralMoney `json:"referral_money"`
	HasBlockingBuyRestrictions            bool          `json:"has_blocking_buy_restrictions"`
	HasMadeAPurchase                      bool          `json:"has_made_a_purchase"`
	HasBuyDepositPaymentMethods           bool          `json:"has_buy_deposit_payment_methods"`
	HasUnverifiedBuyDepositPaymentMethods bool          `json:"has_unverified_buy_deposit_payment_methods"`
	NeedsKYCRemediation                   bool          `json:"needs_kyc_remediation"`
	ShowInstantAchUx                      bool          `json:"show_instant_ach_ux"`
	UserType                              string        `json:"user_type"`
	Email                                 string        `json:"email"`
	SendsDisabled                         bool          `json:"sends_disabled"`
}

// Currency is a sub-struct holding information on a currency, used in WalletData
type Currency struct {
	Code                string `json:"code"`
	Name                string `json:"name"`
	Color               string `json:"color"`
	SortIndex           int32  `json:"sort_index"`
	Exponent            int32  `json:"exponent"`
	Type                string `json:"type"`
	AddressRegex        string `json:"address_regex"`
	AssetID             string `json:"asset_id"`
	DestinationTagName  string `json:"destination_tag_name"`
	DestinationTagRegex string `json:"destination_tag_regex"`
	Slug                string `json:"slug"`
	Rewards             any    `json:"rewards"`
}

// WalletData is a sub-struct holding wallet information, used in GetAllWalletsResponse
type WalletData struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Primary          bool               `json:"primary"`
	Type             string             `json:"type"`
	Currency         Currency           `json:"currency"`
	Balance          AmountWithCurrency `json:"balance"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
	Resource         string             `json:"resource"`
	ResourcePath     string             `json:"resource_path"`
	AllowDeposits    bool               `json:"allow_deposits"`
	AllowWithdrawals bool               `json:"allow_withdrawals"`
}

// GetAllWalletsResponse holds information on many wallets, returned by GetAllWallets
type GetAllWalletsResponse struct {
	Pagination *PaginationResp `json:"pagination"`
	Data       []WalletData    `json:"data"`
}

// AddressInfo holds an address and a destination tag, used in AddressData and AccountStruct
type AddressInfo struct {
	Address        string `json:"address"`
	DestinationTag string `json:"destination_tag"`
}

// TitleSubtitle holds a title and a subtitle, used in AddressData and TransactionData
type TitleSubtitle struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

// Options is a sub-struct used in Warnings
type Options struct {
	Text  string `json:"text"`
	Style string `json:"style"`
	ID    string `json:"id"`
}

// Warnings is a sub-struct used in AddressData
type Warnings struct {
	Type     string    `json:"type"`
	Title    string    `json:"title"`
	Details  string    `json:"details"`
	ImageURL string    `json:"image_url"`
	Options  []Options `json:"options"`
}

// ShareAddressCopy is a sub-struct used in AddressData
type ShareAddressCopy struct {
	Line1 string `json:"line1"`
	Line2 string `json:"line2"`
}

// InlineWarning is a sub-struct used in AddressData
type InlineWarning struct {
	Text    string        `json:"text"`
	Tooltip TitleSubtitle `json:"tooltip"`
}

// AddressData holds address information, used in GetAllAddrResponse
type AddressData struct {
	ID               string           `json:"id"`
	Address          string           `json:"address"`
	AddressInfo      AddressInfo      `json:"address_info"`
	Name             string           `json:"name"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	Network          string           `json:"network"`
	URIScheme        string           `json:"uri_scheme"`
	Resource         string           `json:"resource"`
	ResourcePath     string           `json:"resource_path"`
	Warnings         []Warnings       `json:"warnings"`
	QRCodeImageURL   string           `json:"qr_code_image_url"`
	AddressLabel     string           `json:"address_label"`
	DefaultReceive   bool             `json:"default_receive"`
	DestinationTag   string           `json:"destination_tag"`
	DepositURI       string           `json:"deposit_uri"`
	CallbackURL      string           `json:"callback_url"`
	ShareAddressCopy ShareAddressCopy `json:"share_address_copy"`
	ReceiveSubtitle  string           `json:"receive_subtitle"`
	InlineWarning    InlineWarning    `json:"inline_warning"`
}

// GetAllAddrResponse holds information on many addresses, returned by GetAllAddresses
type GetAllAddrResponse struct {
	Pagination PaginationResp `json:"pagination"`
	Data       []AddressData  `json:"data"`
}

// AdvancedTradeFill is a sub-struct used in TransactionData
type AdvancedTradeFill struct {
	FillPrice  float64       `json:"fill_price,string"`
	ProductID  currency.Pair `json:"product_id"`
	OrderID    string        `json:"order_id"`
	Commission float64       `json:"commission,string"`
	OrderSide  string        `json:"order_side"`
}

// Network is a sub-struct used in TransactionData
type Network struct {
	Status string `json:"status"`
	Hash   string `json:"hash"`
	Name   string `json:"name"`
}

// TransactionData is a sub-type that holds information on a transaction. Used in ManyTransactionsResp
type TransactionData struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Status       string             `json:"status"`
	Amount       AmountWithCurrency `json:"amount"`
	NativeAmount AmountWithCurrency `json:"native_amount"`
	Description  string             `json:"description"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	Resource     string             `json:"resource"`
	ResourcePath string             `json:"resource_path"`
	Details      TitleSubtitle      `json:"details"`
	Network      Network            `json:"network"`
	To           IDResource         `json:"to"`
	From         IDResource         `json:"from"`
}

// ManyTransactionsResp holds information on many transactions. Returned by GetAddressTransactions and GetAllTransactions
type ManyTransactionsResp struct {
	Pagination PaginationResp    `json:"pagination"`
	Data       []TransactionData `json:"data"`
}

// DeposWithdrData is a sub-type that holds information on a deposit/withdrawal. Used in ManyDeposWithdrResp
type DeposWithdrData struct {
	ID            string             `json:"id"`
	Status        string             `json:"status"`
	PaymentMethod IDResource         `json:"payment_method"`
	Transaction   IDResource         `json:"transaction"`
	Amount        AmountWithCurrency `json:"amount"`
	Subtotal      AmountWithCurrency `json:"subtotal"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	Resource      string             `json:"resource"`
	ResourcePath  string             `json:"resource_path"`
	Committed     bool               `json:"committed"`
	Fee           AmountWithCurrency `json:"fee"`
	PayoutAt      time.Time          `json:"payout_at"`
	TransferType  FiatTransferType   `json:"transfer_type"`
}

// ManyDeposWithdrResp holds information on many deposits. Returned by GetAllFiatTransfers
type ManyDeposWithdrResp struct {
	Pagination PaginationResp    `json:"pagination"`
	Data       []DeposWithdrData `json:"data"`
}

// FiatData holds information on fiat currencies. Returned by GetFiatCurrencies
type FiatData struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	MinSize float64 `json:"min_size,string"`
}

// CryptoData holds information on cryptocurrencies. Returned by GetCryptocurrencies
type CryptoData struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Color        string `json:"color"`
	SortIndex    uint16 `json:"sort_index"`
	Exponent     uint8  `json:"exponent"`
	Type         string `json:"type"`
	AddressRegex string `json:"address_regex"`
	AssetID      string `json:"asset_id"`
}

// GetExchangeRatesResp holds information on exchange rates. Returned by GetExchangeRates
type GetExchangeRatesResp struct {
	Currency string                  `json:"currency"`
	Rates    map[string]types.Number `json:"rates"`
}

// GetPriceResp holds information on a price. Returned by GetPrice
type GetPriceResp struct {
	Amount   float64 `json:"amount,string"`
	Base     string  `json:"base"`
	Currency string  `json:"currency"`
}

// ServerTimeV2 holds current requested server time information, returned by GetV2Time
type ServerTimeV2 struct {
	ISO   time.Time `json:"iso"`
	Epoch uint64    `json:"epoch"`
}

// WebsocketRequest is an aspect of constructing a request to the websocket server, used in sendRequest
type WebsocketRequest struct {
	Type       string          `json:"type"`
	ProductIDs []currency.Pair `json:"product_ids,omitempty"`
	Channel    string          `json:"channel,omitempty"`
	Signature  string          `json:"signature,omitempty"`
	Key        string          `json:"api_key,omitempty"`
	Timestamp  string          `json:"timestamp,omitempty"`
	JWT        string          `json:"jwt,omitempty"`
}

// StandardWebsocketResponse is a standard response from the websocket connection
type StandardWebsocketResponse struct {
	Channel   string          `json:"channel"`
	ClientID  string          `json:"client_id"`
	Timestamp time.Time       `json:"timestamp"`
	Sequence  uint64          `json:"sequence_num"`
	Events    json.RawMessage `json:"events"`
	Error     string          `json:"type"`
}

// WebsocketTicker defines a ticker websocket response, used in WebsocketTickerHolder
type WebsocketTicker struct {
	Type                     string        `json:"type"`
	ProductID                currency.Pair `json:"product_id"`
	Price                    float64       `json:"price,string"`
	Volume24H                float64       `json:"volume_24_h,string"`
	Low24H                   float64       `json:"low_24_h,string"`
	High24H                  float64       `json:"high_24_h,string"`
	Low52W                   float64       `json:"low_52_w,string"`
	High52W                  float64       `json:"high_52_w,string"`
	PricePercentageChange24H float64       `json:"price_percent_chg_24_h,string"`
	BestBid                  float64       `json:"best_bid,string"`
	BestBidQuantity          float64       `json:"best_bid_size,string"`
	BestAsk                  float64       `json:"best_ask,string"`
	BestAskQuantity          float64       `json:"best_ask_size,string"`
}

// WebsocketTickerHolder holds a variety of ticker responses, used when wsHandleData processes tickers
type WebsocketTickerHolder struct {
	Type    string            `json:"type"`
	Tickers []WebsocketTicker `json:"tickers"`
}

// WebsocketCandle defines a candle websocket response, used in WebsocketCandleHolder
type WebsocketCandle struct {
	Start     types.Time    `json:"start"`
	Low       float64       `json:"low,string"`
	High      float64       `json:"high,string"`
	Open      float64       `json:"open,string"`
	Close     float64       `json:"close,string"`
	Volume    float64       `json:"volume,string"`
	ProductID currency.Pair `json:"product_id"`
}

// WebsocketCandleHolder holds a variety of candle responses, used when wsHandleData processes candles
type WebsocketCandleHolder struct {
	Type    string            `json:"type"`
	Candles []WebsocketCandle `json:"candles"`
}

// WebsocketMarketTrade defines a market trade websocket response, used in WebsocketMarketTradeHolder
type WebsocketMarketTrade struct {
	TradeID   string        `json:"trade_id"`
	ProductID currency.Pair `json:"product_id"`
	Price     float64       `json:"price,string"`
	Size      float64       `json:"size,string"`
	Side      order.Side    `json:"side"`
	Time      time.Time     `json:"time"`
}

// WebsocketMarketTradeHolder holds a variety of market trade responses, used when wsHandleData processes trades
type WebsocketMarketTradeHolder struct {
	Type   string                 `json:"type"`
	Trades []WebsocketMarketTrade `json:"trades"`
}

// WebsocketProduct defines a product websocket response, used in WebsocketProductHolder
type WebsocketProduct struct {
	ProductType    string        `json:"product_type"`
	ID             currency.Pair `json:"id"`
	BaseCurrency   string        `json:"base_currency"`
	QuoteCurrency  string        `json:"quote_currency"`
	BaseIncrement  float64       `json:"base_increment,string"`
	QuoteIncrement float64       `json:"quote_increment,string"`
	DisplayName    string        `json:"display_name"`
	Status         string        `json:"status"`
	StatusMessage  string        `json:"status_message"`
	MinMarketFunds float64       `json:"min_market_funds,string"`
}

// WebsocketProductHolder holds a variety of product responses, used when wsHandleData processes an update on a product's status
type WebsocketProductHolder struct {
	Type     string             `json:"type"`
	Products []WebsocketProduct `json:"products"`
}

// WebsocketOrderbookData defines a websocket orderbook response, used in WebsocketOrderbookDataHolder
type WebsocketOrderbookData struct {
	Side        string    `json:"side"`
	EventTime   time.Time `json:"event_time"`
	PriceLevel  float64   `json:"price_level,string"`
	NewQuantity float64   `json:"new_quantity,string"`
}

// WebsocketOrderbookDataHolder holds a variety of orderbook responses, used when wsHandleData processes orderbooks, as well as under typical operation of ProcessSnapshot, ProcessUpdate, and processBidAskArray
type WebsocketOrderbookDataHolder struct {
	Type      string                   `json:"type"`
	ProductID currency.Pair            `json:"product_id"`
	Changes   []WebsocketOrderbookData `json:"updates"`
}

// WebsocketOrderData defines a websocket order response, used in WebsocketOrderDataHolder
type WebsocketOrderData struct {
	AveragePrice          float64       `json:"avg_price,string"`
	CancelReason          string        `json:"cancel_reason"`
	ClientOrderID         string        `json:"client_order_id"`
	CompletionPercentage  float64       `json:"completion_percentage,string"`
	ContractExpiryType    string        `json:"contract_expiry_type"`
	CumulativeQuantity    float64       `json:"cumulative_quantity,string"`
	FilledValue           float64       `json:"filled_value,string"`
	LeavesQuantity        float64       `json:"leaves_quantity,string"`
	LimitPrice            float64       `json:"limit_price,string"`
	NumberOfFills         int64         `json:"number_of_fills"`
	OrderID               string        `json:"order_id"`
	OrderSide             string        `json:"order_side"`
	OrderType             string        `json:"order_type"`
	OutstandingHoldAmount float64       `json:"outstanding_hold_amount,string"`
	PostOnly              bool          `json:"post_only"`
	ProductID             currency.Pair `json:"product_id"`
	ProductType           string        `json:"product_type"`
	RejectReason          string        `json:"reject_reason"`
	RetailPortfolioID     string        `json:"retail_portfolio_id"`
	RiskManagedBy         string        `json:"risk_managed_by"`
	Status                string        `json:"status"`
	StopPrice             float64       `json:"stop_price,string"`
	TimeInForce           string        `json:"time_in_force"`
	TotalFees             float64       `json:"total_fees,string"`
	TotalValueAfterFees   float64       `json:"total_value_after_fees,string"`
	TriggerStatus         string        `json:"trigger_status"`
	CreationTime          time.Time     `json:"creation_time"`
	EndTime               time.Time     `json:"end_time"`
	StartTime             time.Time     `json:"start_time"`
}

// WebsocketPerpData defines a websocket perpetual position response, used in WebsocketPositionStruct
type WebsocketPerpData struct {
	ProductID        currency.Pair `json:"product_id"`
	PortfolioUUID    string        `json:"portfolio_uuid"`
	VWAP             float64       `json:"vwap,string"`
	EntryVWAP        float64       `json:"entry_vwap,string"`
	PositionSide     string        `json:"position_side"`
	MarginType       string        `json:"margin_type"`
	NetSize          float64       `json:"net_size,string"`
	BuyOrderSize     float64       `json:"buy_order_size,string"`
	SellOrderSize    float64       `json:"sell_order_size,string"`
	Leverage         float64       `json:"leverage,string"`
	MarkPrice        float64       `json:"mark_price,string"`
	LiquidationPrice float64       `json:"liquidation_price,string"`
	IMNotional       float64       `json:"im_notional,string"`
	MMNotional       float64       `json:"mm_notional,string"`
	PositionNotional float64       `json:"position_notional,string"`
	UnrealizedPNL    float64       `json:"unrealized_pnl,string"`
	AggregatedPNL    float64       `json:"aggregated_pnl,string"`
}

// WebsocketExpData defines a websocket expiring position response, used in WebsocketPositionStruct
type WebsocketExpData struct {
	ProductID         currency.Pair `json:"product_id"`
	Side              string        `json:"side"`
	NumberOfContracts float64       `json:"number_of_contracts,string"`
	RealizedPNL       float64       `json:"realized_pnl,string"`
	UnrealizedPNL     float64       `json:"unrealized_pnl,string"`
	EntryPrice        float64       `json:"entry_price,string"`
}

// WebsocketPositionStruct holds position data, used in WebsocketOrderDataHolder
type WebsocketPositionStruct struct {
	PerpetualFuturesPositions []WebsocketPerpData `json:"perpetual_futures_positions"`
	ExpiringFuturesPositions  []WebsocketExpData  `json:"expiring_futures_positions"`
}

// WebsocketOrderDataHolder holds a variety of order responses, used when wsHandleData processes orders
type WebsocketOrderDataHolder struct {
	Type      string                  `json:"type"`
	Orders    []WebsocketOrderData    `json:"orders"`
	Positions WebsocketPositionStruct `json:"positions"`
}

// Details is a sub-struct used in CurrencyData
type Details struct {
	Type                  string   `json:"type"`
	Symbol                string   `json:"symbol"`
	NetworkConfirmations  int32    `json:"network_confirmations"`
	SortOrder             int32    `json:"sort_order"`
	CryptoAddressLink     string   `json:"crypto_address_link"`
	CryptoTransactionLink string   `json:"crypto_transaction_link"`
	PushPaymentMethods    []string `json:"push_payment_methods"`
	GroupTypes            []string `json:"group_types"`
	DisplayName           string   `json:"display_name"`
	ProcessingTimeSeconds int64    `json:"processing_time_seconds"`
	MinWithdrawalAmount   float64  `json:"min_withdrawal_amount"`
	MaxWithdrawalAmount   float64  `json:"max_withdrawal_amount"`
}

// SupportedNetworks is a sub-struct used in CurrencyData
type SupportedNetworks struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	Status                string  `json:"status"`
	ContractAddress       string  `json:"contract_address"`
	CryptoAddressLink     string  `json:"crypto_address_link"`
	CryptoTransactionLink string  `json:"crypto_transaction_link"`
	MinWithdrawalAmount   float64 `json:"min_withdrawal_amount"`
	MaxWithdrawalAmount   float64 `json:"max_withdrawal_amount"`
	NetworkConfirmations  int32   `json:"network_confirmations"`
	ProcessingTimeSeconds int64   `json:"processing_time_seconds"`
}

// CurrencyData contains information on known currencies, used in GetAllCurrencies and GetACurrency
type CurrencyData struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	MinSize           string              `json:"min_size"`
	Status            string              `json:"status"`
	Message           string              `json:"message"`
	MaxPrecision      float64             `json:"max_precision,string"`
	ConvertibleTo     []string            `json:"convertible_to"`
	Details           Details             `json:"details"`
	DefaultNetwork    string              `json:"default_network"`
	SupportedNetworks []SupportedNetworks `json:"supported_networks"`
	DisplayName       string              `json:"display_name"`
}

// PairData contains information on available trading pairs, used in GetAllTradingPairs
type PairData struct {
	ID                     string       `json:"id"`
	BaseCurrency           string       `json:"base_currency"`
	QuoteCurrency          string       `json:"quote_currency"`
	QuoteIncrement         float64      `json:"quote_increment,string"`
	BaseIncrement          float64      `json:"base_increment,string"`
	DisplayName            string       `json:"display_name"`
	MinMarketFunds         float64      `json:"min_market_funds,string"`
	MarginEnabled          bool         `json:"margin_enabled"`
	PostOnly               bool         `json:"post_only"`
	LimitOnly              bool         `json:"limit_only"`
	CancelOnly             bool         `json:"cancel_only"`
	Status                 string       `json:"status"`
	StatusMessage          string       `json:"status_message"`
	TradingDisabled        bool         `json:"trading_disabled"`
	FXStablecoin           bool         `json:"fx_stablecoin"`
	MaxSlippagePercentage  float64      `json:"max_slippage_percentage,string"`
	AuctionMode            bool         `json:"auction_mode"`
	HighBidLimitPercentage types.Number `json:"high_bid_limit_percentage"`
}

// PairVolumeData contains information on trading pair volume, used in GetAllPairVolumes
type PairVolumeData struct {
	ID                     string       `json:"id"`
	BaseCurrency           string       `json:"base_currency"`
	QuoteCurrency          string       `json:"quote_currency"`
	DisplayName            string       `json:"display_name"`
	MarketTypes            []string     `json:"market_types"`
	SpotVolume24Hour       types.Number `json:"spot_volume_24hour"`
	SpotVolume30Day        types.Number `json:"spot_volume_30day"`
	RFQVolume24Hour        float64      `json:"rfq_volume_24hour,string"`
	RFQVolume30Day         float64      `json:"rfq_volume_30day,string"`
	ConversionVolume24Hour float64      `json:"conversion_volume_24hour,string"`
	ConversionVolume30Day  float64      `json:"conversion_volume_30day,string"`
}

// Auction holds information on an ongoing auction, used as a sub-struct in OrderBookResp and OrderBook
type Auction struct {
	OpenPrice    float64   `json:"open_price,string"`
	OpenSize     float64   `json:"open_size,string"`
	BestBidPrice float64   `json:"best_bid_price,string"`
	BestBidSize  float64   `json:"best_bid_size,string"`
	BestAskPrice float64   `json:"best_ask_price,string"`
	BestAskSize  float64   `json:"best_ask_size,string"`
	AuctionState string    `json:"auction_state"`
	CanOpen      string    `json:"can_open"`
	Time         time.Time `json:"time"`
}

// OrderBookResp holds information on bids and asks for a particular currency pair, used for unmarshalling in GetProductBookV1
type OrderBookResp struct {
	Bids        []Orders  `json:"bids"`
	Asks        []Orders  `json:"asks"`
	Sequence    float64   `json:"sequence"`
	AuctionMode bool      `json:"auction_mode"`
	Auction     Auction   `json:"auction"`
	Time        time.Time `json:"time"`
}

// Orders holds information on orders, used as a sub-struct in OrderBook
type Orders struct {
	Price      types.Number
	Size       types.Number
	OrderCount uint64
	OrderID    uuid.UUID
}

// Candle holds properly formatted candle data, returned by GetProductCandles
type Candle struct {
	Time   types.Time
	Low    float64
	High   float64
	Open   float64
	Close  float64
	Volume float64
}

// ProductStats holds information on a pair's price and volume, returned by GetProductStats
type ProductStats struct {
	Open                    float64 `json:"open,string"`
	High                    float64 `json:"high,string"`
	Low                     float64 `json:"low,string"`
	Last                    float64 `json:"last,string"`
	Volume                  float64 `json:"volume,string"`
	Volume30Day             float64 `json:"volume_30day,string"`
	RFQVolume24Hour         float64 `json:"rfq_volume_24hour,string"`
	RFQVolume30Day          float64 `json:"rfq_volume_30day,string"`
	ConversionsVolume24Hour float64 `json:"conversions_volume_24hour,string"`
	ConversionsVolume30Day  float64 `json:"conversions_volume_30day,string"`
}

// ProductTicker holds information on a pair's price and volume, returned by GetProductTicker
type ProductTicker struct {
	Ask               float64   `json:"ask,string"`
	Bid               float64   `json:"bid,string"`
	Volume            float64   `json:"volume,string"`
	TradeID           int32     `json:"trade_id"`
	Price             float64   `json:"price,string"`
	Size              float64   `json:"size,string"`
	Time              time.Time `json:"time"`
	RFQVolume         float64   `json:"rfq_volume,string"`
	ConversionsVolume float64   `json:"conversions_volume,string"`
}

// ProductTrades holds information on a pair's trades, returned by GetProductTrades
type ProductTrades struct {
	TradeID int32     `json:"trade_id"`
	Side    string    `json:"side"`
	Size    float64   `json:"size,string"`
	Price   float64   `json:"price,string"`
	Time    time.Time `json:"time"`
}

// WrappedAsset holds information on a wrapped asset, used in AllWrappedAssets and returned by GetWrappedAssetDetails
type WrappedAsset struct {
	ID                string       `json:"id"`
	CirculatingSupply float64      `json:"circulating_supply,string"`
	TotalSupply       float64      `json:"total_supply,string"`
	ConversionRate    float64      `json:"conversion_rate,string"`
	APY               types.Number `json:"apy"`
}

// AllWrappedAssets holds information on all wrapped assets, returned by GetAllWrappedAssets
type AllWrappedAssets struct {
	WrappedAssets []WrappedAsset `json:"wrapped_assets"`
}

// WrappedAssetConversionRate holds information on a wrapped asset's conversion rate, returned by GetWrappedAssetConversionRate
type WrappedAssetConversionRate struct {
	Amount float64 `json:"amount,string"`
}

// ManyErrors holds information on errors
type ManyErrors struct {
	Success              bool   `json:"success"`
	FailureReason        string `json:"failure_reason"`
	OrderID              string `json:"order_id"`
	EditFailureReason    string `json:"edit_failure_reason"`
	PreviewFailureReason string `json:"preview_failure_reason"`
}
