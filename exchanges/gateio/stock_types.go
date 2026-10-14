package gateio

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// GetStockUserAssetsRequest holds the profit and loss calculation options for GetStockUserAssets
type GetStockUserAssetsRequest struct {
	PNLCalculationCostType  uint64 // 1: average cost price, the default, 2: diluted cost price
	PNLCalculationPriceType uint64 // 1: intraday price, the default, 2: latest extended-hours price
}

// StockUserAssetsResponse holds the stock account's assets
type StockUserAssetsResponse struct {
	Equity                    types.Number `json:"equity"`
	Balance                   types.Number `json:"balance"`
	Available                 types.Number `json:"available"`
	PositionMarketValue       types.Number `json:"position_market_value"`
	PositionPNL               types.Number `json:"position_pnl"`
	TodayPNL                  types.Number `json:"today_pnl"`
	OptionPositionMarketValue types.Number `json:"option_position_market_value"`
	OptionPositionPNL         types.Number `json:"option_position_pnl"`
	OptionTodayPNL            types.Number `json:"option_today_pnl"`
	UserExists                bool         `json:"user_exists"`
}

// GetStockSymbolsRequest holds the filters for GetStockSymbols
type GetStockSymbolsRequest struct {
	Symbols                      []string
	Exchange                     string // us, hk, kr or jp
	WithMultilingualDescriptions bool
	Page                         uint64
	PageSize                     uint64 // Defaults to 10, up to 500
}

// StockSymbolDescription holds a symbol description in one language
type StockSymbolDescription struct {
	Language string `json:"lang"`
	Value    string `json:"value"`
}

// StockSymbol holds a stock symbol's trading information
type StockSymbol struct {
	Symbol                 string                    `json:"symbol"`
	Exchange               string                    `json:"exchange"`
	ExchangeDescription    string                    `json:"exchange_desc"`
	QuoteCurrency          currency.Code             `json:"quote_currency"`
	QuoteCurrencyPrecision uint64                    `json:"quote_currency_precision"`
	FXRate                 types.Number              `json:"fx_rate"` // Quote currency to USD
	SymbolDescription      string                    `json:"symbol_desc"`
	Category               string                    `json:"category"`
	AssetType              string                    `json:"asset_type"`   // STOCK or ETF
	TradeStatus            string                    `json:"trade_status"` // pre_market, open, post_market, closed or gt_lp
	TradeMode              uint64                    `json:"trade_mode"`
	OrderFillTiming        uint64                    `json:"order_fill_timing"` // 1: immediate, 2: after pre-market opens, 3: after the regular session opens
	IconLink               string                    `json:"icon_link"`
	QuoteCurrencySymbol    string                    `json:"quote_currency_symbol"`
	PricePrecision         uint64                    `json:"price_precision"`
	VolumePrecision        uint64                    `json:"volume_precision"`
	IsIPO                  bool                      `json:"is_ipo"`
	IPOPrice               types.Number              `json:"ipo_price"`
	SellPriceProtection    types.Number              `json:"sell_price_protection"`
	BuyPriceProtection     types.Number              `json:"buy_price_protection"`
	SymbolDescriptions     []*StockSymbolDescription `json:"symbol_descs"` // Only set when multilingual descriptions are requested
}

// StockSymbolsResponse holds a page of stock symbols
type StockSymbolsResponse struct {
	Total      uint64         `json:"total"`
	TotalPages uint64         `json:"total_page"`
	List       []*StockSymbol `json:"list"`
}

// GetStockSymbolDetailsRequest holds the filters for GetStockSymbolDetails
type GetStockSymbolDetailsRequest struct {
	Symbols  []string
	Exchange string // us, hk, kr or jp
	Page     uint64
	PageSize uint64 // Defaults to 10, up to 500
}

// StockSymbolDetail holds a stock symbol's trading rules
type StockSymbolDetail struct {
	Symbol                 string                    `json:"symbol"`
	Exchange               string                    `json:"exchange"`
	ExchangeDescription    string                    `json:"exchange_desc"`
	QuoteCurrency          currency.Code             `json:"quote_currency"`
	QuoteCurrencyPrecision uint64                    `json:"quote_currency_precision"`
	FXRate                 types.Number              `json:"fx_rate"` // Quote currency to USD
	SymbolDescription      string                    `json:"symbol_desc"`
	Category               string                    `json:"category"`
	AssetType              string                    `json:"asset_type"` // STOCK or ETF
	SettlementCurrency     currency.Code             `json:"settlement_currency"`
	MaxOrderVolume         types.Number              `json:"max_order_volume"`
	StepOrderVolume        types.Number              `json:"step_order_volume"`
	MinOrderVolume         types.Number              `json:"min_order_volume"`
	PricePrecision         uint64                    `json:"price_precision"`
	VolumePrecision        uint64                    `json:"volume_precision"`
	IsIPO                  bool                      `json:"is_ipo"`
	IPOPrice               types.Number              `json:"ipo_price"`
	PriceProtection        types.Number              `json:"price_protection"`
	SellPriceProtection    types.Number              `json:"sell_price_protection"`
	BuyPriceProtection     types.Number              `json:"buy_price_protection"`
	SlippageRate           types.Number              `json:"slippage_rate"`
	CommissionRate         types.Number              `json:"commission_rate"`
	TradeStatus            string                    `json:"trade_status"` // pre_market, open, post_market, closed or gt_lp
	TradeMode              uint64                    `json:"trade_mode"`
	OrderFillTiming        uint64                    `json:"order_fill_timing"` // 1: immediate, 2: after pre-market opens, 3: after the regular session opens
	SymbolDescriptions     []*StockSymbolDescription `json:"symbol_descs"`
	IconLink               string                    `json:"icon_link"`
}

// StockSymbolDetailsResponse holds a page of stock symbol details
type StockSymbolDetailsResponse struct {
	Total      uint64               `json:"total"`
	TotalPages uint64               `json:"total_page"`
	List       []*StockSymbolDetail `json:"list"`
}

// StockOrderbookLevel holds an orderbook price level, which the API publishes without a size
type StockOrderbookLevel struct {
	Price     types.Number `json:"p"`
	UserOrder bool         `json:"user_order"`
}

// StockOrderbookResponse holds a stock symbol's orderbook
type StockOrderbookResponse struct {
	Symbol string                 `json:"symbol"`
	Bids   []*StockOrderbookLevel `json:"bids"`
	Asks   []*StockOrderbookLevel `json:"asks"`
}

// StockOrderRequest holds the parameters for CreateStockOrder
type StockOrderRequest struct {
	Symbol    string       `json:"symbol"`
	Side      order.Side   `json:"side"`       // Buy or Sell
	PriceType order.Type   `json:"price_type"` // Market or Limit
	Volume    types.Number `json:"volume"`
	Price     types.Number `json:"price,omitempty"` // Required for a limit order
	// TradingSession may be left empty because each price type supports a single session: all for limit orders and
	// regular for market orders
	TradingSession string `json:"trading_session"`
	// TimeInForce may be left unset because day orders, order.GoodTillDay, are the only ones the API supports
	TimeInForce   order.TimeInForce `json:"time_in_force"`
	ClientOrderID string            `json:"client_order_id,omitempty"`
}

// MarshalJSON converts the side, price type and time in force to the values the Stock API expects and fills in the
// trading session when it is unset
func (r *StockOrderRequest) MarshalJSON() ([]byte, error) {
	side, err := tradFiSideCode(r.Side)
	if err != nil {
		return nil, err
	}
	tradingSession := r.TradingSession
	if tradingSession == "" {
		tradingSession = "regular"
		if r.PriceType == order.Limit {
			tradingSession = "all"
		}
	}
	type Alias StockOrderRequest
	return json.Marshal(&struct {
		Side           uint8  `json:"side"`
		PriceType      string `json:"price_type"`
		TradingSession string `json:"trading_session"`
		TimeInForce    string `json:"time_in_force"`
		*Alias
	}{
		Side:           side,
		PriceType:      r.PriceType.Lower(),
		TradingSession: tradingSession,
		TimeInForce:    "day",
		Alias:          (*Alias)(r),
	})
}

// StockCreateOrderResponse holds the ID of a placed stock order
type StockCreateOrderResponse struct {
	OrderID string `json:"id"`
}

// StockOpenOrder holds an open stock order
type StockOpenOrder struct {
	OrderID             string        `json:"order_id"`
	Symbol              string        `json:"symbol"`
	Exchange            string        `json:"exchange"`
	QuoteCurrency       currency.Code `json:"quote_currency"`
	FXRate              types.Number  `json:"fx_rate"` // Quote currency to USD
	SymbolDescription   string        `json:"symbol_desc"`
	TradeStatus         string        `json:"trade_status"` // pre_market, open, post_market, closed or gt_lp
	TradeMode           uint64        `json:"trade_mode"`
	PriceType           string        `json:"price_type"` // market or limit
	Side                uint64        `json:"side"`       // 1: sell, 2: buy
	Status              uint64        `json:"status"`
	Volume              types.Number  `json:"volume"`
	FillVolume          types.Number  `json:"fill_volume"`
	Price               types.Number  `json:"price"`
	TimeSetup           types.Time    `json:"time_setup"`
	TimeUpdate          types.Time    `json:"time_update"`
	MaxOrderVolume      types.Number  `json:"max_order_volume"`
	StepOrderVolume     types.Number  `json:"step_order_volume"`
	MinOrderVolume      types.Number  `json:"min_order_volume"`
	PricePrecision      uint64        `json:"price_precision"`
	PriceProtection     types.Number  `json:"price_protection"`
	SellPriceProtection types.Number  `json:"sell_price_protection"`
	BuyPriceProtection  types.Number  `json:"buy_price_protection"`
	CommissionRate      types.Number  `json:"commission_rate"`
	SlippageRate        types.Number  `json:"slippage_rate"`
}

// GetStockOrderHistoryRequest holds the filters for GetStockOrderHistory
type GetStockOrderHistoryRequest struct {
	Symbol    string
	OrderIDs  []string  // Up to 20
	BeginTime time.Time // The range may span up to 3 months
	EndTime   time.Time
	Side      order.Side // Buy or Sell; UnknownSide returns both
	Page      uint64
	PageSize  uint64 // Defaults to 10, up to 500
}

// StockOrderStatusDetail holds a readable explanation of a stock order's status
type StockOrderStatusDetail struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// StockHistoricalOrder holds a stock order from the order history
type StockHistoricalOrder struct {
	OrderID           string                  `json:"order_id"`
	Symbol            string                  `json:"symbol"`
	Exchange          string                  `json:"exchange"`
	QuoteCurrency     currency.Code           `json:"quote_currency"`
	FXRate            types.Number            `json:"fx_rate"` // Quote currency to USD
	SymbolDescription string                  `json:"symbol_desc"`
	PriceType         string                  `json:"price_type"` // market or limit
	Status            uint64                  `json:"status"`
	StatusDescription string                  `json:"status_desc"`
	StatusDetail      *StockOrderStatusDetail `json:"status_detail"`
	FinishAs          uint64                  `json:"finish_as"`
	Side              uint64                  `json:"side"` // 1: sell, 2: buy
	TimeInForce       string                  `json:"time_in_force"`
	Volume            types.Number            `json:"volume"`
	FillVolume        types.Number            `json:"fill_volume"`
	Price             types.Number            `json:"price"`
	AverageFillPrice  types.Number            `json:"avg_fill_price"`
	Commission        types.Number            `json:"commission"`
	TimeSetup         types.Time              `json:"time_setup"`
	TimeDone          types.Time              `json:"time_done"`
}

// StockOrderHistoryResponse holds a page of the stock order history
type StockOrderHistoryResponse struct {
	Total      uint64                  `json:"total"`
	TotalPages uint64                  `json:"total_page"`
	List       []*StockHistoricalOrder `json:"list"`
}

// StockOrderUpdateRequest holds the parameters for UpdateStockOrder
type StockOrderUpdateRequest struct {
	Volume types.Number `json:"volume"`
	Price  types.Number `json:"price"`
}

// StockOrderIDResponse holds the ID of an order that was modified or placed to close a position
type StockOrderIDResponse struct {
	OrderID uint64 `json:"order_id"`
}

// GetStockPositionsRequest holds the filters and profit and loss calculation options for GetStockPositions
type GetStockPositionsRequest struct {
	PNLCalculationCostType  uint64 // 1: average cost price, the default, 2: diluted cost price
	PNLCalculationPriceType uint64 // 1: intraday price, the default, 2: latest extended-hours price
	Symbol                  string
	Exchange                string // us, hk, kr or jp
}

// StockPosition holds a stock position
type StockPosition struct {
	Symbol                     string        `json:"symbol"`
	Exchange                   string        `json:"exchange"`
	QuoteCurrency              currency.Code `json:"quote_currency"`
	QuoteCurrencyPrecision     uint64        `json:"quote_currency_precision"`
	FXRate                     types.Number  `json:"fx_rate"`      // Quote currency to USD
	TradeStatus                string        `json:"trade_status"` // pre_market, open, post_market, closed or gt_lp
	SymbolDescription          string        `json:"symbol_desc"`
	PositionPNL                types.Number  `json:"position_pnl"`
	TodayPNL                   types.Number  `json:"today_pnl"`
	PNLRate                    types.Number  `json:"pnl_rate"`
	TodaySellAmount            types.Number  `json:"today_sell_amount"`
	TodayBuyAmount             types.Number  `json:"today_buy_amount"`
	TodaySellVolume            types.Number  `json:"today_sell_volume"`
	TodayBuyVolume             types.Number  `json:"today_buy_volume"`
	YesterdayVolume            types.Number  `json:"yesterday_volume"`
	Volume                     types.Number  `json:"volume"`
	Available                  types.Number  `json:"available"`
	TransferOutPendingQuantity types.Number  `json:"transfer_out_pending_qty"`
	AverageCostPrice           types.Number  `json:"avg_cost_price"`
	DilutedCostPrice           types.Number  `json:"diluted_cost_price"`
	LastPrice                  types.Number  `json:"last_price"`
	ExtendedLastPrice          types.Number  `json:"extended_last_price"`
	MaxOrderVolume             types.Number  `json:"max_order_volume"`
	StepOrderVolume            types.Number  `json:"step_order_volume"`
	MinOrderVolume             types.Number  `json:"min_order_volume"`
	PricePrecision             uint64        `json:"price_precision"`
	PriceProtection            types.Number  `json:"price_protection"`
	SellPriceProtection        types.Number  `json:"sell_price_protection"`
	BuyPriceProtection         types.Number  `json:"buy_price_protection"`
	CommissionRate             types.Number  `json:"commission_rate"`
	SlippageRate               types.Number  `json:"slippage_rate"`
}

// StockClosePositionRequest holds the parameters for CloseStockPosition
type StockClosePositionRequest struct {
	Symbol      string       `json:"symbol"`
	CloseType   uint64       `json:"close_type"`             // 1: partial close of CloseVolume, 2: close all
	CloseVolume types.Number `json:"close_volume,omitempty"` // Required for a partial close
}

// StockTransactionRequest holds the parameters for CreateStockTransaction
type StockTransactionRequest struct {
	Asset       currency.Code `json:"asset"` // Only USDT is supported
	Change      types.Number  `json:"change"`
	Type        string        `json:"type"`   // deposit moves funds from the spot account to the stock account, withdraw moves them back
	ReferenceID string        `json:"ref_id"` // Makes the transfer idempotent
}

// GetStockTransactionsRequest holds the filters for GetStockTransactions
type GetStockTransactionsRequest struct {
	BeginTime   time.Time // The range may span up to 3 months
	EndTime     time.Time
	ReferenceID string // Takes precedence over every other filter
	Type        string // deposit, withdraw, fee, dividend, sell, buy, award, stock_transfer_in or stock_transfer_out
	Page        uint64
	PageSize    uint64 // Defaults to 10, up to 500
}

// StockTransaction holds a stock account transaction
type StockTransaction struct {
	Asset             currency.Code  `json:"asset"`
	Symbol            string         `json:"symbol"`
	SymbolDisplayName string         `json:"symbol_display"`
	Type              string         `json:"type"`
	TypeDescription   string         `json:"type_desc"`
	Change            types.Number   `json:"change"` // Negative for a debit
	Balance           types.Number   `json:"balance"`
	ReferenceID       string         `json:"ref_id"`
	Time              types.Time     `json:"time"`
	UnitText          string         `json:"unit_text"`
	Detail            map[string]any `json:"detail"`
}

// StockTransactionsResponse holds a page of stock account transactions
type StockTransactionsResponse struct {
	Total      uint64              `json:"total"`
	TotalPages uint64              `json:"total_page"`
	List       []*StockTransaction `json:"list"`
}

// StockExchange holds a stock market the API supports
type StockExchange struct {
	Exchange              string `json:"exchange"`
	ExchangeDescription   string `json:"exchange_desc"`
	IconLink              string `json:"icon_link"`
	SupportsStockTransfer bool   `json:"support_transfer"`
}

// StockFeeRate holds the Japanese and Korean stock fee rates for a VIP level
type StockFeeRate struct {
	VIPLevel uint64       `json:"vip_level"`
	MakerFee types.Number `json:"maker_fee"`
	TakerFee types.Number `json:"taker_fee"`
}
