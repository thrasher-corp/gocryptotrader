package coinbase

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const (
	// derivativesMarketDataMaximumCount is the most trades or settlement events a request returns; the gateway refuses a
	// larger count
	derivativesMarketDataMaximumCount = 1000
	// derivativesMaximumOrderBookDepth is the most price levels a side of an order book returns; the gateway refuses a
	// larger depth
	derivativesMaximumOrderBookDepth = 10000
)

var (
	errDerivativesIndexNameRequired          = errors.New("index name required")
	errDerivativesInstrumentIDRequired       = errors.New("instrument ID required")
	errDerivativesInvalidMarketDataKind      = errors.New("invalid instrument kind")
	errDerivativesInvalidFundingChartLength  = errors.New("invalid funding chart length")
	errDerivativesInvalidIndexChartRange     = errors.New("invalid index chart range")
	errDerivativesInvalidIndexType           = errors.New("invalid index type")
	errDerivativesInvalidLastSettlementsType = errors.New("invalid settlement type")
	errDerivativesInvalidLastTradesSorting   = errors.New("invalid trade sorting")
	errDerivativesLastTradesWindowRequired   = errors.New("trade window start or end must be set")
	errDerivativesInvalidDepth               = errors.New("invalid book depth")
)

// GetBookSummaryByCurrency returns the market summaries of a settlement currency's instruments, optionally of one kind;
// the docs recommend the ticker subscription over polling it
func (g GlobalDerivativesAPI) GetBookSummaryByCurrency(ctx context.Context, req *DerivativesBookSummaryByCurrencyRequest) ([]DerivativesBookSummary, error) {
	return g.getBookSummaryByCurrency(ctx, false, req)
}

func (g GlobalDerivativesAPI) getBookSummaryByCurrency(ctx context.Context, overWebsocket bool, req *DerivativesBookSummaryByCurrencyRequest) ([]DerivativesBookSummary, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	// The gateway answers any with no summaries, so only the kinds it summarises are accepted
	if err := validateDerivativesMarketDataKind(req.Kind, DerivativesKindFuture, DerivativesKindOption, DerivativesKindSpot, DerivativesKindFutureCombo, DerivativesKindOptionCombo); err != nil {
		return nil, err
	}
	params := &derivativesBookSummaryByCurrencyParams{Currency: derivativesMarketDataCurrency(req.Currency), Kind: req.Kind}
	var resp []DerivativesBookSummary
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_book_summary_by_currency", params, &resp)
}

// GetBookSummaryByInstrument returns an instrument's market summary, as a list of one
func (g GlobalDerivativesAPI) GetBookSummaryByInstrument(ctx context.Context, instrumentName string) ([]DerivativesBookSummary, error) {
	return g.getBookSummaryByInstrument(ctx, false, instrumentName)
}

func (g GlobalDerivativesAPI) getBookSummaryByInstrument(ctx context.Context, overWebsocket bool, instrumentName string) ([]DerivativesBookSummary, error) {
	if instrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp []DerivativesBookSummary
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_book_summary_by_instrument", &derivativesInstrumentNameParams{InstrumentName: instrumentName}, &resp)
}

// GetContractSize returns an instrument's contract size, a combo's or an expired instrument's included
func (g GlobalDerivativesAPI) GetContractSize(ctx context.Context, instrumentName string) (*DerivativesContractSizeResponse, error) {
	return g.getContractSize(ctx, false, instrumentName)
}

func (g GlobalDerivativesAPI) getContractSize(ctx context.Context, overWebsocket bool, instrumentName string) (*DerivativesContractSizeResponse, error) {
	if instrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp *DerivativesContractSizeResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_contract_size", &derivativesInstrumentNameParams{InstrumentName: instrumentName}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetCurrencies returns every currency the gateway supports and its deposit and withdrawal parameters
func (g GlobalDerivativesAPI) GetCurrencies(ctx context.Context) ([]DerivativesCurrency, error) {
	return g.getCurrencies(ctx, false)
}

func (g GlobalDerivativesAPI) getCurrencies(ctx context.Context, overWebsocket bool) ([]DerivativesCurrency, error) {
	var resp []DerivativesCurrency
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_currencies", nil, &resp)
}

// GetDeliveryPrices returns a page of an index's delivery prices, newest first, at most 100 a page, and the number of
// prices available
func (g GlobalDerivativesAPI) GetDeliveryPrices(ctx context.Context, req *DerivativesDeliveryPricesRequest) (*DerivativesDeliveryPricesResponse, error) {
	return g.getDeliveryPrices(ctx, false, req)
}

func (g GlobalDerivativesAPI) getDeliveryPrices(ctx context.Context, overWebsocket bool, req *DerivativesDeliveryPricesRequest) (*DerivativesDeliveryPricesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.IndexName == "" {
		return nil, errDerivativesIndexNameRequired
	}
	params := &derivativesDeliveryPricesParams{IndexName: req.IndexName, Offset: req.Offset, Count: req.Count}
	var resp *DerivativesDeliveryPricesResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_delivery_prices", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetExpirations returns the expiration labels of listed futures and options, such as 27NOV26 or PERPETUAL, not
// timestamps; an empty request merges every currency's labels
func (g GlobalDerivativesAPI) GetExpirations(ctx context.Context, req *DerivativesExpirationsRequest) (*DerivativesExpirationsResponse, error) {
	return g.getExpirations(ctx, false, req)
}

func (g GlobalDerivativesAPI) getExpirations(ctx context.Context, overWebsocket bool, req *DerivativesExpirationsRequest) (*DerivativesExpirationsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validateDerivativesMarketDataKind(req.Kind, DerivativesKindFuture, DerivativesKindOption, DerivativesKindAny); err != nil {
		return nil, err
	}
	params := &derivativesExpirationsParams{Kind: req.Kind, CurrencyPair: req.CurrencyPair}
	if !req.Currency.IsEmpty() {
		params.Currency = derivativesMarketDataCurrency(req.Currency)
	}
	var resp *DerivativesExpirationsResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_expirations", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetFundingChartData returns a perpetual's current funding and its 8 hour funding and index price over a period, a
// point a minute for 8 and 24 hours and a point an hour for a month
func (g GlobalDerivativesAPI) GetFundingChartData(ctx context.Context, req *DerivativesFundingChartDataRequest) (*DerivativesFundingChartDataResponse, error) {
	return g.getFundingChartData(ctx, false, req)
}

func (g GlobalDerivativesAPI) getFundingChartData(ctx context.Context, overWebsocket bool, req *DerivativesFundingChartDataRequest) (*DerivativesFundingChartDataResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	switch req.Length {
	case DerivativesFundingChartLength8Hour, DerivativesFundingChartLength24Hour, DerivativesFundingChartLengthOneMonth:
	default:
		return nil, fmt.Errorf("%w: %q", errDerivativesInvalidFundingChartLength, req.Length)
	}
	params := &derivativesFundingChartDataParams{InstrumentName: req.InstrumentName, Length: req.Length}
	var resp *DerivativesFundingChartDataResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_funding_chart_data", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetFundingRateHistory returns a perpetual's hourly funding over the hours ending in a time range, oldest first; a
// range of more than 744 hours returns its latest 744
func (g GlobalDerivativesAPI) GetFundingRateHistory(ctx context.Context, req *DerivativesInstrumentTimeRangeRequest) ([]DerivativesFundingRateHistory, error) {
	return g.getFundingRateHistory(ctx, false, req)
}

func (g GlobalDerivativesAPI) getFundingRateHistory(ctx context.Context, overWebsocket bool, req *DerivativesInstrumentTimeRangeRequest) ([]DerivativesFundingRateHistory, error) {
	params, err := req.params()
	if err != nil {
		return nil, err
	}
	var resp []DerivativesFundingRateHistory
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_funding_rate_history", params, &resp)
}

// GetFundingRateValue returns a perpetual's funding rate accumulated over a time range
func (g GlobalDerivativesAPI) GetFundingRateValue(ctx context.Context, req *DerivativesInstrumentTimeRangeRequest) (float64, error) {
	return g.getFundingRateValue(ctx, false, req)
}

func (g GlobalDerivativesAPI) getFundingRateValue(ctx context.Context, overWebsocket bool, req *DerivativesInstrumentTimeRangeRequest) (float64, error) {
	params, err := req.params()
	if err != nil {
		return 0, err
	}
	var resp *float64
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_funding_rate_value", params, &resp); err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, common.ErrNoResponse
	}
	return *resp, nil
}

// params validates an instrument time range request and returns its params
func (r *DerivativesInstrumentTimeRangeRequest) params() (*derivativesInstrumentTimeRangeParams, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := common.StartEndTimeCheck(r.StartTime, r.EndTime); err != nil {
		return nil, err
	}
	return &derivativesInstrumentTimeRangeParams{
		InstrumentName: r.InstrumentName,
		StartTimestamp: r.StartTime.UnixMilli(),
		EndTimestamp:   r.EndTime.UnixMilli(),
	}, nil
}

// GetHistoricalVolatility returns a currency's hourly realised volatility over about the last 16 days, oldest first;
// the gateway repeats the first point
func (g GlobalDerivativesAPI) GetHistoricalVolatility(ctx context.Context, ccy currency.Code) ([]DerivativesHistoricalVolatility, error) {
	return g.getHistoricalVolatility(ctx, false, ccy)
}

func (g GlobalDerivativesAPI) getHistoricalVolatility(ctx context.Context, overWebsocket bool, ccy currency.Code) ([]DerivativesHistoricalVolatility, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp []DerivativesHistoricalVolatility
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_historical_volatility", &derivativesCurrencyParams{Currency: derivativesMarketDataCurrency(ccy)}, &resp)
}

// GetIndexChartData returns an index's average price over a period, oldest first; the points are 6 seconds apart for an
// hour and 6 hours apart for all of the index's history
func (g GlobalDerivativesAPI) GetIndexChartData(ctx context.Context, req *DerivativesIndexChartDataRequest) ([]DerivativesIndexChartPoint, error) {
	return g.getIndexChartData(ctx, false, req)
}

func (g GlobalDerivativesAPI) getIndexChartData(ctx context.Context, overWebsocket bool, req *DerivativesIndexChartDataRequest) ([]DerivativesIndexChartPoint, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.IndexName == "" {
		return nil, errDerivativesIndexNameRequired
	}
	switch req.Range {
	case DerivativesIndexChartRangeOneHour, DerivativesIndexChartRangeOneDay, DerivativesIndexChartRangeTwoDays, DerivativesIndexChartRangeOneMonth, DerivativesIndexChartRangeOneYear, DerivativesIndexChartRangeAll:
	default:
		return nil, fmt.Errorf("%w: %q", errDerivativesInvalidIndexChartRange, req.Range)
	}
	var resp []DerivativesIndexChartPoint
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_index_chart_data", &derivativesIndexChartDataParams{IndexName: req.IndexName, Range: req.Range}, &resp)
}

// GetIndexPrice returns an index's current price and estimated delivery price; index names are lower case, such as
// btc_usdc
func (g GlobalDerivativesAPI) GetIndexPrice(ctx context.Context, indexName string) (*DerivativesIndexPriceResponse, error) {
	return g.getIndexPrice(ctx, false, indexName)
}

func (g GlobalDerivativesAPI) getIndexPrice(ctx context.Context, overWebsocket bool, indexName string) (*DerivativesIndexPriceResponse, error) {
	if indexName == "" {
		return nil, errDerivativesIndexNameRequired
	}
	var resp *DerivativesIndexPriceResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_index_price", &derivativesIndexNameParams{IndexName: indexName}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetIndexPriceNames returns the names of every price index and, when extended is set, whether future and option combos
// can be created on each
func (g GlobalDerivativesAPI) GetIndexPriceNames(ctx context.Context, extended bool) ([]DerivativesIndexPriceName, error) {
	return g.getIndexPriceNames(ctx, false, extended)
}

func (g GlobalDerivativesAPI) getIndexPriceNames(ctx context.Context, overWebsocket, extended bool) ([]DerivativesIndexPriceName, error) {
	var resp []DerivativesIndexPriceName
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_index_price_names", &derivativesMarketDataExtendedParams{Extended: extended}, &resp)
}

// GetInstrument returns an instrument's specification, a combo's or an expired instrument's included
func (g GlobalDerivativesAPI) GetInstrument(ctx context.Context, instrumentName string) (*DerivativesInstrument, error) {
	return g.getInstrument(ctx, false, instrumentName)
}

func (g GlobalDerivativesAPI) getInstrument(ctx context.Context, overWebsocket bool, instrumentName string) (*DerivativesInstrument, error) {
	if instrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp *DerivativesInstrument
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_instrument", &derivativesInstrumentNameParams{InstrumentName: instrumentName}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetInstruments returns a settlement currency's instruments, optionally of one kind; the gateway lists Deribit's whole
// catalogue, although Coinbase accounts only trade its perpetuals, and limits the method to a request a second
func (g GlobalDerivativesAPI) GetInstruments(ctx context.Context, req *DerivativesInstrumentsRequest) ([]DerivativesInstrument, error) {
	return g.getInstruments(ctx, false, req)
}

func (g GlobalDerivativesAPI) getInstruments(ctx context.Context, overWebsocket bool, req *DerivativesInstrumentsRequest) ([]DerivativesInstrument, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateDerivativesMarketDataKind(req.Kind, DerivativesKindFuture, DerivativesKindOption, DerivativesKindSpot, DerivativesKindFutureCombo, DerivativesKindOptionCombo, DerivativesKindAny); err != nil {
		return nil, err
	}
	params := &derivativesInstrumentsParams{
		Currency: derivativesMarketDataCurrency(req.Currency),
		Kind:     req.Kind,
		Expired:  req.Expired,
		Extended: req.Extended,
	}
	var resp []DerivativesInstrument
	return resp, g.sendRequest(ctx, overWebsocket, derivativesGetInstrumentsEPL, "public/get_instruments", params, &resp)
}

// GetLastSettlementsByCurrency returns a page of a settlement currency's settlement, delivery and bankruptcy events,
// newest first; profit and loss figures are aggregated across every position holder
func (g GlobalDerivativesAPI) GetLastSettlementsByCurrency(ctx context.Context, req *DerivativesLastSettlementsByCurrencyRequest) (*DerivativesSettlementsResponse, error) {
	return g.getLastSettlementsByCurrency(ctx, false, req)
}

func (g GlobalDerivativesAPI) getLastSettlementsByCurrency(ctx context.Context, overWebsocket bool, req *DerivativesLastSettlementsByCurrencyRequest) (*DerivativesSettlementsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateDerivativesLastSettlementsType(req.Type); err != nil {
		return nil, err
	}
	if err := validateDerivativesMarketDataCount(req.Count); err != nil {
		return nil, err
	}
	params := &derivativesLastSettlementsByCurrencyParams{
		Currency:             derivativesMarketDataCurrency(req.Currency),
		Type:                 req.Type,
		Count:                req.Count,
		Continuation:         req.Continuation,
		SearchStartTimestamp: derivativesMarketDataUnixMilli(req.SearchStartTime),
	}
	var resp *DerivativesSettlementsResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_last_settlements_by_currency", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLastSettlementsByInstrument returns a page of an instrument's settlement, delivery and bankruptcy events, newest
// first; profit and loss figures are aggregated across every position holder
func (g GlobalDerivativesAPI) GetLastSettlementsByInstrument(ctx context.Context, req *DerivativesLastSettlementsByInstrumentRequest) (*DerivativesSettlementsResponse, error) {
	return g.getLastSettlementsByInstrument(ctx, false, req)
}

func (g GlobalDerivativesAPI) getLastSettlementsByInstrument(ctx context.Context, overWebsocket bool, req *DerivativesLastSettlementsByInstrumentRequest) (*DerivativesSettlementsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateDerivativesLastSettlementsType(req.Type); err != nil {
		return nil, err
	}
	if err := validateDerivativesMarketDataCount(req.Count); err != nil {
		return nil, err
	}
	params := &derivativesLastSettlementsByInstrumentParams{
		InstrumentName:       req.InstrumentName,
		Type:                 req.Type,
		Count:                req.Count,
		Continuation:         req.Continuation,
		SearchStartTimestamp: derivativesMarketDataUnixMilli(req.SearchStartTime),
	}
	var resp *DerivativesSettlementsResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_last_settlements_by_instrument", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLastTradesByCurrency returns a settlement currency's latest trades, optionally within a trade ID or time range;
// the gateway only serves the last 24 hours of trades
func (g GlobalDerivativesAPI) GetLastTradesByCurrency(ctx context.Context, req *DerivativesLastTradesByCurrencyRequest) (*DerivativesLastTradesResponse, error) {
	return g.getLastTradesByCurrency(ctx, false, req)
}

func (g GlobalDerivativesAPI) getLastTradesByCurrency(ctx context.Context, overWebsocket bool, req *DerivativesLastTradesByCurrencyRequest) (*DerivativesLastTradesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateDerivativesMarketDataKind(req.Kind, DerivativesKindFuture, DerivativesKindOption, DerivativesKindSpot, DerivativesKindFutureCombo, DerivativesKindOptionCombo, DerivativesKindCombo, DerivativesKindAny); err != nil {
		return nil, err
	}
	if err := validateDerivativesLastTrades(req.Count, req.Sorting); err != nil {
		return nil, err
	}
	if err := checkWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := &derivativesLastTradesByCurrencyParams{
		Currency:       derivativesMarketDataCurrency(req.Currency),
		Kind:           req.Kind,
		StartID:        req.StartID,
		EndID:          req.EndID,
		StartTimestamp: derivativesMarketDataUnixMilli(req.StartTime),
		EndTimestamp:   derivativesMarketDataUnixMilli(req.EndTime),
		Count:          req.Count,
		Sorting:        req.Sorting,
	}
	var resp *DerivativesLastTradesResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_last_trades_by_currency", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLastTradesByCurrencyAndTime returns a settlement currency's latest trades within a time range, which needs a start
// or an end time; the gateway only serves the last 24 hours of trades
func (g GlobalDerivativesAPI) GetLastTradesByCurrencyAndTime(ctx context.Context, req *DerivativesLastTradesByCurrencyAndTimeRequest) (*DerivativesLastTradesResponse, error) {
	return g.getLastTradesByCurrencyAndTime(ctx, false, req)
}

func (g GlobalDerivativesAPI) getLastTradesByCurrencyAndTime(ctx context.Context, overWebsocket bool, req *DerivativesLastTradesByCurrencyAndTimeRequest) (*DerivativesLastTradesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateDerivativesMarketDataKind(req.Kind, DerivativesKindFuture, DerivativesKindOption, DerivativesKindSpot, DerivativesKindFutureCombo, DerivativesKindOptionCombo, DerivativesKindCombo, DerivativesKindAny); err != nil {
		return nil, err
	}
	if err := validateDerivativesLastTrades(req.Count, req.Sorting); err != nil {
		return nil, err
	}
	if err := checkDerivativesLastTradesWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := &derivativesLastTradesByCurrencyAndTimeParams{
		Currency:       derivativesMarketDataCurrency(req.Currency),
		Kind:           req.Kind,
		StartTimestamp: derivativesMarketDataUnixMilli(req.StartTime),
		EndTimestamp:   derivativesMarketDataUnixMilli(req.EndTime),
		Count:          req.Count,
		Sorting:        req.Sorting,
	}
	var resp *DerivativesLastTradesResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_last_trades_by_currency_and_time", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLastTradesByInstrument returns an instrument's latest trades, optionally within a trade sequence or time range;
// the gateway only serves the last 24 hours of trades
func (g GlobalDerivativesAPI) GetLastTradesByInstrument(ctx context.Context, req *DerivativesLastTradesByInstrumentRequest) (*DerivativesLastTradesResponse, error) {
	return g.getLastTradesByInstrument(ctx, false, req)
}

func (g GlobalDerivativesAPI) getLastTradesByInstrument(ctx context.Context, overWebsocket bool, req *DerivativesLastTradesByInstrumentRequest) (*DerivativesLastTradesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateDerivativesLastTrades(req.Count, req.Sorting); err != nil {
		return nil, err
	}
	if err := checkWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := &derivativesLastTradesByInstrumentParams{
		InstrumentName: req.InstrumentName,
		StartSequence:  req.StartSequence,
		EndSequence:    req.EndSequence,
		StartTimestamp: derivativesMarketDataUnixMilli(req.StartTime),
		EndTimestamp:   derivativesMarketDataUnixMilli(req.EndTime),
		Count:          req.Count,
		Sorting:        req.Sorting,
	}
	var resp *DerivativesLastTradesResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_last_trades_by_instrument", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLastTradesByInstrumentAndTime returns an instrument's latest trades within a time range, which needs a start or an
// end time; the gateway only serves the last 24 hours of trades
func (g GlobalDerivativesAPI) GetLastTradesByInstrumentAndTime(ctx context.Context, req *DerivativesLastTradesByInstrumentAndTimeRequest) (*DerivativesLastTradesResponse, error) {
	return g.getLastTradesByInstrumentAndTime(ctx, false, req)
}

func (g GlobalDerivativesAPI) getLastTradesByInstrumentAndTime(ctx context.Context, overWebsocket bool, req *DerivativesLastTradesByInstrumentAndTimeRequest) (*DerivativesLastTradesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateDerivativesLastTrades(req.Count, req.Sorting); err != nil {
		return nil, err
	}
	if err := checkDerivativesLastTradesWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := &derivativesLastTradesByInstrumentAndTimeParams{
		InstrumentName: req.InstrumentName,
		StartTimestamp: derivativesMarketDataUnixMilli(req.StartTime),
		EndTimestamp:   derivativesMarketDataUnixMilli(req.EndTime),
		Count:          req.Count,
		Sorting:        req.Sorting,
	}
	var resp *DerivativesLastTradesResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_last_trades_by_instrument_and_time", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetMarkPriceHistory returns an option's mark prices within a time range, oldest first; the gateway keeps about the
// last 5 minutes of history, and only for the options its volatility index is calculated from, so other instruments
// return none
func (g GlobalDerivativesAPI) GetMarkPriceHistory(ctx context.Context, req *DerivativesInstrumentTimeRangeRequest) ([]DerivativesMarkPrice, error) {
	return g.getMarkPriceHistory(ctx, false, req)
}

func (g GlobalDerivativesAPI) getMarkPriceHistory(ctx context.Context, overWebsocket bool, req *DerivativesInstrumentTimeRangeRequest) ([]DerivativesMarkPrice, error) {
	params, err := req.params()
	if err != nil {
		return nil, err
	}
	var resp []DerivativesMarkPrice
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_mark_price_history", params, &resp)
}

// GetOrderBook returns an instrument's order book, up to a depth on each side, and its ticker
func (g GlobalDerivativesAPI) GetOrderBook(ctx context.Context, req *DerivativesOrderBookRequest) (*DerivativesOrderBook, error) {
	return g.getOrderBook(ctx, false, req)
}

func (g GlobalDerivativesAPI) getOrderBook(ctx context.Context, overWebsocket bool, req *DerivativesOrderBookRequest) (*DerivativesOrderBook, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateDerivativesOrderBookDepth(req.Depth); err != nil {
		return nil, err
	}
	var resp *DerivativesOrderBook
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_order_book", &derivativesOrderBookParams{InstrumentName: req.InstrumentName, Depth: req.Depth}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOrderBookByInstrumentID returns the order book of an instrument identified by its ID, up to a depth on each side,
// and its ticker
func (g GlobalDerivativesAPI) GetOrderBookByInstrumentID(ctx context.Context, req *DerivativesOrderBookByInstrumentIDRequest) (*DerivativesOrderBook, error) {
	return g.getOrderBookByInstrumentID(ctx, false, req)
}

func (g GlobalDerivativesAPI) getOrderBookByInstrumentID(ctx context.Context, overWebsocket bool, req *DerivativesOrderBookByInstrumentIDRequest) (*DerivativesOrderBook, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentID == 0 {
		return nil, errDerivativesInstrumentIDRequired
	}
	if err := validateDerivativesOrderBookDepth(req.Depth); err != nil {
		return nil, err
	}
	var resp *DerivativesOrderBook
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_order_book_by_instrument_id", &derivativesOrderBookByInstrumentIDParams{InstrumentID: req.InstrumentID, Depth: req.Depth}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validateDerivativesOrderBookDepth checks an order book depth is at most the 10000 levels the gateway allows
func validateDerivativesOrderBookDepth(depth uint64) error {
	if depth > derivativesMaximumOrderBookDepth {
		return fmt.Errorf("%w: %d levels, at most %d", errDerivativesInvalidDepth, depth, derivativesMaximumOrderBookDepth)
	}
	return nil
}

// GetSupportedIndexNames returns the names of the price indexes of a type, all, spot or derivative, or of every type
// when indexType is empty; the docs describe objects, but the gateway sends the names alone
func (g GlobalDerivativesAPI) GetSupportedIndexNames(ctx context.Context, indexType string) ([]string, error) {
	return g.getSupportedIndexNames(ctx, false, indexType)
}

func (g GlobalDerivativesAPI) getSupportedIndexNames(ctx context.Context, overWebsocket bool, indexType string) ([]string, error) {
	switch indexType {
	case "", DerivativesIndexTypeAll, DerivativesIndexTypeSpot, DerivativesIndexTypeDerivative:
	default:
		return nil, fmt.Errorf("%w: %q", errDerivativesInvalidIndexType, indexType)
	}
	var resp []string
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_supported_index_names", &derivativesSupportedIndexNamesParams{Type: indexType}, &resp)
}

// GetTradeVolumes returns each currency's traded volumes over 24 hours and, when extended is set, 7 and 30 days
func (g GlobalDerivativesAPI) GetTradeVolumes(ctx context.Context, extended bool) ([]DerivativesTradeVolume, error) {
	return g.getTradeVolumes(ctx, false, extended)
}

func (g GlobalDerivativesAPI) getTradeVolumes(ctx context.Context, overWebsocket, extended bool) ([]DerivativesTradeVolume, error) {
	var resp []DerivativesTradeVolume
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_trade_volumes", &derivativesMarketDataExtendedParams{Extended: extended}, &resp)
}

// GetTradingViewChartData returns an instrument's candles opening within a time range, as TradingView charts take them;
// a range of more than 5000 candles returns its latest. Over HTTP it is sent as a GET request, as the gateway answers a
// posted JSON-RPC body with Method not found
func (g GlobalDerivativesAPI) GetTradingViewChartData(ctx context.Context, req *DerivativesTradingViewChartDataRequest) (*DerivativesTradingViewChartDataResponse, error) {
	return g.getTradingViewChartData(ctx, false, req)
}

func (g GlobalDerivativesAPI) getTradingViewChartData(ctx context.Context, overWebsocket bool, req *DerivativesTradingViewChartDataRequest) (*DerivativesTradingViewChartDataResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	resolution, err := derivativesChartResolution(req.Resolution)
	if err != nil {
		return nil, err
	}
	params := &derivativesTradingViewChartDataParams{
		InstrumentName: req.InstrumentName,
		StartTimestamp: req.StartTime.UnixMilli(),
		EndTimestamp:   req.EndTime.UnixMilli(),
		Resolution:     resolution,
	}
	var resp *DerivativesTradingViewChartDataResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_tradingview_chart_data", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// derivativesChartResolution returns the gateway's TradingView chart resolution, in minutes or 1D, for a candle interval
func derivativesChartResolution(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin, kline.ThreeMin, kline.FiveMin, kline.TenMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour, kline.ThreeHour, kline.SixHour, kline.TwelveHour:
		return strconv.FormatInt(int64(interval.Duration()/time.Minute), 10), nil
	case kline.OneDay:
		return "1D", nil
	}
	return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
}

// GetVolatilityIndexData returns up to 1000 of a currency's volatility index candles, the latest within a time range,
// oldest first; the response's Continuation is the end time of the next, earlier page
func (g GlobalDerivativesAPI) GetVolatilityIndexData(ctx context.Context, req *DerivativesVolatilityIndexDataRequest) (*DerivativesVolatilityIndexDataResponse, error) {
	return g.getVolatilityIndexData(ctx, false, req)
}

func (g GlobalDerivativesAPI) getVolatilityIndexData(ctx context.Context, overWebsocket bool, req *DerivativesVolatilityIndexDataRequest) (*DerivativesVolatilityIndexDataResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	resolution, err := derivativesVolatilityIndexResolution(req.Resolution)
	if err != nil {
		return nil, err
	}
	params := &derivativesVolatilityIndexDataParams{
		Currency:       derivativesMarketDataCurrency(req.Currency),
		StartTimestamp: req.StartTime.UnixMilli(),
		EndTimestamp:   req.EndTime.UnixMilli(),
		Resolution:     resolution,
	}
	var resp *DerivativesVolatilityIndexDataResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_volatility_index_data", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// derivativesVolatilityIndexResolution returns the gateway's volatility index resolution, in seconds or 1D, for a
// candle interval
func derivativesVolatilityIndexResolution(interval kline.Interval) (string, error) {
	switch interval {
	case kline.ThousandMilliseconds, kline.OneMin, kline.OneHour, kline.TwelveHour:
		return strconv.FormatInt(int64(interval.Duration()/time.Second), 10), nil
	case kline.OneDay:
		return "1D", nil
	}
	return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
}

// GetTicker returns an instrument's ticker: its best prices, mark and index prices, 24 hour statistics and, by kind,
// greeks and implied volatilities, funding or combo prices
func (g GlobalDerivativesAPI) GetTicker(ctx context.Context, instrumentName string) (*DerivativesTicker, error) {
	return g.getTicker(ctx, false, instrumentName)
}

func (g GlobalDerivativesAPI) getTicker(ctx context.Context, overWebsocket bool, instrumentName string) (*DerivativesTicker, error) {
	if instrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp *DerivativesTicker
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/ticker", &derivativesInstrumentNameParams{InstrumentName: instrumentName}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTickersByCurrency returns the tickers of a settlement currency's open instruments of the kinds requested; at a
// depth of 3 or 10 each ticker also holds that many levels of its book, which a depth of 1 leaves out
func (g GlobalDerivativesAPI) GetTickersByCurrency(ctx context.Context, req *DerivativesTickersByCurrencyRequest) ([]DerivativesOrderBook, error) {
	return g.getTickersByCurrency(ctx, false, req)
}

func (g GlobalDerivativesAPI) getTickersByCurrency(ctx context.Context, overWebsocket bool, req *DerivativesTickersByCurrencyRequest) ([]DerivativesOrderBook, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	for _, kind := range req.Kinds {
		if kind == "" {
			return nil, fmt.Errorf("%w: empty kind", errDerivativesInvalidMarketDataKind)
		}
		if err := validateDerivativesMarketDataKind(kind, DerivativesKindFuture, DerivativesKindOption, DerivativesKindSpot, DerivativesKindFutureCombo, DerivativesKindOptionCombo, DerivativesKindAny); err != nil {
			return nil, err
		}
	}
	switch req.Depth {
	case 0, 1, 3, 10:
	default:
		return nil, fmt.Errorf("%w: %d levels, want 1, 3 or 10", errDerivativesInvalidDepth, req.Depth)
	}
	params := &derivativesTickersByCurrencyParams{Currency: derivativesMarketDataCurrency(req.Currency), Kinds: req.Kinds, Depth: req.Depth}
	var resp []DerivativesOrderBook
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/tickers_by_currency", params, &resp)
}

// derivativesMarketDataCurrency returns a currency as the gateway takes it: upper case, as the docs give currencies,
// except any and grouped, which the gateway only accepts in lower case
func derivativesMarketDataCurrency(ccy currency.Code) string {
	if ccy.Equal(currency.ANY) || strings.EqualFold(ccy.String(), DerivativesCurrencyGrouped) {
		return ccy.Lower().String()
	}
	return ccy.Upper().String()
}

// validateDerivativesMarketDataKind checks a kind filter is unset, which the gateway defaults, or one a method accepts
func validateDerivativesMarketDataKind(kind string, accepted ...string) error {
	if kind == "" || slices.Contains(accepted, kind) {
		return nil
	}
	return fmt.Errorf("%w: %q", errDerivativesInvalidMarketDataKind, kind)
}

// validateDerivativesLastSettlementsType checks a settlement type filter is unset or one the gateway filters by; the
// exercise of options can only be returned unfiltered
func validateDerivativesLastSettlementsType(settlementType string) error {
	switch settlementType {
	case "", DerivativesSettlementTypeSettlement, DerivativesSettlementTypeDelivery, DerivativesSettlementTypeBankruptcy:
		return nil
	}
	return fmt.Errorf("%w: %q", errDerivativesInvalidLastSettlementsType, settlementType)
}

// validateDerivativesLastTrades checks a trades request's count is within the gateway's maximum and its sorting is unset
// or one the docs name, as the gateway ignores any other
func validateDerivativesLastTrades(count uint64, sorting string) error {
	if err := validateDerivativesMarketDataCount(count); err != nil {
		return err
	}
	switch sorting {
	case "", DerivativesSortingAscending, DerivativesSortingDescending, DerivativesSortingDefault:
		return nil
	}
	return fmt.Errorf("%w: %q", errDerivativesInvalidLastTradesSorting, sorting)
}

// validateDerivativesMarketDataCount checks a count is at most the 1000 records the gateway returns
func validateDerivativesMarketDataCount(count uint64) error {
	if count > derivativesMarketDataMaximumCount {
		return fmt.Errorf("%w: count %d, at most %d", errInvalidLimit, count, derivativesMarketDataMaximumCount)
	}
	return nil
}

// checkDerivativesLastTradesWindow checks a trades window has a start or an end, either of which the gateway takes
// alone, and does not start after it ends
func checkDerivativesLastTradesWindow(start, end time.Time) error {
	if start.IsZero() && end.IsZero() {
		return errDerivativesLastTradesWindowRequired
	}
	return checkWindow(start, end)
}

// derivativesMarketDataUnixMilli returns a time in Unix milliseconds, or zero for an unset time, which leaves an optional
// timestamp out of a request
func derivativesMarketDataUnixMilli(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
