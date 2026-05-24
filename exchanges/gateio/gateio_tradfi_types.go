package gateio

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var klineIntervalToTypeMap = []struct {
	Interval       kline.Interval
	IntervalString string
}{
	{Interval: kline.OneMin, IntervalString: "1m"},
	{Interval: kline.FifteenMin, IntervalString: "15m"},
	{Interval: kline.OneHour, IntervalString: "1h"},
	{Interval: kline.FourHour, IntervalString: "4h"},
	{Interval: kline.OneDay, IntervalString: "1d"},
	{Interval: kline.OneWeek, IntervalString: "7d"},
	{Interval: kline.OneMonth, IntervalString: "30d"},
}

func klineIntervalToTypeString(interval kline.Interval) (string, error) {
	for _, result := range klineIntervalToTypeMap {
		if result.Interval == interval {
			return result.IntervalString, nil
		}
	}
	return "", fmt.Errorf("%w: %v", kline.ErrUnsupportedInterval, interval)
}

// tradFiResponse is a generic wrapper for all TradFi API responses.
type tradFiResponse[T any] struct {
	Code      int64        `json:"code"`
	Label     string       `json:"label"`
	Message   string       `json:"message"`
	Timestamp types.Number `json:"timestamp"`
	Data      T            `json:"data"`
}

// Error implements the error check interface used by SendAuthenticatedHTTPRequest.
func (r *tradFiResponse[T]) Error() error {
	if r.Code != 0 {
		return fmt.Errorf("code: %d message: %s", r.Code, r.Message)
	}
	if r.Label != "" {
		return fmt.Errorf("label: %s message: %s", r.Label, r.Message)
	}
	return nil
}

// TradFiMt5Account holds MT5 account information.
type TradFiMt5Account struct {
	Mt5UID       int64  `json:"mt5_uid"`
	Leverage     int64  `json:"leverage"`
	StopOutLevel string `json:"stop_out_level"`
	Status       int64  `json:"status"` // Status: 1=not opened, 2=pending review, 3=active.
}

// TradFiCategory holds a single symbol category.
type TradFiCategory struct {
	CategoryID   int64  `json:"category_id"`
	IsFavorite   bool   `json:"is_favorite"`
	CategoryName string `json:"category_name"`
}

// TradFiCategoryList wraps the category list data field.
type TradFiCategoryList struct {
	List []*TradFiCategory `json:"list"`
}

// TradFiSymbol holds trading symbol information.
type TradFiSymbol struct {
	Symbol                   currency.Pair `json:"symbol"`
	SymbolDesc               string        `json:"symbol_desc"`
	CategoryID               int64         `json:"category_id"`
	Status                   string        `json:"status"`     // Status: open=tradable, closed=non-tradable.
	TradeMode                string        `json:"trade_mode"` // TradeMode: 0=disabled, 1=long only, 2=short only, 3=close only, 4=full trading access.
	IconLink                 string        `json:"icon_link"`
	CloseTime                types.Time    `json:"close_time"`
	OpenTime                 types.Time    `json:"open_time"`
	NextOpenTime             types.Time    `json:"next_open_time"`
	SettlementCurrency       currency.Code `json:"settlement_currency"`
	SettlementCurrencySymbol currency.Pair `json:"settlement_currency_symbol"`
}

// TradFiSymbolList wraps the symbol list data field.
type TradFiSymbolList struct {
	List []*TradFiSymbol `json:"list"`
}

// TradFiSymbolDetail holds detailed contract information for a trading symbol.
type TradFiSymbolDetail struct {
	Symbol             currency.Pair `json:"symbol"`
	SymbolDesc         string        `json:"symbol_desc"`
	CategoryName       string        `json:"category_name"`
	ContractVolume     types.Number  `json:"contract_volume"`
	SettlementCurrency currency.Code `json:"settlement_currency"`
	MaxOrderVolume     types.Number  `json:"max_order_volume"`
	MinOrderVolume     types.Number  `json:"min_order_volume"`
	Leverage           float64       `json:"leverage"`
	PricePrecision     int64         `json:"price_precision"`
	PriceStopLossLevel string        `json:"price_sl_level"`
	SwapCostType       string        `json:"swap_cost_type"`
	BuySwapCostRate    types.Number  `json:"buy_swap_cost_rate"`
	SellSwapCostRate   types.Number  `json:"sell_swap_cost_rate"`
	SwapCost3Day       string        `json:"swap_cost_3day"`
	TradeTimezone      string        `json:"trade_timezone"`
	TradeMode          string        `json:"trade_mode"` // TradeMode: 0=disabled, 1=long only, 2=short only, 3=close only, 4=full trading access.
	IconLink           string        `json:"icon_link"`
}

// TradFiSymbolDetailList wraps the symbol detail list data field.
type TradFiSymbolDetailList struct {
	List []*TradFiSymbolDetail `json:"list"`
}

// TradFiKline holds a single candlestick data point.
type TradFiKline struct {
	Open      types.Number `json:"o"`
	Close     types.Number `json:"c"`
	Low       types.Number `json:"l"`
	High      types.Number `json:"h"`
	Timestamp types.Time   `json:"t"`
}

// TradFiKlineList wraps the kline list data field.
type TradFiKlineList struct {
	List []*TradFiKline `json:"list"`
}

// TradFiTicker holds ticker information for a trading symbol.
type TradFiTicker struct {
	HighestPrice        types.Number `json:"highest_price"`
	LowestPrice         types.Number `json:"lowest_price"`
	PriceChange         string       `json:"price_change"`
	PriceChangeAmount   types.Number `json:"price_change_amount"`
	TodayOpenPrice      types.Number `json:"today_open_price"`
	LastTodayClosePrice types.Number `json:"last_today_close_price"`
	LastPrice           types.Number `json:"last_price"`
	BidPrice            types.Number `json:"bid_price"`
	AskPrice            types.Number `json:"ask_price"`
	Favorite            bool         `json:"favorite"`
	Status              string       `json:"status"` // Status: open=tradable, closed=non-tradable.
	CloseTime           types.Time   `json:"close_time"`
	OpenTime            types.Time   `json:"open_time"`
	NextOpenTime        types.Time   `json:"next_open_time"`
	TradeMode           string       `json:"trade_mode"`
	CategoryName        string       `json:"category_name"`
}

// TradFiUserInfo holds TradFi user account information returned after activation.
type TradFiUserInfo struct {
	Status   int64  `json:"status"` // Status: 1=not opened, 2=pending review, 3=opened.
	Leverage int64  `json:"leverage"`
	Mt5UID   string `json:"mt5_uid"`
}

// TradFiUserAssets holds TradFi account asset information.
type TradFiUserAssets struct {
	Equity        string       `json:"equity"`
	MarginLevel   string       `json:"margin_level"`
	Balance       types.Number `json:"balance"`
	Margin        types.Number `json:"margin"`
	MarginFree    types.Number `json:"margin_free"`
	UnrealizedPNL types.Number `json:"unrealized_pnl"`
	Mt5UID        string       `json:"mt5_uid"`
}

// TradFiTransaction holds a single fund transfer transaction record.
type TradFiTransaction struct {
	Asset    currency.Code `json:"asset"`
	Type     string        `json:"type"` // Type: deposit=transfer in, withdraw=transfer out, dividend=dividend payment, fill_negative=cover negative balance.
	TypeDesc string        `json:"type_desc"`
	Change   types.Number  `json:"change"`
	Balance  types.Number  `json:"balance"`
	Time     types.Time    `json:"time"`
}

// TradFiTransactionListData wraps the transaction list data with pagination.
type TradFiTransactionListData struct {
	Total     int64                `json:"total"`
	TotalPage int64                `json:"total_page"`
	List      []*TradFiTransaction `json:"list"`
	Timestamp types.Number         `json:"timestamp"`
}

// TradFiTransactionRequest is the request body for fund deposit or withdrawal.
type TradFiTransactionRequest struct {
	Asset  currency.Code `json:"asset"`         // Asset is the asset type (e.g. USDT; currently only USDT is supported).
	Change float64       `json:"change,string"` // Change is the quantity; supports up to two decimal places.
	Type   string        `json:"type"`          // Type is either 'deposit' or 'withdraw'.
}

// TradFiOrder holds an active pending order.
type TradFiOrder struct {
	OrderID         int64         `json:"order_id"`
	Symbol          currency.Pair `json:"symbol"`
	SymbolDesc      string        `json:"symbol_desc"`
	PriceType       string        `json:"price_type"` // PriceType: market=market price, trigger=trigger price.
	State           int64         `json:"state"`
	StateDesc       string        `json:"state_desc"`
	Finished        int64         `json:"finished"` // Finished: 0=shown in active order list, 1=not shown.
	Side            int64         `json:"side"`     // Side: 1=sell, 2=buy.
	Volume          types.Number  `json:"volume"`
	Price           types.Number  `json:"price"`
	PriceTakeProfit types.Number  `json:"price_tp"`
	PriceStopLoss   types.Number  `json:"price_sl"`
	TimeSetup       string        `json:"time_setup"`
}

// TradFiOrderList wraps the active order list data field.
type TradFiOrderList struct {
	List []*TradFiOrder `json:"list"`
}

// TradFiOrderRequest is the request body for creating an order.
type TradFiOrderRequest struct {
	Price           float64       `json:"price,string"`
	PriceType       string        `json:"price_type"`
	Side            int64         `json:"side"` // Side: 1=sell, 2=buy.
	Symbol          currency.Pair `json:"symbol"`
	Volume          float64       `json:"volume,string"`
	PriceTakeProfit float64       `json:"price_tp,omitempty,string"`
	PriceStopLoss   float64       `json:"price_sl,omitempty,string"`
}

// TradFiCreateOrderResult holds the queue task ID returned after order creation.
type TradFiCreateOrderResult struct {
	ID string `json:"id"`
}

// TradFiOrderUpdateRequest is the request body for modifying an existing order.
type TradFiOrderUpdateRequest struct {
	Price           string  `json:"price"`
	PriceTakeProfit float64 `json:"price_tp,omitempty,string"`
	PriceStopLoss   float64 `json:"price_sl,omitempty,string"`
}

// TradFiUpdatedOrder holds the order state after modification.
type TradFiUpdatedOrder struct {
	OrderID         int64         `json:"order_id"`
	Symbol          currency.Pair `json:"symbol"`
	State           string        `json:"state"`
	Volume          types.Number  `json:"volume"`
	Price           types.Number  `json:"price"`
	PriceTakeProfit types.Number  `json:"price_tp"`
	PriceStopLoss   types.Number  `json:"price_sl"`
}

// TradFiHistoricalOrder holds a completed order record.
type TradFiHistoricalOrder struct {
	OrderID         int64         `json:"order_id"`
	Symbol          currency.Pair `json:"symbol"`
	SymbolDesc      string        `json:"symbol_desc"`
	PriceType       string        `json:"price_type"`     // PriceType: market=market price, trigger=trigger price.
	OrderOptType    int64         `json:"order_opt_type"` // OrderOptType: 1=sell, 2=buy, 3=close long, 4=close short, 5=force close long, 6=force close short.
	State           int64         `json:"state"`
	StateDesc       string        `json:"state_desc"`
	Side            int64         `json:"side"` // Side: 1=sell, 2=buy.
	Volume          types.Number  `json:"volume"`
	FillVolume      types.Number  `json:"fill_volume"`
	ClosePNL        types.Number  `json:"close_pnl"`
	Price           types.Number  `json:"price"`
	TriggerPrice    types.Number  `json:"trigger_price"`
	PriceTakeProfit types.Number  `json:"price_tp"`
	PriceStopLoss   types.Number  `json:"price_sl"`
	TimeSetup       types.Time    `json:"time_setup"`
	TimeDone        types.Time    `json:"time_done"`
}

// TradFiOrderHistoryList wraps the historical order list data field.
type TradFiOrderHistoryList struct {
	List []*TradFiHistoricalOrder `json:"list"`
}

// TradFiPosition holds an active open position.
type TradFiPosition struct {
	PositionID        int64         `json:"position_id"`
	Symbol            currency.Pair `json:"symbol"`
	SymbolDesc        string        `json:"symbol_desc"`
	Margin            string        `json:"margin"`
	UnrealizedPNL     types.Number  `json:"unrealized_pnl"`
	UnrealizedPNLRate types.Number  `json:"unrealized_pnl_rate"`
	Volume            types.Number  `json:"volume"`
	PriceOpen         types.Number  `json:"price_open"`
	PositionDir       string        `json:"position_dir"` // PositionDir: Long=long position, Short=short position.
}

// TradFiPositionList wraps the active position list data field.
type TradFiPositionList struct {
	List []*TradFiPosition `json:"list"`
}

// TradFiPositionUpdateRequest is the request body for modifying a position's TP/SL.
type TradFiPositionUpdateRequest struct {
	PriceTakeProfit float64 `json:"price_tp,omitempty,string"`
	PriceStopLoss   float64 `json:"price_sl,omitempty,string"`
}

// TradFiClosePositionRequest is the request body for closing a position.
type TradFiClosePositionRequest struct {
	CloseType   int64   `json:"close_type"` // CloseType: 1=full close, 2=partial close.
	CloseVolume float64 `json:"close_volume,omitempty,string"`
}

// TradFiLiquidationDetail holds margin details recorded at the time of liquidation.
type TradFiLiquidationDetail struct {
	MarginLevel  string `json:"margin_level"`
	Margin       string `json:"margin"`
	Equity       string `json:"equity"`
	StopOutLevel string `json:"stop_out_level"`
}

// TradFiRealizedPnlDetail holds a breakdown of realized profit and loss.
type TradFiRealizedPnlDetail struct {
	ClosedPNL string       `json:"closed_pnl"`
	Swap      string       `json:"swap"`
	Fee       types.Number `json:"fee"`
}

// TradFiHistoricalPosition holds a closed position record.
type TradFiHistoricalPosition struct {
	PositionID        int64                    `json:"position_id"`
	Symbol            currency.Pair            `json:"symbol"`
	RealizedPNL       types.Number             `json:"realized_pnl"`
	RealizedPNLRate   types.Number             `json:"realized_pnl_rate"`
	Volume            types.Number             `json:"volume"`
	VolumeClosed      types.Number             `json:"volume_closed"`
	PriceOpen         types.Number             `json:"price_open"`
	PositionDir       string                   `json:"position_dir"` // PositionDir: Long=long position, Short=short position.
	PriceTP           types.Number             `json:"price_tp"`
	PriceSL           types.Number             `json:"price_sl"`
	CounterpartyPrice types.Number             `json:"counterparty_price"`
	ClosePrice        types.Number             `json:"close_price"`
	TimeCreate        types.Time               `json:"time_create"`
	TimeClose         types.Time               `json:"time_close"`
	PositionStatus    string                   `json:"position_status"` // PositionStatus: 1=fully closed, 2=forced liquidation.
	CloseDetail       *TradFiLiquidationDetail `json:"close_detail"`
	RealizedPnlDetail TradFiRealizedPnlDetail  `json:"realized_pnl_detail"`
}

// TradFiHistoricalPositionListData wraps historical position data with pagination.
type TradFiHistoricalPositionListData struct {
	Total     int64                       `json:"total"`
	TotalPage int64                       `json:"total_page"`
	List      []*TradFiHistoricalPosition `json:"list"`
}

// GetTradFiKlinesRequest holds the query parameters for the klines endpoint.
type GetTradFiKlinesRequest struct {
	KlineType string // KlineType is required: TradFiKlineType1m, 15m, 1h, 4h, 1d, 7d, or 30d.
	BeginTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// GetTradFiTransactionsRequest holds the query parameters for listing transactions.
type GetTradFiTransactionsRequest struct {
	BeginTime time.Time
	EndTime   time.Time
	Type      string // Type filters by transaction type; one of TradFiTransaction* constants or empty for all.
	Page      uint64
	PageSize  uint64
}

// GetTradFiOrderHistoryRequest holds the query parameters for historical order list.
type GetTradFiOrderHistoryRequest struct {
	BeginTime time.Time
	EndTime   time.Time
	Symbol    currency.Pair
	Side      uint64 // Side filters by order side: 1=sell, 2=buy; 0 means no filter.
}

// GetTradFiPositionHistoryRequest holds the query parameters for historical position list.
type GetTradFiPositionHistoryRequest struct {
	Page        uint64
	PageSize    uint64
	BeginTime   time.Time
	EndTime     time.Time
	Symbol      currency.Pair
	PositionDir string // PositionDir filters by direction: TradFiPositionLong, TradFiPositionShort, or empty for all.
}
