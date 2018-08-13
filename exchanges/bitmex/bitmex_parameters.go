package bitmex

import (
	"errors"
	"net/url"
	"reflect"
	"strconv"

	"github.com/thrasher-/gocryptotrader/common"
)

// Parameter just enforces a check on all outgoing data
type Parameter interface {
	VerifyData() error
	ToURLVals(path string) (string, error)
	IsNil() bool
}

// StructValsToURLVals converts a struct into url.values for easy encoding
// can set json tags for outgoing naming conventions.
func StructValsToURLVals(v interface{}) (url.Values, error) {
	values := url.Values{}

	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return nil, errors.New("address of struct needs to be passed in")
	}

	structVal := reflect.ValueOf(v).Elem()
	structType := structVal.Type()

	for i := 0; i < structVal.NumField(); i++ {
		structField := structVal.Field(i)

		var outgoingTag string
		if structType.Field(i).Tag != "" {
			jsonTag := structType.Field(i).Tag.Get("json")
			if jsonTag != "" {
				split := common.SplitStrings(jsonTag, ",")
				outgoingTag = split[0]
			}
		}

		if outgoingTag == "" {
			outgoingTag = structType.Field(i).Name
		}

		var v string
		switch structField.Interface().(type) {
		case int, int8, int16, int32, int64:
			if structField.Int() == 0 {
				continue
			}
			v = strconv.FormatInt(structField.Int(), 10)
		case uint, uint8, uint16, uint32, uint64:
			if structField.Uint() == 0 {
				continue
			}
			v = strconv.FormatUint(structField.Uint(), 10)
		case float32:
			if structField.Float() == 0 {
				continue
			}
			v = strconv.FormatFloat(structField.Float(), 'f', 4, 32)
		case float64:
			if structField.Float() == 0 {
				continue
			}
			v = strconv.FormatFloat(structField.Float(), 'f', 4, 64)
		case []byte:
			if structField.Bytes() == nil {
				continue
			}
			v = string(structField.Bytes())
		case string:
			if structField.String() == "" {
				continue
			}
			v = structField.String()
		case bool:
			v = strconv.FormatBool(structField.Bool())
		}
		values.Set(outgoingTag, v)
	}
	return values, nil
}

// APIKeyParams contains all the parameters to send to the API endpoint
type APIKeyParams struct {
	//API Key ID (public component).
	APIKeyID string `json:"apiKeyID,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p APIKeyParams) VerifyData() error {
	if p.APIKeyID == "" {
		return errors.New("verifydata APIKeyParams error - APIKeyID not set")
	}
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p APIKeyParams) ToURLVals(path string) (string, error) {
	values, err := StructValsToURLVals(&p)
	if err != nil {
		return "", err
	}
	return common.EncodeURLValues(path, values), nil
}

// IsNil checks to see if any values has been set for the paramater
func (p APIKeyParams) IsNil() bool {
	if p == (APIKeyParams{}) {
		return true
	}
	return false
}

// ChatGetParams contains all the parameters to send to the API endpoint
type ChatGetParams struct {
	// ChannelID - [Optional] Leave blank for all.
	ChannelID float64 `json:"channelID,omitempty"`

	// Count - [Optional] Number of results to fetch.
	Count int32 `json:"count,omitempty"`

	// Reverse - [Optional] If true, will sort results newest first.
	Reverse bool `json:"reverse,omitempty"`

	// Start - [Optional] Starting ID for results.
	Start int32 `json:"start,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p ChatGetParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p ChatGetParams) ToURLVals(path string) (string, error) {
	values, err := StructValsToURLVals(&p)
	if err != nil {
		return "", err
	}
	return common.EncodeURLValues(path, values), nil
}

// IsNil checks to see if any values has been set for the paramater
func (p ChatGetParams) IsNil() bool {
	if p == (ChatGetParams{}) {
		return true
	}
	return false
}

// ChatSendParams contains all the parameters to send to the API endpoint
type ChatSendParams struct {
	// ChannelID - Channel to post to. Default 1 (English).
	ChannelID float64 `json:"channelID,omitempty"`

	// Message to send
	Message string `json:"message,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p ChatSendParams) VerifyData() error {
	if p.ChannelID == 0 || p.Message == "" {
		return errors.New("ChatSendParams error params not correctly set")
	}
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p ChatSendParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p ChatSendParams) IsNil() bool {
	if p == (ChatSendParams{}) {
		return true
	}
	return false
}

// GenericRequestParams contains all the parameters for some general functions
type GenericRequestParams struct {
	// Columns - [Optional] Array of column names to fetch. If omitted, will
	// return all columns.
	// NOTE that this method will always return item keys, even when not
	// specified, so you may receive more columns that you expect.
	Columns string `json:"columns,omitempty"`

	// Count - Number of results to fetch.
	Count int32 `json:"count,omitempty"`

	// EndTime - Ending date filter for results.
	EndTime string `json:"endTime,omitempty"`

	// Filter - Generic table filter. Send JSON key/value pairs, such as
	// `{"key": "value"}`. You can key on individual fields, and do more advanced
	// querying on timestamps. See the
	// [Timestamp Docs](https://testnet.bitmex.com/app/restAPI#Timestamp-Filters)
	// for more details.
	Filter string `json:"filter,omitempty"`

	// Reverse - If true, will sort results newest first.
	Reverse bool `json:"reverse,omitempty"`

	// Start - Starting point for results.
	Start int32 `json:"start,omitempty"`

	// StartTime - Starting date filter for results.
	StartTime string `json:"startTime,omitempty"`

	// Symbol - Instrument symbol. Send a bare series (e.g. XBU) to get data for
	// the nearest expiring contract in that series.
	// You can also send a timeframe, e.g. `XBU:monthly`. Timeframes are `daily`,
	// `weekly`, `monthly`, `quarterly`, and `biquarterly`.
	Symbol string `json:"symbol,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p GenericRequestParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p GenericRequestParams) ToURLVals(path string) (string, error) {
	values, err := StructValsToURLVals(&p)
	if err != nil {
		return "", err
	}
	return common.EncodeURLValues(path, values), nil
}

// IsNil checks to see if any values has been set for the paramater
func (p GenericRequestParams) IsNil() bool {
	if p == (GenericRequestParams{}) {
		return true
	}
	return false
}

// LeaderboardGetParams contains all the parameters to send to the API endpoint
type LeaderboardGetParams struct {
	// MethodRanking - [Optional] type. Options: "notional", "ROE"
	Method string `json:"method,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p LeaderboardGetParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p LeaderboardGetParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p LeaderboardGetParams) IsNil() bool {
	if p == (LeaderboardGetParams{}) {
		return true
	}
	return false
}

// OrderNewParams contains all the parameters to send to the API endpoint
type OrderNewParams struct {
	// ClOrdID - [Optional] Client Order ID. This clOrdID will come back on the
	// order and any related executions.
	ClOrdID string `json:"clOrdID,omitempty"`

	// ClOrdLinkID - [Optional] Client Order Link ID for contingent orders.
	ClOrdLinkID string `json:"clOrdLinkID,omitempty"`

	// ContingencyType - [Optional] contingency type for use with `clOrdLinkID`.
	// Valid options: OneCancelsTheOther, OneTriggersTheOther,
	// OneUpdatesTheOtherAbsolute, OneUpdatesTheOtherProportional.
	ContingencyType string `json:"contingencyType,omitempty"`

	// DisplayQty - [Optional] quantity to display in the book. Use 0 for a fully
	// hidden order.
	DisplayQty int32 `json:"displayQty,omitempty"`

	// ExecInst - [Optional] execution instructions. Valid options:
	// ParticipateDoNotInitiate, AllOrNone, MarkPrice, IndexPrice, LastPrice,
	// Close, ReduceOnly, Fixed. 'AllOrNone' instruction requires `displayQty`
	// to be 0. 'MarkPrice', 'IndexPrice' or 'LastPrice' instruction valid for
	// 'Stop', 'StopLimit', 'MarketIfTouched', and 'LimitIfTouched' orders.
	ExecInst string `json:"execInst,omitempty"`

	// OrdType - Order type. Valid options: Market, Limit, Stop, StopLimit,
	// MarketIfTouched, LimitIfTouched, MarketWithLeftOverAsLimit, Pegged.
	// Defaults to 'Limit' when `price` is specified. Defaults to 'Stop' when
	// `stopPx` is specified. Defaults to 'StopLimit' when `price` and `stopPx`
	// are specified.
	OrdType string `json:"ordType,omitempty"`

	//OrderQty Order quantity in units of the instrument (i.e. contracts).
	OrderQty int32 `json:"orderQty,omitempty"`

	// PegOffsetValue - [Optional] trailing offset from the current price for
	// 'Stop', 'StopLimit', 'MarketIfTouched', and 'LimitIfTouched' orders; use a
	// negative offset for stop-sell orders and buy-if-touched orders. [Optional]
	// offset from the peg price for 'Pegged' orders.
	PegOffsetValue float64 `json:"pegOffsetValue,omitempty"`

	// PegPriceType - [Optional] peg price type. Valid options: LastPeg,
	// MidPricePeg, MarketPeg, PrimaryPeg, TrailingStopPeg.
	PegPriceType string `json:"pegPriceType,omitempty"`

	// Price - [Optional] limit price for 'Limit', 'StopLimit', and
	// 'LimitIfTouched' orders.
	Price float64 `json:"price,omitempty"`

	// Side - Order side. Valid options: Buy, Sell. Defaults to 'Buy' unless
	// `orderQty` or `simpleOrderQty` is negative.
	Side string `json:"side,omitempty"`

	// SimpleOrderQty - Order quantity in units of the underlying instrument
	// (i.e. Bitcoin).
	SimpleOrderQty float64 `json:"simpleOrderQty,omitempty"`

	// StopPx - [Optional] trigger price for 'Stop', 'StopLimit',
	// 'MarketIfTouched', and 'LimitIfTouched' orders. Use a price below the
	// current price for stop-sell orders and buy-if-touched orders. Use
	// `execInst` of 'MarkPrice' or 'LastPrice' to define the current price used
	// for triggering.
	StopPx float64 `json:"stopPx,omitempty"`

	// Symbol - Instrument symbol. e.g. 'XBTUSD'.
	Symbol string `json:"symbol,omitempty"`

	// Text - [Optional] order annotation. e.g. 'Take profit'.
	Text string `json:"text,omitempty"`

	// TimeInForce - Valid options: Day, GoodTillCancel, ImmediateOrCancel,
	// FillOrKill. Defaults to 'GoodTillCancel' for 'Limit', 'StopLimit',
	// 'LimitIfTouched', and 'MarketWithLeftOverAsLimit' orders.
	TimeInForce string `json:"timeInForce,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p OrderNewParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p OrderNewParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p OrderNewParams) IsNil() bool {
	if p == (OrderNewParams{}) {
		return true
	}
	return false
}

// OrderAmendParams contains all the parameters to send to the API endpoint
// for the order amend operation
type OrderAmendParams struct {
	// ClOrdID - [Optional] new Client Order ID, requires `origClOrdID`.
	ClOrdID string `json:"clOrdID,omitempty"`

	// LeavesQty - [Optional] leaves quantity in units of the instrument
	// (i.e. contracts). Useful for amending partially filled orders.
	LeavesQty int32 `json:"leavesQty,omitempty"`

	OrderID string `json:"orderID,omitempty"`

	// OrderQty - [Optional] order quantity in units of the instrument
	// (i.e. contracts).
	OrderQty int32 `json:"orderQty,omitempty"`

	// OrigClOrdID - Client Order ID. See POST /order.
	OrigClOrdID string `json:"origClOrdID,omitempty"`

	// PegOffsetValue - [Optional] trailing offset from the current price for
	// 'Stop', 'StopLimit', 'MarketIfTouched', and 'LimitIfTouched' orders; use a
	// negative offset for stop-sell orders and buy-if-touched orders. [Optional]
	// offset from the peg price for 'Pegged' orders.
	PegOffsetValue float64 `json:"pegOffsetValue,omitempty"`

	// Price - [Optional] limit price for 'Limit', 'StopLimit', and
	// 'LimitIfTouched' orders.
	Price float64 `json:"price,omitempty"`

	// SimpleLeavesQty - [Optional] leaves quantity in units of the underlying
	// instrument (i.e. Bitcoin). Useful for amending partially filled orders.
	SimpleLeavesQty float64 `json:"simpleLeavesQty,omitempty"`

	// SimpleOrderQty - [Optional] order quantity in units of the underlying
	// instrument (i.e. Bitcoin).
	SimpleOrderQty float64 `json:"simpleOrderQty,omitempty"`

	// StopPx - [Optional] trigger price for 'Stop', 'StopLimit',
	// 'MarketIfTouched', and 'LimitIfTouched' orders. Use a price below the
	// current price for stop-sell orders and buy-if-touched orders.
	StopPx float64 `json:"stopPx,omitempty"`

	// Text - [Optional] amend annotation. e.g. 'Adjust skew'.
	Text string `json:"text,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p OrderAmendParams) VerifyData() error {
	if p.OrderID == "" {
		return errors.New("verifydata() OrderNewParams error - OrderID not set")
	}
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p OrderAmendParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p OrderAmendParams) IsNil() bool {
	if p == (OrderAmendParams{}) {
		return true
	}
	return false
}

// OrderCancelParams contains all the parameters to send to the API endpoint
type OrderCancelParams struct {
	// ClOrdID - Client Order ID(s). See POST /order.
	ClOrdID string `json:"clOrdID,omitempty"`

	// OrderID - Order ID(s).
	OrderID string `json:"orderID,omitempty"`

	// Text - [Optional] cancellation annotation. e.g. 'Spread Exceeded'.
	Text string `json:"text,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p OrderCancelParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p OrderCancelParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p OrderCancelParams) IsNil() bool {
	if p == (OrderCancelParams{}) {
		return true
	}
	return false
}

// OrderCancelAllParams contains all the parameters to send to the API endpoint
// for cancelling all your orders
type OrderCancelAllParams struct {
	// Filter - [Optional] filter for cancellation. Use to only cancel some
	// orders, e.g. `{"side": "Buy"}`.
	Filter string `json:"filter,omitempty"`

	// Symbol - [Optional] symbol. If provided, only cancels orders for that
	// symbol.
	Symbol string `json:"symbol,omitempty"`

	// Text - [Optional] cancellation annotation. e.g. 'Spread Exceeded'
	Text string `json:"text,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p OrderCancelAllParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p OrderCancelAllParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p OrderCancelAllParams) IsNil() bool {
	if p == (OrderCancelAllParams{}) {
		return true
	}
	return false
}

// OrderAmendBulkParams contains all the parameters to send to the API endpoint
type OrderAmendBulkParams struct {
	// Orders - An array of orders.
	Orders []OrderAmendParams `json:"orders,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p OrderAmendBulkParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p OrderAmendBulkParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p OrderAmendBulkParams) IsNil() bool {
	if len(p.Orders) == 0 {
		return true
	}
	return false
}

// OrderNewBulkParams contains all the parameters to send to the API endpoint
type OrderNewBulkParams struct {
	// Orders - An array of orders.
	Orders []OrderNewParams `json:"orders,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p OrderNewBulkParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p OrderNewBulkParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p OrderNewBulkParams) IsNil() bool {
	if len(p.Orders) == 0 {
		return true
	}
	return false
}

// OrderCancelAllAfterParams contains all the parameters to send to the API
// endpoint
type OrderCancelAllAfterParams struct {
	// Timeout in ms. Set to 0 to cancel this timer.
	Timeout float64 `json:"timeout,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p OrderCancelAllAfterParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p OrderCancelAllAfterParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p OrderCancelAllAfterParams) IsNil() bool {
	if p == (OrderCancelAllAfterParams{}) {
		return true
	}
	return false
}

// OrderClosePositionParams contains all the parameters to send to the API
// endpoint
type OrderClosePositionParams struct {
	// Price - [Optional] limit price.
	Price float64 `json:"price,omitempty"`

	// Symbol of position to close.
	Symbol string `json:"symbol,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p OrderClosePositionParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p OrderClosePositionParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p OrderClosePositionParams) IsNil() bool {
	if p == (OrderClosePositionParams{}) {
		return true
	}
	return false
}

// OrderBookGetL2Params contains all the parameters to send to the API endpoint
type OrderBookGetL2Params struct {
	// Depth - Orderbook depth per side. Send 0 for full depth.
	Depth int32 `json:"depth,omitempty"`

	// Symbol -Instrument symbol. Send a series (e.g. XBT) to get data for the
	// nearest contract in that series.
	Symbol string `json:"symbol,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p OrderBookGetL2Params) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p OrderBookGetL2Params) ToURLVals(path string) (string, error) {
	values, err := StructValsToURLVals(&p)
	if err != nil {
		return "", err
	}
	return common.EncodeURLValues(path, values), nil
}

// IsNil checks to see if any values has been set for the paramater
func (p OrderBookGetL2Params) IsNil() bool {
	if p == (OrderBookGetL2Params{}) {
		return true
	}
	return false
}

// PositionGetParams contains all the parameters to send to the API endpoint
type PositionGetParams struct {

	// Columns - Which columns to fetch. For example, send ["columnName"].
	Columns string `json:"columns,omitempty"`

	// Count - Number of rows to fetch.
	Count int32 `json:"count,omitempty"`

	// Filter - Table filter. For example, send {"symbol": "XBTUSD"}.
	Filter string `json:"filter,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p PositionGetParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p PositionGetParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p PositionGetParams) IsNil() bool {
	if p == (PositionGetParams{}) {
		return true
	}
	return false
}

// PositionIsolateMarginParams contains all the parameters to send to the API
// endpoint
type PositionIsolateMarginParams struct {
	// Enabled - True for isolated margin, false for cross margin.
	Enabled bool `json:"enabled,omitempty"`

	// Symbol - Position symbol to isolate.
	Symbol string `json:"symbol,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p PositionIsolateMarginParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p PositionIsolateMarginParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p PositionIsolateMarginParams) IsNil() bool {
	if p == (PositionIsolateMarginParams{}) {
		return true
	}
	return false
}

// PositionUpdateLeverageParams contains all the parameters to send to the API
// endpoint
type PositionUpdateLeverageParams struct {
	// Leverage - Leverage value. Send a number between 0.01 and 100 to enable
	// isolated margin with a fixed leverage. Send 0 to enable cross margin.
	Leverage float64 `json:"leverage,omitempty"`

	// Symbol - Symbol of position to adjust.
	Symbol string `json:"symbol,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p PositionUpdateLeverageParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p PositionUpdateLeverageParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p PositionUpdateLeverageParams) IsNil() bool {
	if p == (PositionUpdateLeverageParams{}) {
		return true
	}
	return false
}

// PositionUpdateRiskLimitParams contains all the parameters to send to the API
// endpoint
type PositionUpdateRiskLimitParams struct {
	// RiskLimit - New Risk Limit, in Satoshis.
	RiskLimit int64 `json:"riskLimit,omitempty"`

	// Symbol - Symbol of position to update risk limit on.
	Symbol string `json:"symbol,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p PositionUpdateRiskLimitParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p PositionUpdateRiskLimitParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p PositionUpdateRiskLimitParams) IsNil() bool {
	if p == (PositionUpdateRiskLimitParams{}) {
		return true
	}
	return false
}

// PositionTransferIsolatedMarginParams contains all the parameters to send to
// the API endpoint
type PositionTransferIsolatedMarginParams struct {
	// Amount - Amount to transfer, in Satoshis. May be negative.
	Amount int64 `json:"amount,omitempty"`

	// Symbol - Symbol of position to isolate.
	Symbol string `json:"symbol,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p PositionTransferIsolatedMarginParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p PositionTransferIsolatedMarginParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p PositionTransferIsolatedMarginParams) IsNil() bool {
	if p == (PositionTransferIsolatedMarginParams{}) {
		return true
	}
	return false
}

// QuoteGetBucketedParams contains all the parameters to send to the API
// endpoint
type QuoteGetBucketedParams struct {
	// BinSize - Time interval to bucket by. Available options: [1m,5m,1h,1d].
	BinSize string `json:"binSize,omitempty"`

	// Columns - Array of column names to fetch. If omitted, will return all
	// columns. NOTE that this method will always return item keys, even when not
	// specified, so you may receive more columns that you expect.
	Columns string `json:"columns,omitempty"`

	// Count - Number of results to fetch.
	Count int32 `json:"count,omitempty"`

	// EndTime - Ending date filter for results.
	EndTime string `json:"endTime,omitempty"`

	// Filter - Generic table filter. Send JSON key/value pairs, such as
	// `{"key": "value"}`. You can key on individual fields, and do more advanced
	// querying on timestamps. See the
	// [Timestamp Docs](https://testnet.bitmex.com/app/restAPI#Timestamp-Filters)
	// for more details.
	Filter string `json:"filter,omitempty"`

	// Partial - If true, will send in-progress (incomplete) bins for the current
	// time period.
	Partial bool `json:"partial,omitempty"`

	// Reverse - If true, will sort results newest first.
	Reverse bool `json:"reverse,omitempty"`

	// Start - Starting point for results.
	Start int32 `json:"start,omitempty"`

	// StartTime - Starting date filter for results.
	StartTime string `json:"startTime,omitempty"`

	// Symbol - Instrument symbol. Send a bare series (e.g. XBU) to get data for
	// the nearest expiring contract in that series.You can also send a timeframe,
	// e.g. `XBU:monthly`. Timeframes are `daily`, `weekly`, `monthly`,
	// `quarterly`, and `biquarterly`.
	Symbol string `json:"symbol,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p QuoteGetBucketedParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p QuoteGetBucketedParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p QuoteGetBucketedParams) IsNil() bool {
	if p == (QuoteGetBucketedParams{}) {
		return true
	}
	return false
}

// TradeGetBucketedParams contains all the parameters to send to the API
// endpoint
type TradeGetBucketedParams struct {
	// BinSize - Time interval to bucket by. Available options: [1m,5m,1h,1d].
	BinSize string `json:"binSize,omitempty"`

	// Columns - Array of column names to fetch. If omitted, will return all
	// columns.
	// Note that this method will always return item keys, even when not
	// specified, so you may receive more columns that you expect.
	Columns string `json:"columns,omitempty"`

	// Count - Number of results to fetch.
	Count int32 `json:"count,omitempty"`

	// EndTime - Ending date filter for results.
	EndTime string `json:"endTime,omitempty"`

	// Filter - Generic table filter. Send JSON key/value pairs, such as
	// `{"key": "value"}`. You can key on individual fields, and do more advanced
	// querying on timestamps. See the
	// [Timestamp Docs](https://testnet.bitmex.com/app/restAPI#Timestamp-Filters)
	// for more details.
	Filter string `json:"filter,omitempty"`

	// Partial - If true, will send in-progress (incomplete) bins for the current
	// time period.
	Partial bool `json:"partial,omitempty"`

	// Reverse - If true, will sort results newest first.
	Reverse bool `json:"reverse,omitempty"`

	// Start - Starting point for results.
	Start int32 `json:"start,omitempty"`

	// StartTime - Starting date filter for results.
	StartTime string `json:"startTime,omitempty"`

	// Symbol - Instrument symbol. Send a bare series (e.g. XBU) to get data for
	// the nearest expiring contract in that series.You can also send a timeframe,
	// e.g. `XBU:monthly`. Timeframes are `daily`, `weekly`, `monthly`,
	// `quarterly`, and `biquarterly`.
	Symbol string `json:"symbol,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p TradeGetBucketedParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p TradeGetBucketedParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p TradeGetBucketedParams) IsNil() bool {
	if p == (TradeGetBucketedParams{}) {
		return true
	}
	return false
}

// UserUpdateParams contains all the parameters to send to the API endpoint
type UserUpdateParams struct {
	// Country - Country of residence.
	Country string `json:"country,omitempty"`

	// New Password string
	NewPassword string `json:"newPassword,omitempty"`

	// Confirmation string - must match
	NewPasswordConfirm string `json:"newPasswordConfirm,omitempty"`

	// old password string
	OldPassword string `json:"oldPassword,omitempty"`

	// PGP Public Key. If specified, automated emails will be sent with this key.
	PgpPubKey string `json:"pgpPubKey,omitempty"`

	// Username can only be set once. To reset, email support.
	Username string `json:"username,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p UserUpdateParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p UserUpdateParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p UserUpdateParams) IsNil() bool {
	if p == (UserUpdateParams{}) {
		return true
	}
	return false
}

// UserTokenParams contains all the parameters to send to the API endpoint
type UserTokenParams struct {
	Token string `json:"token,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p UserTokenParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p UserTokenParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p UserTokenParams) IsNil() bool {
	if p == (UserTokenParams{}) {
		return true
	}
	return false
}

// UserCheckReferralCodeParams contains all the parameters to send to the API
// endpoint
type UserCheckReferralCodeParams struct {
	ReferralCode string `json:"referralCode,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p UserCheckReferralCodeParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p UserCheckReferralCodeParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p UserCheckReferralCodeParams) IsNil() bool {
	if p == (UserCheckReferralCodeParams{}) {
		return true
	}
	return false
}

// UserConfirmTFAParams contains all the parameters to send to the API endpoint
type UserConfirmTFAParams struct {
	// Token - Token from your selected TFA type.
	Token string `json:"token,omitempty"`

	// Type - Two-factor auth type. Supported types: 'GA' (Google Authenticator),
	// 'Yubikey'
	Type string `json:"type,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p UserConfirmTFAParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p UserConfirmTFAParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p UserConfirmTFAParams) IsNil() bool {
	if p == (UserConfirmTFAParams{}) {
		return true
	}
	return false
}

// UserCurrencyParams contains all the parameters to send to the API endpoint
type UserCurrencyParams struct {
	Currency string `json:"currency,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p UserCurrencyParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p UserCurrencyParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p UserCurrencyParams) IsNil() bool {
	if p == (UserCurrencyParams{}) {
		return true
	}
	return false
}

// UserPreferencesParams contains all the parameters to send to the API
// endpoint
type UserPreferencesParams struct {
	// Overwrite - If true, will overwrite all existing preferences.
	Overwrite bool `json:"overwrite,omitempty"`
	// Prefs - preferences
	Prefs string `json:"prefs,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p UserPreferencesParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p UserPreferencesParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p UserPreferencesParams) IsNil() bool {
	if p == (UserPreferencesParams{}) {
		return true
	}
	return false
}

// UserRequestWithdrawalParams contains all the parameters to send to the API
// endpoint
type UserRequestWithdrawalParams struct {
	// Address - Destination Address.
	Address string `json:"address,omitempty"`

	// Amount - Amount of withdrawal currency.
	Amount int64 `json:"amount,omitempty"`

	// Currency - Currency you're withdrawing. Options: `XBt`
	Currency string `json:"currency,omitempty"`

	// Fee - Network fee for Bitcoin withdrawals. If not specified, a default
	// value will be calculated based on Bitcoin network conditions. You will have
	// a chance to confirm this via email.
	Fee float64 `json:"fee,omitempty"`

	// OtpToken - 2FA token. Required if 2FA is enabled on your account.
	OtpToken string `json:"otpToken,omitempty"`
}

// VerifyData verifies outgoing data sets
func (p UserRequestWithdrawalParams) VerifyData() error {
	return nil
}

// ToURLVals converts struct values to url.values and encodes it on the supplied
// path
func (p UserRequestWithdrawalParams) ToURLVals(path string) (string, error) {
	return "", nil
}

// IsNil checks to see if any values has been set for the paramater
func (p UserRequestWithdrawalParams) IsNil() bool {
	if p == (UserRequestWithdrawalParams{}) {
		return true
	}
	return false
}
