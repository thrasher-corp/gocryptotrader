package deribit

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	errIndexNameRequired           = errors.New("index name is required")
	errInstrumentIDRequired        = errors.New("instrument ID is required")
	errInvalidFundingChartLength   = errors.New("invalid funding chart length")
	errInvalidIndexChartRange      = errors.New("invalid index chart range")
	errInvalidIndexType            = errors.New("invalid index type")
	errInvalidLastSettlementsType  = errors.New("invalid settlement type")
	errInvalidLastTradesSorting    = errors.New("invalid trade sorting")
	errLastTradesTimeRangeRequired = errors.New("start or end time is required")
)

// GetAPRHistory returns a yield-bearing token's daily APRs, newest first, a page of up to Limit days at a time
func (e *Exchange) GetAPRHistory(ctx context.Context, arg *APRHistoryRequest) (*APRHistoryResponse, error) {
	return e.getAPRHistory(ctx, arg, false)
}

func (e *Exchange) getAPRHistory(ctx context.Context, arg *APRHistoryRequest, overWebsocket bool) (*APRHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *APRHistoryResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_apr_history", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetBasketIndexComposition returns a basket index's latest constituents and weights; COIN50 is the only basket index
// Deribit lists, and its name is case-insensitive
func (e *Exchange) GetBasketIndexComposition(ctx context.Context, indexName string) (*BasketIndexCompositionResponse, error) {
	return e.getBasketIndexComposition(ctx, indexName, false)
}

func (e *Exchange) getBasketIndexComposition(ctx context.Context, indexName string, overWebsocket bool) (*BasketIndexCompositionResponse, error) {
	if indexName == "" {
		return nil, errIndexNameRequired
	}
	var resp *BasketIndexCompositionResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_basket_index_composition", &indexNameParams{IndexName: indexName}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetBookSummaryByCurrency returns the market summaries of a currency's instruments, optionally of one kind; Deribit
// recommends the ticker subscription over polling it
func (e *Exchange) GetBookSummaryByCurrency(ctx context.Context, arg *BookSummaryByCurrencyRequest) ([]BookSummary, error) {
	return e.getBookSummaryByCurrency(ctx, arg, false)
}

func (e *Exchange) getBookSummaryByCurrency(ctx context.Context, arg *BookSummaryByCurrencyRequest, overWebsocket bool) ([]BookSummary, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateMarketDataKind(arg.Kind, KindFuture, KindOption, KindSpot, KindFutureCombo, KindOptionCombo); err != nil {
		return nil, err
	}
	var resp []BookSummary
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_book_summary_by_currency", arg, &resp)
}

// GetBookSummaryByInstrument returns an instrument's market summary, as a list of one
func (e *Exchange) GetBookSummaryByInstrument(ctx context.Context, instrumentName string) ([]BookSummary, error) {
	return e.getBookSummaryByInstrument(ctx, instrumentName, false)
}

func (e *Exchange) getBookSummaryByInstrument(ctx context.Context, instrumentName string, overWebsocket bool) ([]BookSummary, error) {
	if instrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp []BookSummary
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_book_summary_by_instrument", &instrumentNameParams{InstrumentName: instrumentName}, &resp)
}

// GetContractSize returns an instrument's contract size, including a combo's or an expired instrument's
func (e *Exchange) GetContractSize(ctx context.Context, instrumentName string) (*ContractSizeResponse, error) {
	return e.getContractSize(ctx, instrumentName, false)
}

func (e *Exchange) getContractSize(ctx context.Context, instrumentName string, overWebsocket bool) (*ContractSizeResponse, error) {
	if instrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp *ContractSizeResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_contract_size", &instrumentNameParams{InstrumentName: instrumentName}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetCurrencies returns every currency Deribit supports and its deposit and withdrawal parameters
func (e *Exchange) GetCurrencies(ctx context.Context) ([]CurrencyDetail, error) {
	return e.getCurrencies(ctx, false)
}

func (e *Exchange) getCurrencies(ctx context.Context, overWebsocket bool) ([]CurrencyDetail, error) {
	var resp []CurrencyDetail
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_currencies", nil, &resp)
}

// GetDeliveryPrices returns an index's delivery prices, newest first, and the number of prices available
func (e *Exchange) GetDeliveryPrices(ctx context.Context, arg *DeliveryPricesRequest) (*DeliveryPricesResponse, error) {
	return e.getDeliveryPrices(ctx, arg, false)
}

func (e *Exchange) getDeliveryPrices(ctx context.Context, arg *DeliveryPricesRequest, overWebsocket bool) (*DeliveryPricesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.IndexName == "" {
		return nil, errIndexNameRequired
	}
	var resp *DeliveryPricesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_delivery_prices", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetExpirations returns the expiration labels of listed futures and options, such as 27NOV26 or PERPETUAL, not
// timestamps; an empty request merges every currency's labels
func (e *Exchange) GetExpirations(ctx context.Context, arg *ExpirationsRequest) (*ExpirationsResponse, error) {
	return e.getExpirations(ctx, arg, false)
}

func (e *Exchange) getExpirations(ctx context.Context, arg *ExpirationsRequest, overWebsocket bool) (*ExpirationsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateMarketDataKind(arg.Kind, KindFuture, KindOption, KindAny); err != nil {
		return nil, err
	}
	var resp *ExpirationsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_expirations", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetFundingChartData returns a perpetual's current funding and its 8 hour funding and index price over a period, a
// point a minute for 8 and 24 hours and a point an hour for a month
func (e *Exchange) GetFundingChartData(ctx context.Context, arg *FundingChartDataRequest) (*FundingChartDataResponse, error) {
	return e.getFundingChartData(ctx, arg, false)
}

func (e *Exchange) getFundingChartData(ctx context.Context, arg *FundingChartDataRequest, overWebsocket bool) (*FundingChartDataResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	switch arg.Length {
	case FundingChartLength8Hour, FundingChartLength24Hour, FundingChartLengthOneMonth:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidFundingChartLength, arg.Length)
	}
	var resp *FundingChartDataResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_funding_chart_data", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetFundingRateHistory returns a perpetual's hourly funding over the hours ending after StartTime and up to EndTime,
// oldest first; a range of more than 744 hours returns its latest 744
func (e *Exchange) GetFundingRateHistory(ctx context.Context, arg *InstrumentTimeRangeRequest) ([]FundingRateHistory, error) {
	return e.getFundingRateHistory(ctx, arg, false)
}

func (e *Exchange) getFundingRateHistory(ctx context.Context, arg *InstrumentTimeRangeRequest, overWebsocket bool) ([]FundingRateHistory, error) {
	params, err := arg.params()
	if err != nil {
		return nil, err
	}
	var resp []FundingRateHistory
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_funding_rate_history", params, &resp)
}

// GetFundingRateValue returns a perpetual's funding rate accumulated over a time range
func (e *Exchange) GetFundingRateValue(ctx context.Context, arg *InstrumentTimeRangeRequest) (float64, error) {
	return e.getFundingRateValue(ctx, arg, false)
}

func (e *Exchange) getFundingRateValue(ctx context.Context, arg *InstrumentTimeRangeRequest, overWebsocket bool) (float64, error) {
	params, err := arg.params()
	if err != nil {
		return 0, err
	}
	var resp *float64
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_funding_rate_value", params, &resp); err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, common.ErrNoResponse
	}
	return *resp, nil
}

// params returns an instrument time range request's params, checking its instrument and time range are set
func (r *InstrumentTimeRangeRequest) params() (*instrumentTimeRangeParams, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if r.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := common.StartEndTimeCheck(r.StartTime, r.EndTime); err != nil {
		return nil, err
	}
	return &instrumentTimeRangeParams{
		InstrumentTimeRangeRequest: r,
		StartTimestamp:             r.StartTime.UnixMilli(),
		EndTimestamp:               r.EndTime.UnixMilli(),
	}, nil
}

// GetHistoricalVolatility returns a currency's hourly realised volatility over about the last 16 days, oldest first;
// Deribit repeats the first point
func (e *Exchange) GetHistoricalVolatility(ctx context.Context, ccy currency.Code) ([]HistoricalVolatilityPoint, error) {
	return e.getHistoricalVolatility(ctx, ccy, false)
}

func (e *Exchange) getHistoricalVolatility(ctx context.Context, ccy currency.Code, overWebsocket bool) ([]HistoricalVolatilityPoint, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp []HistoricalVolatilityPoint
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_historical_volatility", &currencyParams{Currency: ccy}, &resp)
}

// GetIndexChartData returns an index's average price over a period, oldest first
func (e *Exchange) GetIndexChartData(ctx context.Context, arg *IndexChartDataRequest) ([]IndexChartPoint, error) {
	return e.getIndexChartData(ctx, arg, false)
}

func (e *Exchange) getIndexChartData(ctx context.Context, arg *IndexChartDataRequest, overWebsocket bool) ([]IndexChartPoint, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.IndexName == "" {
		return nil, errIndexNameRequired
	}
	switch arg.Range {
	case IndexChartRangeOneHour, IndexChartRangeOneDay, IndexChartRangeTwoDays, IndexChartRangeOneMonth, IndexChartRangeOneYear, IndexChartRangeAll:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidIndexChartRange, arg.Range)
	}
	var resp []IndexChartPoint
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_index_chart_data", arg, &resp)
}

// GetIndexPrice returns an index's current price and estimated delivery price
func (e *Exchange) GetIndexPrice(ctx context.Context, indexName string) (*IndexPriceResponse, error) {
	return e.getIndexPrice(ctx, indexName, false)
}

func (e *Exchange) getIndexPrice(ctx context.Context, indexName string, overWebsocket bool) (*IndexPriceResponse, error) {
	if indexName == "" {
		return nil, errIndexNameRequired
	}
	var resp *IndexPriceResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_index_price", &indexNameParams{IndexName: indexName}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetIndexPriceNames returns the names of every price index and, when extended is set, whether future and option combos
// can be created on each
func (e *Exchange) GetIndexPriceNames(ctx context.Context, extended bool) ([]IndexPriceName, error) {
	return e.getIndexPriceNames(ctx, extended, false)
}

func (e *Exchange) getIndexPriceNames(ctx context.Context, extended, overWebsocket bool) ([]IndexPriceName, error) {
	var resp []IndexPriceName
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_index_price_names", &indexPriceNamesParams{Extended: extended}, &resp)
}

// GetInstrument returns an instrument's specification, including a combo's or an expired or archivized instrument's
// that GetInstruments no longer lists
func (e *Exchange) GetInstrument(ctx context.Context, instrumentName string) (*Instrument, error) {
	return e.getInstrument(ctx, instrumentName, false)
}

func (e *Exchange) getInstrument(ctx context.Context, instrumentName string, overWebsocket bool) (*Instrument, error) {
	if instrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp *Instrument
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_instrument", &instrumentNameParams{InstrumentName: instrumentName}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetInstruments returns a settlement currency's instruments, optionally of one kind; unexpired instruments include
// inactive ones but not inactive combos, and Deribit limits the method to a request a second
func (e *Exchange) GetInstruments(ctx context.Context, arg *InstrumentsRequest) ([]Instrument, error) {
	return e.getInstruments(ctx, arg, false)
}

func (e *Exchange) getInstruments(ctx context.Context, arg *InstrumentsRequest, overWebsocket bool) ([]Instrument, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateMarketDataKind(arg.Kind, KindFuture, KindOption, KindSpot, KindFutureCombo, KindOptionCombo, KindAny); err != nil {
		return nil, err
	}
	params := *arg
	params.Currency = marketDataCurrency(arg.Currency)
	var resp []Instrument
	return resp, e.sendRequest(ctx, overWebsocket, getInstrumentsEPL, "public/get_instruments", &params, &resp)
}

// GetLastSettlementsByCurrency returns a currency's settlement, delivery and bankruptcy events, newest first; profit
// and loss figures are aggregated across every position holder
func (e *Exchange) GetLastSettlementsByCurrency(ctx context.Context, arg *LastSettlementsByCurrencyRequest) (*SettlementsResponse, error) {
	return e.getLastSettlementsByCurrency(ctx, arg, false)
}

func (e *Exchange) getLastSettlementsByCurrency(ctx context.Context, arg *LastSettlementsByCurrencyRequest, overWebsocket bool) (*SettlementsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateLastSettlementsType(arg.Type); err != nil {
		return nil, err
	}
	params := &lastSettlementsByCurrencyParams{
		LastSettlementsByCurrencyRequest: arg,
		SearchStartTimestamp:             marketDataUnixMilli(arg.SearchStartTime),
	}
	var resp *SettlementsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_last_settlements_by_currency", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLastSettlementsByInstrument returns an instrument's settlement, delivery and bankruptcy events, newest first;
// profit and loss figures are aggregated across every position holder
func (e *Exchange) GetLastSettlementsByInstrument(ctx context.Context, arg *LastSettlementsByInstrumentRequest) (*SettlementsResponse, error) {
	return e.getLastSettlementsByInstrument(ctx, arg, false)
}

func (e *Exchange) getLastSettlementsByInstrument(ctx context.Context, arg *LastSettlementsByInstrumentRequest, overWebsocket bool) (*SettlementsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateLastSettlementsType(arg.Type); err != nil {
		return nil, err
	}
	params := &lastSettlementsByInstrumentParams{
		LastSettlementsByInstrumentRequest: arg,
		SearchStartTimestamp:               marketDataUnixMilli(arg.SearchStartTime),
	}
	var resp *SettlementsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_last_settlements_by_instrument", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLastTradesByCurrency returns a currency's latest trades, optionally within a trade ID or time range; Deribit only
// serves the last 24 hours of trades
func (e *Exchange) GetLastTradesByCurrency(ctx context.Context, arg *LastTradesByCurrencyRequest) (*LastTradesResponse, error) {
	return e.getLastTradesByCurrency(ctx, arg, false)
}

func (e *Exchange) getLastTradesByCurrency(ctx context.Context, arg *LastTradesByCurrencyRequest, overWebsocket bool) (*LastTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateMarketDataKind(arg.Kind, KindFuture, KindOption, KindSpot, KindFutureCombo, KindOptionCombo, KindCombo, KindAny); err != nil {
		return nil, err
	}
	if err := validateLastTradesSorting(arg.Sorting); err != nil {
		return nil, err
	}
	if err := validateLastTradesTimeRange(arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	params := &lastTradesByCurrencyParams{
		LastTradesByCurrencyRequest: arg,
		StartTimestamp:              marketDataUnixMilli(arg.StartTime),
		EndTimestamp:                marketDataUnixMilli(arg.EndTime),
	}
	var resp *LastTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_last_trades_by_currency", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLastTradesByCurrencyAndTime returns a currency's latest trades within a time range, which needs a start or an end
// time; Deribit only serves the last 24 hours of trades
func (e *Exchange) GetLastTradesByCurrencyAndTime(ctx context.Context, arg *LastTradesByCurrencyAndTimeRequest) (*LastTradesResponse, error) {
	return e.getLastTradesByCurrencyAndTime(ctx, arg, false)
}

func (e *Exchange) getLastTradesByCurrencyAndTime(ctx context.Context, arg *LastTradesByCurrencyAndTimeRequest, overWebsocket bool) (*LastTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateMarketDataKind(arg.Kind, KindFuture, KindOption, KindSpot, KindFutureCombo, KindOptionCombo, KindCombo, KindAny); err != nil {
		return nil, err
	}
	if err := validateLastTradesSorting(arg.Sorting); err != nil {
		return nil, err
	}
	if err := validateLastTradesRequiredTimeRange(arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	params := &lastTradesByCurrencyAndTimeParams{
		LastTradesByCurrencyAndTimeRequest: arg,
		StartTimestamp:                     marketDataUnixMilli(arg.StartTime),
		EndTimestamp:                       marketDataUnixMilli(arg.EndTime),
	}
	var resp *LastTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_last_trades_by_currency_and_time", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLastTradesByInstrument returns an instrument's latest trades, optionally within a trade sequence or time range;
// Deribit only serves the last 24 hours of trades
func (e *Exchange) GetLastTradesByInstrument(ctx context.Context, arg *LastTradesByInstrumentRequest) (*LastTradesResponse, error) {
	return e.getLastTradesByInstrument(ctx, arg, false)
}

func (e *Exchange) getLastTradesByInstrument(ctx context.Context, arg *LastTradesByInstrumentRequest, overWebsocket bool) (*LastTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateLastTradesSorting(arg.Sorting); err != nil {
		return nil, err
	}
	if err := validateLastTradesTimeRange(arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	params := &lastTradesByInstrumentParams{
		LastTradesByInstrumentRequest: arg,
		StartTimestamp:                marketDataUnixMilli(arg.StartTime),
		EndTimestamp:                  marketDataUnixMilli(arg.EndTime),
	}
	var resp *LastTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_last_trades_by_instrument", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLastTradesByInstrumentAndTime returns an instrument's latest trades within a time range, which needs a start or an
// end time; Deribit only serves the last 24 hours of trades
func (e *Exchange) GetLastTradesByInstrumentAndTime(ctx context.Context, arg *LastTradesByInstrumentAndTimeRequest) (*LastTradesResponse, error) {
	return e.getLastTradesByInstrumentAndTime(ctx, arg, false)
}

func (e *Exchange) getLastTradesByInstrumentAndTime(ctx context.Context, arg *LastTradesByInstrumentAndTimeRequest, overWebsocket bool) (*LastTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateLastTradesSorting(arg.Sorting); err != nil {
		return nil, err
	}
	if err := validateLastTradesRequiredTimeRange(arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	params := &lastTradesByInstrumentAndTimeParams{
		LastTradesByInstrumentAndTimeRequest: arg,
		StartTimestamp:                       marketDataUnixMilli(arg.StartTime),
		EndTimestamp:                         marketDataUnixMilli(arg.EndTime),
	}
	var resp *LastTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_last_trades_by_instrument_and_time", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetMarkPriceHistory returns an option's mark prices within a time range, oldest first; Deribit keeps about the last 5
// minutes of history, and only for the options its volatility index is calculated from, so other instruments return
// none
func (e *Exchange) GetMarkPriceHistory(ctx context.Context, arg *InstrumentTimeRangeRequest) ([]MarkPricePoint, error) {
	return e.getMarkPriceHistory(ctx, arg, false)
}

func (e *Exchange) getMarkPriceHistory(ctx context.Context, arg *InstrumentTimeRangeRequest, overWebsocket bool) ([]MarkPricePoint, error) {
	params, err := arg.params()
	if err != nil {
		return nil, err
	}
	var resp []MarkPricePoint
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_mark_price_history", params, &resp)
}

// GetOrderBook returns an instrument's order book, up to a depth on each side, and its ticker
func (e *Exchange) GetOrderBook(ctx context.Context, arg *OrderBookRequest) (*OrderBookResponse, error) {
	return e.getOrderBook(ctx, arg, false)
}

func (e *Exchange) getOrderBook(ctx context.Context, arg *OrderBookRequest, overWebsocket bool) (*OrderBookResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp *OrderBookResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_order_book", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOrderBookByInstrumentID returns the order book of an instrument identified by its ID, up to a depth on each side,
// and its ticker
func (e *Exchange) GetOrderBookByInstrumentID(ctx context.Context, arg *OrderBookByInstrumentIDRequest) (*OrderBookResponse, error) {
	return e.getOrderBookByInstrumentID(ctx, arg, false)
}

func (e *Exchange) getOrderBookByInstrumentID(ctx context.Context, arg *OrderBookByInstrumentIDRequest, overWebsocket bool) (*OrderBookResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == 0 {
		return nil, errInstrumentIDRequired
	}
	var resp *OrderBookResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_order_book_by_instrument_id", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetSupportedIndexNames returns the names of the price indexes of a type, all, spot or derivative, or of every type
// when indexType is empty
func (e *Exchange) GetSupportedIndexNames(ctx context.Context, indexType string) ([]string, error) {
	return e.getSupportedIndexNames(ctx, indexType, false)
}

func (e *Exchange) getSupportedIndexNames(ctx context.Context, indexType string, overWebsocket bool) ([]string, error) {
	switch indexType {
	case "", IndexTypeAll, IndexTypeSpot, IndexTypeDerivative:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidIndexType, indexType)
	}
	var resp []string
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_supported_index_names", &supportedIndexNamesParams{Type: indexType}, &resp)
}

// GetTradeVolumes returns each currency's traded volumes over 24 hours and, when extended is set, 7 and 30 days
func (e *Exchange) GetTradeVolumes(ctx context.Context, extended bool) ([]TradeVolume, error) {
	return e.getTradeVolumes(ctx, extended, false)
}

func (e *Exchange) getTradeVolumes(ctx context.Context, extended, overWebsocket bool) ([]TradeVolume, error) {
	var resp []TradeVolume
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_trade_volumes", &tradeVolumesParams{Extended: extended}, &resp)
}

// GetTradingViewChartData returns an instrument's candles opening within a time range, as TradingView charts take them
func (e *Exchange) GetTradingViewChartData(ctx context.Context, arg *TradingViewChartDataRequest) (*TradingViewChartDataResponse, error) {
	return e.getTradingViewChartData(ctx, arg, false)
}

func (e *Exchange) getTradingViewChartData(ctx context.Context, arg *TradingViewChartDataRequest, overWebsocket bool) (*TradingViewChartDataResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	resolution, err := e.tradingViewResolution(arg.Resolution)
	if err != nil {
		return nil, err
	}
	params := &tradingViewChartDataParams{
		TradingViewChartDataRequest: arg,
		StartTimestamp:              arg.StartTime.UnixMilli(),
		EndTimestamp:                arg.EndTime.UnixMilli(),
		ChartResolution:             resolution,
	}
	var resp *TradingViewChartDataResponse
	if overWebsocket {
		err = e.SendWebsocketRequest(ctx, nonMatchingEPL, "public/get_tradingview_chart_data", params, &resp)
	} else {
		// Deribit's production gateway answers a POST of this method to its path with Method not found, while it serves
		// a GET of it, as its documentation specifies
		err = e.sendHTTPQueryRequest(ctx, "public/get_tradingview_chart_data", url.Values{
			"instrument_name": {arg.InstrumentName},
			"start_timestamp": {strconv.FormatInt(params.StartTimestamp, 10)},
			"end_timestamp":   {strconv.FormatInt(params.EndTimestamp, 10)},
			"resolution":      {resolution},
		}, &resp)
	}
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// tradingViewResolution returns Deribit's chart resolution for a candle interval
func (e *Exchange) tradingViewResolution(interval kline.Interval) (string, error) {
	// GetResolutionFromInterval also names the 100ms and raw intervals of subscriptions, which charts do not support
	if interval != kline.HundredMilliseconds && interval != kline.Raw {
		if resolution, err := e.GetResolutionFromInterval(interval); err == nil {
			return resolution, nil
		}
	}
	return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
}

// GetVolatilityIndexData returns up to 1000 of a currency's volatility index candles, the latest within a time range,
// oldest first; the response's Continuation is the end time of the next, earlier page
func (e *Exchange) GetVolatilityIndexData(ctx context.Context, arg *VolatilityIndexDataRequest) (*VolatilityIndexDataResponse, error) {
	return e.getVolatilityIndexData(ctx, arg, false)
}

func (e *Exchange) getVolatilityIndexData(ctx context.Context, arg *VolatilityIndexDataRequest, overWebsocket bool) (*VolatilityIndexDataResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	resolution, err := volatilityIndexResolution(arg.Resolution)
	if err != nil {
		return nil, err
	}
	params := &volatilityIndexDataParams{
		VolatilityIndexDataRequest: arg,
		StartTimestamp:             arg.StartTime.UnixMilli(),
		EndTimestamp:               arg.EndTime.UnixMilli(),
		IndexResolution:            resolution,
	}
	var resp *VolatilityIndexDataResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_volatility_index_data", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// volatilityIndexResolution returns Deribit's volatility index resolution, in seconds or 1D, for a candle interval
func volatilityIndexResolution(interval kline.Interval) (string, error) {
	switch interval {
	case kline.ThousandMilliseconds:
		return "1", nil
	case kline.OneMin:
		return "60", nil
	case kline.OneHour:
		return "3600", nil
	case kline.TwelveHour:
		return "43200", nil
	case kline.OneDay:
		return "1D", nil
	default:
		return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
	}
}

// GetTicker returns an instrument's ticker: its best prices, mark and index prices, 24 hour statistics and, by kind,
// greeks and implied volatilities, funding or combo prices
func (e *Exchange) GetTicker(ctx context.Context, instrumentName string) (*TickerResponse, error) {
	return e.getTicker(ctx, instrumentName, false)
}

func (e *Exchange) getTicker(ctx context.Context, instrumentName string, overWebsocket bool) (*TickerResponse, error) {
	if instrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp *TickerResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/ticker", &instrumentNameParams{InstrumentName: instrumentName}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// marketDataCurrency returns a currency as Deribit's market data methods accept it, lowercasing ANY, which Deribit
// rejects, so currency.ANY selects every currency
func marketDataCurrency(ccy currency.Code) currency.Code {
	if ccy.Equal(currency.ANY) {
		return ccy.Lower()
	}
	return ccy
}

// validateMarketDataKind checks a kind filter is empty, which Deribit defaults, or one the method accepts
func validateMarketDataKind(kind string, accepted ...string) error {
	if kind == "" || slices.Contains(accepted, kind) {
		return nil
	}
	return fmt.Errorf("%w: %q", errInvalidKind, kind)
}

// validateLastTradesSorting checks a trade sorting is empty or one Deribit documents, as Deribit ignores any other
func validateLastTradesSorting(sorting string) error {
	switch sorting {
	case "", "asc", "desc", "default":
		return nil
	default:
		return fmt.Errorf("%w: %q", errInvalidLastTradesSorting, sorting)
	}
}

// validateLastSettlementsType checks a settlement type filter is empty or one Deribit accepts; exercise events can only
// be returned unfiltered
func validateLastSettlementsType(settlementType string) error {
	switch settlementType {
	case "", "settlement", "delivery", "bankruptcy":
		return nil
	default:
		return fmt.Errorf("%w: %q", errInvalidLastSettlementsType, settlementType)
	}
}

// validateLastTradesTimeRange checks an optional trade time range does not start after it ends
func validateLastTradesTimeRange(start, end time.Time) error {
	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return common.ErrStartAfterEnd
	}
	return nil
}

// validateLastTradesRequiredTimeRange checks a trade time range has a start or an end time, either of which Deribit
// accepts alone, and does not start after it ends
func validateLastTradesRequiredTimeRange(start, end time.Time) error {
	if start.IsZero() && end.IsZero() {
		return errLastTradesTimeRangeRequired
	}
	return validateLastTradesTimeRange(start, end)
}

// marketDataUnixMilli returns a time in Unix milliseconds, or zero for a zero time, which leaves an optional timestamp
// out of a request
func marketDataUnixMilli(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
