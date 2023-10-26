package coinbaseinternational

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// AssetItemInfo represents a single an asset item instance.
type AssetItemInfo struct {
	AssetID                  string  `json:"asset_id"`
	AssetUUID                string  `json:"asset_uuid"`
	AssetName                string  `json:"asset_name"`
	Status                   string  `json:"status"`
	CollateralWeight         float64 `json:"collateral_weight"`
	SupportedNetworksEnabled bool    `json:"supported_networks_enabled"`
}

// AssetInfoWithSupportedNetwork represents network information for a specific asset.
type AssetInfoWithSupportedNetwork struct {
	AssetID             string                  `json:"asset_id"`
	AssetUUID           string                  `json:"asset_uuid"`
	AssetName           string                  `json:"asset_name"`
	NetworkName         string                  `json:"network_name"`
	DisplayName         string                  `json:"display_name"`
	NetworkArnID        string                  `json:"network_arn_id"`
	MinWithdrawalAmount convert.StringToFloat64 `json:"min_withdrawal_amt"`
	MaxWithdrawalAmount convert.StringToFloat64 `json:"max_withdrawal_amt"`
	NetworkConfirms     int64                   `json:"network_confirms"`
	ProcessingTime      convert.ExchangeTime    `json:"processing_time"`
	IsDefault           bool                    `json:"is_default"`
}

// InstrumentInfo represents an instrument detail for specific instrument id.
type InstrumentInfo struct {
	InstrumentID        string  `json:"instrument_id"`
	InstrumentUUID      string  `json:"instrument_uuid"`
	Symbol              string  `json:"symbol"`
	Type                string  `json:"type"`
	BaseAssetID         string  `json:"base_asset_id"`
	BaseAssetUUID       string  `json:"base_asset_uuid"`
	BaseAssetName       string  `json:"base_asset_name"`
	QuoteAssetID        string  `json:"quote_asset_id"`
	QuoteAssetUUID      string  `json:"quote_asset_uuid"`
	QuoteAssetName      string  `json:"quote_asset_name"`
	BaseIncrement       string  `json:"base_increment"`
	QuoteIncrement      string  `json:"quote_increment"`
	PriceBandPercent    float64 `json:"price_band_percent"`
	MarketOrderPercent  float64 `json:"market_order_percent"`
	Qty24Hr             string  `json:"qty_24hr"`
	Notional24Hr        string  `json:"notional_24hr"`
	AvgDailyQty         string  `json:"avg_daily_qty"`
	AvgDailyNotional    string  `json:"avg_daily_notional"`
	PreviousDayQty      string  `json:"previous_day_qty"`
	OpenInterest        string  `json:"open_interest"`
	PositionLimitQty    string  `json:"position_limit_qty"`
	PositionLimitAdqPct float64 `json:"position_limit_adq_pct"`
	ReplacementCost     string  `json:"replacement_cost"`
	BaseImf             float64 `json:"base_imf"`
	MinNotionalValue    string  `json:"min_notional_value"`
	FundingInterval     string  `json:"funding_interval"`
	TradingState        string  `json:"trading_state"`
	PositionLimitAdv    float64 `json:"position_limit_adv"`
	InitialMarginAdv    float64 `json:"initial_margin_adv"`
}

// QuoteInformation represents a instrument quote information
type QuoteInformation struct {
	BestBidPrice     convert.StringToFloat64 `json:"best_bid_price"`
	BestBidSize      convert.StringToFloat64 `json:"best_bid_size"`
	BestAskPrice     convert.StringToFloat64 `json:"best_ask_price"`
	BestAskSize      convert.StringToFloat64 `json:"best_ask_size"`
	TradePrice       convert.StringToFloat64 `json:"trade_price"`
	TradeQty         convert.StringToFloat64 `json:"trade_qty"`
	IndexPrice       convert.StringToFloat64 `json:"index_price"`
	MarkPrice        convert.StringToFloat64 `json:"mark_price"`
	SettlementPrice  convert.StringToFloat64 `json:"settlement_price"`
	LimitUp          convert.StringToFloat64 `json:"limit_up"`
	LimitDown        convert.StringToFloat64 `json:"limit_down"`
	PredictedFunding convert.StringToFloat64 `json:"predicted_funding"`
	Timestamp        time.Time               `json:"timestamp"`
}

// OrderRequestParams represents a request parameter for creating order.
type OrderRequestParams struct {
	ClientOrderID string  `json:"client_order_id"`
	Side          string  `json:"side,omitempty"`
	BaseSize      float64 `json:"size,omitempty,string"`
	TimeInForce   string  `json:"tif,omitempty"`        // Possible values: [GTC, IOC, GTT]
	Instrument    string  `json:"instrument,omitempty"` // The name, ID, or UUID of the instrument the order wants to transact
	OrderType     string  `json:"type,omitempty"`
	Price         float64 `json:"price,omitempty,string"`
	StopPrice     float64 `json:"stop_price,omitempty,string"`
	ExpireTime    string  `json:"expire_time,omitempty"` // e.g., 2023-03-16T23:59:53Z
	Portfolio     string  `json:"portfolio,omitempty"`
	User          string  `json:"user,omitempty"`     // The ID or UUID of the user the order belongs to (only used and required for brokers)
	STPMode       string  `json:"stp_mode,omitempty"` // Possible values: [NONE, AGGRESSING, BOTH]
	PostOnly      bool    `json:"post_only,omitempty"`
}

// TradeOrder represents a single order
type TradeOrder struct {
	OrderID        int64                   `json:"order_id"`
	ClientOrderID  string                  `json:"client_order_id"`
	Side           string                  `json:"side"`
	InstrumentID   int64                   `json:"instrument_id"`
	InstrumentUUID string                  `json:"instrument_uuid"`
	Symbol         string                  `json:"symbol"`
	PortfolioID    int64                   `json:"portfolio_id"`
	PortfolioUUID  string                  `json:"portfolio_uuid"`
	Type           string                  `json:"type"`
	Price          float64                 `json:"price"`
	StopPrice      float64                 `json:"stop_price"`
	Size           float64                 `json:"size"`
	TimeInForce    string                  `json:"tif"`
	ExpireTime     time.Time               `json:"expire_time"`
	StpMode        string                  `json:"stp_mode"`
	EventType      string                  `json:"event_type"`
	OrderStatus    string                  `json:"order_status"`
	LeavesQty      convert.StringToFloat64 `json:"leaves_qty"`
	ExecQty        convert.StringToFloat64 `json:"exec_qty"`
	AvgPrice       convert.StringToFloat64 `json:"avg_price"`
	Message        string                  `json:"message"`
	Fee            convert.StringToFloat64 `json:"fee"`
}

// OrderItemDetail represents an open order detail.
type OrderItemDetail struct {
	Pagination struct {
		RefDatetime  time.Time `json:"ref_datetime"`
		ResultLimit  int64     `json:"result_limit"`
		ResultOffset int64     `json:"result_offset"`
	} `json:"pagination"`
	Results []OrderItem `json:"results"`
}

// ModifyOrderParam holds update parameters to modify an order.
type ModifyOrderParam struct {
	ClientOrderID string  `json:"client_order_id,omitempty"`
	Portfolio     string  `json:"portfolio,omitempty"`
	Price         float64 `json:"price,omitempty,string"`
	StopPrice     float64 `json:"stop_price,omitempty,string"`
	Size          float64 `json:"size,omitempty,string"`
}

// OrderItem represents a single order item.
type OrderItem struct {
	OrderID        string                  `json:"order_id"`
	ClientOrderID  string                  `json:"client_order_id"`
	Side           string                  `json:"side"`
	InstrumentID   string                  `json:"instrument_id"`
	InstrumentUUID string                  `json:"instrument_uuid"`
	Symbol         string                  `json:"symbol"`
	PortfolioID    int64                   `json:"portfolio_id"`
	PortfolioUUID  string                  `json:"portfolio_uuid"`
	Type           string                  `json:"type"`
	Price          float64                 `json:"price"`
	StopPrice      float64                 `json:"stop_price"`
	Size           float64                 `json:"size"`
	TimeInForce    string                  `json:"tif"`
	ExpireTime     time.Time               `json:"expire_time"`
	StpMode        string                  `json:"stp_mode"`
	EventType      string                  `json:"event_type"`
	OrderStatus    string                  `json:"order_status"`
	LeavesQuantity convert.StringToFloat64 `json:"leaves_qty"`
	ExecQty        convert.StringToFloat64 `json:"exec_qty"`
	AveragePrice   convert.StringToFloat64 `json:"avg_price"`
	Message        string                  `json:"message"`
	Fee            convert.StringToFloat64 `json:"fee"`
}

// PortfolioItem represents a user portfolio item
// and transaction fee information.
type PortfolioItem struct {
	PortfolioID    string                  `json:"portfolio_id"`
	PortfolioUUID  string                  `json:"portfolio_uuid"`
	Name           string                  `json:"name"`
	UserUUID       string                  `json:"user_uuid"`
	MakerFeeRate   convert.StringToFloat64 `json:"maker_fee_rate"`
	TakerFeeRate   convert.StringToFloat64 `json:"taker_fee_rate"`
	TradingLock    bool                    `json:"trading_lock"`
	BorrowDisabled bool                    `json:"borrow_disabled"`
	IsLSP          bool                    `json:"is_lsp"` // Indicates if the portfolio is setup to take liquidation assignments
}

// PortfolioDetail represents a portfolio detail.
type PortfolioDetail struct {
	Summary struct {
		Collateral             float64 `json:"collateral"`
		UnrealizedPnl          float64 `json:"unrealized_pnl"`
		PositionNotional       float64 `json:"position_notional"`
		OpenPositionNotional   float64 `json:"open_position_notional"`
		PendingFees            float64 `json:"pending_fees"`
		Borrow                 float64 `json:"borrow"`
		AccruedInterest        float64 `json:"accrued_interest"`
		RollingDebt            float64 `json:"rolling_debt"`
		Balance                float64 `json:"balance"`
		BuyingPower            float64 `json:"buying_power"`
		PortfolioCurrentMargin float64 `json:"portfolio_current_margin"`
		PortfolioInitialMargin float64 `json:"portfolio_initial_margin"`
		InLiquidation          string  `json:"in_liquidation"`
	} `json:"summary"`
	Balances []struct {
		AssetID           string  `json:"asset_id"`
		AssetUUID         string  `json:"asset_uuid"`
		AssetName         string  `json:"asset_name"`
		Quantity          float64 `json:"quantity"`
		Hold              float64 `json:"hold"`
		TransferHold      float64 `json:"transfer_hold"`
		CollateralValue   float64 `json:"collateral_value"`
		MaxWithdrawAmount float64 `json:"max_withdraw_amount"`
	} `json:"balances"`
	Positions []struct {
		InstrumentID   string  `json:"instrument_id"`
		InstrumentUUID string  `json:"instrument_uuid"`
		Symbol         string  `json:"symbol"`
		Vwap           float64 `json:"vwap"`
		NetSize        float64 `json:"net_size"`
		BuyOrderSize   float64 `json:"buy_order_size"`
		SellOrderSize  float64 `json:"sell_order_size"`
		ImContribution float64 `json:"im_contribution"`
		UnrealizedPnl  float64 `json:"unrealized_pnl"`
		MarkPrice      float64 `json:"mark_price"`
	} `json:"positions"`
}

// PortfolioSummary represents a portfolio summary detailed instance.
type PortfolioSummary struct {
	Collateral             float64 `json:"collateral"`
	UnrealizedPnl          float64 `json:"unrealized_pnl"`
	PositionNotional       float64 `json:"position_notional"`
	OpenPositionNotional   float64 `json:"open_position_notional"`
	PendingFees            float64 `json:"pending_fees"`
	Borrow                 float64 `json:"borrow"`
	AccruedInterest        float64 `json:"accrued_interest"`
	RollingDebt            float64 `json:"rolling_debt"`
	Balance                float64 `json:"balance"`
	BuyingPower            float64 `json:"buying_power"`
	PortfolioCurrentMargin float64 `json:"portfolio_current_margin"`
	PortfolioInitialMargin float64 `json:"portfolio_initial_margin"`
	InLiquidation          string  `json:"in_liquidation"`
}

// PortfolioBalance represents a portfolio balance instance.
type PortfolioBalance struct {
	AssetID           string                  `json:"asset_id"`
	AssetUUID         string                  `json:"asset_uuid"`
	AssetName         string                  `json:"asset_name"`
	Quantity          convert.StringToFloat64 `json:"quantity"`
	Hold              convert.StringToFloat64 `json:"hold"`
	TransferHold      convert.StringToFloat64 `json:"transfer_hold"`
	CollateralValue   convert.StringToFloat64 `json:"collateral_value"`
	MaxWithdrawAmount convert.StringToFloat64 `json:"max_withdraw_amount"`
}

// PortfolioPosition represents a portfolio positions instance.
type PortfolioPosition struct {
	InstrumentID              string  `json:"instrument_id"`
	InstrumentUUID            string  `json:"instrument_uuid"`
	Symbol                    string  `json:"symbol"`
	Vwap                      float64 `json:"vwap"`
	NetSize                   float64 `json:"net_size"`
	BuyOrderSize              float64 `json:"buy_order_size"`
	SellOrderSize             float64 `json:"sell_order_size"`
	InitialMarginContribution float64 `json:"im_contribution"`
	UnrealizedPnl             float64 `json:"unrealized_pnl"`
	MarkPrice                 float64 `json:"mark_price"`
}

// PortfolioFill represents a portfolio fill information.
type PortfolioFill struct {
	Pagination struct {
		RefDatetime  time.Time `json:"ref_datetime"`
		ResultLimit  int64     `json:"result_limit"`
		ResultOffset int64     `json:"result_offset"`
	} `json:"pagination"`
	Results []struct {
		FillID         string    `json:"fill_id"`
		OrderID        string    `json:"order_id"`
		InstrumentID   string    `json:"instrument_id"`
		InstrumentUUID string    `json:"instrument_uuid"`
		Symbol         string    `json:"symbol"`
		MatchID        string    `json:"match_id"`
		FillPrice      float64   `json:"fill_price"`
		FillQty        float64   `json:"fill_qty"`
		ClientID       string    `json:"client_id"`
		ClientOrderID  string    `json:"client_order_id"`
		OrderQty       float64   `json:"order_qty"`
		LimitPrice     int64     `json:"limit_price"`
		TotalFilled    float64   `json:"total_filled"`
		FilledVwap     float64   `json:"filled_vwap"`
		ExpireTime     time.Time `json:"expire_time"`
		StopPrice      float64   `json:"stop_price"`
		Side           string    `json:"side"`
		Tif            string    `json:"tif"`
		StpMode        string    `json:"stp_mode"`
		Flags          string    `json:"flags"`
		Fee            float64   `json:"fee"`
		FeeAsset       string    `json:"fee_asset"`
		OrderStatus    string    `json:"order_status"`
		EventTime      time.Time `json:"event_time"`
	} `json:"results"`
}

// Transfers returns a list of fund transfers.
type Transfers struct {
	Pagination struct {
		ResultLimit  int `json:"result_limit"`
		ResultOffset int `json:"result_offset"`
	} `json:"pagination"`
	Results []FundTransfer `json:"results"`
}

// FundTransfer represents a fund transfer instance.
type FundTransfer struct {
	TransferUUID string    `json:"transfer_uuid"`
	Type         string    `json:"type"` // TODO: ?
	Amount       float64   `json:"amount"`
	Asset        string    `json:"asset"`
	Status       string    `json:"status"`
	NetworkName  string    `json:"network_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WithdrawToCoinbaseINTXParam holds withdraw funds parameters.
type WithdrawToCoinbaseINTXParam struct {
	ProfileID         string        `json:"profile_id"`
	Amount            float64       `json:"amount,omitempty,string"`
	CoinbaseAccountID string        `json:"coinbase_account_id,omitempty"`
	Currency          currency.Code `json:"currency"`
}

// WithdrawToCoinbaseResponse represents a response after withdrawing to coinbase account.
type WithdrawToCoinbaseResponse struct {
	ID       string                  `json:"id"`
	Amount   convert.StringToFloat64 `json:"amount"`
	Fee      convert.StringToFloat64 `json:"fee"`
	Currency string                  `json:"currency"`
	PayoutAt string                  `json:"payout_at"`
	Subtotal string                  `json:"subtotal"`
}

// WithdrawCryptoParams holds crypto fund withdrawal information.
type WithdrawCryptoParams struct {
	Portfolio            string  `json:"portfolio,omitempty"` // Identifies the portfolio by UUID
	AssetIdentifier      string  `json:"asset"`               // Identifies the asset by name
	Amount               float64 `json:"amount,string"`
	AddNetworkFeeToTotal bool    `json:"add_network_fee_to_total,omitempty"` // if true, deducts network fee from the portfolio, otherwise deduct fee from the withdrawal
	NetworkArnID         string  `json:"network_arn_id,omitempty"`           // Identifies the blockchain network
	Address              string  `json:"address"`
	Nonce                string  `json:"nonce,omitempty"`
}

// WithdrawalResponse holds crypto withdrawal ID information
type WithdrawalResponse struct {
	Idem string `json:"idem"` // Idempotent uuid representing the successful withdraw
}

// CryptoAddressParam holds crypto address creation parameters.
type CryptoAddressParam struct {
	Portfolio       string `json:"portfolio"`      // Identifies the portfolio by UUID
	AssetIdentifier string `json:"asset"`          // Identifies the asset by name (e.g., BTC), UUID (e.g., 291efb0f-2396-4d41-ad03-db3b2311cb2c), or asset ID (e.g., 1482439423963469)
	NetworkArnID    string `json:"network_arn_id"` // Identifies the blockchain network
}

// CryptoAddressInfo holds crypto address information after creation
type CryptoAddressInfo struct {
	Address      string `json:"address"`
	NetworkArnID string `json:"network_arn_id"`
}

// SubscriptionInput holds channel subscription information
type SubscriptionInput struct {
	Type           string         `json:"type"` // SUBSCRIBE or UNSUBSCRIBE
	ProductIDPairs currency.Pairs `json:"-"`
	ProductIDs     []string       `json:"product_ids"`
	Channels       []string       `json:"channels"`
	Time           string         `json:"time,omitempty"`
	Key            string         `json:"key,omitempty"`
	Passphrase     string         `json:"passphrase,omitempty"`
	Signature      string         `json:"signature,omitempty"`
}

// SubscriptionRespnse represents a subscription response
type SubscriptionRespnse struct {
	Channels      []SubscribedChannel `json:"channels,omitempty"`
	Authenticated bool                `json:"authenticated,omitempty"`
	Channel       string              `json:"channel,omitempty"`
	Type          string              `json:"type,omitempty"`
	Time          time.Time           `json:"time,omitempty"`

	// Error message and failure reason information.
	Message string `json:"message,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// SubscribedChannel represents a subscribed channel name and product ID(instrument) list.
type SubscribedChannel struct {
	Name       string   `json:"name"`
	ProductIDs []string `json:"product_ids"`
}

// WsInstrument holds response information to websocket
type WsInstrument struct {
	Sequence            int64     `json:"sequence"`
	ProductID           string    `json:"product_id"`
	InstrumentType      string    `json:"instrument_type"`
	BaseAssetName       string    `json:"base_asset_name"`
	QuoteAssetName      string    `json:"quote_asset_name"`
	BaseIncrement       string    `json:"base_increment"`
	QuoteIncrement      string    `json:"quote_increment"`        // 30 day average daily traded vol, updated daily
	AvgDailyQuantity    string    `json:"avg_daily_quantity"`     // 30 day avg daily traded notional amt in USDC, updated daily
	AvgDailyVolume      string    `json:"avg_daily_volume"`       // Max leverage allowed when trading on margin, in margin fraction
	Total30DayQuantity  string    `json:"total_30_day_quantity"`  // 30 day total traded vol, updated daily
	Total30DayVolume    string    `json:"total_30_day_volume"`    // 30 day total traded notional amt in USDC, updated daily
	Total24HourQuantity string    `json:"total_24_hour_quantity"` // 24 hr total traded vol, updated hourly
	Total24HourVolume   string    `json:"total_24_hour_volume"`   // 24 hr total traded notional amt in USDC, updated hourly
	BaseImf             string    `json:"base_imf"`               // Smallest qty allowed to place an order
	MinQuantity         string    `json:"min_quantity"`           // Max size allowed for a position
	PositionSizeLimit   string    `json:"position_size_limit"`
	FundingInterval     string    `json:"funding_interval"` // Time in nanoseconds between funding intervals
	TradingState        string    `json:"trading_state"`    // ALLOWED: offline, trading, paused
	LastUpdateTime      time.Time `json:"last_update_time"`
	Time                time.Time `json:"time"` // Gateway timestamp
	Channel             string    `json:"channel"`
	Type                string    `json:"type"`
}

// WsMatch holds push data information through the channel MATCH.
type WsMatch struct {
	Sequence   int64     `json:"sequence"`
	ProductID  string    `json:"product_id"`
	Time       time.Time `json:"time"`
	MatchID    string    `json:"match_id"`
	TradeQty   string    `json:"trade_qty"`
	TradePrice string    `json:"trade_price"`
	Channel    string    `json:"channel"`
	Type       string    `json:"type"`
}

// WsFunding holds push data information through the FUNDING channel.
type WsFunding struct {
	Sequence    int64     `json:"sequence"`
	ProductID   string    `json:"product_id"`
	Time        time.Time `json:"time"`
	FundingRate string    `json:"funding_rate"`
	IsFinal     bool      `json:"is_final"`
	Channel     string    `json:"channel"`
	Type        string    `json:"type"`
}

// WsRisk holds push data information through the RISK channel.
type WsRisk struct {
	Sequence        int64     `json:"sequence"`
	ProductID       string    `json:"product_id"`
	Time            time.Time `json:"time"`
	LimitUp         string    `json:"limit_up"`
	LimitDown       string    `json:"limit_down"`
	IndexPrice      string    `json:"index_price"`
	MarkPrice       string    `json:"mark_price"`
	SettlementPrice string    `json:"settlement_price"`
	Channel         string    `json:"channel"`
	Type            string    `json:"type"`
}

// WsOrderbookLevel1 holds Level-1 orderbook information
type WsOrderbookLevel1 struct {
	Sequence  int64                   `json:"sequence"`
	ProductID string                  `json:"product_id"`
	Time      time.Time               `json:"time"`
	BidPrice  convert.StringToFloat64 `json:"bid_price"`
	BidQty    convert.StringToFloat64 `json:"bid_qty"`
	Channel   string                  `json:"channel"`
	Type      string                  `json:"type"`
	AskPrice  convert.StringToFloat64 `json:"ask_price,omitempty"`
	AskQty    convert.StringToFloat64 `json:"ask_qty,omitempty"`
}

// WsOrderbookLevel2 holds Level-2 orderbook information.
type WsOrderbookLevel2 struct {
	Sequence  int64       `json:"sequence"`
	ProductID string      `json:"product_id"`
	Time      time.Time   `json:"time"`
	Asks      [][2]string `json:"asks"`
	Bids      [][2]string `json:"bids"`
	Channel   string      `json:"channel"`
	Type      string      `json:"type"` // Possible values: UPDATE and SNAPSHOT

	// Changes when the data is UPDATE
	Changes [][3]string `json:"changes"`
}
