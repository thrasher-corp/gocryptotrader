package mexc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// futuresDefaultPageSizeLimit is the page size cap of most paged futures endpoints
	futuresDefaultPageSizeLimit = 100
	// futuresFundingRateHistoryPageSizeLimit is the page size cap of the funding rate history
	futuresFundingRateHistoryPageSizeLimit = 1000
	// futuresRecentTradesLimit is the most trades a recent trades request returns
	futuresRecentTradesLimit = 100
	// futuresDepthSnapshotsLimit is the most order book changes a depth snapshots request returns
	futuresDepthSnapshotsLimit = 1000
)

var futuresIntervals = []struct {
	interval kline.Interval
	value    string
}{
	{kline.OneMin, "Min1"},
	{kline.FiveMin, "Min5"},
	{kline.FifteenMin, "Min15"},
	{kline.ThirtyMin, "Min30"},
	{kline.OneHour, "Min60"},
	{kline.FourHour, "Hour4"},
	{kline.EightHour, "Hour8"},
	{kline.OneDay, "Day1"},
	{kline.OneWeek, "Week1"},
	{kline.OneMonth, "Month1"},
}

// futuresIntervalString returns the futures API's name for a candle interval
func futuresIntervalString(interval kline.Interval) (string, error) {
	for _, i := range futuresIntervals {
		if i.interval == interval {
			return i.value, nil
		}
	}
	return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
}

// futuresIntervalFromString returns the candle interval the futures API names
func futuresIntervalFromString(value string) (kline.Interval, error) {
	for _, i := range futuresIntervals {
		if i.value == value {
			return i.interval, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", kline.ErrInvalidInterval, value)
}

// setFuturesPage sets the paging parameters of a paged futures request after checking the page size against the
// endpoint's cap; an unset page asks for the first page and an unset size leaves the venue's default of 20
func setFuturesPage(params url.Values, page *FuturesPageRequest, sizeLimit uint64) error {
	pageNumber, pageSize := uint64(1), uint64(0)
	if page != nil {
		pageNumber, pageSize = max(page.PageNumber, 1), page.PageSize
	}
	if pageSize > sizeLimit {
		return fmt.Errorf("%w: page size %d, at most %d", errInvalidLimit, pageSize, sizeLimit)
	}
	params.Set("page_num", strconv.FormatUint(pageNumber, 10))
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatUint(pageSize, 10))
	}
	return nil
}

// GetFuturesServerTime returns the futures API's server time
func (e *Exchange) GetFuturesServerTime(ctx context.Context) (time.Time, error) {
	var resp types.Time
	if err := e.SendFuturesHTTPRequest(ctx, futuresServerTimeEPL, http.MethodGet, "contract/ping", nil, nil, &resp, false); err != nil {
		return time.Time{}, err
	}
	return resp.Time(), nil
}

// GetFuturesContracts returns the details of every contract, or of one when symbol is set
func (e *Exchange) GetFuturesContracts(ctx context.Context, symbol currency.Pair) ([]*FuturesContractResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp oneOrMany[*FuturesContractResponse]
	if err := e.SendFuturesHTTPRequest(ctx, futuresContractsEPL, http.MethodGet, "contract/detail/country", params, nil, &resp, false); err != nil {
		return nil, err
	}
	e.storeFuturesContracts(resp)
	return resp, nil
}

// GetFuturesTransferableCurrencies returns the currencies which can be transferred to and from the futures account
func (e *Exchange) GetFuturesTransferableCurrencies(ctx context.Context) ([]currency.Code, error) {
	var resp []currency.Code
	return resp, e.SendFuturesHTTPRequest(ctx, futuresTransferableCurrenciesEPL, http.MethodGet, "contract/support_currencies", nil, nil, &resp, false)
}

// GetFuturesOrderbook returns a contract's order book. limit sets the number of levels on each side; 0 leaves the
// venue's default
func (e *Exchange) GetFuturesOrderbook(ctx context.Context, symbol currency.Pair, limit uint64) (*FuturesOrderbookResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *FuturesOrderbookResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOrderbookEPL, http.MethodGet, "contract/depth/"+symbol.String(), params, nil, &resp, false)
}

// GetFuturesDepthSnapshots returns a contract's latest order book changes, newest first, one per version
func (e *Exchange) GetFuturesDepthSnapshots(ctx context.Context, symbol currency.Pair, limit uint64) ([]*FuturesDepthSnapshotResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if limit == 0 || limit > futuresDepthSnapshotsLimit {
		return nil, fmt.Errorf("%w: %d, must be between 1 and %d", errInvalidLimit, limit, futuresDepthSnapshotsLimit)
	}
	var resp []*FuturesDepthSnapshotResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresDepthSnapshotsEPL, http.MethodGet, "contract/depth_commits/"+symbol.String()+"/"+strconv.FormatUint(limit, 10), nil, nil, &resp, false)
}

// GetFuturesIndexPrice returns a contract's index price
func (e *Exchange) GetFuturesIndexPrice(ctx context.Context, symbol currency.Pair) (*FuturesIndexPriceResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesIndexPriceResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresIndexPriceEPL, http.MethodGet, "contract/index_price/"+symbol.String(), nil, nil, &resp, false)
}

// GetFuturesFairPrice returns a contract's fair price
func (e *Exchange) GetFuturesFairPrice(ctx context.Context, symbol currency.Pair) (*FuturesFairPriceResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesFairPriceResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresFairPriceEPL, http.MethodGet, "contract/fair_price/"+symbol.String(), nil, nil, &resp, false)
}

// GetFuturesFundingRates returns the current funding rate of a contract, or of every contract when symbol is unset
func (e *Exchange) GetFuturesFundingRates(ctx context.Context, symbol currency.Pair) ([]*FuturesFundingRateResponse, error) {
	path := "contract/funding_rate"
	if !symbol.IsEmpty() {
		path += "/" + symbol.String()
	}
	var resp oneOrMany[*FuturesFundingRateResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresFundingRateEPL, http.MethodGet, path, nil, nil, &resp, false)
}

// GetFuturesCandles returns a contract's candles. At most 2000 are returned: from the start time when only it is set,
// and the latest ones up to the end time otherwise
func (e *Exchange) GetFuturesCandles(ctx context.Context, req *FuturesCandlesRequest) (*FuturesCandlesResponse, error) {
	return e.getFuturesCandles(ctx, futuresCandlesEPL, "contract/kline/", req)
}

// GetFuturesIndexPriceCandles returns a contract's index price candles, as GetFuturesCandles does
func (e *Exchange) GetFuturesIndexPriceCandles(ctx context.Context, req *FuturesCandlesRequest) (*FuturesCandlesResponse, error) {
	return e.getFuturesCandles(ctx, futuresIndexPriceCandlesEPL, "contract/kline/index_price/", req)
}

// GetFuturesFairPriceCandles returns a contract's fair price candles, as GetFuturesCandles does
func (e *Exchange) GetFuturesFairPriceCandles(ctx context.Context, req *FuturesCandlesRequest) (*FuturesCandlesResponse, error) {
	return e.getFuturesCandles(ctx, futuresFairPriceCandlesEPL, "contract/kline/fair_price/", req)
}

func (e *Exchange) getFuturesCandles(ctx context.Context, epl request.EndpointLimit, path string, req *FuturesCandlesRequest) (*FuturesCandlesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.Interval != 0 {
		interval, err := futuresIntervalString(req.Interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", interval)
	}
	if !req.StartTime.IsZero() {
		params.Set("start", strconv.FormatInt(req.StartTime.Unix(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("end", strconv.FormatInt(req.EndTime.Unix(), 10))
	}
	var resp *FuturesCandlesResponse
	return resp, e.SendFuturesHTTPRequest(ctx, epl, http.MethodGet, path+req.Symbol.String(), params, nil, &resp, false)
}

// GetFuturesRecentTrades returns a contract's latest trades, newest first; limit is at most 100 and 0 asks for 100
func (e *Exchange) GetFuturesRecentTrades(ctx context.Context, symbol currency.Pair, limit uint64) ([]*FuturesTradeResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if limit > futuresRecentTradesLimit {
		return nil, fmt.Errorf("%w: %d, at most %d", errInvalidLimit, limit, futuresRecentTradesLimit)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesTradeResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresRecentTradesEPL, http.MethodGet, "contract/deals/"+symbol.String(), params, nil, &resp, false)
}

// GetFuturesTickers returns the ticker of a contract, or of every contract when symbol is unset
func (e *Exchange) GetFuturesTickers(ctx context.Context, symbol currency.Pair) ([]*FuturesTickerResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp oneOrMany[*FuturesTickerResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresTickersEPL, http.MethodGet, "contract/ticker", params, nil, &resp, false)
}

// GetFuturesInsuranceFundBalances returns the insurance fund balance of a contract, or of every contract when symbol
// is unset
func (e *Exchange) GetFuturesInsuranceFundBalances(ctx context.Context, symbol currency.Pair) ([]*FuturesInsuranceFundResponse, error) {
	path := "contract/risk_reverse"
	if !symbol.IsEmpty() {
		path += "/" + symbol.String()
	}
	var resp oneOrMany[*FuturesInsuranceFundResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresInsuranceFundEPL, http.MethodGet, path, nil, nil, &resp, false)
}

// GetFuturesInsuranceFundBalanceHistory returns a page of a contract's insurance fund balance snapshots, newest
// first; the page size is at most 100
func (e *Exchange) GetFuturesInsuranceFundBalanceHistory(ctx context.Context, symbol currency.Pair, page *FuturesPageRequest) (*FuturesPageResponse[FuturesInsuranceFundSnapshotResponse], error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if err := setFuturesPage(params, page, futuresDefaultPageSizeLimit); err != nil {
		return nil, err
	}
	var resp *FuturesPageResponse[FuturesInsuranceFundSnapshotResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresInsuranceFundHistoryEPL, http.MethodGet, "contract/risk_reverse/history", params, nil, &resp, false)
}

// GetFuturesFundingRateHistory returns a page of a contract's settled funding rates, newest first; the page size is
// at most 1000
func (e *Exchange) GetFuturesFundingRateHistory(ctx context.Context, symbol currency.Pair, page *FuturesPageRequest) (*FuturesPageResponse[FuturesFundingRateHistoryResponse], error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if err := setFuturesPage(params, page, futuresFundingRateHistoryPageSizeLimit); err != nil {
		return nil, err
	}
	var resp *FuturesPageResponse[FuturesFundingRateHistoryResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresFundingRateHistoryEPL, http.MethodGet, "contract/funding_rate/history", params, nil, &resp, false)
}

// storeFuturesContracts records the details of contracts read, by symbol
func (e *Exchange) storeFuturesContracts(contracts []*FuturesContractResponse) {
	e.futuresContractsMu.Lock()
	defer e.futuresContractsMu.Unlock()
	if e.futuresContracts == nil {
		e.futuresContracts = make(map[string]*FuturesContractResponse, len(contracts))
	}
	for _, c := range contracts {
		if c != nil && c.Symbol != "" {
			e.futuresContracts[c.Symbol] = c
		}
	}
}

// futuresContract returns the details last read for a contract, reading every contract's when it is not yet known
func (e *Exchange) futuresContract(ctx context.Context, symbol string) (*FuturesContractResponse, error) {
	e.futuresContractsMu.RLock()
	c, ok := e.futuresContracts[symbol]
	e.futuresContractsMu.RUnlock()
	if ok {
		return c, nil
	}
	if _, err := e.GetFuturesContracts(ctx, currency.EMPTYPAIR); err != nil {
		return nil, err
	}
	e.futuresContractsMu.RLock()
	defer e.futuresContractsMu.RUnlock()
	if c, ok = e.futuresContracts[symbol]; !ok {
		return nil, fmt.Errorf("%w: %s", errContractNotFound, symbol)
	}
	return c, nil
}
