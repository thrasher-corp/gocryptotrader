package kraken

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// futuresChartIntervals are the intervals the charts API serves candles and analytics in, mapped to the resolution
// candles are requested by
var futuresChartIntervals = map[kline.Interval]string{
	kline.OneMin:     "1m",
	kline.FiveMin:    "5m",
	kline.FifteenMin: "15m",
	kline.ThirtyMin:  "30m",
	kline.OneHour:    "1h",
	kline.FourHour:   "4h",
	kline.TwelveHour: "12h",
	kline.OneDay:     "1d",
	kline.OneWeek:    "1w",
}

// GetFuturesTickTypes calls Tick Types, returning the tick types candles are served for
func (e *Exchange) GetFuturesTickTypes(ctx context.Context) ([]string, error) {
	var resp []string
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFuturesSupplementary, "/charts/v1/", &resp)
}

// GetFuturesChartMarkets calls Markets, returning the symbols of the markets candles of a tick type are served for
func (e *Exchange) GetFuturesChartMarkets(ctx context.Context, tickType string) ([]string, error) {
	if tickType == "" {
		return nil, errTickTypeEmpty
	}
	var resp []string
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFuturesSupplementary, "/charts/v1/"+url.PathEscape(tickType), &resp)
}

// GetFuturesChartResolutions calls Resolutions, returning the resolutions a market's candles of a tick type are served
// in, such as 1m or 1d
func (e *Exchange) GetFuturesChartResolutions(ctx context.Context, tickType string, pair currency.Pair) ([]string, error) {
	if tickType == "" {
		return nil, errTickTypeEmpty
	}
	symbol, err := e.futuresMarketSymbol(pair)
	if err != nil {
		return nil, err
	}
	var resp []string
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFuturesSupplementary, "/charts/v1/"+url.PathEscape(tickType)+"/"+url.PathEscape(symbol), &resp)
}

// GetFuturesCandles calls Market Candles, returning a market's candles from From, or the most recent ones up to To or
// now. A response holds up to Count candles, 2000 when it is 0, and an unknown market returns none
func (e *Exchange) GetFuturesCandles(ctx context.Context, req *FuturesCandlesRequest) (*FuturesCandlesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TickType == "" {
		return nil, errTickTypeEmpty
	}
	symbol, err := e.futuresMarketSymbol(req.Pair)
	if err != nil {
		return nil, err
	}
	resolution, ok := futuresChartIntervals[req.Interval]
	if !ok {
		return nil, fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, req.Interval)
	}
	if !req.From.IsZero() && !req.To.IsZero() && req.From.After(req.To) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if !req.From.IsZero() {
		params.Set("from", strconv.FormatInt(req.From.Unix(), 10))
	}
	if !req.To.IsZero() {
		params.Set("to", strconv.FormatInt(req.To.Unix(), 10))
	}
	if req.Count != 0 {
		params.Set("count", strconv.FormatUint(req.Count, 10))
	}
	path := "/charts/v1/" + url.PathEscape(req.TickType) + "/" + url.PathEscape(symbol) + "/" + resolution
	var resp *FuturesCandlesResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFuturesSupplementary, common.EncodeURLValues(path, params), &resp)
}

// GetFuturesLiquidityPoolStatistic calls Get liquidity pool statistic, returning the liquidity pool's USD value in
// time buckets from Since
func (e *Exchange) GetFuturesLiquidityPoolStatistic(ctx context.Context, req *FuturesLiquidityPoolStatisticRequest) (*FuturesAnalyticsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := futuresAnalyticsParams(req.Since, req.To, req.Interval)
	if err != nil {
		return nil, err
	}
	return e.futuresAnalytics(ctx, common.EncodeURLValues("/charts/v1/analytics/liquidity-pool", params))
}

// GetFuturesMarketAnalytics calls Market Analytics, returning a market's statistic in time buckets from Since. An
// unknown market returns no buckets
func (e *Exchange) GetFuturesMarketAnalytics(ctx context.Context, req *FuturesMarketAnalyticsRequest) (*FuturesAnalyticsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.futuresMarketSymbol(req.Pair)
	if err != nil {
		return nil, err
	}
	if req.AnalyticsType == "" {
		return nil, errAnalyticsTypeEmpty
	}
	params, err := futuresAnalyticsParams(req.Since, req.To, req.Interval)
	if err != nil {
		return nil, err
	}
	path := "/charts/v1/analytics/" + url.PathEscape(symbol) + "/" + url.PathEscape(req.AnalyticsType)
	return e.futuresAnalytics(ctx, common.EncodeURLValues(path, params))
}

// futuresAnalyticsParams returns the parameters the analytics endpoints take
func futuresAnalyticsParams(since, to time.Time, interval kline.Interval) (url.Values, error) {
	if since.IsZero() {
		return nil, fmt.Errorf("%w: since", common.ErrDateUnset)
	}
	if _, ok := futuresChartIntervals[interval]; !ok {
		return nil, fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
	}
	if !to.IsZero() && since.After(to) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("since", strconv.FormatInt(since.Unix(), 10))
	params.Set("interval", strconv.FormatInt(int64(interval.Duration().Seconds()), 10))
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return params, nil
}

// futuresAnalytics requests an analytics endpoint, returning the errors a successful response reports as an APIError
// and logging its warnings, as Spot REST does with the same severities
func (e *Exchange) futuresAnalytics(ctx context.Context, path string) (*FuturesAnalyticsResponse, error) {
	var resp futuresAnalyticsEnvelope
	if err := e.SendFuturesHTTPRequest(ctx, exchange.RestFuturesSupplementary, path, &resp); err != nil {
		return nil, err
	}
	var apiErr *APIError
	for i := range resp.Errors {
		msg := resp.Errors[i].message()
		if resp.Errors[i].Severity == "W" {
			log.Warnf(log.ExchangeSys, "%s REST request warning: %s", e.Name, msg)
			continue
		}
		if apiErr == nil {
			apiErr = new(APIError)
		}
		apiErr.Errors = append(apiErr.Errors, msg)
	}
	if apiErr != nil {
		return nil, apiErr
	}
	if resp.Result == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Result, nil
}
