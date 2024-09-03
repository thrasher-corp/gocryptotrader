package coinbasepro

import (
	"net/url"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CoinbasePro is the overarching type across the coinbasepro package
type CoinbasePro struct {
	exchange.Base
	jwt          string
	jwtLastRegen time.Time
}

// Version is used for the niche cases where the Version of the API must be specified and passed
// around for proper functionality
type Version bool

// FiatTransferType is used so that we don't need to duplicate the four fiat transfer-related
// endpoints under version 2 of the API
type FiatTransferType bool

// ValCur is a sub-struct used in the types Account, NativeAndRaw, DetailedPortfolioResponse,
// FuturesBalanceSummary, ListFuturesSweepsResponse, PerpetualsPortfolioSummary, PerpPositionDetail,
// FeeStruct, AmScale, and ConvertResponse
type ValCur struct {
	Value    float64 `json:"value,string"`
	Currency string  `json:"currency"`
}

// Account holds details for a trading account, returned by GetAccountByID and used as
// a sub-struct in the types AllAccountsResponse and OneAccountResponse
type Account struct {
	UUID             string    `json:"uuid"`
	Name             string    `json:"name"`
	Currency         string    `json:"currency"`
	AvailableBalance ValCur    `json:"available_balance"`
	Default          bool      `json:"default"`
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	DeletedAt        time.Time `json:"deleted_at"`
	Type             string    `json:"type"`
	Ready            bool      `json:"ready"`
	Hold             ValCur    `json:"hold"`
}

// AllAccountsResponse holds many Account structs, as well as pagination information,
// returned by GetAllAccounts
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

// OneAccountResponse is a temporary struct used for unmarshalling in GetAccountByID
type OneAccountResponse struct {
	Account Account `json:"account"`
}

// PriSiz is a sub-struct used in the type ProductBook
type PriSiz struct {
	Price float64 `json:"price,string"`
	Size  float64 `json:"size,string"`
}

// ProductBook holds bid and ask prices for a particular product, returned by GetProductBookV3
// and used as a sub-struct in the types BestBidAsk and ProductBookResponse
type ProductBook struct {
	ProductID string    `json:"product_id"`
	Bids      []PriSiz  `json:"bids"`
	Asks      []PriSiz  `json:"asks"`
	Time      time.Time `json:"time"`
}

// BestBidAsk holds the best bid and ask prices for a variety of products, used for
// unmarshalling in GetBestBidAsk
type BestBidAsk struct {
	Pricebooks []ProductBook `json:"pricebooks"`
}

// ProductBookResponse is a temporary struct used for unmarshalling in GetProductBookV3
type ProductBookResponse struct {
	Pricebook ProductBook `json:"pricebook"`
}

// Product holds product information, returned by GetProductByID, and used as a sub-struct
// in the type AllProducts
type Product struct {
	ID                        string       `json:"product_id"`
	Price                     types.Number `json:"price"`
	PricePercentageChange24H  types.Number `json:"price_percentage_change_24h"`
	Volume24H                 types.Number `json:"volume_24h"`
	VolumePercentageChange24H types.Number `json:"volume_percentage_change_24h"`
	BaseIncrement             types.Number `json:"base_increment"`
	QuoteIncrement            types.Number `json:"quote_increment"`
	QuoteMinSize              types.Number `json:"quote_min_size"`
	QuoteMaxSize              types.Number `json:"quote_max_size"`
	BaseMinSize               types.Number `json:"base_min_size"`
	BaseMaxSize               types.Number `json:"base_max_size"`
	BaseName                  string       `json:"base_name"`
	QuoteName                 string       `json:"quote_name"`
	Watched                   bool         `json:"watched"`
	IsDisabled                bool         `json:"is_disabled"`
	New                       bool         `json:"new"`
	Status                    string       `json:"status"`
	CancelOnly                bool         `json:"cancel_only"`
	LimitOnly                 bool         `json:"limit_only"`
	PostOnly                  bool         `json:"post_only"`
	TradingDisabled           bool         `json:"trading_disabled"`
	AuctionMode               bool         `json:"auction_mode"`
	ProductType               string       `json:"product_type"`
	QuoteCurrencyID           string       `json:"quote_currency_id"`
	BaseCurrencyID            string       `json:"base_currency_id"`
	FCMTradingSessionDetails  struct {
		IsSessionOpen                bool      `json:"is_session_open"`
		OpenTime                     time.Time `json:"open_time"`
		CloseTime                    time.Time `json:"close_time"`
		SessionState                 string    `json:"session_state"`
		AfterHoursOrderEntryDisabled bool      `json:"after_hours_order_entry_disabled"`
	} `json:"fcm_trading_session_details"`
	MidMarketPrice            types.Number `json:"mid_market_price"`
	Alias                     string       `json:"alias"`
	AliasTo                   []string     `json:"alias_to"`
	BaseDisplaySymbol         string       `json:"base_display_symbol"`
	QuoteDisplaySymbol        string       `json:"quote_display_symbol"`
	ViewOnly                  bool         `json:"view_only"`
	PriceIncrement            types.Number `json:"price_increment"`
	DisplayName               string       `json:"display_name"`
	ProductVenue              string       `json:"product_venue"`
	ApproximateQuote24HVolume types.Number `json:"approximate_quote_24h_volume"`
	FutureProductDetails      struct {
		Venue                  string       `json:"venue"`
		ContractCode           string       `json:"contract_code"`
		ContractExpiry         time.Time    `json:"contract_expiry"`
		ContractSize           types.Number `json:"contract_size"`
		ContractRootUnit       string       `json:"contract_root_unit"`
		GroupDescription       string       `json:"group_description"`
		ContractExpiryTimezone string       `json:"contract_expiry_timezone"`
		GroupShortDescription  string       `json:"group_short_description"`
		RiskManagedBy          string       `json:"risk_managed_by"`
		ContractExpiryType     string       `json:"contract_expiry_type"`
		PerpetualDetails       struct {
			OpenInterest  types.Number `json:"open_interest"`
			FundingRate   types.Number `json:"funding_rate"`
			FundingTime   time.Time    `json:"funding_time"`
			MaxLeverage   types.Number `json:"max_leverage"`
			BaseAssetUUID uuid.UUID    `json:"base_asset_uuid"`
		} `json:"perpetual_details"`
		ContractDisplayName string `json:"contract_display_name"`
		TimeToExpiryMS      uint64 `json:"time_to_expiry_ms,string"`
		NonCrypto           bool   `json:"non_crypto"`
		ContractExpiryName  string `json:"contract_expiry_name"`
	} `json:"future_product_details"`
}

// AllProducts holds information on a lot of available currency pairs, returned by
// GetAllProducts
type AllProducts struct {
	Products    []Product `json:"products"`
	NumProducts int32     `json:"num_products"`
}

// UnixTimestamp is a type used to unmarshal unix timestamps returned from
// the exchange, used in the types History and WebsocketCandle
type UnixTimestamp time.Time

// CandleStruct holds historic trade information, used as a sub-struct in History,
// and returned by GetHistoricRates
type CandleStruct struct {
	Start  UnixTimestamp `json:"start"`
	Low    float64       `json:"low,string"`
	High   float64       `json:"high,string"`
	Open   float64       `json:"open,string"`
	Close  float64       `json:"close,string"`
	Volume float64       `json:"volume,string"`
}

// History holds historic rate information, used for unmarshalling in GetHistoricRates
type History struct {
	Candles []CandleStruct `json:"candles"`
}

// Ticker holds basic ticker information, returned by GetTicker
type Ticker struct {
	Trades []struct {
		TradeID   string       `json:"trade_id"`
		ProductID string       `json:"product_id"`
		Price     float64      `json:"price,string"`
		Size      float64      `json:"size,string"`
		Time      time.Time    `json:"time"`
		Side      string       `json:"side"`
		Bid       types.Number `json:"bid"`
		Ask       types.Number `json:"ask"`
	} `json:"trades"`
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
	BaseSize   types.Number `json:"base_size"`
	LimitPrice types.Number `json:"limit_price"`
	PostOnly   bool         `json:"post_only"`
}

// LimitLimitGTD is a sub-struct used in the type OrderConfiguration
type LimitLimitGTD struct {
	BaseSize   types.Number `json:"base_size"`
	LimitPrice types.Number `json:"limit_price"`
	EndTime    time.Time    `json:"end_time"`
	PostOnly   bool         `json:"post_only"`
}

// StopLimitStopLimitGTC is a sub-struct used in the type OrderConfiguration
type StopLimitStopLimitGTC struct {
	BaseSize      types.Number `json:"base_size"`
	LimitPrice    types.Number `json:"limit_price"`
	StopPrice     types.Number `json:"stop_price"`
	StopDirection string       `json:"stop_direction"`
}

// StopLimitStopLimitGTD is a sub-struct used in the type OrderConfiguration
type StopLimitStopLimitGTD struct {
	BaseSize      types.Number `json:"base_size"`
	LimitPrice    types.Number `json:"limit_price"`
	StopPrice     types.Number `json:"stop_price"`
	EndTime       time.Time    `json:"end_time"`
	StopDirection string       `json:"stop_direction"`
}

// OrderConfiguration is a struct used in the formation of requests in PrepareOrderConfig, and is
// a sub-struct used in the types PlaceOrderResp and GetOrderResponse
type OrderConfiguration struct {
	MarketMarketIOC       *MarketMarketIOC       `json:"market_market_ioc,omitempty"`
	LimitLimitGTC         *LimitLimitGTC         `json:"limit_limit_gtc,omitempty"`
	LimitLimitGTD         *LimitLimitGTD         `json:"limit_limit_gtd,omitempty"`
	StopLimitStopLimitGTC *StopLimitStopLimitGTC `json:"stop_limit_stop_limit_gtc,omitempty"`
	StopLimitStopLimitGTD *StopLimitStopLimitGTD `json:"stop_limit_stop_limit_gtd,omitempty"`
}

// PlaceOrderResp contains information on an order, returned by PlaceOrder
type PlaceOrderResp struct {
	Success         bool   `json:"success"`
	FailureReason   string `json:"failure_reason"`
	OrderID         string `json:"order_id"`
	SuccessResponse struct {
		OrderID       string `json:"order_id"`
		ProductID     string `json:"product_id"`
		Side          string `json:"side"`
		ClientOrderID string `json:"client_oid"`
	} `json:"success_response"`
	OrderConfiguration OrderConfiguration `json:"order_configuration"`
}

// OrderCancelDetail contains information on attempted order cancellations, used as a
// sub-struct by CancelOrdersResp, and returned by CancelOrders
type OrderCancelDetail struct {
	Success       bool   `json:"success"`
	FailureReason string `json:"failure_reason"`
	OrderID       string `json:"order_id"`
}

// CancelOrderResp contains information on attempted order cancellations, used for unmarshalling
// by CancelOrders
type CancelOrderResp struct {
	Results []OrderCancelDetail `json:"results"`
}

// EditOrderPreviewResp contains information on the effects of editing an order,
// returned by EditOrderPreview
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

// SingleOrder provides information on an order in a format that the exchange provides for the GetOrderByID function
type SingleOrder struct {
	Order GetOrderResponse `json:"order"`
}

// GetOrderResponse contains information on an order, returned by GetOrderByID IterativeGetAllOrders, and used in
// GetAllOrdersResp
type GetOrderResponse struct {
	OrderID               string             `json:"order_id"`
	ProductID             string             `json:"product_id"`
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
	EditHistory           []struct {
		Price                  float64   `json:"price,string"`
		Size                   float64   `json:"size,string"`
		ReplaceAcceptTimestamp time.Time `json:"replace_accept_timestamp"`
	} `json:"edit_history"`
	Leverage          types.Number `json:"leverage"`
	MarginType        string       `json:"margin_type"`
	RetailPortfolioID string       `json:"retail_portfolio_id"`
}

// FillResponse contains fill information, returned by GetFills
type FillResponse struct {
	Fills []struct {
		EntryID            string    `json:"entry_id"`
		TradeID            string    `json:"trade_id"`
		OrderID            string    `json:"order_id"`
		TradeTime          time.Time `json:"trade_time"`
		TradeType          string    `json:"trade_type"`
		Price              float64   `json:"price,string"`
		Size               float64   `json:"size,string"`
		Commission         float64   `json:"commission,string"`
		ProductID          string    `json:"product_id"`
		SequenceTimestamp  time.Time `json:"sequence_timestamp"`
		LiquidityIndicator string    `json:"liquidity_indicator"`
		SizeInQuote        bool      `json:"size_in_quote"`
		UserID             string    `json:"user_id"`
		Side               string    `json:"side"`
	} `json:"fills"`
	Cursor string `json:"cursor"`
}

// PreviewOrderResp contains information on the effects of placing an order, returned by
// PreviewOrder
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

// SimplePortfolioData is a sub-struct used in the types AllPortfolioResponse,
// SimplePortfolioResponse, and DetailedPortfolioResponse
type SimplePortfolioData struct {
	Name    string `json:"name"`
	UUID    string `json:"uuid"`
	Type    string `json:"type"`
	Deleted bool   `json:"deleted"`
}

// AllPortfolioResponse contains a brief overview of the user's portfolios, used in unmarshalling
// for GetAllPortfolios
type AllPortfolioResponse struct {
	Portfolios []SimplePortfolioData `json:"portfolios"`
}

// SimplePortfolioResponse contains a small amount of information on a single portfolio.
// Returned by CreatePortfolio and EditPortfolio
type SimplePortfolioResponse struct {
	Portfolio SimplePortfolioData `json:"portfolio"`
}

// MovePortfolioFundsResponse contains the UUIDs of the portfolios involved. Returned by
// MovePortfolioFunds
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
	UserNativeCurrency ValCur `json:"userNativeCurrency"`
	RawCurrency        ValCur `json:"rawCurrency"`
}

// DetailedPortfolioResponse contains a great deal of information on a single portfolio.
// Returned by GetPortfolioByID
type DetailedPortfolioResponse struct {
	Breakdown struct {
		Portfolio         SimplePortfolioData `json:"portfolio"`
		PortfolioBalances struct {
			TotalBalance               ValCur `json:"total_balance"`
			TotalFuturesBalance        ValCur `json:"total_futures_balance"`
			TotalCashEquivalentBalance ValCur `json:"total_cash_equivalent_balance"`
			TotalCryptoBalance         ValCur `json:"total_crypto_balance"`
			FuturesUnrealizedPNL       ValCur `json:"futures_unrealized_pnl"`
			PerpUnrealizedPNL          ValCur `json:"perp_unrealized_pnl"`
		} `json:"portfolio_balances"`
		SpotPositions []struct {
			Asset                 string  `json:"asset"`
			AccountUUID           string  `json:"account_uuid"`
			TotalBalanceFiat      float64 `json:"total_balance_fiat"`
			TotalBalanceCrypto    float64 `json:"total_balance_crypto"`
			AvailableToTreadeFiat float64 `json:"available_to_trade_fiat"`
			Allocation            float64 `json:"allocation"`
			OneDayChange          float64 `json:"one_day_change"`
			CostBasis             ValCur  `json:"cost_basis"`
			AssetImgURL           string  `json:"asset_img_url"`
			IsCash                bool    `json:"is_cash"`
		} `json:"spot_positions"`
		PerpPositions []struct {
			ProductID             string       `json:"product_id"`
			ProductUUID           string       `json:"product_uuid"`
			Symbol                string       `json:"symbol"`
			AssetImageURL         string       `json:"asset_image_url"`
			VWAP                  NativeAndRaw `json:"vwap"`
			PositionSide          string       `json:"position_side"`
			NetSize               float64      `json:"net_size,string"`
			BuyOrderSize          float64      `json:"buy_order_size,string"`
			SellOrderSize         float64      `json:"sell_order_size,string"`
			IMContribution        float64      `json:"im_contribution,string"`
			UnrealizedPNL         NativeAndRaw `json:"unrealized_pnl"`
			MarkPrice             NativeAndRaw `json:"mark_price"`
			LiquidationPrice      NativeAndRaw `json:"liquidation_price"`
			Leverage              float64      `json:"leverage,string"`
			IMNotional            NativeAndRaw `json:"im_notional"`
			MMNotional            NativeAndRaw `json:"mm_notional"`
			PositionNotional      NativeAndRaw `json:"position_notional"`
			MarginType            string       `json:"margin_type"`
			LiquidationBuffer     float64      `json:"liquidation_buffer,string"`
			LiquidationPercentage float64      `json:"liquidation_percentage,string"`
		} `json:"perp_positions"`
		FuturesPositions []struct {
			ProductID       string    `json:"product_id"`
			ContractSize    float64   `json:"contract_size,string"`
			Side            string    `json:"side"`
			Amount          float64   `json:"amount,string"`
			AvgEntryPrice   float64   `json:"avg_entry_price,string"`
			CurrentPrice    float64   `json:"current_price,string"`
			UnrealizedPNL   float64   `json:"unrealized_pnl,string"`
			Expiry          time.Time `json:"expiry"`
			UnderlyingAsset string    `json:"underlying_asset"`
			AssetImgURL     string    `json:"asset_img_url"`
			ProductName     string    `json:"product_name"`
			Venue           string    `json:"venue"`
			NotionalValue   float64   `json:"notional_value,string"`
		} `json:"futures_positions"`
	} `json:"breakdown"`
}

// FuturesBalanceSummary contains information on futures balances, returned by
// GetFuturesBalanceSummary
type FuturesBalanceSummary struct {
	BalanceSummary struct {
		FuturesBuyingPower          ValCur  `json:"futures_buying_power"`
		TotalUSDBalance             ValCur  `json:"total_usd_balance"`
		CBIUSDBalance               ValCur  `json:"cbi_usd_balance"`
		CFMUSDBalance               ValCur  `json:"cfm_usd_balance"`
		TotalOpenOrdersHoldAmount   ValCur  `json:"total_open_orders_hold_amount"`
		UnrealizedPNL               ValCur  `json:"unrealized_pnl"`
		DailyRealizedPNL            ValCur  `json:"daily_realized_pnl"`
		InitialMargin               ValCur  `json:"initial_margin"`
		AvailableMargin             ValCur  `json:"available_margin"`
		LiquidationThreshold        ValCur  `json:"liquidation_threshold"`
		LiquidationBufferAmount     ValCur  `json:"liquidation_buffer_amount"`
		LiquidationBufferPercentage float64 `json:"liquidation_buffer_percentage,string"`
	} `json:"balance_summary"`
}

// FuturesPosition contains information on a single futures position, returned by
// GetFuturesPositionByID and used as a sub-struct in the type AllFuturesPositions
type FuturesPosition struct {
	// This may belong in a struct of its own called "position", requiring a bit
	// more abstraction, but for the moment I'll assume it doesn't
	ProductID         string    `json:"product_id"`
	ExpirationTime    time.Time `json:"expiration_time"`
	Side              string    `json:"side"`
	NumberOfContracts float64   `json:"number_of_contracts,string"`
	CurrentPrice      float64   `json:"current_price,string"`
	AverageEntryPrice float64   `json:"avg_entry_price,string"`
	UnrealizedPNL     float64   `json:"unrealized_pnl,string"`
	DailyRealizedPNL  float64   `json:"daily_realized_pnl,string"`
}

// AllFuturesPositions contains information on all futures positions, used in unmarshalling
// by GetAllFuturesPositions
type AllFuturesPositions struct {
	Positions []FuturesPosition `json:"positions"`
}

// SuccessBool is returned by some endpoints to indicate a failure or a success. Used in
// unmarshalling by EditOrder, ScheduleFuturesSweep, and CancelPendingFuturesSweep
type SuccessBool struct {
	Success bool `json:"success"`
}

// SweepData contains information on pending and processing sweep requests, used as a
// sub-struct in ListFuturesSweepsResponse, and returned by ListFuturesSweeps
type SweepData struct {
	ID              string    `json:"id"`
	RequestedAmount ValCur    `json:"requested_amount"`
	ShouldSweepAll  bool      `json:"should_sweep_all"`
	Status          string    `json:"status"`
	ScheduledTime   time.Time `json:"scheduled_time"`
}

// ListFuturesSweepsResponse contains information on pending and processing sweep
// requests. Used in unmarshalling by ListFuturesSweeps
type ListFuturesSweepsResponse struct {
	Sweeps []SweepData `json:"sweeps"`
}

// PerpetualsPortfolioSummary contains information on perpetuals portfolio balances, used as
// a sub-struct in the types PerpetualPortResponse, PerpPositionDetail, AllPerpPosResponse, and
// OnePerpPosResponse
type PerpetualsPortfolioSummary struct {
	PortfolioUUID              string  `json:"portfolio_uuid"`
	Collateral                 float64 `json:"collateral,string"`
	PositionNotional           float64 `json:"position_notional,string"`
	OpenPositionNotional       float64 `json:"open_position_notional,string"`
	PendingFees                float64 `json:"pending_fees,string"`
	Borrow                     float64 `json:"borrow,string"`
	AccruedInterest            float64 `json:"accrued_interest,string"`
	RollingDebt                float64 `json:"rolling_debt,string"`
	PortfolioInitialMargin     float64 `json:"portfolio_initial_margin,string"`
	PortfolioIMNotional        ValCur  `json:"portfolio_im_notional"`
	PortfolioMaintenanceMargin float64 `json:"portfolio_maintenance_margin,string"`
	PortfolioMMNotional        ValCur  `json:"portfolio_mm_notional"`
	LiquidationPercentage      float64 `json:"liquidation_percentage,string"`
	LiquidationBuffer          float64 `json:"liquidation_buffer,string"`
	MarginType                 string  `json:"margin_type"`
	MarginFlags                string  `json:"margin_flags"`
	LiquidationStatus          string  `json:"liquidation_status"`
	UnrealizedPNL              ValCur  `json:"unrealized_pnl"`
	BuyingPower                ValCur  `json:"buying_power"`
	TotalBalance               ValCur  `json:"total_balance"`
	MaxWithDrawal              ValCur  `json:"max_withdrawal"`
}

// PerpetualPortResponse contains information on perpetuals portfolio balances, returned by
// GetPerpetualsPortfolioSummary
type PerpetualPortResponse struct {
	Summary PerpetualsPortfolioSummary `json:"summary"`
}

// PerpPositionDetail contains information on a single perpetuals position, used as a sub-struct
// in the types AllPerpPosResponse and OnePerpPosResponse
type PerpPositionDetail struct {
	ProductID             string                     `json:"product_id"`
	ProductUUID           string                     `json:"product_uuid"`
	Symbol                string                     `json:"symbol"`
	VWAP                  ValCur                     `json:"vwap"`
	PositionSide          string                     `json:"position_side"`
	NetSize               float64                    `json:"net_size,string"`
	BuyOrderSize          float64                    `json:"buy_order_size,string"`
	SellOrderSize         float64                    `json:"sell_order_size,string"`
	IMContribution        float64                    `json:"im_contribution,string"`
	UnrealizedPNL         ValCur                     `json:"unrealized_pnl"`
	MarkPrice             ValCur                     `json:"mark_price"`
	LiquidationPrice      ValCur                     `json:"liquidation_price"`
	Leverage              float64                    `json:"leverage,string"`
	IMNotional            ValCur                     `json:"im_notional"`
	MMNotional            ValCur                     `json:"mm_notional"`
	PositionNotional      ValCur                     `json:"position_notional"`
	MarginType            string                     `json:"margin_type"`
	LiquidationBuffer     float64                    `json:"liquidation_buffer,string"`
	LiquidationPercentage float64                    `json:"liquidation_percentage,string"`
	PortfolioSummary      PerpetualsPortfolioSummary `json:"portfolio_summary"`
}

// AllPerpPosResponse contains information on perpetuals positions, returned by
// GetAllPerpetualsPositions
type AllPerpPosResponse struct {
	Positions        []PerpPositionDetail       `json:"positions"`
	PortfolioSummary PerpetualsPortfolioSummary `json:"portfolio_summary"`
}

// OnePerpPosResponse contains information on a single perpetuals position, returned by
// GetPerpetualsPositionByID
type OnePerpPosResponse struct {
	Position         PerpPositionDetail         `json:"position"`
	PortfolioSummary PerpetualsPortfolioSummary `json:"portfolio_summary"`
}

// TransactionSummary contains a summary of transaction fees, volume, and the like. Returned
// by GetTransactionSummary
type TransactionSummary struct {
	TotalVolume float64 `json:"total_volume"`
	TotalFees   float64 `json:"total_fees"`
	FeeTier     struct {
		PricingTier  string       `json:"pricing_tier"`
		USDFrom      float64      `json:"usd_from,string"`
		USDTo        float64      `json:"usd_to,string"`
		TakerFeeRate float64      `json:"taker_fee_rate,string"`
		MakerFeeRate float64      `json:"maker_fee_rate,string"`
		AOPFrom      types.Number `json:"aop_from"`
		AOPTo        types.Number `json:"aop_to"`
	} `json:"fee_tier"`
	MarginRate struct {
		Value float64 `json:"value,string"`
	} `json:"margin_rate"`
	GoodsAndServicesTax struct {
		Rate float64 `json:"rate,string"`
		Type string  `json:"type"`
	} `json:"goods_and_services_tax"`
	AdvancedTradeOnlyVolume float64      `json:"advanced_trade_only_volume"`
	AdvancedTradeOnlyFees   float64      `json:"advanced_trade_only_fees"`
	CoinbaseProVolume       float64      `json:"coinbase_pro_volume"`
	CoinbaseProFees         float64      `json:"coinbase_pro_fees"`
	TotalBalance            types.Number `json:"total_balance"`
	HasPromoFee             bool         `json:"has_promo_fee"`
}

// GetAllOrdersResp contains information on a lot of orders, returned by GetAllOrders
type GetAllOrdersResp struct {
	Orders   []GetOrderResponse `json:"orders"`
	Sequence int64              `json:"sequence,string"`
	HasNext  bool               `json:"has_next"`
	Cursor   string             `json:"cursor"`
}

// LinkStruct is a sub-struct storing information on links, used in FeeStruct and
// ConvertResponse
type LinkStruct struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

// FeeStruct is a sub-struct storing information on fees, used in ConvertResponse
type FeeStruct struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Amount      ValCur `json:"amount"`
	Label       string `json:"label"`
	Disclosure  struct {
		Title       string     `json:"title"`
		Description string     `json:"description"`
		Link        LinkStruct `json:"link"`
	} `json:"disclosure"`
}

// AccountStruct is a sub-struct storing information on accounts, used in ConvertResponse
type AccountStruct struct {
	Type          string `json:"type"`
	Network       string `json:"network"`
	LedgerAccount struct {
		AccountID string `json:"account_id"`
		Currency  string `json:"currency"`
		Owner     struct {
			ID       string `json:"id"`
			UUID     string `json:"uuid"`
			UserUUID string `json:"user_uuid"`
			Type     string `json:"type"`
		} `json:"owner"`
	} `json:"ledger_account"`
}

// AmScale is a sub-struct storing information on amounts and scales, used in ConvertResponse
type AmScale struct {
	Amount ValCur `json:"amount"`
	Scale  int32  `json:"scale"`
}

// ConvertResponse contains information on a convert trade, returned by CreateConvertQuote,
// CommitConvertTrade, and GetConvertTradeByID
type ConvertResponse struct {
	Trade struct {
		ID                string        `json:"id"`
		Status            string        `json:"status"`
		UserEnteredAmount ValCur        `json:"user_entered_amount"`
		Amount            ValCur        `json:"amount"`
		Subtotal          ValCur        `json:"subtotal"`
		Total             ValCur        `json:"total"`
		Fees              []FeeStruct   `json:"fees"`
		TotalFee          FeeStruct     `json:"total_fee"`
		Source            AccountStruct `json:"source"`
		Target            AccountStruct `json:"target"`
		UnitPrice         struct {
			TargetToFiat   AmScale `json:"target_to_fiat"`
			TargetToSource AmScale `json:"target_to_source"`
			SourceToFiat   AmScale `json:"source_to_fiat"`
		} `json:"unit_price"`
		UserWarnings []struct {
			ID      string     `json:"id"`
			Link    LinkStruct `json:"link"`
			Context struct {
				Details  []string `json:"details"`
				Title    string   `json:"title"`
				LinkText string   `json:"link_text"`
			} `json:"context"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"user_warnings"`
		UserReference      string `json:"user_reference"`
		SourceCurrency     string `json:"source_currency"`
		TargetCurrency     string `json:"target_currency"`
		CancellationReason struct {
			Message   string `json:"message"`
			Code      string `json:"code"`
			ErrorCode string `json:"error_code"`
			ErrorCTA  string `json:"error_cta"`
		} `json:"cancellation_reason"`
		SourceID     string `json:"source_id"`
		TargetID     string `json:"target_id"`
		ExchangeRate ValCur `json:"exchange_rate"`
		TaxDetails   []struct {
			Name   string `json:"name"`
			Amount ValCur `json:"amount"`
		} `json:"tax_details"`
		TradeIncentiveInfo struct {
			AppliedIncentive    bool      `json:"applied_incentive"`
			UserIncentiveID     string    `json:"user_incentive_id"`
			CodeVal             string    `json:"code_val"`
			EndsAt              time.Time `json:"ends_at"`
			FeeWithoutIncentive ValCur    `json:"fee_without_incentive"`
			Redeemed            bool      `json:"redeemed"`
		} `json:"trade_incentive_info"`
		TotalFeeWithoutTax FeeStruct `json:"total_fee_without_tax"`
		FiatDenotedTotal   ValCur    `json:"fiat_denoted_total"`
	} `json:"trade"`
}

// ServerTimeV3 holds information on the server's time, returned by GetV3Time
type ServerTimeV3 struct {
	Iso               time.Time `json:"iso"`
	EpochSeconds      int64     `json:"epochSeconds,string"`
	EpochMilliseconds int64     `json:"epochMillis,string"`
}

// PaymentMethodData is a sub-type that holds information on a payment method. Used in
// GetAllPaymentMethodsResp and GenPaymentMethodResp
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

// GetAllPaymentMethodsResp holds information on many payment methods. Returned by
// GetAllPaymentMethods
type GetAllPaymentMethodsResp struct {
	PaymentMethods []PaymentMethodData `json:"payment_methods"`
}

// GenPaymentMethodResp holds information on a payment method. Returned by
// GetPaymentMethodByID
type GenPaymentMethodResp struct {
	PaymentMethod PaymentMethodData `json:"payment_method"`
}

// IDResource holds an ID, resource type, and associated data, used in ListNotificationsResponse,
// TransactionData, DeposWithdrData, and PaymentMethodData
type IDResource struct {
	ID           string `json:"id"`
	Resource     string `json:"resource"`
	ResourcePath string `json:"resource_path"`
	Email        string `json:"email"`
}

// PaginationResp holds pagination information, used in ListNotificationsResponse,
// GetAllWalletsResponse, GetAllAddrResponse, ManyTransactionsResp, ManyDeposWithdrResp,
// and GetAllPaymentMethodsResp
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

// PaginationInp holds information needed to engage in pagination with Sign in With
// Coinbase. Used in ListNotifications, GetAllWallets, GetAllAddresses, GetAddressTransactions,
// GetAllTransactions, GetAllFiatTransfers, GetAllPaymentMethods, and preparePagination
type PaginationInp struct {
	Limit         uint8
	OrderAscend   bool
	StartingAfter string
	EndingBefore  string
}

// AmCur is a sub-struct used in ListNotificationsResponse, WalletData, TransactionData,
// DeposWithdrData, and PaymentMethodData
type AmCur struct {
	Amount   float64 `json:"amount,string"`
	Currency string  `json:"currency"`
}

// ListNotificationsResponse holds information on notifications that the user is subscribed
// to. Returned by ListNotifications
type ListNotificationsResponse struct {
	Pagination PaginationResp `json:"pagination"`
	Data       []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			ID            string     `json:"id"`
			Address       string     `json:"address"`
			Name          string     `json:"name"`
			Status        string     `json:"status"`
			PaymentMethod IDResource `json:"payment_method"`
			Transaction   IDResource `json:"transaction"`
			Amount        AmCur      `json:"amount"`
			Total         AmCur      `json:"total"`
			Subtotal      AmCur      `json:"subtotal"`
			CreatedAt     time.Time  `json:"created_at"`
			UpdatedAt     time.Time  `json:"updated_at"`
			Resource      string     `json:"resource"`
			ResourcePath  string     `json:"resource_path"`
			Committed     bool       `json:"committed"`
			Instant       bool       `json:"instant"`
			Fee           AmCur      `json:"fee"`
			Fees          []struct {
				Type   string `json:"type"`
				Amount AmCur  `json:"amount"`
			} `json:"fees"`
			PayoutAt time.Time `json:"payout_at"`
		} `json:"data"`
		AdditionalData struct {
			Hash   string `json:"hash"`
			Amount AmCur  `json:"amount"`
		} `json:"additional_data"`
		User             IDResource `json:"user"`
		Account          IDResource `json:"account"`
		DeliveryAttempts int32      `json:"delivery_attempts"`
		CreatedAt        time.Time  `json:"created_at"`
		Resource         string     `json:"resource"`
		ResourcePath     string     `json:"resource_path"`
		Transaction      IDResource `json:"transaction"`
	} `json:"data"`
}

// CodeName is a sub-struct holding a code and a name, used in UserResponse
type CodeName struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// UserResponse holds information on a user, returned by GetUserByID and GetCurrentUser
type UserResponse struct {
	Data struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		Username        string `json:"username"`
		ProfileLocation string `json:"profile_location"`
		ProfileBio      string `json:"profile_bio"`
		ProfileURL      string `json:"profile_url"`
		AvatarURL       string `json:"avatar_url"`
		Resource        string `json:"resource"`
		ResourcePath    string `json:"resource_path"`
		LegacyID        string `json:"legacy_id"`
		TimeZone        string `json:"time_zone"`
		NativeCurrency  string `json:"native_currency"`
		BitcoinUnit     string `json:"bitcoin_unit"`
		State           string `json:"state"`
		Country         struct {
			Code       string `json:"code"`
			Name       string `json:"name"`
			IsInEurope bool   `json:"is_in_europe"`
		} `json:"country"`
		Nationality                           CodeName  `json:"nationality"`
		RegionSupportsFiatTransfers           bool      `json:"region_supports_fiat_transfers"`
		RegionSupportsCryptoToCryptoTransfers bool      `json:"region_supports_crypto_to_crypto_transfers"`
		CreatedAt                             time.Time `json:"created_at"`
		SupportsRewards                       bool      `json:"supports_rewards"`
		Tiers                                 struct {
			CompletedDescription string `json:"completed_description"`
			UpgradeButtonText    string `json:"upgrade_button_text"`
			Header               string `json:"header"`
			Body                 string `json:"body"`
		} `json:"tiers"`
		ReferralMoney struct {
			Amount            float64 `json:"amount,string"`
			Currency          string  `json:"currency"`
			CurrencySymbol    string  `json:"currency_symbol"`
			ReferralThreshold float64 `json:"referral_threshold,string"`
		} `json:"referral_money"`
		HasBlockingBuyRestrictions            bool   `json:"has_blocking_buy_restrictions"`
		HasMadeAPurchase                      bool   `json:"has_made_a_purchase"`
		HasBuyDepositPaymentMethods           bool   `json:"has_buy_deposit_payment_methods"`
		HasUnverifiedBuyDepositPaymentMethods bool   `json:"has_unverified_buy_deposit_payment_methods"`
		NeedsKYCRemediation                   bool   `json:"needs_kyc_remediation"`
		ShowInstantAchUx                      bool   `json:"show_instant_ach_ux"`
		UserType                              string `json:"user_type"`
		Email                                 string `json:"email"`
		SendsDisabled                         bool   `json:"sends_disabled"`
	} `json:"data"`
}

// WalletData is a sub-struct holding wallet information, used in GenWalletResponse and GetAllWalletsResponse
type WalletData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Primary  bool   `json:"primary"`
	Type     string `json:"type"`
	Currency struct {
		Code                string      `json:"code"`
		Name                string      `json:"name"`
		Color               string      `json:"color"`
		SortIndex           int32       `json:"sort_index"`
		Exponent            int32       `json:"exponent"`
		Type                string      `json:"type"`
		AddressRegex        string      `json:"address_regex"`
		AssetID             string      `json:"asset_id"`
		DestinationTagName  string      `json:"destination_tag_name"`
		DestinationTagRegex string      `json:"destination_tag_regex"`
		Slug                string      `json:"slug"`
		Rewards             interface{} `json:"rewards"`
	} `json:"currency"`
	Balance          AmCur     `json:"balance"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Resource         string    `json:"resource"`
	ResourcePath     string    `json:"resource_path"`
	AllowDeposits    bool      `json:"allow_deposits"`
	AllowWithdrawals bool      `json:"allow_withdrawals"`
}

// GenWalletResponse holds information on a single wallet, returned by GetWalletByID
type GenWalletResponse struct {
	Data WalletData `json:"data"`
}

// GetAllWalletsResponse holds information on many wallets, returned by GetAllWallets
type GetAllWalletsResponse struct {
	Pagination *PaginationResp `json:"pagination"`
	Data       []WalletData    `json:"data"`
}

// AddressInfo holds an address and a destination tag, used in AddressData
type AddressInfo struct {
	Address        string `json:"address"`
	DestinationTag string `json:"destination_tag"`
}

// TitleSubtitle holds a title and a subtitle, used in AddressData and TransactionData
type TitleSubtitle struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

// AddressData holds address information, used in GenAddrResponse and GetAllAddrResponse
type AddressData struct {
	ID           string      `json:"id"`
	Address      string      `json:"address"`
	AddressInfo  AddressInfo `json:"address_info"`
	Name         string      `json:"name"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Network      string      `json:"network"`
	URIScheme    string      `json:"uri_scheme"`
	Resource     string      `json:"resource"`
	ResourcePath string      `json:"resource_path"`
	Warnings     []struct {
		Type     string `json:"type"`
		Title    string `json:"title"`
		Details  string `json:"details"`
		ImageURL string `json:"image_url"`
		Options  []struct {
			Text  string `json:"text"`
			Style string `json:"style"`
			ID    string `json:"id"`
		} `json:"options"`
	} `json:"warnings"`
	QRCodeImageURL   string `json:"qr_code_image_url"`
	AddressLabel     string `json:"address_label"`
	DefaultReceive   bool   `json:"default_receive"`
	DestinationTag   string `json:"destination_tag"`
	DepositURI       string `json:"deposit_uri"`
	CallbackURL      string `json:"callback_url"`
	ShareAddressCopy struct {
		Line1 string `json:"line1"`
		Line2 string `json:"line2"`
	} `json:"share_address_copy"`
	ReceiveSubtitle string `json:"receive_subtitle"`
	InlineWarning   struct {
		Text    string        `json:"text"`
		Tooltip TitleSubtitle `json:"tooltip"`
	} `json:"inline_warning"`
}

// GenAddrResponse holds information on a generated address, returned by CreateAddress and
// GetAddressByID
type GenAddrResponse struct {
	Data AddressData `json:"data"`
}

// GetAllAddrResponse holds information on many addresses, returned by GetAllAddresses
type GetAllAddrResponse struct {
	Pagination PaginationResp `json:"pagination"`
	Data       []AddressData  `json:"data"`
}

// TransactionData is a sub-type that holds information on a transaction. Used in
// ManyTransactionsResp and GenTransactionResp
type TransactionData struct {
	ID                string     `json:"id"`
	Type              string     `json:"type"`
	Status            string     `json:"status"`
	Amount            AmCur      `json:"amount"`
	NativeAmount      AmCur      `json:"native_amount"`
	Description       string     `json:"description"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Resource          string     `json:"resource"`
	ResourcePath      string     `json:"resource_path"`
	InstantExchange   bool       `json:"instant_exchange"`
	Buy               IDResource `json:"buy"`
	AdvancedTradeFill struct {
		FillPrice  float64 `json:"fill_price,string"`
		ProductID  string  `json:"product_id"`
		OrderID    string  `json:"order_id"`
		Commission float64 `json:"commission,string"`
		OrderSide  string  `json:"order_side"`
	} `json:"advanced_trade_fill"`
	Details TitleSubtitle `json:"details"`
	Network struct {
		Status string `json:"status"`
		Hash   string `json:"hash"`
		Name   string `json:"name"`
	} `json:"network"`
	To      IDResource `json:"to"`
	From    IDResource `json:"from"`
	Address struct {
	} `json:"address"`
	Application struct {
	} `json:"application"`
}

// ManyTransactionsResp holds information on many transactions. Returned by
// GetAddressTransactions and GetAllTransactions
type ManyTransactionsResp struct {
	Pagination PaginationResp    `json:"pagination"`
	Data       []TransactionData `json:"data"`
}

// GenTransactionResp holds information on one transaction. Returned by SendMoney and GetTransactionByID
type GenTransactionResp struct {
	Data TransactionData `json:"data"`
}

// DeposWithdrData is a sub-type that holds information on a deposit/withdrawal. Used in
// GenDeposWithdrResp and ManyDeposWithdrResp
type DeposWithdrData struct {
	ID            string     `json:"id"`
	Status        string     `json:"status"`
	PaymentMethod IDResource `json:"payment_method"`
	Transaction   IDResource `json:"transaction"`
	Amount        AmCur      `json:"amount"`
	Subtotal      AmCur      `json:"subtotal"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Resource      string     `json:"resource"`
	ResourcePath  string     `json:"resource_path"`
	Committed     bool       `json:"committed"`
	Fee           AmCur      `json:"fee"`
	PayoutAt      time.Time  `json:"payout_at"`
	TransferType  FiatTransferType
}

// GenDeposWithdrResp holds information on a deposit. Returned by FiatTransfer, CommitTransfer, and
// GetFiatTransferByID
type GenDeposWithdrResp struct {
	Data DeposWithdrData `json:"data"`
}

// ManyDeposWithdrResp holds information on many deposits. Returned by GetAllFiatTransfers
type ManyDeposWithdrResp struct {
	Pagination PaginationResp    `json:"pagination"`
	Data       []DeposWithdrData `json:"data"`
}

// FiatData holds information on fiat currencies. Used as a sub-struct in
// GetFiatCurrenciesResp, and returned by GetFiatCurrencies
type FiatData struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	MinSize float64 `json:"min_size,string"`
}

// GetFiatCurrenciesResp holds information on fiat currencies. Used for
// unmarshalling in GetFiatCurrencies
type GetFiatCurrenciesResp struct {
	Data []FiatData `json:"data"`
}

// CryptoData holds information on cryptocurrencies. Used as a sub-struct in
// GetCryptocurrenciesResp, and returned by GetCryptocurrencies
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

// GetCryptocurrenciesResp holds information on cryptocurrencies. Used for
// unmarshalling in GetCryptocurrencies
type GetCryptocurrenciesResp struct {
	Data []CryptoData `json:"data"`
}

// GetExchangeRatesResp holds information on exchange rates. Returned by GetExchangeRates
type GetExchangeRatesResp struct {
	Data struct {
		Currency string                  `json:"currency"`
		Rates    map[string]types.Number `json:"rates"`
	} `json:"data"`
}

// GetPriceResp holds information on a price. Returned by GetPrice
type GetPriceResp struct {
	Data struct {
		Amount   float64 `json:"amount,string"`
		Base     string  `json:"base"`
		Currency string  `json:"currency"`
	} `json:"data"`
}

// ServerTimeV2 holds current requested server time information, returned by GetV2Time
type ServerTimeV2 struct {
	Data struct {
		ISO   time.Time `json:"iso"`
		Epoch uint64    `json:"epoch"`
	} `json:"data"`
}

// WebsocketRequest is an aspect of constructing a request to the websocket server, used in sendRequest
type WebsocketRequest struct {
	Type       string   `json:"type"`
	ProductIDs []string `json:"product_ids,omitempty"`
	Channel    string   `json:"channel,omitempty"`
	Signature  string   `json:"signature,omitempty"`
	Key        string   `json:"api_key,omitempty"`
	Timestamp  string   `json:"timestamp,omitempty"`
	JWT        string   `json:"jwt,omitempty"`
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
}

// WebsocketTickerHolder holds a variety of ticker responses, used when wsHandleData processes tickers
type WebsocketTickerHolder struct {
	Type    string            `json:"type"`
	Tickers []WebsocketTicker `json:"tickers"`
}

// WebsocketCandle defines a candle websocket response, used in WebsocketCandleHolder
type WebsocketCandle struct {
	Start     UnixTimestamp `json:"start"`
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

// WebsocketMarketTradeHolder holds a variety of market trade responses, used when wsHandleData
// processes trades
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

// WebsocketProductHolder holds a variety of product responses, used when wsHandleData processes
// an update on a product's status
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

// WebsocketOrderbookDataHolder holds a variety of orderbook responses, used when wsHandleData processes
// orderbooks, as well as under typical operation of ProcessSnapshot, ProcessUpdate, and processBidAskArray
type WebsocketOrderbookDataHolder struct {
	Type      string                   `json:"type"`
	ProductID currency.Pair            `json:"product_id"`
	Changes   []WebsocketOrderbookData `json:"updates"`
}

// WebsocketOrderData defines a websocket order response, used in WebsocketOrderDataHolder
type WebsocketOrderData struct {
	OrderID            string        `json:"order_id"`
	ClientOrderID      string        `json:"client_order_id"`
	CumulativeQuantity float64       `json:"cumulative_quantity,string"`
	LeavesQuantity     float64       `json:"leaves_quantity,string"`
	AveragePrice       float64       `json:"avg_price,string"`
	TotalFees          float64       `json:"total_fees,string"`
	Status             string        `json:"status"`
	ProductID          currency.Pair `json:"product_id"`
	CreationTime       time.Time     `json:"creation_time"`
	OrderSide          string        `json:"order_side"`
	OrderType          string        `json:"order_type"`
}

// WebsocketOrderDataHolder holds a variety of order responses, used when wsHandleData processes orders
type WebsocketOrderDataHolder struct {
	Type   string               `json:"type"`
	Orders []WebsocketOrderData `json:"orders"`
}

// CurrencyData contains information on known currencies, used in GetAllCurrencies and GetACurrency
type CurrencyData struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	MinSize       string   `json:"min_size"`
	Status        string   `json:"status"`
	Message       string   `json:"message"`
	MaxPrecision  float64  `json:"max_precision,string"`
	ConvertibleTo []string `json:"convertible_to"`
	Details       struct {
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
	} `json:"details"`
	DefaultNetwork    string `json:"default_network"`
	SupportedNetworks []struct {
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
	} `json:"supported_networks"`
	DisplayName string `json:"display_name"`
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

// OrderBookResp holds information on bids and asks for a particular currency pair, used for unmarshalling in
// GetProductBookV1
type OrderBookResp struct {
	Bids        [][3]interface{} `json:"bids"`
	Asks        [][3]interface{} `json:"asks"`
	Sequence    float64          `json:"sequence"`
	AuctionMode bool             `json:"auction_mode"`
	Auction     Auction          `json:"auction"`
	Time        time.Time        `json:"time"`
}

// Orders holds information on orders, used as a sub-struct in OrderBook
type Orders struct {
	Price      float64
	Size       float64
	OrderCount uint64
	OrderID    uuid.UUID
}

// OrderBook holds information on bids and asks for a particular currency pair, used in GetProductBookV1
type OrderBook struct {
	Bids        []Orders
	Asks        []Orders
	Sequence    float64
	AuctionMode bool
	Auction     Auction
	Time        time.Time
}

// RawCandles holds raw candle data, used in unmarshalling for GetProductCandles
type RawCandles [6]interface{}

// Candle holds properly formatted candle data, returned by GetProductCandles
type Candle struct {
	Time   time.Time
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

// WrappedAssetConversionRate holds information on a wrapped asset's conversion rate, returned by
// GetWrappedAssetConversionRate
type WrappedAssetConversionRate struct {
	Amount float64 `json:"amount,string"`
}
