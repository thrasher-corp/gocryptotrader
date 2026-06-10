package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CrossexSymbol holds symbol information for a CrossEx trading pair.
type CrossexSymbol struct {
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

// CrossexRiskLimitTier holds a single risk limit tier for a CrossEx symbol.
type CrossexRiskLimitTier struct {
	MinRiskLimitValue types.Number `json:"min_risk_limit_value"`
	QuickAdjAmount    types.Number `json:"quick_adj_amount"`
	LeverageMax       types.Number `json:"leverage_max"`
	MaintenanceRate   types.Number `json:"maintenance_rate"`
}

// CrossexRiskLimit holds the risk limit tiers for a CrossEx symbol.
type CrossexRiskLimit struct {
	Symbol string                  `json:"symbol"`
	Tiers  []*CrossexRiskLimitTier `json:"lec"`
}

// CrossexTransferCoin holds information about a currency supported for CrossEx transfers.
type CrossexTransferCoin struct {
	Coin           currency.Code `json:"coin"`
	MinTransAmount types.Number  `json:"min_trans_amount"`
	EstimatedFee   types.Number  `json:"est_fee"`
	Precision      int64         `json:"precision"`
	IsDisabled     int64         `json:"is_disabled"`
}

// GetCrossexTransferHistoryRequest holds query parameters for the transfer history endpoint.
type GetCrossexTransferHistoryRequest struct {
	Coin          currency.Code
	OrderID       string
	ClientOrderID string
	To            int64
	From          int64
	Limit         uint64
}

// CrossexTransferRecord holds a single fund transfer record.
type CrossexTransferRecord struct {
	ID              string        `json:"id"`
	ClientOrderID   string        `json:"client_order_id"`
	FromAccountType string        `json:"from_account_type"`
	ToAccountType   string        `json:"to_account_type"`
	Coin            currency.Code `json:"coin"`
	AmountActual    string        `json:"amount_actual"`
	Status          string        `json:"status"`
	FailReason      string        `json:"fail_reason"`
}

// CrossexTransferRequest is the request body for a CrossEx fund transfer.
type CrossexTransferRequest struct {
	Coin   currency.Code `json:"coin"`
	Amount float64       `json:"amount,string"`
	From   string        `json:"from"`
	To     string        `json:"to"`
	Text   string        `json:"text,omitempty"`
}

// CrossexTransferResponse holds the result of a CrossEx fund transfer.
type CrossexTransferResponse struct {
	TxID string `json:"tx_id"`
}

// CrossexOrderCreateRequest is the request body for creating a CrossEx order.
type CrossexOrderCreateRequest struct {
	Text         string     `json:"text,omitempty"`
	ExchangeType string     `json:"exchange_type"`
	Symbol       string     `json:"symbol"`
	Side         order.Side `json:"side"`
	Type         string     `json:"type"`
	Quantity     float64    `json:"qty,string,omitempty"`
	Price        float64    `json:"price,string,omitempty"`
	QuickQty     float64    `json:"quick_qty,string,omitempty"`
	ReduceOnly   bool       `json:"reduce_only,omitempty"`
	PositionSide order.Side `json:"position_side,omitempty"`
}

// CrossexOrder holds the full detail of a CrossEx order.
type CrossexOrder struct {
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

// CrossexOrderUpdateRequest is the request body for modifying a CrossEx order.
type CrossexOrderUpdateRequest struct {
	Quantity float64 `json:"qty,string,omitempty"`
	Price    float64 `json:"price,string,omitempty"`
}

// CrossexOrderActionResponse holds the result of a CrossEx order action (modify/cancel).
type CrossexOrderActionResponse struct {
	OrderID string `json:"order_id"`
	Text    string `json:"text"`
}

// CrossexConvertQuoteRequest is the request body for a CrossEx flash swap inquiry.
type CrossexConvertQuoteRequest struct {
	ExchangeType string        `json:"exchange_type"`
	FromCoin     currency.Code `json:"from_coin"`
	ToCoin       currency.Code `json:"to_coin"`
	FromAmount   float64       `json:"from_amount,string"`
}

// CrossexConvertQuoteResponse holds the quote returned from a flash swap inquiry.
type CrossexConvertQuoteResponse struct {
	QuoteID    string        `json:"quote_id"`
	ValidMs    string        `json:"valid_ms"`
	FromCoin   currency.Code `json:"from_coin"`
	ToCoin     currency.Code `json:"to_coin"`
	FromAmount types.Number  `json:"from_amount"`
	ToAmount   types.Number  `json:"to_amount"`
	Price      types.Number  `json:"price"`
}

// CrossexConvertOrderRequest is the request body for executing a CrossEx flash swap.
type CrossexConvertOrderRequest struct {
	QuoteID string `json:"quote_id"`
}

// CrossexConvertOrderResponse holds the result of a CrossEx flash swap execution.
type CrossexConvertOrderResponse struct {
	OrderID string `json:"order_id"`
	Text    string `json:"text"`
}

// CrossexAccountAsset holds per-exchange asset information within a CrossEx account.
type CrossexAccountAsset struct {
	UserID  string       `json:"user_id"`
	Balance types.Number `json:"balance"`
	Equity  types.Number `json:"equity"`
	PNL     types.Number `json:"pnl"`
}

// CrossexAccount holds CrossEx account asset information.
type CrossexAccount struct {
	UserID                string                 `json:"user_id"`
	AvailableMargin       types.Number           `json:"available_margin"`
	MarginBalance         types.Number           `json:"margin_balance"`
	InitialMargin         types.Number           `json:"initial_margin"`
	MaintenanceMargin     types.Number           `json:"maintenance_margin"`
	InitialMarginRate     types.Number           `json:"initial_margin_rate"`
	MaintenanceMarginRate types.Number           `json:"maintenance_margin_rate"`
	PositionMode          string                 `json:"position_mode"`
	AccountLimit          types.Number           `json:"account_limit"`
	CreateTime            types.Time             `json:"create_time"`
	ExchangeType          string                 `json:"exchange_type"`
	AccountMode           string                 `json:"account_mode"`
	UpdateTime            types.Time             `json:"update_time"`
	Assets                []*CrossexAccountAsset `json:"assets,omitempty"`
}

// CrossexAccountUpdateRequest is the request body for modifying CrossEx account settings.
type CrossexAccountUpdateRequest struct {
	PositionMode string `json:"position_mode,omitempty"`
	AccountMode  string `json:"account_mode,omitempty"`
	ExchangeType string `json:"exchange_type,omitempty"`
}

// CrossexAccountUpdateResponse holds the result of a CrossEx account settings update.
type CrossexAccountUpdateResponse struct {
	PositionMode string `json:"position_mode"`
	AccountMode  string `json:"account_mode"`
	ExchangeType string `json:"exchange_type"`
}

// CrossexLeverageRequest is the request body for setting CrossEx leverage.
type CrossexLeverageRequest struct {
	Symbol   string  `json:"symbol"`
	Leverage float64 `json:"leverage,string"`
}

// CrossexLeverageResponse holds the result of a CrossEx leverage change.
type CrossexLeverageResponse struct {
	Symbol   string       `json:"symbol"`
	Leverage types.Number `json:"leverage"`
}

// CrossexClosePositionRequest is the request body for fully closing a CrossEx position.
type CrossexClosePositionRequest struct {
	Symbol       string `json:"symbol"`
	PositionSide string `json:"position_side,omitempty"`
}

// CrossexInterestRate holds margin asset interest rate information.
type CrossexInterestRate struct {
	Coin             currency.Code `json:"coin"`
	ExchangeType     string        `json:"exchange_type"`
	HourInterestRate types.Number  `json:"hour_interest_rate"`
	Time             types.Time    `json:"time"`
}

// CrossexSpecialFee holds the special fee rates for a specific CrossEx symbol.
type CrossexSpecialFee struct {
	Symbol       currency.Pair `json:"symbol"`
	MakerFeeRate types.Number  `json:"maker_fee_rate,omitempty"`
	TakerFeeRate types.Number  `json:"taker_fee_rate"`
}

// CrossexFee holds the fee rate information for a CrossEx exchange type.
type CrossexFee struct {
	ExchangeType   string               `json:"exchange_type"`
	SpotMakerFee   types.Number         `json:"spot_maker_fee"`
	SpotTakerFee   types.Number         `json:"spot_taker_fee"`
	FutureMakerFee types.Number         `json:"future_maker_fee"`
	FutureTakerFee types.Number         `json:"future_taker_fee"`
	SpecialFeeList []*CrossexSpecialFee `json:"special_fee_list"`
}

// CrossexPosition holds a CrossEx contract position.
type CrossexPosition struct {
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

// CrossexMarginPosition holds a CrossEx leveraged (margin) position.
type CrossexMarginPosition struct {
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

// CrossexAdlRank holds the ADL position reduction ranking for a CrossEx position.
type CrossexAdlRank struct {
	UserID          string `json:"user_id"`
	Symbol          string `json:"symbol"`
	CrossexAdlRank  string `json:"crossex_adl_rank"`
	ExchangeAdlRank string `json:"exchange_adl_rank"`
}

// GetCrossexOpenOrdersRequest holds query parameters for the open orders endpoint.
type GetCrossexOpenOrdersRequest struct {
	Symbol       string
	ExchangeType string
	BusinessType string
}

// GetCrossexOrderHistoryRequest holds query parameters for the order history endpoint.
type GetCrossexOrderHistoryRequest struct {
	Page      uint64
	Limit     uint64
	Symbol    string
	From      int64
	To        int64
	Attribute string
}

// GetCrossexPositionHistoryRequest holds query parameters for the position history endpoints.
type GetCrossexPositionHistoryRequest struct {
	Page   uint64
	Limit  uint64
	Symbol string
	From   int64
	To     int64
}

// CrossexHistoricalPosition holds a closed CrossEx contract position record.
type CrossexHistoricalPosition struct {
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

// CrossexHistoricalMarginPosition holds a closed CrossEx leveraged position record.
type CrossexHistoricalMarginPosition struct {
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

// GetCrossexMarginInterestHistoryRequest holds query parameters for the margin interest history endpoint.
type GetCrossexMarginInterestHistoryRequest struct {
	Symbol       string
	From         int64
	To           int64
	Page         uint64
	Limit        uint64
	ExchangeType string
}

// CrossexMarginInterestRecord holds a single leveraged interest deduction record.
type CrossexMarginInterestRecord struct {
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

// GetCrossexTradeHistoryRequest holds query parameters for the trade history endpoint.
type GetCrossexTradeHistoryRequest struct {
	Page   uint64
	Limit  uint64
	Symbol string
	From   int64
	To     int64
}

// CrossexTrade holds a single CrossEx trade record.
type CrossexTrade struct {
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

// GetCrossexAccountBookRequest holds query parameters for the account book endpoint.
type GetCrossexAccountBookRequest struct {
	Page          uint64
	Limit         uint64
	Coin          currency.Code
	StatementType string
	From          int64
	To            int64
}

// CrossexAccountBookRecord holds a single account asset change record.
type CrossexAccountBookRecord struct {
	ID            string       `json:"id"`
	UserID        string       `json:"user_id"`
	BusinessID    string       `json:"business_id"`
	StatementType string       `json:"statement_type"`
	ExchangeType  string       `json:"exchange_type"`
	Change        string       `json:"change"`
	Balance       types.Number `json:"balance"`
	CreateTime    types.Time   `json:"create_time"`
}

// CrossexCoinDiscountRate holds the currency discount rate information for a CrossEx asset.
type CrossexCoinDiscountRate struct {
	Coin         currency.Code `json:"coin"`
	ExchangeType string        `json:"exchange_type"`
	Tier         string        `json:"tier"`
	MinValue     types.Number  `json:"min_value"`
	MaxValue     types.Number  `json:"max_value"`
	DiscountRate types.Number  `json:"discount_rate"`
}
