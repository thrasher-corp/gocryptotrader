package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CrossExchangeSymbol holds symbol information for a CrossEx trading pair.
type CrossExchangeSymbol struct {
	Name         string       `json:"name"`
	ExchangeType string       `json:"exchange_type"`
	BusinessType string       `json:"business_type"`
	State        string       `json:"state"`
	TickSize     types.Number `json:"tick_size"`
	LotSize      types.Number `json:"lot_size"`
	MinNotional  types.Number `json:"min_notional"`
	MinSize      types.Number `json:"min_size"`
	MaxNumOrders types.Number `json:"max_num_orders"`
}

// CrossExchangeRiskLimitTier holds a single risk limit tier for a CrossEx symbol.
type CrossExchangeRiskLimitTier struct {
	MinRiskLimitValue types.Number `json:"min_risk_limit_value"`
	QuickAdjAmount    types.Number `json:"quick_adj_amount"`
	LeverageMax       types.Number `json:"leverage_max"`
	MaintenanceRate   types.Number `json:"maintenance_rate"`
}

// CrossExchangeRiskLimit holds the risk limit tiers for a CrossEx symbol.
type CrossExchangeRiskLimit struct {
	Symbol string                        `json:"symbol"`
	Tiers  []*CrossExchangeRiskLimitTier `json:"lec"`
}

// CrossExchangeTransferCoin holds information about a currency supported for CrossEx transfers.
type CrossExchangeTransferCoin struct {
	Coin           currency.Code `json:"coin"`
	MinTransAmount types.Number  `json:"min_trans_amount"`
	EstimatedFee   types.Number  `json:"est_fee"`
	Precision      int64         `json:"precision"`
	IsDisabled     int64         `json:"is_disabled"`
}

// GetCrossExchangeTransferHistoryRequest holds query parameters for the transfer history endpoint.
type GetCrossExchangeTransferHistoryRequest struct {
	Coin          currency.Code
	OrderID       string
	ClientOrderID string
	To            int64
	From          int64
	Limit         uint64
}

// CrossExchangeTransferRecord holds a single fund transfer record.
type CrossExchangeTransferRecord struct {
	ID              string        `json:"id"`
	ClientOrderID   string        `json:"client_order_id"`
	FromAccountType string        `json:"from_account_type"`
	ToAccountType   string        `json:"to_account_type"`
	Coin            currency.Code `json:"coin"`
	AmountActual    string        `json:"amount_actual"`
	Status          string        `json:"status"`
	FailReason      string        `json:"fail_reason"`
}

// CrossExchangeTransferRequest is the request body for a CrossEx fund transfer.
type CrossExchangeTransferRequest struct {
	Coin   currency.Code `json:"coin"`
	Amount float64       `json:"amount,string"`
	Text   string        `json:"text,omitempty"`
	From   string        `json:"from"`
	To     string        `json:"to"`
}

// CrossExchangeTransferResponse holds the result of a CrossEx fund transfer.
type CrossExchangeTransferResponse struct {
	TxID string `json:"tx_id"`
}

// CrossExchangeOrderCreateRequest is the request body for creating a CrossEx order.
type CrossExchangeOrderCreateRequest struct {
	Text         string       `json:"text,omitempty"`
	ExchangeType string       `json:"exchange_type"`
	Symbol       string       `json:"symbol"`
	Side         order.Side   `json:"side"`
	Type         string       `json:"type"`
	Quantity     types.Number `json:"qty,omitempty"`
	Price        types.Number `json:"price,omitempty"`
	QuickQty     types.Number `json:"quick_qty,omitempty"`
	ReduceOnly   bool         `json:"reduce_only,omitempty"`
	PositionSide order.Side   `json:"position_side,omitempty"`
}

// CrossExchangeOrder holds the full detail of a CrossEx order.
type CrossExchangeOrder struct {
	UserID            string        `json:"user_id"`
	OrderID           string        `json:"order_id"`
	Text              string        `json:"text"`
	State             string        `json:"state"`
	Symbol            string        `json:"symbol"`
	Side              order.Side    `json:"side"`
	Type              string        `json:"type"`
	Attribute         string        `json:"attribute"`
	ExchangeType      string        `json:"exchange_type"`
	BusinessType      string        `json:"business_type"`
	Quantity          types.Number  `json:"qty"`
	Price             types.Number  `json:"price"`
	ExecutedQuantity  types.Number  `json:"executed_qty"`
	ExecutedAmount    types.Number  `json:"executed_amount"`
	Fee               types.Number  `json:"fee"`
	FeeAsset          currency.Code `json:"fee_asset"`
	TimeInForce       string        `json:"time_in_force"`
	Leverage          types.Number  `json:"leverage"`
	LastExecutedQty   types.Number  `json:"last_executed_qty"`
	LastExecutedPrice types.Number  `json:"last_executed_price"`
	PositionSide      string        `json:"position_side"`
	ReduceOnly        types.Boolean `json:"reduce_only"`
	LastExecutedTime  types.Time    `json:"last_executed_time"`
	CreateTime        types.Time    `json:"create_time"`
	UpdateTime        types.Time    `json:"update_time"`
}

// CrossExchangeOrderUpdateRequest is the request body for modifying a CrossEx order.
type CrossExchangeOrderUpdateRequest struct {
	Quantity types.Number `json:"qty,omitempty"`
	Price    types.Number `json:"price,omitempty"`
}

// CrossExchangeOrderActionResponse holds the result of a CrossEx order action (modify/cancel).
type CrossExchangeOrderActionResponse struct {
	OrderID string `json:"order_id"`
	Text    string `json:"text"`
}

// CrossExchangeConvertQuoteRequest is the request body for a CrossEx flash swap inquiry.
type CrossExchangeConvertQuoteRequest struct {
	ExchangeType string        `json:"exchange_type"`
	FromCoin     currency.Code `json:"from_coin"`
	ToCoin       currency.Code `json:"to_coin"`
	FromAmount   float64       `json:"from_amount,string"`
}

// CrossExchangeConvertQuoteResponse holds the quote returned from a flash swap inquiry.
type CrossExchangeConvertQuoteResponse struct {
	QuoteID    string        `json:"quote_id"`
	ValidMs    string        `json:"valid_ms"`
	FromCoin   currency.Code `json:"from_coin"`
	ToCoin     currency.Code `json:"to_coin"`
	FromAmount types.Number  `json:"from_amount"`
	ToAmount   types.Number  `json:"to_amount"`
	Price      types.Number  `json:"price"`
}

// CrossExchangeConvertOrderRequest is the request body for executing a CrossEx flash swap.
type CrossExchangeConvertOrderRequest struct {
	QuoteID string `json:"quote_id"`
}

// CrossExchangeConvertOrderResponse holds the result of a CrossEx flash swap execution.
type CrossExchangeConvertOrderResponse struct {
	OrderID string `json:"order_id"`
	Text    string `json:"text"`
}

// CrossExchangeAccountAsset holds per-exchange asset information within a CrossEx account.
type CrossExchangeAccountAsset struct {
	UserID  string       `json:"user_id"`
	Balance types.Number `json:"balance"`
	Equity  types.Number `json:"equity"`
	PNL     types.Number `json:"pnl"`
}

// CrossExchangeAccount holds CrossEx account asset information.
type CrossExchangeAccount struct {
	UserID                string                       `json:"user_id"`
	AvailableMargin       types.Number                 `json:"available_margin"`
	MarginBalance         types.Number                 `json:"margin_balance"`
	InitialMargin         types.Number                 `json:"initial_margin"`
	MaintenanceMargin     types.Number                 `json:"maintenance_margin"`
	InitialMarginRate     types.Number                 `json:"initial_margin_rate"`
	MaintenanceMarginRate types.Number                 `json:"maintenance_margin_rate"`
	PositionMode          string                       `json:"position_mode"`
	AccountLimit          types.Number                 `json:"account_limit"`
	CreateTime            types.Time                   `json:"create_time"`
	ExchangeType          string                       `json:"exchange_type"`
	AccountMode           string                       `json:"account_mode"`
	UpdateTime            types.Time                   `json:"update_time"`
	Assets                []*CrossExchangeAccountAsset `json:"assets,omitempty"`
}

// CrossExchangeAccountUpdateRequest is the request body for modifying CrossEx account settings.
type CrossExchangeAccountUpdateRequest struct {
	PositionMode string `json:"position_mode,omitempty"`
	AccountMode  string `json:"account_mode,omitempty"`
	ExchangeType string `json:"exchange_type,omitempty"`
}

// CrossExchangeAccountUpdateResponse holds the result of a CrossEx account settings update.
type CrossExchangeAccountUpdateResponse struct {
	PositionMode string `json:"position_mode"`
	AccountMode  string `json:"account_mode"`
	ExchangeType string `json:"exchange_type"`
}

// CrossExchangeLeverageRequest is the request body for setting CrossEx leverage.
type CrossExchangeLeverageRequest struct {
	Symbol   string  `json:"symbol"`
	Leverage float64 `json:"leverage,string"`
}

// CrossExchangeLeverageResponse holds the result of a CrossEx leverage change.
type CrossExchangeLeverageResponse struct {
	Symbol   string       `json:"symbol"`
	Leverage types.Number `json:"leverage"`
}

// CrossExchangeClosePositionRequest is the request body for fully closing a CrossEx position.
type CrossExchangeClosePositionRequest struct {
	Symbol       string `json:"symbol"`
	PositionSide string `json:"position_side,omitempty"`
}

// CrossExchangeInterestRate holds margin asset interest rate information.
type CrossExchangeInterestRate struct {
	Coin             currency.Code `json:"coin"`
	ExchangeType     string        `json:"exchange_type"`
	HourInterestRate types.Number  `json:"hour_interest_rate"`
	Time             types.Time    `json:"time"`
}

// CrossExchangeSpecialFee holds the special fee rates for a specific CrossEx symbol.
type CrossExchangeSpecialFee struct {
	Symbol       currency.Pair `json:"symbol"`
	MakerFeeRate types.Number  `json:"maker_fee_rate,omitempty"`
	TakerFeeRate types.Number  `json:"taker_fee_rate"`
}

// CrossExchangeFee holds the fee rate information for a CrossEx exchange type.
type CrossExchangeFee struct {
	ExchangeType   string                     `json:"exchange_type"`
	SpotMakerFee   types.Number               `json:"spot_maker_fee"`
	SpotTakerFee   types.Number               `json:"spot_taker_fee"`
	FutureMakerFee types.Number               `json:"future_maker_fee"`
	FutureTakerFee types.Number               `json:"future_taker_fee"`
	SpecialFeeList []*CrossExchangeSpecialFee `json:"special_fee_list"`
}

// CrossExchangePosition holds a CrossEx contract position.
type CrossExchangePosition struct {
	UserID            string       `json:"user_id"`
	PositionID        string       `json:"position_id"`
	Symbol            string       `json:"symbol"`
	PositionSide      string       `json:"position_side"`
	InitialMargin     string       `json:"initial_margin"`
	MaintenanceMargin string       `json:"maintenance_margin"`
	PositionQuantity  string       `json:"position_qty"`
	PositionValue     string       `json:"position_value"`
	UnrealizedPNL     string       `json:"upnl"`
	Leverage          types.Number `json:"leverage"`
	MaxLeverage       types.Number `json:"max_leverage"`
	OpenAvgPrice      types.Number `json:"open_avg_price"`
	IndexPrice        types.Number `json:"index_price"`
	MarkPrice         types.Number `json:"mark_price"`
	LastPrice         types.Number `json:"last_price"`
	CreateTime        types.Time   `json:"create_time"`
	UpdateTime        types.Time   `json:"update_time"`
	ExchangeType      string       `json:"exchange_type"`
}

// CrossExchangeMarginPosition holds a CrossEx leveraged (margin) position.
type CrossExchangeMarginPosition struct {
	UserID            string        `json:"user_id"`
	PositionID        string        `json:"position_id"`
	Symbol            string        `json:"symbol"`
	PositionSide      string        `json:"position_side"`
	InitialMargin     string        `json:"initial_margin"`
	MaintenanceMargin string        `json:"maintenance_margin"`
	AssetQuantity     string        `json:"asset_qty"`
	AssetCoin         currency.Code `json:"asset_coin"`
	PositionValue     types.Number  `json:"position_value"`
	Leverage          types.Number  `json:"leverage"`
	CreateTime        types.Time    `json:"create_time"`
	UpdateTime        types.Time    `json:"update_time"`
	ExchangeType      string        `json:"exchange_type"`
}

// CrossExchangeAdlRank holds the ADL position reduction ranking for a CrossEx position.
type CrossExchangeAdlRank struct {
	UserID               string `json:"user_id"`
	Symbol               string `json:"symbol"`
	CrossExchangeAdlRank string `json:"crossex_adl_rank"`
	ExchangeAdlRank      string `json:"exchange_adl_rank"`
}

// GetCrossExchangeOpenOrdersRequest holds query parameters for the open orders endpoint.
type GetCrossExchangeOpenOrdersRequest struct {
	Symbol       string
	ExchangeType string
	BusinessType string
}

// GetCrossExchangeOrderHistoryRequest holds query parameters for the order history endpoint.
type GetCrossExchangeOrderHistoryRequest struct {
	Page      uint64
	Limit     uint64
	Symbol    string
	From      int64
	To        int64
	Attribute string
}

// GetCrossExchangePositionHistoryRequest holds query parameters for the position history endpoints.
type GetCrossExchangePositionHistoryRequest struct {
	Page   uint64
	Limit  uint64
	Symbol string
	From   int64
	To     int64
}

// CrossExchangeHistoricalPosition holds a closed CrossEx contract position record.
type CrossExchangeHistoricalPosition struct {
	PositionID     string        `json:"position_id"`
	UserID         string        `json:"user_id"`
	Symbol         string        `json:"symbol"`
	PositionSide   currency.Code `json:"position_side"`
	ClosedType     string        `json:"closed_type"`
	ClosedPNL      string        `json:"closed_pnl"`
	ClosedPNLRate  types.Number  `json:"closed_pnl_rate"`
	OpenAvgPrice   types.Number  `json:"open_avg_price"`
	ClosedAvgPrice types.Number  `json:"closed_avg_price"`
	MaxPositionQty types.Number  `json:"max_position_qty"`
	ExchangeType   string        `json:"exchange_type"`
	CreateTime     types.Time    `json:"create_time"`
	UpdateTime     types.Time    `json:"update_time"`
}

// CrossExchangeHistoricalMarginPosition holds a closed CrossEx leveraged position record.
type CrossExchangeHistoricalMarginPosition struct {
	PositionID          string       `json:"position_id"`
	UserID              string       `json:"user_id"`
	Symbol              string       `json:"symbol"`
	PositionSide        string       `json:"position_side"`
	ClosedType          string       `json:"closed_type"`
	ClosedPNL           types.Number `json:"closed_pnl"`
	ClosedPNLRate       types.Number `json:"closed_pnl_rate"`
	OpenAveragePrice    types.Number `json:"open_avg_price"`
	ClosedAvgPrice      types.Number `json:"closed_avg_price"`
	MaxPositionQuantity types.Number `json:"max_position_qty"`
	ExchangeType        string       `json:"exchange_type"`
	CreateTime          types.Time   `json:"create_time"`
	UpdateTime          types.Time   `json:"update_time"`
}

// GetCrossExchangeMarginInterestHistoryRequest holds query parameters for the margin interest history endpoint.
type GetCrossExchangeMarginInterestHistoryRequest struct {
	Symbol       string
	From         int64
	To           int64
	Page         uint64
	Limit        uint64
	ExchangeType string
}

// CrossExchangeMarginInterestRecord holds a single leveraged interest deduction record.
type CrossExchangeMarginInterestRecord struct {
	UserID        string        `json:"user_id"`
	Symbol        string        `json:"symbol"`
	InterestID    string        `json:"interest_id"`
	LiabilityID   string        `json:"liability_id"`
	Liability     string        `json:"liability"`
	LiabilityCoin currency.Code `json:"liability_coin"`
	Interest      string        `json:"interest"`
	InterestRate  types.Number  `json:"interest_rate"`
	InterestType  string        `json:"interest_type"`
	CreateTime    types.Time    `json:"create_time"`
	ExchangeType  string        `json:"exchange_type"`
}

// GetCrossExchangeTradeHistoryRequest holds query parameters for the trade history endpoint.
type GetCrossExchangeTradeHistoryRequest struct {
	Page   uint64
	Limit  uint64
	Symbol string
	From   int64
	To     int64
}

// CrossExchangeTrade holds a single CrossEx trade record.
type CrossExchangeTrade struct {
	UserID         string        `json:"user_id"`
	TransactionID  string        `json:"transaction_id"`
	FilledRecordID string        `json:"filled_record_id"`
	OrderID        string        `json:"order_id"`
	Text           string        `json:"text"`
	Symbol         string        `json:"symbol"`
	ExchangeType   string        `json:"exchange_type"`
	BusinessType   string        `json:"business_type"`
	Side           currency.Code `json:"side"`
	Quantity       types.Number  `json:"qty"`
	Price          types.Number  `json:"price"`
	Fee            types.Number  `json:"fee"`
	FeeCoin        currency.Code `json:"fee_coin"`
	CreateTime     types.Time    `json:"create_time"`
}

// GetCrossExchangeAccountBookRequest holds query parameters for the account book endpoint.
type GetCrossExchangeAccountBookRequest struct {
	Page          uint64
	Limit         uint64
	Coin          currency.Code
	StatementType string
	From          int64
	To            int64
}

// CrossExchangeAccountBookRecord holds a single account asset change record.
type CrossExchangeAccountBookRecord struct {
	ID            string       `json:"id"`
	UserID        string       `json:"user_id"`
	BusinessID    string       `json:"business_id"`
	StatementType string       `json:"statement_type"`
	ExchangeType  string       `json:"exchange_type"`
	Change        string       `json:"change"`
	Balance       types.Number `json:"balance"`
	CreateTime    types.Time   `json:"create_time"`
}

// CrossExchangeCoinDiscountRate holds the currency discount rate information for a CrossEx asset.
type CrossExchangeCoinDiscountRate struct {
	Coin         currency.Code `json:"coin"`
	ExchangeType string        `json:"exchange_type"`
	Tier         string        `json:"tier"`
	MinValue     types.Number  `json:"min_value"`
	MaxValue     types.Number  `json:"max_value"`
	DiscountRate types.Number  `json:"discount_rate"`
}
