package gateio

import (
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CrossEx spellings of time in force values order.TimeInForce cannot express: post-only renders as POSTONLY, and
// retail price improvement has no order.TimeInForce flag
const (
	crossExPostOnlyTIF               = "POC"
	crossExRetailPriceImprovementTIF = "RPI"
)

// CrossExchangeSymbolIdentifier identifies a CrossEx market without relying on a caller-composed symbol string.
type CrossExchangeSymbolIdentifier struct {
	Exchange string
	Asset    asset.Item
	Pair     currency.Pair
}

// Validate ensures the identifier holds everything needed to render a CrossEx symbol. Which venues and markets Gate
// actually lists is left for Gate to reject, as that set changes without notice.
func (c CrossExchangeSymbolIdentifier) Validate() error {
	if strings.TrimSpace(c.Exchange) == "" {
		return errCrossExchangeExchangeTypeRequired
	}
	if !c.Pair.IsPopulated() {
		return currency.ErrCurrencyPairEmpty
	}
	switch c.Asset {
	case asset.Spot, asset.Margin, asset.Futures:
	default:
		return asset.ErrNotSupported
	}
	return nil
}

// Format returns the identifier in Gate's {Exchange}_{Business}_{Base}_{Counter} format, erroring when it cannot be
// rendered as a whole symbol.
func (c CrossExchangeSymbolIdentifier) Format() (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}
	return c.String(), nil
}

// String renders the identifier in Gate's {Exchange}_{Business}_{Base}_{Counter} format from whatever fields are set.
// Use Validate or Format when the result must be a symbol Gate can act on.
func (c CrossExchangeSymbolIdentifier) String() string {
	pair := c.Pair.Format(currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true})
	return strings.ToUpper(strings.TrimSpace(c.Exchange)) + "_" + c.businessType() + "_" + pair.String()
}

// MarshalJSON implements json.Marshaler.
func (c CrossExchangeSymbolIdentifier) MarshalJSON() ([]byte, error) {
	formatted, err := c.Format()
	if err != nil {
		return nil, err
	}
	return json.Marshal(formatted)
}

// IsEmpty returns whether no part of the identifier has been set.
func (c CrossExchangeSymbolIdentifier) IsEmpty() bool {
	return strings.TrimSpace(c.Exchange) == "" && c.Asset == asset.Empty && c.Pair.IsEmpty()
}

func (c CrossExchangeSymbolIdentifier) businessType() string {
	if c.Asset == asset.Futures {
		return "FUTURE"
	}
	return c.Asset.Upper()
}

// CrossExchangeSymbol holds symbol information for a CrossEx trading pair.
type CrossExchangeSymbol struct {
	Symbol          string        `json:"symbol"`
	ExchangeType    string        `json:"exchange_type"`
	BusinessType    string        `json:"business_type"`
	State           string        `json:"state"`
	MinSize         types.Number  `json:"min_size"`
	MinNotional     types.Number  `json:"min_notional"`
	LotSize         types.Number  `json:"lot_size"`
	TickSize        types.Number  `json:"tick_size"`
	MaxNumOrders    types.Number  `json:"max_num_orders"`
	MaxMarketSize   types.Number  `json:"max_market_size"`
	MaxLimitSize    types.Number  `json:"max_limit_size"`
	ContractSize    types.Number  `json:"contract_size"` // Deprecated by Gate, as quantity is used uniformly
	LiquidationFee  types.Number  `json:"liquidation_fee"`
	DefaultLeverage types.Number  `json:"default_leverage"` // Returned by the live API although undocumented
	DelistTime      types.Time    `json:"delist_time"`
	SupportRPI      types.Boolean `json:"support_rpi"`
	SupportCross    types.Boolean `json:"support_cross"`
}

// CrossExchangeRiskLimitTier holds a single risk limit tier for a CrossEx symbol.
type CrossExchangeRiskLimitTier struct {
	MinRiskLimitValue      types.Number `json:"min_risk_limit_value"`
	MaxRiskLimitValue      types.Number `json:"max_risk_limit_value"`
	QuickCalculationAmount types.Number `json:"quick_cal_amount"`
	LeverageMax            types.Number `json:"leverage_max"`
	MaintenanceRate        types.Number `json:"maintenance_rate"`
	Tier                   string       `json:"tier"`
}

// CrossExchangeRiskLimit holds the risk limit tiers for a CrossEx symbol.
type CrossExchangeRiskLimit struct {
	Symbol string                        `json:"symbol"`
	Tiers  []*CrossExchangeRiskLimitTier `json:"tiers"`
}

// CrossExchangeTransferCoin holds information about a currency supported for CrossEx transfers.
type CrossExchangeTransferCoin struct {
	Coin           currency.Code `json:"coin"`
	MinTransAmount types.Number  `json:"min_trans_amount"`
	EstimatedFee   types.Number  `json:"est_fee"`
	Precision      uint64        `json:"precision"`
	IsDisabled     uint64        `json:"is_disabled"`
}

// CrossExchangeTransferRequest is the request body for a CrossEx fund transfer.
type CrossExchangeTransferRequest struct {
	Coin   currency.Code `json:"coin"`
	Amount types.Number  `json:"amount"`
	From   string        `json:"from"` // SPOT, CROSSEX or a CROSSEX_{exchange} account
	To     string        `json:"to"`   // SPOT, CROSSEX or a CROSSEX_{exchange} account
	Text   string        `json:"text,omitempty"`
}

// CrossExchangeTransferResponse holds the result of a CrossEx fund transfer.
type CrossExchangeTransferResponse struct {
	TxID string `json:"tx_id"`
	Text string `json:"text"`
}

// GetCrossExchangeTransferHistoryRequest holds query parameters for the transfer history endpoint.
type GetCrossExchangeTransferHistoryRequest struct {
	Coin      currency.Code
	OrderID   string // OrderID matches either a transfer tx_id or client-defined text ID.
	StartTime time.Time
	EndTime   time.Time
	Page      uint64
	Limit     uint64
}

// CrossExchangeTransferRecord holds a single fund transfer record.
type CrossExchangeTransferRecord struct {
	ID              string        `json:"id"`
	Text            string        `json:"text"`
	FromAccountType string        `json:"from_account_type"`
	ToAccountType   string        `json:"to_account_type"`
	Coin            currency.Code `json:"coin"`
	Amount          types.Number  `json:"amount"`
	ActualReceive   types.Number  `json:"actual_receive"` // Only set once the transfer succeeds
	Status          string        `json:"status"`
	FailReason      string        `json:"fail_reason"`
	CreateTime      types.Time    `json:"create_time"`
	UpdateTime      types.Time    `json:"update_time"`
}

// CrossExchangeOrderCreateRequest is the request body for creating a CrossEx order.
type CrossExchangeOrderCreateRequest struct {
	Text                   string                        `json:"text,omitempty"`
	Symbol                 CrossExchangeSymbolIdentifier `json:"symbol"`
	Side                   order.Side                    `json:"side"`
	OrderType              order.Type                    `json:"type,omitempty"`
	TimeInForce            order.TimeInForce             `json:"time_in_force,omitempty"`
	RetailPriceImprovement bool                          `json:"-"` // Sends the RPI time in force, which order.TimeInForce cannot express
	Quantity               types.Number                  `json:"qty,omitempty"`
	Price                  types.Number                  `json:"price,omitempty"`
	QuoteQuantity          types.Number                  `json:"quote_qty,omitempty"`
	ReduceOnly             bool                          `json:"reduce_only,omitempty,string"`
	PositionSide           order.Side                    `json:"position_side,omitempty"` // Omission defaults to NONE in single position mode.
}

// MarshalJSON renders OrderType as a string and maps order.PostOnly and RetailPriceImprovement to CrossEx's POC and
// RPI values.
func (c *CrossExchangeOrderCreateRequest) MarshalJSON() ([]byte, error) {
	type Alias CrossExchangeOrderCreateRequest
	aux := &struct {
		OrderType   string `json:"type,omitempty"`
		TimeInForce string `json:"time_in_force,omitempty"`
		*Alias
	}{Alias: (*Alias)(c)}
	if c.OrderType != order.UnknownType {
		aux.OrderType = c.OrderType.String()
	}
	switch {
	case c.RetailPriceImprovement:
		aux.TimeInForce = crossExRetailPriceImprovementTIF
	case c.TimeInForce.Is(order.PostOnly):
		aux.TimeInForce = crossExPostOnlyTIF
	case c.TimeInForce != order.UnknownTIF:
		aux.TimeInForce = c.TimeInForce.String()
	}
	return json.Marshal(aux)
}

// CrossExchangeOrderActionResponse holds the result of a CrossEx order action.
type CrossExchangeOrderActionResponse struct {
	OrderID string `json:"order_id"`
	Text    string `json:"text"`
}

// CrossExchangeBatchCancelOrderRequest identifies a CrossEx order to cancel.
type CrossExchangeBatchCancelOrderRequest struct {
	OrderID string `json:"order_id,omitempty"`
	Text    string `json:"text,omitempty"`
}

// CrossExchangeBatchCancelOrderResponse holds an individual CrossEx batch cancellation result.
type CrossExchangeBatchCancelOrderResponse struct {
	OrderID  string        `json:"order_id"`
	Text     string        `json:"text"`
	Accepted types.Boolean `json:"accepted"`
	Label    string        `json:"label"`
	Message  string        `json:"message"`
}

// CrossExchangeOrderUpdateRequest is the request body for modifying a CrossEx order.
type CrossExchangeOrderUpdateRequest struct {
	Quantity types.Number `json:"qty,omitempty"`
	Price    types.Number `json:"price,omitempty"`
}

// CrossExchangeOrder holds the full detail of a CrossEx order.
type CrossExchangeOrder struct {
	UserID               string        `json:"user_id"`
	OrderID              string        `json:"order_id"`
	Text                 string        `json:"text"`
	State                string        `json:"state"`
	Symbol               string        `json:"symbol"`
	Side                 string        `json:"side"`
	Type                 string        `json:"type"`
	Attribute            string        `json:"attribute"`
	ExchangeType         string        `json:"exchange_type"`
	BusinessType         string        `json:"business_type"`
	Quantity             types.Number  `json:"qty"`
	QuoteQuantity        types.Number  `json:"quote_qty"`
	Price                types.Number  `json:"price"`
	TimeInForce          string        `json:"time_in_force"` // GTC, IOC, FOK, POC or RPI
	ExecutedQuantity     types.Number  `json:"executed_qty"`
	ExecutedAmount       types.Number  `json:"executed_amount"`
	ExecutedAveragePrice types.Number  `json:"executed_avg_price"`
	FeeCoin              currency.Code `json:"fee_coin"`
	Fee                  types.Number  `json:"fee"`
	ReduceOnly           types.Boolean `json:"reduce_only"`
	Leverage             types.Number  `json:"leverage"`
	Reason               string        `json:"reason"`
	LastExecutedQuantity types.Number  `json:"last_executed_qty"`
	LastExecutedPrice    types.Number  `json:"last_executed_price"`
	LastExecutedAmount   types.Number  `json:"last_executed_amount"`
	PositionSide         string        `json:"position_side"`
	CreateTime           types.Time    `json:"create_time"`
	UpdateTime           types.Time    `json:"update_time"`
}

// CrossExchangeConvertQuoteRequest is the request body for a CrossEx flash swap inquiry.
type CrossExchangeConvertQuoteRequest struct {
	ExchangeType string        `json:"exchange_type"`
	FromCoin     currency.Code `json:"from_coin"`
	ToCoin       currency.Code `json:"to_coin"`
	FromAmount   types.Number  `json:"from_amount"`
}

// CrossExchangeConvertQuoteResponse holds the quote returned from a flash swap inquiry.
type CrossExchangeConvertQuoteResponse struct {
	QuoteID           string        `json:"quote_id"`
	ValidMilliseconds types.Number  `json:"valid_ms"` // How long the quote remains valid
	FromCoin          currency.Code `json:"from_coin"`
	ToCoin            currency.Code `json:"to_coin"`
	FromAmount        types.Number  `json:"from_amount"`
	ToAmount          types.Number  `json:"to_amount"`
	Price             types.Number  `json:"price"`
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

// CrossExchangeAccountUpdateRequest is the request body for modifying CrossEx account settings. Setting
// PositionMode, with ExchangeType in isolated exchange mode, changes the futures position mode, while AccountMode
// changes the account mode.
type CrossExchangeAccountUpdateRequest struct {
	PositionMode string `json:"position_mode,omitempty"` // SINGLE or DUAL
	AccountMode  string `json:"account_mode,omitempty"`  // CROSS_EXCHANGE or ISOLATED_EXCHANGE
	ExchangeType string `json:"exchange_type,omitempty"`
}

// CrossExchangeAccountUpdateResponse holds the result of a CrossEx account settings update.
type CrossExchangeAccountUpdateResponse struct {
	PositionMode string `json:"position_mode"`
	AccountMode  string `json:"account_mode"`
	ExchangeType string `json:"exchange_type"`
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
	UpdateTime            types.Time                   `json:"update_time"`
	AccountMode           string                       `json:"account_mode"`
	ExchangeType          string                       `json:"exchange_type"`
	Assets                []*CrossExchangeAccountAsset `json:"assets"`
}

// CrossExchangeAccountAsset holds per-exchange asset information within a CrossEx account.
type CrossExchangeAccountAsset struct {
	UserID                     string        `json:"user_id"`
	Coin                       currency.Code `json:"coin"`
	ExchangeType               string        `json:"exchange_type"`
	Balance                    types.Number  `json:"balance"`
	UnrealisedPNL              types.Number  `json:"upnl"`
	Equity                     types.Number  `json:"equity"`
	FuturesInitialMargin       types.Number  `json:"futures_initial_margin"`
	FuturesMaintenanceMargin   types.Number  `json:"futures_maintenance_margin"`
	BorrowingInitialMargin     types.Number  `json:"borrowing_initial_margin"`
	BorrowingMaintenanceMargin types.Number  `json:"borrowing_maintenance_margin"`
	AvailableBalance           types.Number  `json:"available_balance"`
	Liability                  types.Number  `json:"liability"`
}

// CrossExchangeLeverageRequest is the request body for setting CrossEx leverage.
type CrossExchangeLeverageRequest struct {
	Symbol   CrossExchangeSymbolIdentifier `json:"symbol"`
	Leverage types.Number                  `json:"leverage"`
}

// CrossExchangeLeverageResponse holds the result of a CrossEx leverage change.
type CrossExchangeLeverageResponse struct {
	Symbol   string       `json:"symbol"`
	Leverage types.Number `json:"leverage"`
}

// CrossExchangeClosePositionRequest is the request body for fully closing a CrossEx position.
type CrossExchangeClosePositionRequest struct {
	Symbol       CrossExchangeSymbolIdentifier `json:"symbol"`
	PositionSide string                        `json:"position_side,omitempty"` // Required for margin positions; NONE, LONG or SHORT
}

// CrossExchangeMarginModeRequest is the request body for changing a futures position's margin mode
type CrossExchangeMarginModeRequest struct {
	Symbol     CrossExchangeSymbolIdentifier `json:"symbol"`
	MarginMode string                        `json:"margin_mode"` // CROSS or ISOLATED
}

// CrossExchangeMarginModeResponse holds a futures position's margin mode
type CrossExchangeMarginModeResponse struct {
	Symbol     string `json:"symbol"`
	MarginMode string `json:"margin_mode"`
}

// CrossExchangeIsolatedMarginRequest is the request body for adjusting an isolated futures position's margin
type CrossExchangeIsolatedMarginRequest struct {
	Symbol CrossExchangeSymbolIdentifier `json:"symbol"`
	// Margin increases the position's margin when positive and decreases it when negative; Gate truncates it to two
	// decimal places
	Margin       types.Number `json:"margin"`
	PositionSide string       `json:"position_side,omitempty"` // NONE, LONG or SHORT; NONE when omitted
}

// CrossExchangeIsolatedMarginResponse holds the result of an isolated margin adjustment
type CrossExchangeIsolatedMarginResponse struct {
	Symbol       string       `json:"symbol"`
	Margin       types.Number `json:"margin"`
	PositionSide string       `json:"position_side"`
}

// CrossExchangeInterestRate holds margin asset interest rate information.
type CrossExchangeInterestRate struct {
	Coin             currency.Code `json:"coin"`
	ExchangeType     string        `json:"exchange_type"`
	HourInterestRate types.Number  `json:"hour_interest_rate"`
	Time             types.Time    `json:"time"`
}

// CrossExchangeFee holds the fee rate information for a CrossEx exchange type.
type CrossExchangeFee struct {
	ExchangeType      string                     `json:"exchange_type"`
	SpotMakerFee      types.Number               `json:"spot_maker_fee"`
	SpotTakerFee      types.Number               `json:"spot_taker_fee"`
	SpotRPIMakerFee   types.Number               `json:"spot_rpi_maker_fee"`
	FutureMakerFee    types.Number               `json:"future_maker_fee"`
	FutureTakerFee    types.Number               `json:"future_taker_fee"`
	FutureRPIMakerFee types.Number               `json:"future_rpi_maker_fee"`
	SpecialFeeList    []*CrossExchangeSpecialFee `json:"special_fee_list"`
}

// CrossExchangeSpecialFee holds the special fee rates for a specific CrossEx symbol.
type CrossExchangeSpecialFee struct {
	Symbol       string       `json:"symbol"`
	TakerFeeRate types.Number `json:"taker_fee_rate"`
	MakerFeeRate types.Number `json:"maker_fee_rate"`
	RPIFeeRate   types.Number `json:"rpi_fee_rate"`
}

// CrossExchangePosition holds a CrossEx contract position.
type CrossExchangePosition struct {
	UserID            string       `json:"user_id"`
	PositionID        string       `json:"position_id"`
	Symbol            string       `json:"symbol"`
	PositionSide      string       `json:"position_side"`
	InitialMargin     types.Number `json:"initial_margin"`
	IsolatedMargin    types.Number `json:"isolated_margin"` // Zero in cross margin mode
	MarginMode        string       `json:"margin_mode"`
	MaintenanceMargin types.Number `json:"maintenance_margin"`
	PositionQuantity  types.Number `json:"position_qty"`
	PositionValue     types.Number `json:"position_value"`
	UnrealisedPNL     types.Number `json:"upnl"`
	UnrealisedPNLRate types.Number `json:"upnl_rate"`
	EntryPrice        types.Number `json:"entry_price"`
	LiquidationPrice  types.Number `json:"liq_price"` // Zero in cross margin mode, or for an isolated position that cannot be liquidated
	MarkPrice         types.Number `json:"mark_price"`
	Leverage          types.Number `json:"leverage"`
	MaxLeverage       types.Number `json:"max_leverage"`
	RiskLimit         types.Number `json:"risk_limit"`
	Fee               types.Number `json:"fee"`
	FundingFee        types.Number `json:"funding_fee"` // Accumulated; positive values are gains
	FundingTime       types.Time   `json:"funding_time"`
	CreateTime        types.Time   `json:"create_time"`
	UpdateTime        types.Time   `json:"update_time"`
	ClosedPNL         types.Number `json:"closed_pnl"`
}

// CrossExchangeMarginPosition holds a CrossEx leveraged (margin) position.
type CrossExchangeMarginPosition struct {
	UserID              string        `json:"user_id"`
	PositionID          string        `json:"position_id"`
	Symbol              string        `json:"symbol"`
	PositionSide        string        `json:"position_side"`
	InitialMargin       types.Number  `json:"initial_margin"`
	MaintenanceMargin   types.Number  `json:"maintenance_margin"`
	AssetQuantity       types.Number  `json:"asset_qty"`
	AssetCoin           currency.Code `json:"asset_coin"`
	PositionValue       types.Number  `json:"position_value"`
	Liability           types.Number  `json:"liability"`
	LiabilityCoin       currency.Code `json:"liability_coin"`
	Interest            types.Number  `json:"interest"`
	MaxPositionQuantity types.Number  `json:"max_position_qty"`
	EntryPrice          types.Number  `json:"entry_price"`
	IndexPrice          types.Number  `json:"index_price"`
	UnrealisedPNL       types.Number  `json:"upnl"`
	UnrealisedPNLRate   types.Number  `json:"upnl_rate"`
	Leverage            types.Number  `json:"leverage"`
	MaxLeverage         types.Number  `json:"max_leverage"`
	CreateTime          types.Time    `json:"create_time"`
	UpdateTime          types.Time    `json:"update_time"`
}

// CrossExchangeADLRank holds the ADL position reduction ranking for a CrossEx position.
type CrossExchangeADLRank struct {
	UserID               string       `json:"user_id"`
	Symbol               string       `json:"symbol"`
	CrossExchangeADLRank types.Number `json:"crossex_adl_rank"`  // 1 to 5, with 5 reduced first
	ExchangeADLRank      types.Number `json:"exchange_adl_rank"` // The venue's own ranking, whose order differs by venue
}

// GetCrossExchangeOpenOrdersRequest holds query parameters for the open orders endpoint.
type GetCrossExchangeOpenOrdersRequest struct {
	Symbol       CrossExchangeSymbolIdentifier
	ExchangeType string
	BusinessType string
}

// GetCrossExchangeOrderHistoryRequest holds query parameters for the order history endpoint.
type GetCrossExchangeOrderHistoryRequest struct {
	Page       uint64
	Limit      uint64
	Symbol     CrossExchangeSymbolIdentifier
	StartTime  time.Time
	EndTime    time.Time
	Attributes []string // COMMON, LIQ, REDUCE, ADL or SETTLEMENT
}

// GetCrossExchangePositionHistoryRequest holds query parameters for the position history endpoints.
type GetCrossExchangePositionHistoryRequest struct {
	Page      uint64
	Limit     uint64
	Symbol    CrossExchangeSymbolIdentifier
	StartTime time.Time
	EndTime   time.Time
}

// CrossExchangeHistoricalPosition holds a closed CrossEx contract position record.
type CrossExchangeHistoricalPosition struct {
	PositionID          string       `json:"position_id"`
	UserID              string       `json:"user_id"`
	Symbol              string       `json:"symbol"`
	ClosedType          string       `json:"closed_type"` // PARTIAL_CLOSED or COMPLETE_CLOSED
	ClosedPNL           types.Number `json:"closed_pnl"`
	ClosedPNLRate       types.Number `json:"closed_pnl_rate"`
	OpenAveragePrice    types.Number `json:"open_avg_price"`
	ClosedAveragePrice  types.Number `json:"closed_avg_price"`
	MaxPositionQuantity types.Number `json:"max_position_qty"`
	ClosedQuantity      types.Number `json:"closed_qty"`
	ClosedValue         types.Number `json:"closed_value"`
	Fee                 types.Number `json:"fee"`
	LiquidationFee      types.Number `json:"liq_fee"`
	FundingFee          types.Number `json:"funding_fee"`
	PositionSide        string       `json:"position_side"`
	PositionMode        string       `json:"position_mode"`
	Leverage            types.Number `json:"leverage"`
	MarginMode          string       `json:"margin_mode"`
	BusinessType        string       `json:"business_type"`
	CreateTime          types.Time   `json:"create_time"`
	UpdateTime          types.Time   `json:"update_time"`
}

// CrossExchangeHistoricalMarginPosition holds a closed CrossEx leveraged position record.
type CrossExchangeHistoricalMarginPosition struct {
	PositionID          string       `json:"position_id"`
	UserID              string       `json:"user_id"`
	Symbol              string       `json:"symbol"`
	ClosedType          string       `json:"closed_type"` // PARTIAL_CLOSED or COMPLETE_CLOSED
	ClosedPNL           types.Number `json:"closed_pnl"`
	ClosedPNLRate       types.Number `json:"closed_pnl_rate"`
	OpenAveragePrice    types.Number `json:"open_avg_price"`
	ClosedAveragePrice  types.Number `json:"closed_avg_price"`
	MaxPositionQuantity types.Number `json:"max_position_qty"`
	ClosedQuantity      types.Number `json:"closed_qty"`
	ClosedValue         types.Number `json:"closed_value"`
	LiquidationFee      types.Number `json:"liq_fee"`
	PositionSide        string       `json:"position_side"`
	Leverage            types.Number `json:"leverage"`
	Interest            types.Number `json:"interest"`
	BusinessType        string       `json:"business_type"`
	CreateTime          types.Time   `json:"create_time"`
	UpdateTime          types.Time   `json:"update_time"`
}

// GetCrossExchangeMarginInterestHistoryRequest holds query parameters for the margin interest history endpoint.
type GetCrossExchangeMarginInterestHistoryRequest struct {
	Symbol       CrossExchangeSymbolIdentifier
	StartTime    time.Time
	EndTime      time.Time
	Page         uint64
	Limit        uint64
	ExchangeType string
}

// CrossExchangeMarginInterestRecord holds a single leveraged interest deduction record.
type CrossExchangeMarginInterestRecord struct {
	UserID        string        `json:"user_id"`
	Symbol        string        `json:"symbol"`
	InterestID    string        `json:"interest_id"`
	LiabilityID   string        `json:"liability_id"` // The order or position the liability arose from
	Liability     types.Number  `json:"liability"`
	LiabilityCoin currency.Code `json:"liability_coin"`
	Interest      types.Number  `json:"interest"`
	InterestRate  types.Number  `json:"interest_rate"`
	InterestType  string        `json:"interest_type"`
	CreateTime    types.Time    `json:"create_time"`
	ExchangeType  string        `json:"exchange_type"`
}

// GetCrossExchangeTradeHistoryRequest holds query parameters for the trade history endpoint.
type GetCrossExchangeTradeHistoryRequest struct {
	Page      uint64
	Limit     uint64
	Symbol    CrossExchangeSymbolIdentifier
	StartTime time.Time
	EndTime   time.Time
}

// CrossExchangeTrade holds a single CrossEx trade record.
type CrossExchangeTrade struct {
	UserID        string        `json:"user_id"`
	TransactionID string        `json:"transaction_id"`
	OrderID       string        `json:"order_id"`
	Text          string        `json:"text"`
	Symbol        string        `json:"symbol"`
	ExchangeType  string        `json:"exchange_type"`
	BusinessType  string        `json:"business_type"`
	Side          string        `json:"side"`
	Quantity      types.Number  `json:"qty"`
	Price         types.Number  `json:"price"`
	Fee           types.Number  `json:"fee"`
	FeeCoin       currency.Code `json:"fee_coin"`
	FeeRate       types.Number  `json:"fee_rate"`
	MatchRole     string        `json:"match_role"`
	RealisedPNL   types.Number  `json:"rpnl"`
	PositionMode  string        `json:"position_mode"`
	PositionSide  string        `json:"position_side"`
	CreateTime    types.Time    `json:"create_time"`
}

// GetCrossExchangeAccountBookRequest holds query parameters for the account book endpoint.
type GetCrossExchangeAccountBookRequest struct {
	Page          uint64
	Limit         uint64
	Coin          currency.Code
	StatementType string // Such as TRANSACTION, TRADING_FEE, FUNDING_FEE or TRANSFER_IN
	StartTime     time.Time
	EndTime       time.Time
}

// CrossExchangeAccountBookRecord holds a single account asset change record.
type CrossExchangeAccountBookRecord struct {
	ID            string        `json:"id"`
	UserID        string        `json:"user_id"`
	BusinessID    string        `json:"business_id"` // Its meaning depends on StatementType, such as an order ID for TRANSACTION
	StatementType string        `json:"statement_type"`
	ExchangeType  string        `json:"exchange_type"`
	Coin          currency.Code `json:"coin"`
	Symbol        string        `json:"symbol"`
	Change        types.Number  `json:"change"`
	Balance       types.Number  `json:"balance"`
	CreateTime    types.Time    `json:"create_time"`
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

// CrossExchangeMarketTickerResponse holds an exchange's ticker for a CrossEx symbol. Fields a market does not
// support, such as the mark price of a spot market, are zero.
type CrossExchangeMarketTickerResponse struct {
	Symbol            string       `json:"symbol"`
	LastPrice         types.Number `json:"last_price"`
	Open24Hour        types.Number `json:"open_24h"`
	Low24Hour         types.Number `json:"low_24h"`
	High24Hour        types.Number `json:"high_24h"`
	BaseVolume24Hour  types.Number `json:"volume_24h_base"`
	QuoteVolume24Hour types.Number `json:"volume_24h_quote"`
	MarkPrice         types.Number `json:"mark_price"`
	IndexPrice        types.Number `json:"index_price"`
	OpenInterest      types.Number `json:"open_interest"`
	OpenInterestQuote types.Number `json:"open_interest_quote"`
	Timestamp         types.Time   `json:"timestamp"`
}

// CrossExchangeFundingInfoResponse holds an exchange's funding rate information for a CrossEx futures symbol
type CrossExchangeFundingInfoResponse struct {
	Symbol                 string       `json:"symbol"`
	FundingRate            types.Number `json:"funding_rate"`
	FundingIntervalSeconds types.Number `json:"funding_interval"`
	NextFundingTime        types.Time   `json:"funding_time"`
}
