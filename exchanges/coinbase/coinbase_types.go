package coinbase

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

type jwtManager struct {
	token     string
	expiresAt time.Time
	m         sync.RWMutex
}

type pairAliases struct {
	associatedAliases map[currency.Pair]currency.Pairs
	m                 sync.RWMutex
}

// Exchange is the overarching type across the coinbase package
type Exchange struct {
	exchange.Base
	jwt         jwtManager
	pairAliases pairAliases
}

// Version is used for the niche cases where the Version of the API must be specified and passed around for proper functionality
type Version bool

// FiatTransferType is used so that we don't need to duplicate the four fiat transfer-related endpoints under version 2 of the API
type FiatTransferType bool

// Integer is used to represent an integer in the API, which is represented as a string in the JSON response
type Integer int64

// CurrencyAmount holds a currency code and amount
type CurrencyAmount struct {
	Value    types.Number  `json:"value"`
	Currency currency.Code `json:"currency"`
}

// Account holds details for a trading account
type Account struct {
	UUID              string         `json:"uuid"`
	Name              string         `json:"name"`
	Currency          currency.Code  `json:"currency"`
	AvailableBalance  CurrencyAmount `json:"available_balance"`
	Default           bool           `json:"default"`
	Active            bool           `json:"active"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         time.Time      `json:"deleted_at"`
	Type              string         `json:"type"`
	Ready             bool           `json:"ready"`
	Hold              CurrencyAmount `json:"hold"`
	RetailPortfolioID string         `json:"retail_portfolio_id"`
	Platform          string         `json:"platform"`
}

// AllAccountsResponse holds many Account structs, as well as pagination information, returned by ListAccounts
type AllAccountsResponse struct {
	Accounts []*Account `json:"accounts"`
	HasNext  bool       `json:"has_next"`
	Cursor   Integer    `json:"cursor"`
	Size     uint8      `json:"size"`
}

// PermissionsResponse holds information on the permissions of a user, returned by GetPermissions
type PermissionsResponse struct {
	CanView       bool   `json:"can_view"`
	CanTrade      bool   `json:"can_trade"`
	CanTransfer   bool   `json:"can_transfer"`
	PortfolioUUID string `json:"portfolio_uuid"`
	PortfolioType string `json:"portfolio_type"`
}

// Params is used within functions to make the setting of parameters easier
type Params struct {
	url.Values
}

// MarginWindow is a sub-struct used in the type CurrentMarginWindow
type MarginWindow struct {
	MarginWindowType string    `json:"margin_window_type"`
	EndTime          time.Time `json:"end_time"`
}

// SuccessResp holds information on the success of an API request, returned by CancelPendingFuturesSweep, ScheduleFuturesSweep, and EditOrder
type SuccessResp struct {
	Success bool `json:"success"`
}

type futuresSweepReqBase struct {
	USDAmount float64 `json:"usd_amount,string,omitempty"`
}

type marginSettingReqBase struct {
	Setting string `json:"setting"`
}

// CurrentMarginWindow holds information on the current margin window, returned by GetCurrentMarginWindow
type CurrentMarginWindow struct {
	MarginWindow                                MarginWindow `json:"margin_window"`
	IsIntradayMarginKillswitchEnabled           bool         `json:"is_intraday_margin_killswitch_enabled"`
	IsIntradayMarginEnrollmentKillswitchEnabled bool         `json:"is_intraday_margin_enrollment_killswitch_enabled"`
}

// PriceSize is a sub-struct used in the type ProductBook
type PriceSize struct {
	Price types.Number `json:"price"`
	Size  types.Number `json:"size"`
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
	Pricebook      ProductBook  `json:"pricebook"`
	Last           types.Number `json:"last"`
	MidMarket      types.Number `json:"mid_market"`
	SpreadBPs      types.Number `json:"spread_bps"`
	SpreadAbsolute types.Number `json:"spread_absolute"`
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
	OpenInterest   types.Number `json:"open_interest"`
	FundingRate    types.Number `json:"funding_rate"`
	FundingTime    time.Time    `json:"funding_time"`
	MaxLeverage    types.Number `json:"max_leverage"`
	BaseAssetUUID  string       `json:"base_asset_uuid"`
	UnderlyingType string       `json:"underlying_type"`
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
	TwentyFourBySeven      bool             `json:"twenty_four_by_seven"`
	FundingInterval        string           `json:"funding_interval"`
	OpenInterest           types.Number     `json:"open_interest"`
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
	// Typically shows whether an FCM product is available for trading. If the request is authenticated, and the "get_tradability_status" bool is set to true, and the product is SPOT, and you're using our GetAllProducts function, this will instead reflect whether the product is available for trading.
	ViewOnly                  bool         `json:"view_only"`
	PriceIncrement            types.Number `json:"price_increment"`
	DisplayName               string       `json:"display_name"`
	ProductVenue              string       `json:"product_venue"`
	ApproximateQuote24HVolume types.Number `json:"approximate_quote_24h_volume"`
	NewAt                     time.Time    `json:"new_at"`
	// The following field only appears for future products
	FutureProductDetails FutureProductDetails `json:"future_product_details"`
}

// AllProducts holds information on a lot of available currency pairs, returned by GetAllProducts
type AllProducts struct {
	Products    []Product `json:"products"`
	NumProducts int32     `json:"num_products"`
}

// Klines holds historic trade information, returned by GetHistoricKlines
type Klines struct {
	Start  types.Time   `json:"start"`
	Low    types.Number `json:"low"`
	High   types.Number `json:"high"`
	Open   types.Number `json:"open"`
	Close  types.Number `json:"close"`
	Volume types.Number `json:"volume"`
}

// Trades is a sub-struct used in the type Ticker
type Trades struct {
	TradeID   string        `json:"trade_id"`
	ProductID currency.Pair `json:"product_id"`
	Price     types.Number  `json:"price"`
	Size      types.Number  `json:"size"`
	Time      time.Time     `json:"time"`
	Side      string        `json:"side"`
	Bid       types.Number  `json:"bid"`
	Ask       types.Number  `json:"ask"`
	Exchange  string        `json:"exchange"`
}

// Ticker holds basic ticker information, returned by GetTicker
type Ticker struct {
	Trades  []Trades     `json:"trades"`
	BestBid types.Number `json:"best_bid"`
	BestAsk types.Number `json:"best_ask"`
}

// MarketMarketIOC is a sub-struct used in the type OrderConfiguration
type MarketMarketIOC struct {
	QuoteSize   types.Number `json:"quote_size,omitempty"`
	BaseSize    types.Number `json:"base_size,omitempty"`
	RFQDisabled bool         `json:"rfq_disabled"`
	RFQEnabled  *bool        `json:"rfq_enabled,omitempty"`
	ReduceOnly  *bool        `json:"reduce_only,omitempty"`
}

// QuoteBaseLimit is a sub-struct used in the type OrderConfiguration
type QuoteBaseLimit struct {
	QuoteSize   types.Number `json:"quote_size,omitempty"`
	BaseSize    types.Number `json:"base_size,omitempty"`
	LimitPrice  types.Number `json:"limit_price"`
	RFQDisabled bool         `json:"rfq_disabled"`
}

// LimitLimitGTC is a sub-struct used in the type OrderConfiguration
type LimitLimitGTC struct {
	BaseSize    types.Number `json:"base_size,omitempty"`
	QuoteSize   types.Number `json:"quote_size,omitempty"`
	LimitPrice  types.Number `json:"limit_price"`
	PostOnly    bool         `json:"post_only"`
	RFQDisabled bool         `json:"rfq_disabled"`
	ReduceOnly  *bool        `json:"reduce_only,omitempty"`
}

// LimitLimitGTD is a sub-struct used in the type OrderConfiguration
type LimitLimitGTD struct {
	BaseSize    types.Number `json:"base_size,omitempty"`
	QuoteSize   types.Number `json:"quote_size,omitempty"`
	LimitPrice  types.Number `json:"limit_price"`
	EndTime     time.Time    `json:"end_time"`
	PostOnly    bool         `json:"post_only"`
	ReduceOnly  *bool        `json:"reduce_only,omitempty"`
	RFQDisabled bool         `json:"rfq_disabled,omitempty"`
}

// TWAPLimitGTD is a sub-struct used in the type OrderConfiguration
type TWAPLimitGTD struct {
	QuoteSize      types.Number `json:"quote_size,omitempty"`
	BaseSize       types.Number `json:"base_size,omitempty"`
	StartTime      time.Time    `json:"start_time"`
	EndTime        time.Time    `json:"end_time"`
	LimitPrice     types.Number `json:"limit_price"`
	NumberBuckets  int64        `json:"number_buckets,string"`
	BucketSize     types.Number `json:"bucket_size"`
	BucketDuration string       `json:"bucket_duration"`
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

// TriggerBracketGTC is a sub-struct used in the type OrderConfiguration
type TriggerBracketGTC struct {
	BaseSize         types.Number `json:"base_size,omitempty"`
	LimitPrice       types.Number `json:"limit_price"`
	StopTriggerPrice types.Number `json:"stop_trigger_price"`
}

// TriggerBracketGTD is a sub-struct used in the type OrderConfiguration
type TriggerBracketGTD struct {
	BaseSize         types.Number `json:"base_size,omitempty"`
	LimitPrice       types.Number `json:"limit_price"`
	StopTriggerPrice types.Number `json:"stop_trigger_price"`
	EndTime          time.Time    `json:"end_time"`
}

// OrderConfiguration is a struct used in the formation of requests in PrepareOrderConfig, and is a sub-struct used in the types SuccessFailureConfig and GetOrderResponse
type OrderConfiguration struct {
	MarketMarketIOC       *MarketMarketIOC       `json:"market_market_ioc,omitempty"`
	SORLimitIOC           *QuoteBaseLimit        `json:"sor_limit_ioc,omitempty"`
	LimitLimitGTC         *LimitLimitGTC         `json:"limit_limit_gtc,omitempty"`
	LimitLimitGTD         *LimitLimitGTD         `json:"limit_limit_gtd,omitempty"`
	LimitLimitFOK         *QuoteBaseLimit        `json:"limit_limit_fok,omitempty"`
	TWAPLimitGTD          *TWAPLimitGTD          `json:"twap_limit_gtd,omitempty"`
	StopLimitStopLimitGTC *StopLimitStopLimitGTC `json:"stop_limit_stop_limit_gtc,omitempty"`
	StopLimitStopLimitGTD *StopLimitStopLimitGTD `json:"stop_limit_stop_limit_gtd,omitempty"`
	TriggerBracketGTC     *TriggerBracketGTC     `json:"trigger_bracket_gtc,omitempty"`
	TriggerBracketGTD     *TriggerBracketGTD     `json:"trigger_bracket_gtd,omitempty"`
}

// SuccessResponse is a sub-struct used in the type SuccessFailureConfig
type SuccessResponse struct {
	OrderID         string        `json:"order_id"`
	ProductID       currency.Pair `json:"product_id"`
	Side            string        `json:"side"`
	ClientOrderID   string        `json:"client_order_id"`
	AttachedOrderID string        `json:"attached_order_id"`
}

// ErrorResponse is a sub-struct used in unmarshalling errors from the exchange in general
type ErrorResponse struct {
	ErrorType             string `json:"error"`
	Message               string `json:"message"`
	ErrorDetails          string `json:"error_details"`
	EditFailureReason     string `json:"edit_failure_reason"`
	PreviewFailureReason  string `json:"preview_failure_reason"`
	NewOrderFailureReason string `json:"new_order_failure_reason"`
}

// OrderInfo contains order configuration information used in both PlaceOrderInfo and PreviewOrderInfo
type OrderInfo struct {
	OrderType      order.Type
	TimeInForce    order.TimeInForce
	StopDirection  string
	BaseAmount     float64
	QuoteAmount    float64
	LimitPrice     float64
	StopPrice      float64
	BucketSize     float64
	EndTime        time.Time
	PostOnly       bool
	RFQDisabled    bool
	BucketNumber   int64
	BucketDuration time.Duration
}

// PlaceOrderInfo is a struct used in the formation of requests in PlaceOrder
type PlaceOrderInfo struct {
	ClientOID                  string
	ProductID                  string
	Side                       string
	MarginType                 string
	RetailPortfolioID          string
	PreviewID                  string
	Leverage                   float64
	AttachedOrderConfiguration OrderConfiguration
	OrderInfo
}

// SuccessFailureConfig contains information on an order, returned by PlaceOrder
type SuccessFailureConfig struct {
	Success            bool               `json:"success"`
	SuccessResponse    SuccessResponse    `json:"success_response"`
	OrderConfiguration OrderConfiguration `json:"order_configuration"`
}

// OrderCancelDetail contains information on attempted order cancellations, returned by CancelOrders
type OrderCancelDetail struct {
	Success       bool   `json:"success"`
	FailureReason string `json:"failure_reason"`
	OrderID       string `json:"order_id"`
}

type cancelOrdersReqBase struct {
	OrderIDs []string `json:"order_ids"`
}

type closePositionReqBase struct {
	ClientOrderID string        `json:"client_order_id"`
	ProductID     currency.Pair `json:"product_id"`
	Size          float64       `json:"size,string"`
}

type placeOrderReqbase struct {
	ClientOID                  string              `json:"client_order_id"`
	ProductID                  string              `json:"product_id"`
	Side                       string              `json:"side"`
	OrderConfiguration         *OrderConfiguration `json:"order_configuration"`
	RetailPortfolioID          string              `json:"retail_portfolio_id"`
	PreviewID                  string              `json:"preview_id"`
	AttachedOrderConfiguration *OrderConfiguration `json:"attached_order_configuration"`
	MarginType                 string              `json:"margin_type,omitempty"`
	Leverage                   float64             `json:"leverage,omitempty,string"`
}

type editOrderReqBase struct {
	OrderID string  `json:"order_id"`
	Size    float64 `json:"size,string"`
	Price   float64 `json:"price,string"`
}

// EditOrderPreviewResp contains information on the effects of editing an order, returned by EditOrderPreview
type EditOrderPreviewResp struct {
	Slippage           types.Number `json:"slippage"`
	OrderTotal         types.Number `json:"order_total"`
	CommissionTotal    types.Number `json:"commission_total"`
	QuoteSize          types.Number `json:"quote_size"`
	BaseSize           types.Number `json:"base_size"`
	BestBid            types.Number `json:"best_bid"`
	BestAsk            types.Number `json:"best_ask"`
	AverageFilledPrice types.Number `json:"average_filled_price"`
	OrderMarginTotal   types.Number `json:"order_margin_total"`
}

// EditHistory is a sub-struct used in the type GetOrderResponse
type EditHistory struct {
	Price                  types.Number `json:"price"`
	Size                   types.Number `json:"size"`
	ReplaceAcceptTimestamp time.Time    `json:"replace_accept_timestamp"`
}

// GetOrderResponse contains information on an order, returned by GetOrderByID IterativeGetAllOrders, and used in GetAllOrdersResp
type GetOrderResponse struct {
	OrderID                    string             `json:"order_id"`
	ProductID                  currency.Pair      `json:"product_id"`
	UserID                     string             `json:"user_id"`
	OrderConfiguration         OrderConfiguration `json:"order_configuration"`
	Side                       string             `json:"side"`
	ClientOID                  string             `json:"client_order_id"`
	Status                     string             `json:"status"`
	TimeInForce                string             `json:"time_in_force"`
	CreatedTime                time.Time          `json:"created_time"`
	CompletionPercentage       types.Number       `json:"completion_percentage"`
	FilledSize                 types.Number       `json:"filled_size"`
	AverageFilledPrice         types.Number       `json:"average_filled_price"`
	Fee                        types.Number       `json:"fee"`
	NumberOfFills              int64              `json:"num_fills,string"`
	FilledValue                types.Number       `json:"filled_value"`
	PendingCancel              bool               `json:"pending_cancel"`
	SizeInQuote                bool               `json:"size_in_quote"`
	TotalFees                  types.Number       `json:"total_fees"`
	SizeInclusiveOfFees        bool               `json:"size_inclusive_of_fees"`
	TotalValueAfterFees        types.Number       `json:"total_value_after_fees"`
	TriggerStatus              string             `json:"trigger_status"`
	OrderType                  string             `json:"order_type"`
	RejectReason               string             `json:"reject_reason"`
	Settled                    bool               `json:"settled"`
	ProductType                string             `json:"product_type"`
	RejectMessage              string             `json:"reject_message"`
	CancelMessage              string             `json:"cancel_message"`
	OrderPlacementSource       string             `json:"order_placement_source"`
	OutstandingHoldAmount      types.Number       `json:"outstanding_hold_amount"`
	IsLiquidation              bool               `json:"is_liquidation"`
	LastFillTime               time.Time          `json:"last_fill_time"`
	EditHistory                []EditHistory      `json:"edit_history"`
	Leverage                   types.Number       `json:"leverage"`
	MarginType                 string             `json:"margin_type"`
	RetailPortfolioID          string             `json:"retail_portfolio_id"`
	OriginatingOrderID         string             `json:"originating_order_id"`
	AttachedOrderID            string             `json:"attached_order_id"`
	AttachedOrderConfiguration OrderConfiguration `json:"attached_order_configuration"`
	CurrentPendingReplace      json.RawMessage    `json:"current_pending_replace"`
	CommissionDetailTotal      json.RawMessage    `json:"commission_detail_total"`
}

// Fills is a sub-struct used in the type FillResponse
type Fills struct {
	EntryID               string          `json:"entry_id"`
	TradeID               string          `json:"trade_id"`
	OrderID               string          `json:"order_id"`
	TradeTime             time.Time       `json:"trade_time"`
	TradeType             string          `json:"trade_type"`
	Price                 types.Number    `json:"price"`
	Size                  types.Number    `json:"size"`
	Commission            types.Number    `json:"commission"`
	ProductID             currency.Pair   `json:"product_id"`
	SequenceTimestamp     time.Time       `json:"sequence_timestamp"`
	LiquidityIndicator    string          `json:"liquidity_indicator"`
	SizeInQuote           bool            `json:"size_in_quote"`
	UserID                string          `json:"user_id"`
	Side                  string          `json:"side"`
	RetailPortfolioID     string          `json:"retail_portfolio_id"`
	FillSource            string          `json:"fill_source"`
	CommissionDetailTotal json.RawMessage `json:"commission_detail_total"`
}

// FillResponse contains fill information, returned by ListFills
type FillResponse struct {
	Fills  []Fills `json:"fills"`
	Cursor Integer `json:"cursor"`
}

// PreviewOrderInfo is a struct used in the formation of requests in PreviewOrder
type PreviewOrderInfo struct {
	ProductID                  string
	Side                       string
	MarginType                 string
	RetailPortfolioID          string
	Leverage                   float64
	AttachedOrderConfiguration OrderConfiguration
	OrderInfo
}

// TriggerBracketPNL is a sub-struct used in the type PreviewOrderResp
type TriggerBracketPNL struct {
	TakeProfitPNL types.Number `json:"take_profit_pnl"`
	StopLossPNL   types.Number `json:"stop_loss_pnl"`
}

// TWAPBucketMetadata is a sub-struct used in the type PreviewOrderResp
type TWAPBucketMetadata struct {
	BucketDuration time.Duration `json:"bucket_duration"`
	BucketSize     types.Number  `json:"bucket_size"`
	BucketNumber   Integer       `json:"bucket_number"`
}

// MarginRatioData is a sub-struct used in the type PreviewOrderResp
type MarginRatioData struct {
	CurrentMarginRatio   types.Number `json:"current_margin_ratio"`
	ProjectedMarginRatio types.Number `json:"projected_margin_ratio"`
}

// PreviewOrderResp contains information on the effects of placing an order, returned by PreviewOrder
type PreviewOrderResp struct {
	OrderTotal                     types.Number       `json:"order_total"`
	CommissionTotal                types.Number       `json:"commission_total"`
	Errs                           []string           `json:"errs"`
	Warning                        []string           `json:"warning"`
	QuoteSize                      types.Number       `json:"quote_size"`
	BaseSize                       types.Number       `json:"base_size"`
	BestBid                        types.Number       `json:"best_bid"`
	BestAsk                        types.Number       `json:"best_ask"`
	IsMax                          bool               `json:"is_max"`
	OrderMarginTotal               types.Number       `json:"order_margin_total"`
	Leverage                       types.Number       `json:"leverage"`
	LongLeverage                   types.Number       `json:"long_leverage"`
	ShortLeverage                  types.Number       `json:"short_leverage"`
	Slippage                       types.Number       `json:"slippage"`
	PreviewID                      string             `json:"preview_id"`
	CurrentLiquidationBuffer       types.Number       `json:"current_liquidation_buffer"`
	ProjectedLiquidationBuffer     types.Number       `json:"projected_liquidation_buffer"`
	MaxLeverage                    types.Number       `json:"max_leverage"`
	PNLConfiguration               TriggerBracketPNL  `json:"pnl_configuration"`
	TWAPBucketMetadata             TWAPBucketMetadata `json:"twap_bucket_metadata"`
	PositionNotionalLimit          types.Number       `json:"position_notional_limit"`
	MaxNotionalAtRequestedLeverage types.Number       `json:"max_notional_at_requested_leverage"`
	MarginRatioData                MarginRatioData    `json:"margin_ratio_data"`
	CommissionDetailTotal          json.RawMessage    `json:"commission_detail_total"`
}

type previewOrderReqBase struct {
	ProductID                  string              `json:"product_id"`
	Side                       string              `json:"side"`
	OrderConfiguration         *OrderConfiguration `json:"order_configuration"`
	RetailPortfolioID          string              `json:"retail_portfolio_id"`
	Leverage                   float64             `json:"leverage,string"`
	AttachedOrderConfiguration *OrderConfiguration `json:"attached_order_configuration"`
	MarginType                 string              `json:"margin_type,omitempty"`
}

// SimplePortfolioData is a sub-struct used in the type DetailedPortfolioResponse
type SimplePortfolioData struct {
	Name    string `json:"name"`
	UUID    string `json:"uuid"`
	Type    string `json:"type"`
	Deleted bool   `json:"deleted"`
}

type nameReqBase struct {
	Name string `json:"name"`
}

// MovePortfolioFundsResponse contains the UUIDs of the portfolios involved. Returned by MovePortfolioFunds
type MovePortfolioFundsResponse struct {
	SourcePortfolioUUID string `json:"source_portfolio_uuid"`
	TargetPortfolioUUID string `json:"target_portfolio_uuid"`
}

type fundsData struct {
	Value    float64       `json:"value,string"`
	Currency currency.Code `json:"currency"`
}

type movePortfolioFundsReqBase struct {
	SourcePortfolioUUID string    `json:"source_portfolio_uuid"`
	TargetPortfolioUUID string    `json:"target_portfolio_uuid"`
	Funds               fundsData `json:"funds"`
}

// NativeAndRaw is a sub-struct used in the type DetailedPortfolioResponse
type NativeAndRaw struct {
	UserNativeCurrency CurrencyAmount `json:"userNativeCurrency"`
	RawCurrency        CurrencyAmount `json:"rawCurrency"`
}

// PortfolioBalances is a sub-struct used in the type DetailedPortfolioResponse
type PortfolioBalances struct {
	TotalBalance               CurrencyAmount `json:"total_balance"`
	TotalFuturesBalance        CurrencyAmount `json:"total_futures_balance"`
	TotalCashEquivalentBalance CurrencyAmount `json:"total_cash_equivalent_balance"`
	TotalCryptoBalance         CurrencyAmount `json:"total_crypto_balance"`
	FuturesUnrealizedPNL       CurrencyAmount `json:"futures_unrealized_pnl"`
	PerpUnrealizedPNL          CurrencyAmount `json:"perp_unrealized_pnl"`
}

// SpotPositions is a sub-struct used in the type DetailedPortfolioResponse
type SpotPositions struct {
	Asset                 string         `json:"asset"`
	AccountUUID           string         `json:"account_uuid"`
	TotalBalanceFiat      float64        `json:"total_balance_fiat"`
	TotalBalanceCrypto    float64        `json:"total_balance_crypto"`
	AvailableToTreadeFiat float64        `json:"available_to_trade_fiat"`
	Allocation            float64        `json:"allocation"`
	OneDayChange          float64        `json:"one_day_change"`
	CostBasis             CurrencyAmount `json:"cost_basis"`
	AssetImgURL           string         `json:"asset_img_url"`
	IsCash                bool           `json:"is_cash"`
}

// PerpPositions is a sub-struct used in the type DetailedPortfolioResponse
type PerpPositions struct {
	ProductID             currency.Pair `json:"product_id"`
	ProductUUID           string        `json:"product_uuid"`
	Symbol                string        `json:"symbol"`
	AssetImageURL         string        `json:"asset_image_url"`
	VWAP                  NativeAndRaw  `json:"vwap"`
	PositionSide          string        `json:"position_side"`
	NetSize               types.Number  `json:"net_size"`
	BuyOrderSize          types.Number  `json:"buy_order_size"`
	SellOrderSize         types.Number  `json:"sell_order_size"`
	IMContribution        types.Number  `json:"im_contribution"`
	UnrealizedPNL         NativeAndRaw  `json:"unrealized_pnl"`
	MarkPrice             NativeAndRaw  `json:"mark_price"`
	LiquidationPrice      NativeAndRaw  `json:"liquidation_price"`
	Leverage              types.Number  `json:"leverage"`
	IMNotional            NativeAndRaw  `json:"im_notional"`
	MMNotional            NativeAndRaw  `json:"mm_notional"`
	PositionNotional      NativeAndRaw  `json:"position_notional"`
	MarginType            string        `json:"margin_type"`
	LiquidationBuffer     types.Number  `json:"liquidation_buffer"`
	LiquidationPercentage types.Number  `json:"liquidation_percentage"`
}

// FuturesPositions is a sub-struct used in the type DetailedPortfolioResponse
type FuturesPositions []struct {
	ProductID       currency.Pair `json:"product_id"`
	ContractSize    types.Number  `json:"contract_size"`
	Side            string        `json:"side"`
	Amount          types.Number  `json:"amount"`
	AvgEntryPrice   types.Number  `json:"avg_entry_price"`
	CurrentPrice    types.Number  `json:"current_price"`
	UnrealizedPNL   types.Number  `json:"unrealized_pnl"`
	Expiry          time.Time     `json:"expiry"`
	UnderlyingAsset string        `json:"underlying_asset"`
	AssetImgURL     string        `json:"asset_img_url"`
	ProductName     string        `json:"product_name"`
	Venue           string        `json:"venue"`
	NotionalValue   types.Number  `json:"notional_value"`
}

// DetailedPortfolioResponse contains a great deal of information on a single portfolio. Returned by GetPortfolioByID
type DetailedPortfolioResponse struct {
	Portfolio         SimplePortfolioData `json:"portfolio"`
	PortfolioBalances PortfolioBalances   `json:"portfolio_balances"`
	SpotPositions     []SpotPositions     `json:"spot_positions"`
	PerpPositions     []PerpPositions     `json:"perp_positions"`
	FuturesPositions  []FuturesPositions  `json:"futures_positions"`
}

// MarginWindowMeasurement is a sub-struct used in the type FuturesBalanceSummary
type MarginWindowMeasurement struct {
	MarginWindowType   string `json:"margin_window_type"`
	MarginLevel        string `json:"margin_level"`
	InitialMargin      string `json:"initial_margin"`
	MaintenanceMargin  string `json:"maintenance_margin"`
	LiquidationBuffer  string `json:"liquidation_buffer"`
	TotalHold          string `json:"total_hold_amount"`
	FuturesBuyingPower string `json:"futures_buying_power"`
}

// FuturesBalanceSummary contains information on futures balances, returned by GetFuturesBalanceSummary
type FuturesBalanceSummary struct {
	FuturesBuyingPower               CurrencyAmount          `json:"futures_buying_power"`
	TotalUSDBalance                  CurrencyAmount          `json:"total_usd_balance"`
	CBIUSDBalance                    CurrencyAmount          `json:"cbi_usd_balance"`
	CFMUSDBalance                    CurrencyAmount          `json:"cfm_usd_balance"`
	TotalOpenOrdersHoldAmount        CurrencyAmount          `json:"total_open_orders_hold_amount"`
	UnrealizedPNL                    CurrencyAmount          `json:"unrealized_pnl"`
	DailyRealizedPNL                 CurrencyAmount          `json:"daily_realized_pnl"`
	InitialMargin                    CurrencyAmount          `json:"initial_margin"`
	AvailableMargin                  CurrencyAmount          `json:"available_margin"`
	LiquidationThreshold             CurrencyAmount          `json:"liquidation_threshold"`
	LiquidationBufferAmount          CurrencyAmount          `json:"liquidation_buffer_amount"`
	LiquidationBufferPercentage      types.Number            `json:"liquidation_buffer_percentage"`
	IntradayMarginWindowMeasurement  MarginWindowMeasurement `json:"intraday_margin_window_measure"`
	OvernightMarginWindowMeasurement MarginWindowMeasurement `json:"overnight_margin_window_measure"`
}

// FuturesPosition contains information on a single futures position, returned by GetFuturesPositionByID and ListFuturesPositions
type FuturesPosition struct {
	ProductID         currency.Pair `json:"product_id"`
	ExpirationTime    time.Time     `json:"expiration_time"`
	Side              string        `json:"side"`
	NumberOfContracts types.Number  `json:"number_of_contracts"`
	CurrentPrice      types.Number  `json:"current_price"`
	AverageEntryPrice types.Number  `json:"avg_entry_price"`
	UnrealizedPNL     types.Number  `json:"unrealized_pnl"`
	DailyRealizedPNL  types.Number  `json:"daily_realized_pnl"`
}

// SweepData contains information on pending and processing sweep requests, returned by ListFuturesSweeps
type SweepData struct {
	ID              string         `json:"id"`
	RequestedAmount CurrencyAmount `json:"requested_amount"`
	ShouldSweepAll  bool           `json:"should_sweep_all"`
	Status          string         `json:"status"`
	ScheduledTime   time.Time      `json:"scheduled_time"`
}

// PerpPositionSummary contains information on perpetuals portfolio balances, used as a sub-struct in the types PerpPositionDetail, AllPerpPosResponse, and OnePerpPosResponse
type PerpPositionSummary struct {
	PortfolioUUID              string         `json:"portfolio_uuid"`
	Collateral                 types.Number   `json:"collateral"`
	PositionNotional           types.Number   `json:"position_notional"`
	OpenPositionNotional       types.Number   `json:"open_position_notional"`
	PendingFees                types.Number   `json:"pending_fees"`
	Borrow                     types.Number   `json:"borrow"`
	AccruedInterest            types.Number   `json:"accrued_interest"`
	RollingDebt                types.Number   `json:"rolling_debt"`
	PortfolioInitialMargin     types.Number   `json:"portfolio_initial_margin"`
	PortfolioIMNotional        CurrencyAmount `json:"portfolio_im_notional"`
	PortfolioMaintenanceMargin types.Number   `json:"portfolio_maintenance_margin"`
	PortfolioMMNotional        CurrencyAmount `json:"portfolio_mm_notional"`
	LiquidationPercentage      types.Number   `json:"liquidation_percentage"`
	LiquidationBuffer          types.Number   `json:"liquidation_buffer"`
	MarginType                 string         `json:"margin_type"`
	MarginFlags                string         `json:"margin_flags"`
	LiquidationStatus          string         `json:"liquidation_status"`
	UnrealizedPNL              CurrencyAmount `json:"unrealized_pnl"`
	BuyingPower                CurrencyAmount `json:"buying_power"` // Not in the GetPerpetualsPortfolioSummary response
	TotalBalance               CurrencyAmount `json:"total_balance"`
	MaxWithdrawal              CurrencyAmount `json:"max_withdrawal"` // Not in the GetPerpetualsPortfolioSummary response
}

// PerpetualPortfolioSummary contains information on perpetuals portfolio balances, used as a sub-struct in the type PerpetualPortfolioResponse
type PerpetualPortfolioSummary struct {
	UnrealisedPNL           CurrencyAmount `json:"unrealized_pnl"`
	BuyingPower             CurrencyAmount `json:"buying_power"`
	TotalBalance            CurrencyAmount `json:"total_balance"`
	MaximumWithdrawalAmount CurrencyAmount `json:"maximum_withdrawal_amount"`
}

// PerpetualPortfolioResponse contains information on perpetuals portfolio balances, returned by GetPerpetualsPortfolioSummary
type PerpetualPortfolioResponse struct {
	Portfolios []PerpPositionSummary     `json:"portfolios"`
	Summary    PerpetualPortfolioSummary `json:"summary"`
}

// PerpPositionDetail contains information on a single perpetuals position, used as a sub-struct in the type AllPerpPosResponse, and returned by GetPerpetualsPositionByID
type PerpPositionDetail struct {
	ProductID             currency.Pair       `json:"product_id"`
	ProductUUID           string              `json:"product_uuid"`
	PortfolioUUID         string              `json:"portfolio_uuid"`
	Symbol                string              `json:"symbol"`
	VWAP                  CurrencyAmount      `json:"vwap"`
	EntryVWAP             CurrencyAmount      `json:"entry_vwap"`
	PositionSide          string              `json:"position_side"`
	MarginType            string              `json:"margin_type"`
	NetSize               types.Number        `json:"net_size"`
	BuyOrderSize          types.Number        `json:"buy_order_size"`
	SellOrderSize         types.Number        `json:"sell_order_size"`
	IMContribution        types.Number        `json:"im_contribution"`
	UnrealizedPNL         CurrencyAmount      `json:"unrealized_pnl"`
	MarkPrice             CurrencyAmount      `json:"mark_price"`
	LiquidationPrice      CurrencyAmount      `json:"liquidation_price"`
	Leverage              types.Number        `json:"leverage"`
	IMNotional            CurrencyAmount      `json:"im_notional"`
	MMNotional            CurrencyAmount      `json:"mm_notional"`
	PositionNotional      CurrencyAmount      `json:"position_notional"`
	AggregatedPNL         CurrencyAmount      `json:"aggregated_pnl"`
	LiquidationBuffer     types.Number        `json:"liquidation_buffer"`     // Not in GetPerpetualsPositionByID
	LiquidationPercentage types.Number        `json:"liquidation_percentage"` // Not in GetPerpetualsPositionByID
	PortfolioSummary      PerpPositionSummary `json:"portfolio_summary"`      // Not in GetPerpetualsPositionByID
}

// PortfolioAsset contains information on a single portfolio asset, used as a sub-struct in the type PortfolioBalance
type PortfolioAsset struct {
	AssetID                          string       `json:"asset_id"`
	AssetUUID                        string       `json:"asset_uuid"`
	AssetName                        string       `json:"asset_name"`
	Status                           string       `json:"status"`
	CollateralWeight                 types.Number `json:"collateral_weight"`
	AccountCollateralLimit           types.Number `json:"account_collateral_limit"`
	EcosystemCollateralLimitBreached bool         `json:"ecosystem_collateral_limit_breached"`
	AssetIconURL                     string       `json:"asset_icon_url"`
	SupportedNetworksEnabled         bool         `json:"supported_networks_enabled"`
}

// PortfolioBalance contains information on a single portfolio balance, used as a sub-struct in the type PortfolioBalancesResponse
type PortfolioBalance struct {
	Asset                        PortfolioAsset `json:"asset"`
	Quantity                     types.Number   `json:"quantity"`
	Hold                         types.Number   `json:"hold"`
	TransferHold                 types.Number   `json:"transfer_hold"`
	CollateralValue              types.Number   `json:"collateral_value"`
	CollateralWeight             types.Number   `json:"collateral_weight"`
	MaxWithdrawAmount            types.Number   `json:"max_withdraw_amount"`
	Loan                         types.Number   `json:"loan"`
	LoanCollateralRequirementUSD types.Number   `json:"loan_collateral_requirement_usd"`
	PledgedQuantity              types.Number   `json:"pledged_quantity"`
}

// PortfolioBalancesResponse contains information on a portfolio's balances, returned by GetPortfolioBalances
type PortfolioBalancesResponse struct {
	PortfolioUUID        string             `json:"portfolio_uuid"`
	Balances             []PortfolioBalance `json:"balances"`
	IsMarginLimitReached bool               `json:"is_margin_limit_reached"`
}

// PerpetualSummary contains information on a perpetual position's summary, used as a sub-struct in the type AllPerpPosResponse
type PerpetualSummary struct {
	AggregatedPNL CurrencyAmount `json:"aggregated_pnl"`
}

// AllPerpPosResponse contains information on perpetuals positions, returned by GetAllPerpetualsPositions
type AllPerpPosResponse struct {
	Positions []PerpPositionDetail `json:"positions"`
	Summary   PerpetualSummary     `json:"summary"`
}

type assetCollateralToggleReqBase struct {
	PortfolioUUID string `json:"portfolio_uuid"`
	Enabled       bool   `json:"multi_asset_collateral_enabled"`
}

// FeeTier is a sub-struct used in the type TransactionSummary
type FeeTier struct {
	PricingTier  string       `json:"pricing_tier"`
	USDFrom      types.Number `json:"usd_from"`
	USDTo        types.Number `json:"usd_to"`
	TakerFeeRate types.Number `json:"taker_fee_rate"`
	MakerFeeRate types.Number `json:"maker_fee_rate"`
	AOPFrom      types.Number `json:"aop_from"`
	AOPTo        types.Number `json:"aop_to"`
	PerpsVolFrom types.Number `json:"perps_vol_from"`
	PerpsVolTo   types.Number `json:"perps_vol_to"`
}

// MarginRate is a sub-struct used in the type TransactionSummary
type MarginRate struct {
	Value types.Number `json:"value"`
}

// GoodsAndServicesTax is a sub-struct used in the type TransactionSummary
type GoodsAndServicesTax struct {
	Rate types.Number `json:"rate"`
	Type string       `json:"type"`
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

// ListOrdersReq contains the parameters for the ListOrders request
type ListOrdersReq struct {
	OrderIDs             []string
	OrderStatus          []string
	TimeInForces         []string
	OrderTypes           []string
	AssetFilters         []string
	ProductIDs           currency.Pairs
	ProductType          string
	OrderSide            string
	OrderPlacementSource string
	ContractExpiryType   string
	RetailPortfolioID    string
	SortBy               string
	Cursor               int64
	Limit                int32
	StartDate            time.Time
	EndDate              time.Time
	UserNativeCurrency   currency.Code
}

// ListOrdersResp contains information on a lot of orders, returned by ListOrders
type ListOrdersResp struct {
	Orders   []GetOrderResponse `json:"orders"`
	Sequence Integer            `json:"sequence"`
	HasNext  bool               `json:"has_next"`
	Cursor   Integer            `json:"cursor"`
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
	Amount CurrencyAmount `json:"amount"`
	Source string         `json:"source"`
}

// FeeStruct is a sub-struct storing information on fees, used in ConvertResponse
type FeeStruct struct {
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Amount        CurrencyAmount `json:"amount"`
	Label         string         `json:"label"`
	Disclosure    Disclosure     `json:"disclosure"`
	WaivedDetails WaivedDetails  `json:"waived_details"`
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

// MostlyFullAddress is a sub-struct, used in CardInfo and UKAccountHolder
type MostlyFullAddress struct {
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
	FirstDataToken              ValueWithStoreID  `json:"first_data_token"`
	ExpiryDate                  MonthYear         `json:"expiry_date"`
	PostalCode                  string            `json:"postal_code"`
	Merchant                    MerchantID        `json:"merchant"`
	VaultToken                  VaultToken        `json:"vault_token"`
	WorldpayParams              WorldplayParams   `json:"worldpay_params"`
	PreviousSchemeTransactionID string            `json:"previous_scheme_tx_id"`
	CustomerName                string            `json:"customer_name"`
	Address                     MostlyFullAddress `json:"address"`
	PhoneNumber                 string            `json:"phone_number"`
	UserID                      string            `json:"user_id"`
	CustomerFirstName           string            `json:"customer_first_name"`
	CustomerLastName            string            `json:"customer_last_name"`
	SixDigitBin                 string            `json:"six_digit_bin"`
	CustomerDateOfBirth         FullDate          `json:"customer_dob"`
	Scheme                      string            `json:"scheme"`
	EightDigitBin               string            `json:"eight_digit_bin"`
	CheckoutToken               SourceID          `json:"checkout_token"`
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
	LegalName     string            `json:"legal_name"`
	BBAN          string            `json:"bban"`
	SortCode      string            `json:"sort_code"`
	AccountNumber string            `json:"account_number"`
	Address       MostlyFullAddress `json:"address"`
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

// LedgerAccount is a sub-struct, used in AccountStruct and AccountHolder
type LedgerAccount struct {
	AccountID string `json:"account_id"`
	Currency  string `json:"currency"`
	Owner     Owner  `json:"owner"`
}

// PaymentMethodID is a sub-struct, used in AccountStruct and AccountHolder
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
	Address        MostlyFullAddress    `json:"address"`
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
	Address        MostlyFullAddress    `json:"address"`
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
	VPAID             string            `json:"vpa_id"`
	CustomerFirstName string            `json:"customer_first_name"`
	CustomerLastName  string            `json:"customer_last_name"`
	Email             string            `json:"email"`
	PhoneNumber       PhoneNumber       `json:"phone_number"`
	PAN               string            `json:"pan"`
	Address           MostlyFullAddress `json:"address"`
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
	IFSCCode          string            `json:"ifsc_code"`
	AccountNumber     string            `json:"account_number"`
	CustomerFirstName string            `json:"customer_first_name"`
	CustomerLastName  string            `json:"customer_last_name"`
	Email             string            `json:"email"`
	PhoneNumber       PhoneNumber       `json:"phone_number"`
	PAN               string            `json:"pan"`
	Address           MostlyFullAddress `json:"address"`
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
	Account             NameAndIBAN       `json:"account"`
	CustomerFirstName   string            `json:"customer_first_name"`
	CustomerLastName    string            `json:"customer_last_name"`
	Email               string            `json:"email"`
	PhoneNumber         PhoneNumber       `json:"phone_number"`
	CustomerCountry     string            `json:"customer_country"`
	Address             MostlyFullAddress `json:"address"`
	SupportsOpenBanking bool              `json:"supports_open_banking"`
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
	Amount CurrencyAmount `json:"amount"`
	Scale  int32          `json:"scale"`
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
	LimitAmount CurrencyAmount `json:"limit_amount"`
}

// CancellationReason is a sub-struct used in ConvertResponse and DeposWithdrData
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
	FreeTradingResetDate                       time.Time      `json:"free_trading_reset_date"`
	UsedZeroFeeTrading                         CurrencyAmount `json:"used_zero_fee_trading"`
	RemainingFreeTradingVolume                 CurrencyAmount `json:"remaining_free_trading_volume"`
	MaxFreeTradingVolume                       CurrencyAmount `json:"max_free_trading_volume"`
	HasBenefitCap                              bool           `json:"has_benefit_cap"`
	AppliedSubscriptionBenefit                 bool           `json:"applied_subscription_benefit"`
	FeeWithoutSubscriptionBenefit              CurrencyAmount `json:"fee_without_subscription_benefit"`
	PaymentMethodFeeWithoutSubscriptionBenefit CurrencyAmount `json:"payment_method_fee_without_subscription_benefit"`
}

// TaxDetails is a sub-struct used in ConvertResponse
type TaxDetails struct {
	Name   string         `json:"name"`
	Amount CurrencyAmount `json:"amount"`
}

// TradeIncentiveInfo is a sub-struct used in ConvertResponse
type TradeIncentiveInfo struct {
	AppliedIncentive    bool           `json:"applied_incentive"`
	UserIncentiveID     string         `json:"user_incentive_id"`
	CodeVal             string         `json:"code_val"`
	EndsAt              time.Time      `json:"ends_at"`
	FeeWithoutIncentive CurrencyAmount `json:"fee_without_incentive"`
	Redeemed            bool           `json:"redeemed"`
}

// ConvertWrapper wraps a ConvertResponse, used by CreateConvertQuote, CommitConvertTrade, and GetConvertTradeByID
type ConvertWrapper struct {
	Trade ConvertResponse `json:"trade"`
}

type convertTradeReqBase struct {
	FromAccount string `json:"from_account"`
	ToAccount   string `json:"to_account"`
}

type convertQuoteReqBase struct {
	FromAccount string                 `json:"from_account"`
	ToAccount   string                 `json:"to_account"`
	Amount      float64                `json:"amount,string"`
	Metadata    tradeIncentiveMetadata `json:"trade_incentive_metadata"`
}

type tradeIncentiveMetadata struct {
	UserIncentiveID string `json:"user_incentive_id"`
	CodeVal         string `json:"code_val"`
}

// ConvertResponse contains information on a convert trade, returned by CreateConvertQuote, CommitConvertTrade, and GetConvertTradeByID
type ConvertResponse struct {
	// Many of these fields and subfields could, in truth, be types.Number, but documentation lists them as strings, and these endpoints can't be tested with Australian accounts
	ID                 string             `json:"id"`
	Status             string             `json:"status"`
	UserEnteredAmount  CurrencyAmount     `json:"user_entered_amount"`
	Amount             CurrencyAmount     `json:"amount"`
	Subtotal           CurrencyAmount     `json:"subtotal"`
	Total              CurrencyAmount     `json:"total"`
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
	ExchangeRate       CurrencyAmount     `json:"exchange_rate"`
	TaxDetails         []TaxDetails       `json:"tax_details"`
	TradeIncentiveInfo TradeIncentiveInfo `json:"trade_incentive_info"`
	TotalFeeWithoutTax FeeStruct          `json:"total_fee_without_tax"`
	FiatDenotedTotal   CurrencyAmount     `json:"fiat_denoted_total"`
}

// ServerTimeV3 holds information on the server's time, returned by GetV3Time
type ServerTimeV3 struct {
	Iso               time.Time  `json:"iso"`
	EpochSeconds      types.Time `json:"epochSeconds"`
	EpochMilliseconds types.Time `json:"epochMillis"`
}

// PaymentMethodData is a sub-type that holds information on a payment method
type PaymentMethodData struct {
	ID            string        `json:"id"`
	Type          string        `json:"type"`
	Name          string        `json:"name"`
	Currency      currency.Code `json:"currency"`
	Verified      bool          `json:"verified"`
	AllowBuy      bool          `json:"allow_buy"`
	AllowSell     bool          `json:"allow_sell"`
	AllowDeposit  bool          `json:"allow_deposit"`
	AllowWithdraw bool          `json:"allow_withdraw"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type paymentMethodReqBase struct {
	Currency currency.Code `json:"currency"`
}

type allocatePortfolioReqBase struct {
	PortfolioUUID string  `json:"portfolio_uuid"`
	Symbol        string  `json:"symbol"`
	Currency      string  `json:"currency"`
	Amount        float64 `json:"amount,string"`
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
	PreviousEndingBefore string `json:"previous_ending_before"` // This is only present on some endpoints
	NextStartingAfter    string `json:"next_starting_after"`    // This is only present on some endpoints
	Limit                uint8  `json:"limit"`                  // This is only present on some endpoints
	Order                string `json:"order"`
	PreviousURI          string `json:"previous_uri"`       // Might only be present on some endpoints
	NextURI              string `json:"next_uri"`           // Might only be present on some endpoints
	Page                 uint8  `json:"page"`               // This is only present on some endpoints
	TotalCount           uint32 `json:"total_count,string"` // This is only present on some endpoints
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
	Amount   types.Number `json:"amount"`
	Currency string       `json:"currency"`
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
	Amount            types.Number `json:"amount"`
	Currency          string       `json:"currency"`
	CurrencySymbol    string       `json:"currency_symbol"`
	ReferralThreshold types.Number `json:"referral_threshold"`
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
	Code     string          `json:"code"`
	Name     string          `json:"name"`
	Color    string          `json:"color"`
	Exponent int32           `json:"exponent"`
	Type     string          `json:"type"`
	AssetID  string          `json:"asset_id"`
	Slug     string          `json:"slug"`
	Rewards  json.RawMessage `json:"rewards"`
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
	PortfolioID      string             `json:"portfolio_id"`
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

// AddressData holds address information, used in GetAllAddrResponse, and returned by CreateAddress and GetAddressByID
type AddressData struct {
	ID             string        `json:"id"`
	Address        string        `json:"address"`
	Currency       currency.Code `json:"currency"`
	Name           string        `json:"name"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Network        string        `json:"network"`
	Resource       string        `json:"resource"`
	ResourcePath   string        `json:"resource_path"`
	DestinationTag string        `json:"destination_tag"`
}

// GetAllAddrResponse holds information on many addresses, returned by GetAllAddresses
type GetAllAddrResponse struct {
	Pagination PaginationResp `json:"pagination"`
	Data       []AddressData  `json:"data"`
}

// AdvancedTradeFill is a sub-struct used in TransactionData
type AdvancedTradeFill struct {
	FillPrice  types.Number  `json:"fill_price"`
	ProductID  currency.Pair `json:"product_id"`
	OrderID    string        `json:"order_id"`
	Commission types.Number  `json:"commission"`
	OrderSide  string        `json:"order_side"`
}

// Network is a sub-struct used in TransactionData
type Network struct {
	Status string `json:"status"`
	Hash   string `json:"hash"`
	Name   string `json:"name"`
}

// FullAddress is a sub-struct, used in TravelRule
type FullAddress struct {
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	Address3   string `json:"address3"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	PostalCode string `json:"postal_code"`
}

// TravelRule contains information that may need to be provided to comply with local regulations. Used as a parameter for SendMoney
type TravelRule struct {
	BeneficiaryWalletType           string      `json:"beneficiary_wallet_type"`
	IsSelf                          string      `json:"is_self"`
	BeneficiaryName                 string      `json:"beneficiary_name"`
	BeneficiaryAddress              FullAddress `json:"beneficiary_address"`
	BeneficiaryFinancialInstitution string      `json:"beneficiary_financial_institution"`
	TransferPurpose                 string      `json:"transfer_purpose"`
}

type sendMoneyReqBase struct {
	Type              string        `json:"type"`
	To                string        `json:"to"`
	Amount            float64       `json:"amount,string"`
	Currency          currency.Code `json:"currency"`
	Description       string        `json:"description"`
	SkipNotifications bool          `json:"skip_notifications"`
	Idem              string        `json:"idem"`
	DestinationTag    string        `json:"destination_tag"`
	Network           string        `json:"network"`
	TravelRuleData    *TravelRule   `json:"travel_rule_data"`
}

// TransactionData is a sub-type that holds information on a transaction. Used in ManyTransactionsResp and returned by SendMoney
type TransactionData struct {
	ID                string             `json:"id"`
	Type              string             `json:"type"`
	Status            string             `json:"status"`
	Amount            AmountWithCurrency `json:"amount"`
	NativeAmount      AmountWithCurrency `json:"native_amount"`
	CreatedAt         time.Time          `json:"created_at"`
	Resource          string             `json:"resource"`
	ResourcePath      string             `json:"resource_path"`
	AdvancedTradeFill AdvancedTradeFill  `json:"advanced_trade_fill,omitzero"`
}

// ManyTransactionsResp holds information on many transactions. Returned by GetAddressTransactions and GetAllTransactions
type ManyTransactionsResp struct {
	Pagination PaginationResp    `json:"pagination"`
	Data       []TransactionData `json:"data"`
}

// AccountHolder is a sub-type that holds information on an account holder. Used in DeposWithdrData
type AccountHolder struct {
	Type                  string           `json:"type"`
	Network               string           `json:"network"`
	PaymentMethodID       string           `json:"payment_method_id"`
	ExternalPaymentMethod *PaymentMethodID `json:"external_payment_method,omitempty"`
	LedgerAccount         *LedgerAccount   `json:"ledger_account,omitempty"`
}

// FeeDetail is a sub-type that holds information on a fee. Used in DeposWithdrData
type FeeDetail struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Amount      CurrencyAmount `json:"amount"`
	Type        string         `json:"type"`
}

// DeposWithdrData is a sub-type that holds information on a deposit/withdrawal. Returned by FiatTransfer and CommitTransfer, and used in ManyDeposWithdrResp
type DeposWithdrData struct {
	UserEnteredAmount      CurrencyAmount     `json:"user_entered_amount"`
	Amount                 CurrencyAmount     `json:"amount"`
	Total                  CurrencyAmount     `json:"total"`
	Subtotal               CurrencyAmount     `json:"subtotal"`
	Idempotency            string             `json:"idempotency"`
	Committed              bool               `json:"committed"`
	ID                     string             `json:"id"`
	Instant                bool               `json:"instant"`
	Source                 AccountHolder      `json:"source"`
	Target                 AccountHolder      `json:"target"`
	PayoutAt               time.Time          `json:"payout_at"`
	Status                 string             `json:"status"`
	UserReference          string             `json:"user_reference"`
	Type                   string             `json:"type"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
	UserWarnings           []string           `json:"user_warnings"`
	Fees                   json.RawMessage    `json:"fees"`
	TotalFee               FeeDetail          `json:"total_fee"`
	CancellationReason     CancellationReason `json:"cancellation_reason"`
	HoldDays               int32              `json:"hold_days"`
	NextStep               json.RawMessage    `json:"next_step"`
	CheckoutURL            string             `json:"checkout_url"`
	RequiresCompletionStep bool               `json:"requires_completion_step"`
}

type fiatTransferReqBase struct {
	Currency      string  `json:"currency"`
	PaymentMethod string  `json:"payment_method"`
	Amount        float64 `json:"amount,string"`
	Commit        bool    `json:"commit"`
}

// ManyDeposWithdrResp holds information on many deposits. Returned by GetAllFiatTransfers
type ManyDeposWithdrResp struct {
	Pagination PaginationResp    `json:"pagination"`
	Data       []DeposWithdrData `json:"data"`
}

// FiatData holds information on fiat currencies. Returned by GetFiatCurrencies
type FiatData struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	MinSize types.Number `json:"min_size"`
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
	Amount   types.Number `json:"amount"`
	Base     string       `json:"base"`
	Currency string       `json:"currency"`
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
	Price                    types.Number  `json:"price"`
	Volume24H                types.Number  `json:"volume_24_h"`
	Low24H                   types.Number  `json:"low_24_h"`
	High24H                  types.Number  `json:"high_24_h"`
	Low52W                   types.Number  `json:"low_52_w"`
	High52W                  types.Number  `json:"high_52_w"`
	PricePercentageChange24H types.Number  `json:"price_percent_chg_24_h"`
	BestBid                  types.Number  `json:"best_bid"`
	BestBidQuantity          types.Number  `json:"best_bid_size"`
	BestAsk                  types.Number  `json:"best_ask"`
	BestAskQuantity          types.Number  `json:"best_ask_size"`
}

// WebsocketTickerHolder holds a variety of ticker responses, used when wsHandleData processes tickers
type WebsocketTickerHolder struct {
	Type    string            `json:"type"`
	Tickers []WebsocketTicker `json:"tickers"`
}

// WebsocketCandle defines a candle websocket response, used in WebsocketCandleHolder
type WebsocketCandle struct {
	Start     types.Time    `json:"start"`
	Low       types.Number  `json:"low"`
	High      types.Number  `json:"high"`
	Open      types.Number  `json:"open"`
	Close     types.Number  `json:"close"`
	Volume    types.Number  `json:"volume"`
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
	Price     types.Number  `json:"price"`
	Size      types.Number  `json:"size"`
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
	BaseIncrement  types.Number  `json:"base_increment"`
	QuoteIncrement types.Number  `json:"quote_increment"`
	DisplayName    string        `json:"display_name"`
	Status         string        `json:"status"`
	StatusMessage  string        `json:"status_message"`
	MinMarketFunds types.Number  `json:"min_market_funds"`
}

// WebsocketProductHolder holds a variety of product responses, used when wsHandleData processes an update on a product's status
type WebsocketProductHolder struct {
	Type     string             `json:"type"`
	Products []WebsocketProduct `json:"products"`
}

// WebsocketOrderbookData defines a websocket orderbook response, used in WebsocketOrderbookDataHolder
type WebsocketOrderbookData struct {
	Side        string       `json:"side"`
	EventTime   time.Time    `json:"event_time"`
	PriceLevel  types.Number `json:"price_level"`
	NewQuantity types.Number `json:"new_quantity"`
}

// WebsocketOrderbookDataHolder holds a variety of orderbook responses, used when wsHandleData processes orderbooks, as well as under typical operation of ProcessSnapshot, ProcessUpdate, and processBidAskArray
type WebsocketOrderbookDataHolder struct {
	Type      string                   `json:"type"`
	ProductID currency.Pair            `json:"product_id"`
	Changes   []WebsocketOrderbookData `json:"updates"`
}

// WebsocketOrderData defines a websocket order response, used in WebsocketOrderDataHolder
type WebsocketOrderData struct {
	AveragePrice          types.Number  `json:"avg_price"`
	CancelReason          string        `json:"cancel_reason"`
	ClientOrderID         string        `json:"client_order_id"`
	CompletionPercentage  types.Number  `json:"completion_percentage"`
	ContractExpiryType    string        `json:"contract_expiry_type"`
	CumulativeQuantity    types.Number  `json:"cumulative_quantity"`
	FilledValue           types.Number  `json:"filled_value"`
	LeavesQuantity        types.Number  `json:"leaves_quantity"`
	LimitPrice            types.Number  `json:"limit_price"`
	NumberOfFills         int64         `json:"number_of_fills"`
	OrderID               string        `json:"order_id"`
	OrderSide             string        `json:"order_side"`
	OrderType             string        `json:"order_type"`
	OutstandingHoldAmount types.Number  `json:"outstanding_hold_amount"`
	PostOnly              bool          `json:"post_only"`
	ProductID             currency.Pair `json:"product_id"`
	ProductType           string        `json:"product_type"`
	RejectReason          string        `json:"reject_reason"`
	RetailPortfolioID     string        `json:"retail_portfolio_id"`
	RiskManagedBy         string        `json:"risk_managed_by"`
	Status                string        `json:"status"`
	StopPrice             types.Number  `json:"stop_price"`
	TimeInForce           string        `json:"time_in_force"`
	TotalFees             types.Number  `json:"total_fees"`
	TotalValueAfterFees   types.Number  `json:"total_value_after_fees"`
	TriggerStatus         string        `json:"trigger_status"`
	CreationTime          time.Time     `json:"creation_time"`
	EndTime               time.Time     `json:"end_time"`
	StartTime             time.Time     `json:"start_time"`
}

// WebsocketPerpData defines a websocket perpetual position response, used in WebsocketPositionStruct
type WebsocketPerpData struct {
	ProductID        currency.Pair `json:"product_id"`
	PortfolioUUID    string        `json:"portfolio_uuid"`
	VWAP             types.Number  `json:"vwap"`
	EntryVWAP        types.Number  `json:"entry_vwap"`
	PositionSide     string        `json:"position_side"`
	MarginType       string        `json:"margin_type"`
	NetSize          types.Number  `json:"net_size"`
	BuyOrderSize     types.Number  `json:"buy_order_size"`
	SellOrderSize    types.Number  `json:"sell_order_size"`
	Leverage         types.Number  `json:"leverage"`
	MarkPrice        types.Number  `json:"mark_price"`
	LiquidationPrice types.Number  `json:"liquidation_price"`
	IMNotional       types.Number  `json:"im_notional"`
	MMNotional       types.Number  `json:"mm_notional"`
	PositionNotional types.Number  `json:"position_notional"`
	UnrealizedPNL    types.Number  `json:"unrealized_pnl"`
	AggregatedPNL    types.Number  `json:"aggregated_pnl"`
}

// WebsocketExpData defines a websocket expiring position response, used in WebsocketPositionStruct
type WebsocketExpData struct {
	ProductID         currency.Pair `json:"product_id"`
	Side              string        `json:"side"`
	NumberOfContracts types.Number  `json:"number_of_contracts"`
	RealizedPNL       types.Number  `json:"realized_pnl"`
	UnrealizedPNL     types.Number  `json:"unrealized_pnl"`
	EntryPrice        types.Number  `json:"entry_price"`
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
	MaxPrecision      types.Number        `json:"max_precision"`
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
	QuoteIncrement         types.Number `json:"quote_increment"`
	BaseIncrement          types.Number `json:"base_increment"`
	DisplayName            string       `json:"display_name"`
	MinMarketFunds         types.Number `json:"min_market_funds"`
	MarginEnabled          bool         `json:"margin_enabled"`
	PostOnly               bool         `json:"post_only"`
	LimitOnly              bool         `json:"limit_only"`
	CancelOnly             bool         `json:"cancel_only"`
	Status                 string       `json:"status"`
	StatusMessage          string       `json:"status_message"`
	TradingDisabled        bool         `json:"trading_disabled"`
	FXStablecoin           bool         `json:"fx_stablecoin"`
	MaxSlippagePercentage  types.Number `json:"max_slippage_percentage"`
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
	RFQVolume24Hour        types.Number `json:"rfq_volume_24hour"`
	RFQVolume30Day         types.Number `json:"rfq_volume_30day"`
	ConversionVolume24Hour types.Number `json:"conversion_volume_24hour"`
	ConversionVolume30Day  types.Number `json:"conversion_volume_30day"`
}

// Auction holds information on an ongoing auction, used as a sub-struct in OrderBookResp and OrderBook
type Auction struct {
	OpenPrice    types.Number `json:"open_price"`
	OpenSize     types.Number `json:"open_size"`
	BestBidPrice types.Number `json:"best_bid_price"`
	BestBidSize  types.Number `json:"best_bid_size"`
	BestAskPrice types.Number `json:"best_ask_price"`
	BestAskSize  types.Number `json:"best_ask_size"`
	AuctionState string       `json:"auction_state"`
	CanOpen      string       `json:"can_open"`
	Time         time.Time    `json:"time"`
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
	Open                    types.Number `json:"open"`
	High                    types.Number `json:"high"`
	Low                     types.Number `json:"low"`
	Last                    types.Number `json:"last"`
	Volume                  types.Number `json:"volume"`
	Volume30Day             types.Number `json:"volume_30day"`
	RFQVolume24Hour         types.Number `json:"rfq_volume_24hour"`
	RFQVolume30Day          types.Number `json:"rfq_volume_30day"`
	ConversionsVolume24Hour types.Number `json:"conversions_volume_24hour"`
	ConversionsVolume30Day  types.Number `json:"conversions_volume_30day"`
}

// ProductTicker holds information on a pair's price and volume, returned by GetProductTicker
type ProductTicker struct {
	Ask               types.Number `json:"ask"`
	Bid               types.Number `json:"bid"`
	Volume            types.Number `json:"volume"`
	TradeID           int32        `json:"trade_id"`
	Price             types.Number `json:"price"`
	Size              types.Number `json:"size"`
	Time              time.Time    `json:"time"`
	RFQVolume         types.Number `json:"rfq_volume"`
	ConversionsVolume types.Number `json:"conversions_volume"`
}

// ProductTrades holds information on a pair's trades, returned by GetProductTrades
type ProductTrades struct {
	TradeID int32        `json:"trade_id"`
	Side    string       `json:"side"`
	Size    types.Number `json:"size"`
	Price   types.Number `json:"price"`
	Time    time.Time    `json:"time"`
}

// WrappedAsset holds information on a wrapped asset, used in AllWrappedAssets and returned by GetWrappedAssetDetails
type WrappedAsset struct {
	ID                string       `json:"id"`
	CirculatingSupply types.Number `json:"circulating_supply"`
	TotalSupply       types.Number `json:"total_supply"`
	ConversionRate    types.Number `json:"conversion_rate"`
	APY               types.Number `json:"apy"`
}

// AllWrappedAssets holds information on all wrapped assets, returned by GetAllWrappedAssets
type AllWrappedAssets struct {
	WrappedAssets []WrappedAsset `json:"wrapped_assets"`
}

// WrappedAssetConversionRate holds information on a wrapped asset's conversion rate, returned by GetWrappedAssetConversionRate
type WrappedAssetConversionRate struct {
	Amount types.Number `json:"amount"`
}

// ManyErrors holds information on errors
type ManyErrors struct {
	Success              bool   `json:"success"`
	FailureReason        string `json:"failure_reason"`
	OrderID              string `json:"order_id"`
	EditFailureReason    string `json:"edit_failure_reason"`
	PreviewFailureReason string `json:"preview_failure_reason"`
}
