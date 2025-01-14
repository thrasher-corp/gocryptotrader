package coinbaseinternational

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
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
	AssetID             string       `json:"asset_id"`
	AssetUUID           string       `json:"asset_uuid"`
	AssetName           string       `json:"asset_name"`
	NetworkName         string       `json:"network_name"`
	DisplayName         string       `json:"display_name"`
	NetworkArnID        string       `json:"network_arn_id"`
	MinWithdrawalAmount types.Number `json:"min_withdrawal_amt"`
	MaxWithdrawalAmount types.Number `json:"max_withdrawal_amt"`
	NetworkConfirms     int64        `json:"network_confirms"`
	ProcessingTime      types.Time   `json:"processing_time"`
	IsDefault           bool         `json:"is_default"`
}

// InstrumentInfo represents an instrument detail for specific instrument id.
type InstrumentInfo struct {
	InstrumentID        string              `json:"instrument_id"`
	InstrumentUUID      string              `json:"instrument_uuid"`
	Symbol              string              `json:"symbol"`
	Type                string              `json:"type"`
	BaseAssetID         string              `json:"base_asset_id"`
	BaseAssetUUID       string              `json:"base_asset_uuid"`
	BaseAssetName       string              `json:"base_asset_name"`
	QuoteAssetID        string              `json:"quote_asset_id"`
	QuoteAssetUUID      string              `json:"quote_asset_uuid"`
	QuoteAssetName      string              `json:"quote_asset_name"`
	BaseIncrement       types.Number        `json:"base_increment"`
	QuoteIncrement      types.Number        `json:"quote_increment"`
	PriceBandPercent    float64             `json:"price_band_percent"`
	MarketOrderPercent  float64             `json:"market_order_percent"`
	Qty24Hr             types.Number        `json:"qty_24hr"`
	Notional24Hr        types.Number        `json:"notional_24hr"`
	AvgDailyQty         types.Number        `json:"avg_daily_qty"`
	AvgDailyNotional    types.Number        `json:"avg_daily_notional"`
	PreviousDayQty      types.Number        `json:"previous_day_qty"`
	OpenInterest        types.Number        `json:"open_interest"`
	PositionLimitQty    types.Number        `json:"position_limit_qty"`
	PositionLimitAdqPct float64             `json:"position_limit_adq_pct"`
	ReplacementCost     types.Number        `json:"replacement_cost"`
	BaseImf             float64             `json:"base_imf"`
	MinNotionalValue    string              `json:"min_notional_value"`
	FundingInterval     string              `json:"funding_interval"`
	TradingState        string              `json:"trading_state"`
	PositionLimitAdv    float64             `json:"position_limit_adv"`
	InitialMarginAdv    float64             `json:"initial_margin_adv"`
	Mode                string              `json:"mode"`
	Avg30DayNotional    string              `json:"avg_30day_notional"`
	Avg30DayQty         string              `json:"avg_30day_qty"`
	Quote               ContractQuoteDetail `json:"quote,omitempty"`
	DefaultImf          float64             `json:"default_imf,omitempty"`
	BaseAssetMultiplier string              `json:"base_asset_multiplier"`
	UnderlyingType      string              `json:"underlying_type"`
}

// ContractQuoteDetail represents a contract quote detail
type ContractQuoteDetail struct {
	BestBidPrice     types.Number `json:"best_bid_price"`
	BestBidSize      types.Number `json:"best_bid_size"`
	BestAskPrice     types.Number `json:"best_ask_price"`
	BestAskSize      types.Number `json:"best_ask_size"`
	TradePrice       types.Number `json:"trade_price"`
	TradeQty         types.Number `json:"trade_qty"`
	IndexPrice       types.Number `json:"index_price"`
	MarkPrice        types.Number `json:"mark_price"`
	SettlementPrice  types.Number `json:"settlement_price"`
	LimitUp          types.Number `json:"limit_up"`
	LimitDown        types.Number `json:"limit_down"`
	PredictedFunding types.Number `json:"predicted_funding"`
	Timestamp        time.Time    `json:"timestamp"`
}

// IndexMetadata represents an index metadata detail
type IndexMetadata struct {
	ProductID          string    `json:"product_id"`
	Divisor            float64   `json:"divisor"`
	Timestamp          time.Time `json:"timestamp"`
	InceptionTimestamp time.Time `json:"inception_timestamp"`
	LastRebalance      time.Time `json:"last_rebalance"`
	Constituents       []struct {
		Symbol         string       `json:"symbol"`
		Name           string       `json:"name"`
		Rank           int64        `json:"rank"`
		CapFactor      string       `json:"cap_factor"` // constituent market cap factor (or multiplier)
		Amount         types.Number `json:"amount"`
		MarketCap      types.Number `json:"market_cap"`
		IndexMarketCap types.Number `json:"index_market_cap"`
		Weight         types.Number `json:"weight"`
		RunningWeight  types.Number `json:"running_weight"`
	} `json:"constituents"`
}

// IndexPriceInfo represents latest index price info
type IndexPriceInfo struct {
	ProductID       string       `json:"product_id"`
	Status          string       `json:"status"`
	Timestamp       time.Time    `json:"timestamp"`
	Price           types.Number `json:"price"`
	Price24HrChange types.Number `json:"price_24hr_change"`
}

// IndexPriceCandlesticks represents a index price candlestick data of instruments
type IndexPriceCandlesticks struct {
	Aggregations []struct {
		Start time.Time    `json:"start"`
		Open  types.Number `json:"open"`
		High  types.Number `json:"high"`
		Low   types.Number `json:"low"`
		Close types.Number `json:"close"`
	} `json:"aggregations"`
}

// InstrumentsTradingVolumeInfo represents a daily trading volume information
type InstrumentsTradingVolumeInfo struct {
	Pagination struct {
		ResultLimit  float64 `json:"result_limit"`
		ResultOffset float64 `json:"result_offset"`
	} `json:"pagination"`
	Results []struct {
		Timestamp   time.Time `json:"timestamp"`
		Instruments []struct {
			Symbol   string       `json:"symbol"`
			Volume   types.Number `json:"volume"`
			Notional types.Number `json:"notional"`
		} `json:"instruments"`
		Totals struct {
			TotalInstrumentsVolume   types.Number `json:"total_instruments_volume"`
			TotalInstrumentsNotional types.Number `json:"total_instruments_notional"`
			TotalExchangeVolume      types.Number `json:"total_exchange_volume"`
			TotalExchangeNotional    types.Number `json:"total_exchange_notional"`
		} `json:"totals"`
	} `json:"results"`
}

// CandlestickDataHistory represents aggregated candles data
type CandlestickDataHistory struct {
	Aggregations []struct {
		Start  time.Time    `json:"start"`
		Open   types.Number `json:"open"`
		High   types.Number `json:"high"`
		Low    types.Number `json:"low"`
		Close  types.Number `json:"close"`
		Volume types.Number `json:"volume"`
	} `json:"aggregations"`
}

// FundingRateHistory represents a funding rate history detail
type FundingRateHistory struct {
	Pagination struct {
		ResultLimit  int64 `json:"result_limit"`
		ResultOffset int64 `json:"result_offset"`
	} `json:"pagination"`
	Results []struct {
		InstrumentID string       `json:"instrument_id"`
		FundingRate  types.Number `json:"funding_rate"`
		MarkPrice    types.Number `json:"mark_price"`
		EventTime    time.Time    `json:"event_time"`
	} `json:"results"`
}

// PositionsOffset represents a position offset detail
type PositionsOffset struct {
	PositionOffsets []struct {
		PrimaryInstrumentID   string  `json:"primary_instrument_id"`
		SecondaryInstrumentID string  `json:"secondary_instrument_id"`
		Offset                float64 `json:"offset"`
	} `json:"position_offsets"`
}

// QuoteInformation represents a instrument quote information
type QuoteInformation struct {
	BestBidPrice     types.Number `json:"best_bid_price"`
	BestBidSize      types.Number `json:"best_bid_size"`
	BestAskPrice     types.Number `json:"best_ask_price"`
	BestAskSize      types.Number `json:"best_ask_size"`
	TradePrice       types.Number `json:"trade_price"`
	TradeQty         types.Number `json:"trade_qty"`
	IndexPrice       types.Number `json:"index_price"`
	MarkPrice        types.Number `json:"mark_price"`
	SettlementPrice  types.Number `json:"settlement_price"`
	LimitUp          types.Number `json:"limit_up"`
	LimitDown        types.Number `json:"limit_down"`
	PredictedFunding types.Number `json:"predicted_funding"`
	Timestamp        time.Time    `json:"timestamp"`
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
	OrderID        int64        `json:"order_id"`
	ClientOrderID  string       `json:"client_order_id"`
	Side           string       `json:"side"`
	InstrumentID   int64        `json:"instrument_id"`
	InstrumentUUID string       `json:"instrument_uuid"`
	Symbol         string       `json:"symbol"`
	PortfolioID    int64        `json:"portfolio_id"`
	PortfolioUUID  string       `json:"portfolio_uuid"`
	Type           string       `json:"type"`
	Price          float64      `json:"price"`
	StopPrice      float64      `json:"stop_price"`
	Size           float64      `json:"size"`
	TimeInForce    string       `json:"tif"`
	ExpireTime     time.Time    `json:"expire_time"`
	StpMode        string       `json:"stp_mode"`
	EventType      string       `json:"event_type"`
	OrderStatus    string       `json:"order_status"`
	LeavesQty      types.Number `json:"leaves_qty"`
	ExecQty        types.Number `json:"exec_qty"`
	AvgPrice       types.Number `json:"avg_price"`
	Message        string       `json:"message"`
	Fee            types.Number `json:"fee"`
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
	OrderID        string       `json:"order_id"`
	ClientOrderID  string       `json:"client_order_id"`
	Side           string       `json:"side"`
	InstrumentID   string       `json:"instrument_id"`
	InstrumentUUID string       `json:"instrument_uuid"`
	Symbol         string       `json:"symbol"`
	PortfolioID    int64        `json:"portfolio_id"`
	PortfolioUUID  string       `json:"portfolio_uuid"`
	Type           string       `json:"type"`
	Price          float64      `json:"price"`
	StopPrice      float64      `json:"stop_price"`
	Size           float64      `json:"size"`
	TimeInForce    string       `json:"tif"`
	ExpireTime     time.Time    `json:"expire_time"`
	StpMode        string       `json:"stp_mode"`
	EventType      string       `json:"event_type"`
	OrderStatus    string       `json:"order_status"`
	LeavesQuantity types.Number `json:"leaves_qty"`
	ExecQty        types.Number `json:"exec_qty"`
	AveragePrice   types.Number `json:"avg_price"`
	Message        string       `json:"message"`
	Fee            types.Number `json:"fee"`
}

// PortfolioItem represents a user portfolio item
// and transaction fee information.
type PortfolioItem struct {
	PortfolioID             string       `json:"portfolio_id"`
	PortfolioUUID           string       `json:"portfolio_uuid"`
	Name                    string       `json:"name"`
	UserUUID                string       `json:"user_uuid"`
	MakerFeeRate            types.Number `json:"maker_fee_rate"`
	TakerFeeRate            types.Number `json:"taker_fee_rate"`
	TradingLock             bool         `json:"trading_lock"`
	BorrowDisabled          bool         `json:"borrow_disabled"`
	IsLSP                   bool         `json:"is_lsp"` // Indicates if the portfolio is setup to take liquidation assignments
	IsDefault               string       `json:"is_default"`
	CrossCollateralEnabled  string       `json:"cross_collateral_enabled"`
	PreLaunchTradingEnabled string       `json:"pre_launch_trading_enabled"`
}

// PatchPortfolioParams represents a request body for patching a portfolio
type PatchPortfolioParams struct {
	AutoMarginEnabled       bool   `json:"auto_margin_enabled,omitempty"`
	CrossCollateralEnabled  bool   `json:"cross_collateral_enabled,omitempty"`
	PositionOffsetEnabled   bool   `json:"position_offsets_enabled,omitempty"`
	PreLaunchTradingEnabled bool   `json:"pre_launch_trading_enabled,omitempty"`
	PortfolioName           string `json:"portfolio_name,omitempty"`
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

// PortfolioLoanDetail represents a portfolio loan detail
type PortfolioLoanDetail struct {
	PortfolioID                   string  `json:"portfolio_id"`
	AssetID                       string  `json:"asset_id"`
	AssetUUID                     string  `json:"asset_uuid"`
	AssetName                     string  `json:"asset_name"`
	TotalLoan                     float64 `json:"total_loan"`
	CollateralBackedOverdraftLoan float64 `json:"collateral_backed_overdraft_loan"`
	UserRequestedLoan             float64 `json:"user_requested_loan"`
	CollateralRequirement         float64 `json:"collateral_requirement"`
	InitialMarginContribution     float64 `json:"initial_margin_contribution"`
	InitialMarginRequirement      float64 `json:"initial_margin_requirement"`
	CurrentInterestRate           float64 `json:"current_interest_rate"`
	PendingInterestCharge         float64 `json:"pending_interest_charge"`
}

// AcquireRepayLoanResponse represents a response data for a loan acquire and repayment
type AcquireRepayLoanResponse struct {
	PortfolioID   string  `json:"portfolio_id"`
	AssetID       string  `json:"asset_id"`
	Delta         float64 `json:"delta"`
	Total         float64 `json:"total"`
	AssetUUID     string  `json:"asset_uuid"`
	PortfolioUUID string  `json:"portfolio_uuid"`
}

// LoanActionAmountParam represents a request parameters for loan orders which require action and amount
type LoanActionAmountParam struct {
	Action string  `json:"action,omitempty"`
	Amount float64 `json:"amount,omitempty"`
}

// LoanUpdate represents a loan update detail
type LoanUpdate struct {
	InitialMarginContribution      float64 `json:"initial_margin_contribution"`
	InitialMarginDelta             float64 `json:"initial_margin_delta"`
	PortfolioInitialMargin         float64 `json:"portfolio_initial_margin"`
	PortfolioInitialMarginNotional float64 `json:"portfolio_initial_margin_notional"`
	LoanCollateralRequirement      float64 `json:"loan_collateral_requirement"`
	LoanCollateralRequirementDelta float64 `json:"loan_collateral_requirement_delta"`
	TotalLoan                      float64 `json:"total_loan"`
	LoanDelta                      float64 `json:"loan_delta"`
	MaxAvailable                   float64 `json:"max_available"`
	RejectDetails                  string  `json:"reject_details"`
	IsValid                        string  `json:"is_valid"`
}

// MaxLoanAvailability represents a maximum loan availability information
type MaxLoanAvailability struct {
	Available float64 `json:"available"`
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
	AssetID           string       `json:"asset_id"`
	AssetUUID         string       `json:"asset_uuid"`
	AssetName         string       `json:"asset_name"`
	Quantity          types.Number `json:"quantity"`
	Hold              types.Number `json:"hold"`
	TransferHold      types.Number `json:"transfer_hold"`
	CollateralValue   types.Number `json:"collateral_value"`
	MaxWithdrawAmount types.Number `json:"max_withdraw_amount"`
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
		LimitPrice     float64   `json:"limit_price"`
		TotalFilled    float64   `json:"total_filled"`
		FilledVwap     float64   `json:"filled_vwap"`
		ExpireTime     time.Time `json:"expire_time"`
		StopPrice      float64   `json:"stop_price"`
		Side           string    `json:"side"`
		TimeInForce    string    `json:"tif"`
		StpMode        string    `json:"stp_mode"`
		Flags          string    `json:"flags"`
		Fee            float64   `json:"fee"`
		FeeAsset       string    `json:"fee_asset"`
		OrderStatus    string    `json:"order_status"`
		EventTime      time.Time `json:"event_time"`
	} `json:"results"`
}

// PortfolioCrossCollateralDetail represents a portfolio cross-collateral response detail
type PortfolioCrossCollateralDetail struct {
	PortfolioID             string  `json:"portfolio_id"`
	PortfolioUUID           string  `json:"portfolio_uuid"`
	Name                    string  `json:"name"`
	UserUUID                string  `json:"user_uuid"`
	MakerFeeRate            float64 `json:"maker_fee_rate"`
	TakerFeeRate            float64 `json:"taker_fee_rate"`
	TradingLock             string  `json:"trading_lock"`
	BorrowDisabled          string  `json:"borrow_disabled"`
	IsLsp                   string  `json:"is_lsp"`
	IsDefault               string  `json:"is_default"`
	CrossCollateralEnabled  string  `json:"cross_collateral_enabled"`
	PreLaunchTradingEnabled string  `json:"pre_launch_trading_enabled"`
}

// PortfolioMarginOverrideResponse represents margin override value for a portfolio
type PortfolioMarginOverrideResponse struct {
	PortfolioID    string  `json:"portfolio_id"`
	MarginOverride float64 `json:"margin_override"`
}

// TransferFundsBetweenPortfoliosParams transfer assets from one portfolio to another
type TransferFundsBetweenPortfoliosParams struct {
	From   string        `json:"from,omitempty"`
	To     string        `json:"to,omitempty"`
	Asset  currency.Code `json:"asset,omitempty"`
	Amount float64       `json:"amount,omitempty"`
}

// TransferPortfolioParams represents a response detail for transfer an existing position from one portfolio to another
type TransferPortfolioParams struct {
	From       string  `json:"from"`
	To         string  `json:"to"`
	Instrument string  `json:"instrument"`
	Quantity   float64 `json:"quantity"`
	Side       string  `json:"side"`
}

// PortfolioMarginOverrideParams represents a margin override value for a portfolio parameter
type PortfolioMarginOverrideParams struct {
	PortfolioID    string  `json:"portfolio_id,omitempty"`
	MarginOverride float64 `json:"margin_override,omitempty"`
}

// PortfolioFeeRate represents Perpetual Future and Spot fee rate
type PortfolioFeeRate struct {
	InstrumentType          string  `json:"instrument_type"`
	FeeTierID               int64   `json:"fee_tier_id"`
	IsVipTier               string  `json:"is_vip_tier"`
	FeeTierName             string  `json:"fee_tier_name"`
	MakerFeeRate            float64 `json:"maker_fee_rate"`
	TakerFeeRate            float64 `json:"taker_fee_rate"`
	IsOverride              string  `json:"is_override"`
	Trailing30DayVolume     float64 `json:"trailing_30day_volume"`
	Trailing24HrUsdcBalance float64 `json:"trailing_24hr_usdc_balance"`
}

// VolumeRankingInfo represents a volume ranking information
type VolumeRankingInfo struct {
	LastUpdated time.Time `json:"last_updated"`
	Statistics  struct {
		Maker struct {
			Rank            float64 `json:"rank"`
			RelativePercent float64 `json:"relative_percent"`
			Volume          float64 `json:"volume"`
		} `json:"maker"`
		Taker struct {
			Rank            float64 `json:"rank"`
			RelativePercent float64 `json:"relative_percent"`
			Volume          float64 `json:"volume"`
		} `json:"taker"`
		Total struct {
			Rank            float64 `json:"rank"`
			RelativePercent float64 `json:"relative_percent"`
			Volume          float64 `json:"volume"`
		} `json:"total"`
	} `json:"statistics"`
}

// Transfers returns a list of fund transfers.
type Transfers struct {
	Pagination struct {
		ResultLimit  int64 `json:"result_limit"`
		ResultOffset int64 `json:"result_offset"`
	} `json:"pagination"`
	Results []FundTransfer `json:"results"`
}

// FundTransfer represents a fund transfer instance.
type FundTransfer struct {
	TransferUUID   string    `json:"transfer_uuid"`
	TransferType   string    `json:"type"`
	Amount         float64   `json:"amount"`
	Asset          string    `json:"asset"`
	TransferStatus string    `json:"status"`
	NetworkName    string    `json:"network_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	FromPortfolio  struct {
		ID   string `json:"id"`
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"from_portfolio"`
	ToPortfolio struct {
		ID   string `json:"id"`
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"to_portfolio"`
	FromAddress         int64  `json:"from_address"`
	ToAddress           int64  `json:"to_address"`
	FromCoinbaseAccount string `json:"from_cb_account"`
	ToCoinbaseAccount   string `json:"to_cb_account"`
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
	ID       string       `json:"id"`
	Amount   types.Number `json:"amount"`
	Fee      types.Number `json:"fee"`
	Currency string       `json:"currency"`
	PayoutAt string       `json:"payout_at"`
	Subtotal string       `json:"subtotal"`
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

// CounterpartyIDCreationResponse represents a counterparty ID creation response
type CounterpartyIDCreationResponse struct {
	PortfolioUUID  string `json:"portfolio_uuid"`
	CounterpartyID string `json:"counterparty_id"`
}

// CounterpartyValidationResponse represents a counterparty validation response
type CounterpartyValidationResponse struct {
	CounterpartyID string `json:"counterparty_id"`
	Valid          bool   `json:"valid"`
}

// AssetCounterpartyWithdrawalResponse represents an asset counterparty withdrawal information
type AssetCounterpartyWithdrawalResponse struct {
	Portfolio      string  `json:"portfolio"`
	CounterpartyID string  `json:"counterparty_id"`
	Asset          string  `json:"asset"`
	Amount         float64 `json:"amount"`
	Nonce          string  `json:"nonce"`
}

// CounterpartyWithdrawalResponse an asset withdrawal response
type CounterpartyWithdrawalResponse struct {
	Idem                 string  `json:"idem"`
	PortfolioUUID        string  `json:"portfolio_uuid"`
	SourceCounterpartyID string  `json:"source_counterparty_id"`
	TargetCounterpartyID string  `json:"target_counterparty_id"`
	Asset                string  `json:"asset"`
	Amount               float64 `json:"amount"`
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

// SubscriptionResponse represents a subscription response
type SubscriptionResponse struct {
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
	Sequence            int64        `json:"sequence"`
	ProductID           string       `json:"product_id"`
	InstrumentType      string       `json:"instrument_type"`
	BaseAssetName       string       `json:"base_asset_name"`
	QuoteAssetName      string       `json:"quote_asset_name"`
	BaseIncrement       types.Number `json:"base_increment"`
	QuoteIncrement      types.Number `json:"quote_increment"`
	AvgDailyQuantity    types.Number `json:"avg_daily_quantity"`
	AvgDailyVolume      types.Number `json:"avg_daily_volume"`
	Total30DayQuantity  types.Number `json:"total_30_day_quantity"`
	Total30DayVolume    types.Number `json:"total_30_day_volume"`
	Total24HourQuantity types.Number `json:"total_24_hour_quantity"`
	Total24HourVolume   types.Number `json:"total_24_hour_volume"`
	BaseImf             string       `json:"base_imf"`
	MinQuantity         types.Number `json:"min_quantity"`
	PositionSizeLimit   types.Number `json:"position_size_limit"`
	FundingInterval     string       `json:"funding_interval"`
	TradingState        string       `json:"trading_state"`
	LastUpdateTime      time.Time    `json:"last_update_time"`
	GatewayTime         time.Time    `json:"time"`
	Channel             string       `json:"channel"`
	Type                string       `json:"type"`
}

// WsMatch holds push data information through the channel MATCH.
type WsMatch struct {
	Sequence   int64        `json:"sequence"`
	ProductID  string       `json:"product_id"`
	Time       time.Time    `json:"time"`
	MatchID    string       `json:"match_id"`
	TradeQty   types.Number `json:"trade_qty"`
	TradePrice types.Number `json:"trade_price"`
	Channel    string       `json:"channel"`
	Type       string       `json:"type"`
}

// WsFunding holds push data information through the FUNDING channel.
type WsFunding struct {
	Sequence    int64        `json:"sequence"`
	ProductID   string       `json:"product_id"`
	Time        time.Time    `json:"time"`
	FundingRate types.Number `json:"funding_rate"`
	IsFinal     bool         `json:"is_final"`
	Channel     string       `json:"channel"`
	Type        string       `json:"type"`
}

// WsRisk holds push data information through the RISK channel.
type WsRisk struct {
	Sequence        int64        `json:"sequence"`
	ProductID       string       `json:"product_id"`
	Time            time.Time    `json:"time"`
	LimitUp         string       `json:"limit_up"`
	LimitDown       string       `json:"limit_down"`
	IndexPrice      types.Number `json:"index_price"`
	MarkPrice       types.Number `json:"mark_price"`
	SettlementPrice types.Number `json:"settlement_price"`
	Channel         string       `json:"channel"`
	Type            string       `json:"type"`
}

// WsOrderbookLevel1 holds Level-1 orderbook information
type WsOrderbookLevel1 struct {
	Sequence  int64        `json:"sequence"`
	ProductID string       `json:"product_id"`
	Time      time.Time    `json:"time"`
	BidPrice  types.Number `json:"bid_price"`
	BidQty    types.Number `json:"bid_qty"`
	Channel   string       `json:"channel"`
	Type      string       `json:"type"`
	AskPrice  types.Number `json:"ask_price,omitempty"`
	AskQty    types.Number `json:"ask_qty,omitempty"`
}

// WsOrderbookLevel2 holds Level-2 orderbook information.
type WsOrderbookLevel2 struct {
	Sequence  int64             `json:"sequence"`
	ProductID string            `json:"product_id"`
	Time      time.Time         `json:"time"`
	Asks      [][2]types.Number `json:"asks"`
	Bids      [][2]types.Number `json:"bids"`
	Channel   string            `json:"channel"`
	Type      string            `json:"type"` // Possible values: UPDATE and SNAPSHOT

	// Changes when the data is UPDATE
	Changes [][3]string `json:"changes"`
}

// FeeRateInfo represents a fee tier detail of fee rate tiers.
type FeeRateInfo struct {
	FeeTierType             string       `json:"fee_tier_type"`
	InstrumentType          string       `json:"instrument_type"`
	FeeTierID               string       `json:"fee_tier_id"`
	FeeTierName             string       `json:"fee_tier_name"`
	MakerFeeRate            types.Number `json:"maker_fee_rate"`
	TakerFeeRate            types.Number `json:"taker_fee_rate"`
	RebateRate              string       `json:"rebate_rate"`
	MinBalance              types.Number `json:"min_balance"`
	MinVolume               types.Number `json:"min_volume"`
	RequireBalanceAndVolume types.Number `json:"require_balance_and_volume"`
}
