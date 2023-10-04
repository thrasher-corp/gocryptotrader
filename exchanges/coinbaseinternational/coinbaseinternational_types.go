package coinbaseinternational

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
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
	AssetID          int64   `json:"asset_id"`
	AssetUUID        string  `json:"asset_uuid"`
	AssetName        string  `json:"asset_name"`
	IsDefault        string  `json:"is_default"`
	NetworkName      string  `json:"network_name"`
	DisplayName      string  `json:"display_name"`
	NetworkArnID     string  `json:"network_arn_id"`
	MinWithdrawalAmt float64 `json:"min_withdrawal_amt"`
	MaxWithdrawalAmt float64 `json:"max_withdrawal_amt"`
	NetworkConfirms  int64   `json:"network_confirms"`
	ProcessingTime   int64   `json:"processing_time"`
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

// InstrumentQuoteInformation represents a quote information
type InstrumentQuoteInformation struct {
	BestBidPrice     float64              `json:"best_bid_price"`
	BestBidSize      float64              `json:"best_bid_size"`
	BestAskPrice     float64              `json:"best_ask_price"`
	BestAskSize      float64              `json:"best_ask_size"`
	TradePrice       float64              `json:"trade_price"`
	TradeQty         float64              `json:"trade_qty"`
	IndexPrice       float64              `json:"index_price"`
	MarkPrice        float64              `json:"mark_price"`
	SettlementPrice  float64              `json:"settlement_price"`
	LimitUp          float64              `json:"limit_up"`
	LimitDown        float64              `json:"limit_down"`
	PredictedFunding float64              `json:"predicted_funding"`
	Timestamp        convert.ExchangeTime `json:"timestamp"`
}

// OrderRequestParams represents a request paramter for creating order.
type OrderRequestParams struct {
	ClientOrderID string  `json:"client_order_id,omitempty"`
	Side          string  `json:"side,omitempty"`
	BaseSize      float64 `json:"size,omitempty,string"`
	TimeInForce   string  `json:"tif,omitempty"`
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
	OrderID        int64     `json:"order_id"`
	ClientOrderID  string    `json:"client_order_id"`
	Side           string    `json:"side"`
	InstrumentID   int64     `json:"instrument_id"`
	InstrumentUUID string    `json:"instrument_uuid"`
	Symbol         string    `json:"symbol"`
	PortfolioID    int64     `json:"portfolio_id"`
	PortfolioUUID  string    `json:"portfolio_uuid"`
	Type           string    `json:"type"`
	Price          float64   `json:"price"`
	StopPrice      float64   `json:"stop_price"`
	Size           float64   `json:"size"`
	Tif            string    `json:"tif"`
	ExpireTime     time.Time `json:"expire_time"`
	StpMode        string    `json:"stp_mode"`
	EventType      string    `json:"event_type"`
	OrderStatus    string    `json:"order_status"`
	LeavesQty      string    `json:"leaves_qty"`
	ExecQty        string    `json:"exec_qty"`
	AvgPrice       string    `json:"avg_price"`
	Message        string    `json:"message"`
	Fee            string    `json:"fee"`
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

// OrderItem represents a single order item.
type OrderItem struct {
	OrderID        int64                   `json:"order_id"`
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
	Tif            string                  `json:"tif"`
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

// PortfolioItem represents a user portfolio item.
type PortfolioItem struct {
	PortfolioID    string  `json:"portfolio_id"`
	PortfolioUUID  string  `json:"portfolio_uuid"`
	Name           string  `json:"name"`
	UserUUID       string  `json:"user_uuid"`
	MakerFeeRate   float64 `json:"maker_fee_rate"`
	TakerFeeRate   float64 `json:"taker_fee_rate"`
	TradingLock    string  `json:"trading_lock"`
	BorrowDisabled string  `json:"borrow_disabled"`
	IsLSP          string  `json:"is_lsp"` // Indicates if the portfolio is setup to take liquidation assignments
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
