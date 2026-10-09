package kraken

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// spotIntervals are the candle intervals Get OHLC Data serves, and the websocket ohlc channel streams
var spotIntervals = []kline.Interval{kline.OneMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.FourHour, kline.OneDay, kline.OneWeek, kline.FifteenDay}

// GetCurrentServerTime calls Get Server Time
func (e *Exchange) GetCurrentServerTime(ctx context.Context) (*ServerTimeResponse, error) {
	var resp *ServerTimeResponse
	return resp, e.SendHTTPRequest(ctx, "/0/public/Time", &resp)
}

// GetSystemStatus calls Get System Status, returning the trading engine's status with any maintenance due within 72
// hours and any unresolved incident
func (e *Exchange) GetSystemStatus(ctx context.Context) (*SystemStatusResponse, error) {
	var resp *SystemStatusResponse
	return resp, e.SendHTTPRequest(ctx, "/0/public/SystemStatus", &resp)
}

// GetMaintenanceSchedule calls Get Maintenance Schedule, returning the maintenance scheduled in the next 7 days.
// Incidents are not scheduled, so only GetSystemStatus reports them
func (e *Exchange) GetMaintenanceSchedule(ctx context.Context) (*MaintenanceScheduleResponse, error) {
	var resp *MaintenanceScheduleResponse
	return resp, e.SendHTTPRequest(ctx, "/0/public/MaintenanceSchedule", &resp)
}

// GetAssets calls Get Asset Info, returning assets keyed by internal name, or by display name with DisplayNames set. A
// nil request returns every asset
func (e *Exchange) GetAssets(ctx context.Context, req *AssetsRequest) (map[string]AssetInfo, error) {
	params := url.Values{}
	if req != nil {
		if len(req.Assets) != 0 {
			codes := make([]string, len(req.Assets))
			for i := range req.Assets {
				if req.Assets[i].IsEmpty() {
					return nil, currency.ErrCurrencyCodeEmpty
				}
				codes[i] = req.Assets[i].Upper().String()
			}
			params.Set("asset", strings.Join(codes, ","))
		}
		if req.AssetClass != "" {
			params.Set("aclass", req.AssetClass)
		}
		if req.DisplayNames {
			params.Set("assetVersion", "1")
		}
	}
	var resp map[string]AssetInfo
	return resp, e.SendHTTPRequest(ctx, common.EncodeURLValues("/0/public/Assets", params), &resp)
}

// GetAssetPairs calls Get Tradable Asset Pairs, returning pairs keyed by internal name, or by display name with
// DisplayNames set. A nil request returns every pair's details
func (e *Exchange) GetAssetPairs(ctx context.Context, req *AssetPairsRequest) (map[string]AssetPair, error) {
	params := url.Values{}
	if req != nil {
		if len(req.Pairs) != 0 {
			pairs, err := e.formatSpotPairs(req.Pairs)
			if err != nil {
				return nil, err
			}
			params.Set("pair", pairs)
		}
		if req.BaseAssetClass != "" {
			params.Set("aclass_base", req.BaseAssetClass)
		}
		if req.Info != "" {
			params.Set("info", req.Info)
		}
		if req.CountryCode != "" {
			params.Set("country_code", req.CountryCode)
		}
		if len(req.ExecutionVenues) != 0 {
			params.Set("execution_venue", strings.Join(req.ExecutionVenues, ","))
		}
		if req.DisplayNames {
			params.Set("assetVersion", "1")
		}
	}
	var resp map[string]AssetPair
	return resp, e.SendHTTPRequest(ctx, common.EncodeURLValues("/0/public/AssetPairs", params), &resp)
}

// GetTickerInformation calls Get Ticker Information, returning tickers keyed by internal pair name, or by display name
// with DisplayNames set. A nil request returns every tradable pair's ticker
func (e *Exchange) GetTickerInformation(ctx context.Context, req *TickerInformationRequest) (map[string]TickerInformation, error) {
	params := url.Values{}
	if req != nil {
		if len(req.Pairs) != 0 {
			pairs, err := e.formatSpotPairs(req.Pairs)
			if err != nil {
				return nil, err
			}
			params.Set("pair", pairs)
		}
		if req.AssetClass != "" {
			params.Set("asset_class", req.AssetClass)
		}
		if req.DisplayNames {
			params.Set("assetVersion", "1")
		}
	}
	var resp map[string]TickerInformation
	return resp, e.SendHTTPRequest(ctx, common.EncodeURLValues("/0/public/Ticker", params), &resp)
}

// GetOHLCData calls Get OHLC Data, returning up to the 720 most recent candles of an interval
func (e *Exchange) GetOHLCData(ctx context.Context, req *OHLCDataRequest) (*OHLCDataResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := e.spotPairParams(req.Pair, req.AssetClass, req.DisplayNames)
	if err != nil {
		return nil, err
	}
	if req.Interval != 0 {
		if !slices.Contains(spotIntervals, req.Interval) {
			return nil, fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, req.Interval)
		}
		params.Set("interval", strconv.FormatFloat(req.Interval.Duration().Minutes(), 'f', -1, 64))
	}
	if !req.Since.IsZero() {
		params.Set("since", strconv.FormatInt(req.Since.Unix(), 10))
	}
	var resp *OHLCDataResponse
	return resp, e.SendHTTPRequest(ctx, common.EncodeURLValues("/0/public/OHLC", params), &resp)
}

// GetOrderBook calls Get Order Book, returning a pair's level 2 order book keyed by pair name
func (e *Exchange) GetOrderBook(ctx context.Context, req *OrderBookRequest) (map[string]OrderBook, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Count > 500 {
		return nil, fmt.Errorf("%w: %d exceeds 500", errInvalidCount, req.Count)
	}
	params, err := e.spotPairParams(req.Pair, req.AssetClass, req.DisplayNames)
	if err != nil {
		return nil, err
	}
	if req.Count != 0 {
		params.Set("count", strconv.FormatUint(req.Count, 10))
	}
	var resp map[string]OrderBook
	return resp, e.SendHTTPRequest(ctx, common.EncodeURLValues("/0/public/Depth", params), &resp)
}

// GetLevel3OrderBook calls Query L3 Order Book, returning each order resting in a pair's book
func (e *Exchange) GetLevel3OrderBook(ctx context.Context, req *Level3OrderBookRequest) (*Level3OrderBookResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.FullBook && req.Depth != 0 {
		return nil, fmt.Errorf("%w: depth cannot be set with the full book", errInvalidDepth)
	}
	if req.Depth != 0 && !slices.Contains([]uint64{10, 25, 100, 250, 1000}, req.Depth) {
		return nil, fmt.Errorf("%w: %d", errInvalidDepth, req.Depth)
	}
	pair, err := e.FormatSymbol(req.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"pair": pair}
	switch {
	case req.FullBook:
		body["depth"] = 0
	case req.Depth != 0:
		body["depth"] = req.Depth
	}
	var resp *Level3OrderBookResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/Level3", nil, body, &resp)
}

// GetGroupedOrderBook calls Get Grouped Order Book, returning a pair's order book aggregated over several ticks a level
func (e *Exchange) GetGroupedOrderBook(ctx context.Context, req *GroupedOrderBookRequest) (*GroupedOrderBookResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Depth != 0 && !slices.Contains([]uint64{10, 25, 100, 250, 1000}, req.Depth) {
		return nil, fmt.Errorf("%w: %d", errInvalidDepth, req.Depth)
	}
	if req.Grouping != 0 && !slices.Contains([]uint64{1, 5, 10, 25, 50, 100, 250, 500, 1000}, req.Grouping) {
		return nil, fmt.Errorf("%w: %d", errInvalidGrouping, req.Grouping)
	}
	pair, err := e.FormatSymbol(req.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("pair", pair)
	if req.Depth != 0 {
		params.Set("depth", strconv.FormatUint(req.Depth, 10))
	}
	if req.Grouping != 0 {
		params.Set("grouping", strconv.FormatUint(req.Grouping, 10))
	}
	var resp *GroupedOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, common.EncodeURLValues("/0/public/GroupedBook", params), &resp)
}

// GetTrades calls Get Recent Trades, returning up to 1000 of a pair's trades
func (e *Exchange) GetTrades(ctx context.Context, req *RecentTradesRequest) (*RecentTradesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Count > 1000 {
		return nil, fmt.Errorf("%w: %d exceeds 1000", errInvalidCount, req.Count)
	}
	params, err := e.spotPairParams(req.Pair, req.AssetClass, req.DisplayNames)
	if err != nil {
		return nil, err
	}
	if !req.Since.IsZero() {
		// Last is a nanosecond timestamp, which Since takes as well as seconds
		params.Set("since", strconv.FormatInt(req.Since.UnixNano(), 10))
	}
	if req.Count != 0 {
		params.Set("count", strconv.FormatUint(req.Count, 10))
	}
	var resp *RecentTradesResponse
	return resp, e.SendHTTPRequest(ctx, common.EncodeURLValues("/0/public/Trades", params), &resp)
}

// GetRecentSpreads calls Get Recent Spreads, returning a pair's last ~200 top of book spreads
func (e *Exchange) GetRecentSpreads(ctx context.Context, req *RecentSpreadsRequest) (*RecentSpreadsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := e.spotPairParams(req.Pair, req.AssetClass, req.DisplayNames)
	if err != nil {
		return nil, err
	}
	if !req.Since.IsZero() {
		params.Set("since", strconv.FormatInt(req.Since.Unix(), 10))
	}
	var resp *RecentSpreadsResponse
	return resp, e.SendHTTPRequest(ctx, common.EncodeURLValues("/0/public/Spread", params), &resp)
}

// spotPairParams returns the parameters every single pair market data endpoint shares
func (e *Exchange) spotPairParams(pair currency.Pair, assetClass string, displayNames bool) (url.Values, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbol, err := e.FormatSymbol(pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("pair", symbol)
	if assetClass != "" {
		params.Set("asset_class", assetClass)
	}
	if displayNames {
		params.Set("assetVersion", "1")
	}
	return params, nil
}

// formatSpotPairs returns pairs as the comma separated list Kraken takes, each formatted as its alternative name, such
// as XBTUSD
func (e *Exchange) formatSpotPairs(pairs currency.Pairs) (string, error) {
	symbols := make([]string, len(pairs))
	for i := range pairs {
		if pairs[i].IsEmpty() {
			return "", currency.ErrCurrencyPairEmpty
		}
		symbol, err := e.FormatSymbol(pairs[i], asset.Spot)
		if err != nil {
			return "", err
		}
		symbols[i] = symbol
	}
	return strings.Join(symbols, ","), nil
}
