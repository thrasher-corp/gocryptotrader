package coinbaseinternational

import "github.com/thrasher-corp/gocryptotrader/common/convert"

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
