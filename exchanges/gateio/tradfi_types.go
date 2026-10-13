package gateio

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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

// tradFiResponse is the envelope shared by the TradFi (CFD) and Stock APIs, which report business failures through a
// non-empty label
type tradFiResponse[T any] struct {
	Label   string `json:"label"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// AsError returns the response label and message as an error, or nil when the request succeeded
func (r *tradFiResponse[T]) AsError() error {
	if r.Label != "" {
		return fmt.Errorf("label: %s message: %s", r.Label, r.Message)
	}
	return nil
}

// tradFiList holds the list carried in the data field of TradFi and Stock list responses
type tradFiList[T any] struct {
	List []T `json:"list"`
}

// TradFiMT5Account holds MT5 account information
type TradFiMT5Account struct {
	Leverage     uint64       `json:"leverage"`
	StopOutLevel types.Number `json:"stop_out_level"`
	Status       uint64       `json:"status"` // 1: not opened, 2: pending review, 3: active
}

// TradFiCategory holds a trading symbol category
type TradFiCategory struct {
	CategoryID   uint64 `json:"category_id"`
	IsFavourite  bool   `json:"is_favorite"`
	CategoryName string `json:"category_name"`
}

// TradFiSymbolCommission holds the commission charged per lot when trading a symbol
type TradFiSymbolCommission struct {
	CategoryCode string       `json:"category_code"`
	Symbol       string       `json:"symbol"`
	FeePerLot    types.Number `json:"fee_per_lot"`
}

// TradFiSymbol holds trading symbol information
type TradFiSymbol struct {
	Symbol                   string        `json:"symbol"`
	SymbolDescription        string        `json:"symbol_desc"`
	CategoryID               uint64        `json:"category_id"`
	Status                   string        `json:"status"`     // open: tradable, closed: not tradable
	TradeMode                string        `json:"trade_mode"` // 0: disabled, 1: long only, 2: short only, 3: close only, 4: full trading access
	IconLink                 string        `json:"icon_link"`
	CloseTime                types.Time    `json:"close_time"`
	OpenTime                 types.Time    `json:"open_time"`
	NextOpenTime             types.Time    `json:"next_open_time"`
	SettlementCurrency       currency.Code `json:"settlement_currency"`
	SettlementCurrencySymbol string        `json:"settlement_currency_symbol"`
	PricePrecision           uint64        `json:"price_precision"`
	// IsBase and Leverages are returned by the live API but not documented. Orders are placed on base symbols, using
	// one of the leverages they allow.
	IsBase    bool           `json:"is_base"`
	Leverages []types.Number `json:"leverages"`
}

// TradFiSymbolDetail holds detailed contract information for a trading symbol
type TradFiSymbolDetail struct {
	Symbol             string        `json:"symbol"`
	SymbolDescription  string        `json:"symbol_desc"`
	CategoryName       string        `json:"category_name"`
	ContractVolume     types.Number  `json:"contract_volume"`
	SettlementCurrency currency.Code `json:"settlement_currency"`
	MaxOrderVolume     types.Number  `json:"max_order_volume"`
	MinOrderVolume     types.Number  `json:"min_order_volume"`
	Leverage           types.Number  `json:"leverage"`
	PricePrecision     uint64        `json:"price_precision"`
	PriceStopLossLevel types.Number  `json:"price_sl_level"`
	SwapCostType       string        `json:"swap_cost_type"`
	BuySwapCostRate    types.Number  `json:"buy_swap_cost_rate"`
	SellSwapCostRate   types.Number  `json:"sell_swap_cost_rate"`
	SwapCost3Day       string        `json:"swap_cost_3day"`
	TradeTimezone      string        `json:"trade_timezone"`
	TradeMode          string        `json:"trade_mode"` // 0: disabled, 1: long only, 2: short only, 3: close only, 4: full trading access
	IconLink           string        `json:"icon_link"`
}

// GetTradFiKlinesRequest holds the query parameters for GetTradFiKlines. The documentation treats the time range and
// limit as independent, but the live API rejects a BeginTime without an EndTime and a BeginTime with a Limit, so
// request either a range or a Limit counting back from EndTime.
type GetTradFiKlinesRequest struct {
	Interval  kline.Interval // One of 1m, 15m, 1h, 4h, 1d, 7d or 30d
	BeginTime time.Time
	EndTime   time.Time
	Limit     uint64 // Up to 500
}

// TradFiKline holds a single candlestick
type TradFiKline struct {
	Open      types.Number `json:"o"`
	Close     types.Number `json:"c"`
	Low       types.Number `json:"l"`
	High      types.Number `json:"h"`
	Timestamp types.Time   `json:"t"`
}

// TradFiTicker holds ticker information for a trading symbol
type TradFiTicker struct {
	HighestPrice          types.Number `json:"highest_price"`
	LowestPrice           types.Number `json:"lowest_price"`
	PriceChangePercentage types.Number `json:"price_change"`
	PriceChangeAmount     types.Number `json:"price_change_amount"`
	TodayOpenPrice        types.Number `json:"today_open_price"`
	PreviousClosePrice    types.Number `json:"last_today_close_price"`
	LastPrice             types.Number `json:"last_price"`
	BidPrice              types.Number `json:"bid_price"`
	AskPrice              types.Number `json:"ask_price"`
	Favourite             bool         `json:"favorite"`
	Status                string       `json:"status"` // open: tradable, closed: not tradable
	CloseTime             types.Time   `json:"close_time"`
	OpenTime              types.Time   `json:"open_time"`
	NextOpenTime          types.Time   `json:"next_open_time"`
	TradeMode             string       `json:"trade_mode"`
	CategoryName          string       `json:"category_name"`
	// ExchangeRate and SettlementCurrency are returned by the live API but not documented
	ExchangeRate       types.Number  `json:"exchange_rate"`
	SettlementCurrency currency.Code `json:"settlement_currency"`
}

// TradFiUserInfo holds the TradFi account state returned after activation
type TradFiUserInfo struct {
	Status   uint64 `json:"status"` // 1: not opened, 2: pending review, 3: opened
	Leverage uint64 `json:"leverage"`
}

// TradFiUserAssets holds TradFi account balance and margin information
type TradFiUserAssets struct {
	Equity        types.Number `json:"equity"`
	MarginLevel   types.Number `json:"margin_level"` // Percentage
	Balance       types.Number `json:"balance"`
	Margin        types.Number `json:"margin"`
	MarginFree    types.Number `json:"margin_free"`
	UnrealisedPNL types.Number `json:"unrealized_pnl"`
}

// TradFiTransactionRequest holds the parameters for CreateTradFiTransaction
type TradFiTransactionRequest struct {
	Asset  currency.Code `json:"asset"`  // Only USDT is supported
	Change types.Number  `json:"change"` // Up to two decimal places
	Type   string        `json:"type"`   // deposit or withdraw
}

// GetTradFiTransactionsRequest holds the filters for GetTradFiTransactions
type GetTradFiTransactionsRequest struct {
	BeginTime time.Time
	EndTime   time.Time
	Type      string // deposit, withdraw, dividend or fill_negative; empty returns every type
	Page      uint64
	PageSize  uint64 // Defaults to 10, up to 50
}

// TradFiTransaction holds a single fund transfer record
type TradFiTransaction struct {
	Asset           currency.Code `json:"asset"`
	Type            string        `json:"type"` // deposit, withdraw, dividend or fill_negative
	TypeDescription string        `json:"type_desc"`
	Change          types.Number  `json:"change"`
	Balance         types.Number  `json:"balance"`
	Time            types.Time    `json:"time"`
}

// TradFiTransactionListData holds a page of fund transfer records
type TradFiTransactionListData struct {
	Total      uint64               `json:"total"`
	TotalPages uint64               `json:"total_page"`
	List       []*TradFiTransaction `json:"list"`
}

// TradFiOrderRequest holds the parameters for CreateTradFiOrder
type TradFiOrderRequest struct {
	Symbol          string       `json:"symbol"`
	Side            order.Side   `json:"side"`       // Buy or Sell
	PriceType       order.Type   `json:"price_type"` // Market, or Trigger to wait for Price
	Price           types.Number `json:"price"`
	Volume          types.Number `json:"volume"`
	PriceTakeProfit types.Number `json:"price_tp,omitempty"`
	PriceStopLoss   types.Number `json:"price_sl,omitempty"`
	Leverage        uint64       `json:"leverage,omitempty"` // Must be one of the leverages the symbol allows
}

// MarshalJSON converts the side and price type to the values the TradFi API expects
func (r *TradFiOrderRequest) MarshalJSON() ([]byte, error) {
	side, err := tradFiSideCode(r.Side)
	if err != nil {
		return nil, err
	}
	type Alias TradFiOrderRequest
	return json.Marshal(&struct {
		Side      uint8  `json:"side"`
		PriceType string `json:"price_type"`
		*Alias
	}{
		Side:      side,
		PriceType: r.PriceType.Lower(),
		Alias:     (*Alias)(r),
	})
}

// TradFiCreateOrderResult holds the queue task ID returned after placing an order, which is the log ID that
// GetTradFiOrderLog takes rather than an order ID
type TradFiCreateOrderResult struct {
	ID string `json:"id"`
}

// TradFiOrder holds an active order
type TradFiOrder struct {
	OrderID           uint64       `json:"order_id"`
	Symbol            string       `json:"symbol"`
	SymbolDescription string       `json:"symbol_desc"`
	PriceType         string       `json:"price_type"` // market or trigger
	State             uint64       `json:"state"`
	StateDescription  string       `json:"state_desc"`
	Finished          uint64       `json:"finished"` // 0: shown in the active order list, 1: not shown
	Side              uint64       `json:"side"`     // 1: sell, 2: buy
	Volume            types.Number `json:"volume"`
	Price             types.Number `json:"price"`
	PriceTakeProfit   types.Number `json:"price_tp"`
	PriceStopLoss     types.Number `json:"price_sl"`
	TimeSetup         types.Time   `json:"time_setup"`
}

// TradFiOrderUpdateRequest holds the parameters for UpdateTradFiOrder
type TradFiOrderUpdateRequest struct {
	Price types.Number `json:"price"`
	// Omitting a take profit or stop loss price clears it, so pass the current price to keep it
	PriceTakeProfit types.Number `json:"price_tp,omitempty"`
	PriceStopLoss   types.Number `json:"price_sl,omitempty"`
}

// TradFiUpdatedOrder holds the order state after modification
type TradFiUpdatedOrder struct {
	OrderID         uint64       `json:"order_id"`
	Symbol          string       `json:"symbol"`
	State           string       `json:"state"`
	Volume          types.Number `json:"volume"`
	Price           types.Number `json:"price"`
	PriceTakeProfit types.Number `json:"price_tp"`
	PriceStopLoss   types.Number `json:"price_sl"`
}

// GetTradFiOrderHistoryRequest holds the filters for GetTradFiOrderHistory
type GetTradFiOrderHistoryRequest struct {
	BeginTime time.Time // No earlier than one month ago
	EndTime   time.Time
	Symbol    string
	Side      order.Side // Buy or Sell; UnknownSide returns both
}

// TradFiHistoricalOrder holds a completed order
type TradFiHistoricalOrder struct {
	OrderID           uint64       `json:"order_id"`
	Symbol            string       `json:"symbol"`
	SymbolDescription string       `json:"symbol_desc"`
	PriceType         string       `json:"price_type"`     // market or trigger
	OrderOptType      uint64       `json:"order_opt_type"` // 1: sell, 2: buy, 3: close long, 4: close short, 5: force close long, 6: force close short
	State             uint64       `json:"state"`
	StateDescription  string       `json:"state_desc"`
	Side              uint64       `json:"side"` // 1: sell, 2: buy
	Volume            types.Number `json:"volume"`
	FillVolume        types.Number `json:"fill_volume"`
	ClosePNL          types.Number `json:"close_pnl"`
	Price             types.Number `json:"price"` // Average fill price
	TriggerPrice      types.Number `json:"trigger_price"`
	PriceTakeProfit   types.Number `json:"price_tp"`
	PriceStopLoss     types.Number `json:"price_sl"`
	TimeSetup         types.Time   `json:"time_setup"`
	TimeDone          types.Time   `json:"time_done"`
}

// TradFiOrderLogResponse holds the order recorded against an order placement log ID
type TradFiOrderLogResponse struct {
	OrderID   uint64       `json:"order_id"`
	LogID     uint64       `json:"log_id"`
	Symbol    string       `json:"symbol"`
	PriceType string       `json:"price_type"` // market or trigger
	State     uint64       `json:"state"`      // 1: placed, 2: cancelled, 3: partially filled, 4: filled, 5: rejected
	Side      uint64       `json:"side"`       // 1: sell, 2: buy
	Volume    types.Number `json:"volume"`
	Price     types.Number `json:"price"` // Average fill price
}

// TradFiPosition holds an open position
type TradFiPosition struct {
	PositionID        uint64       `json:"position_id"`
	Symbol            string       `json:"symbol"`
	SymbolDescription string       `json:"symbol_desc"`
	Margin            types.Number `json:"margin"`
	UnrealisedPNL     types.Number `json:"unrealized_pnl"`
	UnrealisedPNLRate types.Number `json:"unrealized_pnl_rate"`
	Volume            types.Number `json:"volume"`
	PriceOpen         types.Number `json:"price_open"`
	PositionDirection string       `json:"position_dir"` // Long or Short
	PriceTakeProfit   types.Number `json:"price_tp"`
	PriceStopLoss     types.Number `json:"price_sl"`
	CounterpartyPrice types.Number `json:"counterparty_price"`
	TimeCreate        types.Time   `json:"time_create"`
}

// TradFiPositionUpdateRequest holds the parameters for UpdateTradFiPosition. Omitting a take profit or stop loss price
// clears it, so pass the current price to keep it.
type TradFiPositionUpdateRequest struct {
	PriceTakeProfit types.Number `json:"price_tp,omitempty"`
	PriceStopLoss   types.Number `json:"price_sl,omitempty"`
}

// TradFiClosePositionRequest holds the parameters for CloseTradFiPosition
type TradFiClosePositionRequest struct {
	CloseType   uint64       `json:"close_type"`             // 1: partial close of CloseVolume, 2: full close
	CloseVolume types.Number `json:"close_volume,omitempty"` // Required for a partial close
}

// GetTradFiPositionHistoryRequest holds the filters for GetTradFiPositionHistory
type GetTradFiPositionHistoryRequest struct {
	Page              uint64
	PageSize          uint64    // Defaults to 10, up to 100
	BeginTime         time.Time // No earlier than one month ago
	EndTime           time.Time
	Symbol            string
	PositionDirection order.Side // Long or Short; UnknownSide returns both
}

// TradFiLiquidationDetail holds the margin state recorded when a position was liquidated
type TradFiLiquidationDetail struct {
	MarginLevel  types.Number `json:"margin_level"` // Multiplied by 100
	Margin       types.Number `json:"margin"`
	Equity       types.Number `json:"equity"`
	StopOutLevel types.Number `json:"stop_out_level"` // Multiplied by 100
}

// TradFiRealisedPNLDetail holds a breakdown of realised profit and loss
type TradFiRealisedPNLDetail struct {
	ClosedPNL types.Number `json:"closed_pnl"`
	Swap      types.Number `json:"swap"`
	Fee       types.Number `json:"fee"`
}

// TradFiHistoricalPosition holds a closed position
type TradFiHistoricalPosition struct {
	PositionID        uint64                   `json:"position_id"`
	Symbol            string                   `json:"symbol"`
	RealisedPNL       types.Number             `json:"realized_pnl"`
	RealisedPNLRate   types.Number             `json:"realized_pnl_rate"`
	Volume            types.Number             `json:"volume"`
	VolumeClosed      types.Number             `json:"volume_closed"`
	PriceOpen         types.Number             `json:"price_open"`
	PositionDirection string                   `json:"position_dir"` // Long or Short
	PriceTakeProfit   types.Number             `json:"price_tp"`
	PriceStopLoss     types.Number             `json:"price_sl"`
	CounterpartyPrice types.Number             `json:"counterparty_price"`
	ClosePrice        types.Number             `json:"close_price"`
	TimeCreate        types.Time               `json:"time_create"`
	TimeClose         types.Time               `json:"time_close"`
	PositionStatus    string                   `json:"position_status"` // 1: fully closed, 2: forced liquidation
	CloseDetail       *TradFiLiquidationDetail `json:"close_detail"`    // Only set for a forced liquidation
	RealisedPNLDetail TradFiRealisedPNLDetail  `json:"realized_pnl_detail"`
}

// TradFiHistoricalPositionListData holds a page of closed positions
type TradFiHistoricalPositionListData struct {
	Total      uint64                      `json:"total"`
	TotalPages uint64                      `json:"total_page"`
	List       []*TradFiHistoricalPosition `json:"list"`
}
